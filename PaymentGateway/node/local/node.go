package local

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/local/paymentregestry"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database"
	"paidpiper.com/payment-gateway/regestry"
	"paidpiper.com/payment-gateway/root"

	"github.com/go-errors/errors"
	"github.com/rs/xid"
	"github.com/stellar/go/support/log"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
)

const nodeTransactionFee = 10

type LocalPPNode interface {
	node.PPNode
	GetStellarAddress() *models.GetStellarAddressResponse
	NewPaymentRequest(ctx context.Context, request *models.CreatePaymentInfo) (*models.PaymentRequest, error)
	ValidatePayment(ctx context.Context, request *models.ValidatePaymentRequest) (*models.ValidatePaymentResponse, error)
	GetTransactions() []*models.PaymentTransaction
	GetTransaction(sessionId string) *models.PaymentTransaction
	FlushTransactions(context context.Context) error
	ProcessResponse(ctx context.Context, response *models.UtilityResponse) error
	CommandHandler(ctx context.Context, cmd *models.UtilityCommand) (models.OutCommandType, error)
	SetTransactionValiditySecs(transactionValiditySecs int64)
	SetAutoFlush(autoFlush time.Duration)
	//CLIENT PORPS
	ProcessPayment(ctx context.Context, request *models.ProcessPaymentRequest) (*models.ProcessPaymentAccepted, error)
	ProcessCommand(ctx context.Context, command *models.UtilityCommand) (models.OutCommandType, error)
	// Additional
	GetBookHistory(commodity string, bins int, hours int) (*models.BookHistoryResponse, error)
	GetBookBalance() (*models.BookBalanceResponse, error)
}

type nodeImpl struct {
	db                           database.Db
	rootClient                   root.RootApi
	accumulatingTransactionsMode bool
	transactionFee               models.TransactionAmount
	paymentRegistry              paymentregestry.PaymentRegistry
	paymentManagerRegestry       regestry.PaymentManagerRegestry
	commodityManager             commodity.Manager
	tracer                       trace.Tracer
	autoFlushPeriod              *time.Ticker
	flushMux                     sync.Mutex
	asyncMode                    bool
	callbackerFactory            CallbackerFactory
}

func New(rootClient root.RootApi,
	paymentManager regestry.PaymentManagerRegestry,
	callbackerFactory CallbackerFactory,
	nodeConfig config.NodeConfig,
) (LocalPPNode, error) {

	log.SetLevel(log.InfoLevel)
	db, err := database.NewLiteDB()
	if err != nil {
		return nil, err
	}
	paymentRegestry, err := paymentregestry.NewWithDB(db)
	if err != nil {
		return nil, err
	}
	node := &nodeImpl{
		db:                           db,
		rootClient:                   rootClient,
		transactionFee:               nodeTransactionFee,
		paymentRegistry:              paymentRegestry,
		commodityManager:             commodity.New(),
		paymentManagerRegestry:       paymentManager,
		tracer:                       common.CreateTracer("node"),
		callbackerFactory:            callbackerFactory,
		flushMux:                     sync.Mutex{}, //TODO MOVE TO PAYMENT REGESTRY
		accumulatingTransactionsMode: nodeConfig.AccumulateTransactions,
		asyncMode:                    nodeConfig.AsyncMode,
	}
	node.runTicker(nodeConfig.AutoFlushPeriod)
	return node, nil
}

func (n *nodeImpl) runTicker(autoFlushPeriod time.Duration) {
	if n.autoFlushPeriod != nil {
		if autoFlushPeriod == 0 {
			n.autoFlushPeriod.Stop()
		} else {
			n.autoFlushPeriod.Reset(autoFlushPeriod)
		}
		return
	}
	if autoFlushPeriod > 0 {
		n.autoFlushPeriod = time.NewTicker(autoFlushPeriod)
		go func(n *nodeImpl) {
			address := n.GetAddress()
			for now := range n.autoFlushPeriod.C {
				log.Debugf("Node %s autoflush tick: %s", address, now.String())
				err := n.FlushTransactions(context.Background())
				if err != nil {
					log.Errorf("Error during autoflush of node %s: %s", address, err.Error())
				}
			}
		}(n)

	} else {
		log.Debugf("node %s: autoflush disabled.", n.GetAddress())
	}

}

func (n *nodeImpl) GetStellarAddress() *models.GetStellarAddressResponse {
	return &models.GetStellarAddressResponse{
		Address: n.GetAddress(),
	}
}

func (n *nodeImpl) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	n.accumulatingTransactionsMode = accumulateTransactions
}

func (n *nodeImpl) SetAutoFlush(autoFlush time.Duration) {
	n.runTicker(autoFlush)
}

func (n *nodeImpl) SetTransactionValiditySecs(transactionValiditySecs int64) {
	n.rootClient.SetTransactionValiditySecs(transactionValiditySecs)
}

func (n *nodeImpl) GetFee() uint32 {
	return 0
}

func (n *nodeImpl) NewPaymentRequest(ctx context.Context, request *models.CreatePaymentInfo) (*models.PaymentRequest, error) {
	nodeAddress := n.GetAddress()
	_, span := n.tracer.Start(ctx, "node-CreatePaymentRequest "+nodeAddress)
	defer span.End()
	paymentRequest, err := n.commodityManager.Calculate(request)
	if err != nil {
		return nil, errors.Errorf("invalid commodity")
	}
	sessionId := xid.New().String()
	pr := &models.PaymentRequest{
		Address:          nodeAddress,
		ServiceSessionId: sessionId,
		Amount:           paymentRequest.Amount,
		Asset:            paymentRequest.Asset,
		ServiceRef:       paymentRequest.ServiceRef,
	}
	log.Infof("CreatePaymentRequest: Starting %d  %s/%s ", request.Amount, pr.Asset, pr.ServiceRef)

	n.paymentRegistry.AddServiceUsage(sessionId, pr)
	return pr, nil
}

func (n *nodeImpl) CreateTransaction(context context.Context, command *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {
	fee := command.TotalIn - command.TotalOut
	return n.createTransactionWithFee(context, fee, command)
}

func (n *nodeImpl) ValidatePayment(ctx context.Context, request *models.ValidatePaymentRequest) (*models.ValidatePaymentResponse, error) {
	return n.commodityManager.ReverseCalculate(request.ServiceType, request.CommodityType, request.PaymentRequest.Amount, request.PaymentRequest.Asset)

}

func (n *nodeImpl) GetAddress() string {
	return n.rootClient.GetAddress()
}

func (n *nodeImpl) GetAccount() (*horizon.Account, error) {
	return n.rootClient.GetAccount()
}

func (n *nodeImpl) createTransactionWrapper(internalTransaction *models.PaymentTransaction) (*models.PaymentTransactionReplacing, error) {
	refTransaction := n.paymentRegistry.GetActiveTransaction(internalTransaction.PaymentSourceAddress)
	return n.createReferenceTransaction(internalTransaction, refTransaction)
}

func (n *nodeImpl) createReferenceTransaction(pt *models.PaymentTransaction, ref *models.PaymentTransaction) (*models.PaymentTransactionReplacing, error) {
	if ref != nil && !ref.XDR.Empty() {

		if pt.PaymentDestinationAddress != ref.PaymentDestinationAddress {
			log.Error("Error creating accumulating transactions, two transactions have different destination addresses")
			return nil, errors.Errorf("error creating accumulating transactions, two transaction have different destination addresses")
		}

		if pt.PaymentSourceAddress != ref.PaymentSourceAddress {
			log.Error("Error creating accumulating transactions, two transactions have different source addresses")
			return nil, errors.Errorf("error creating accumulating transactions, two transactions have different source addresses")
		}
		pt.AmountOut = ref.AmountOut + pt.AmountOut
		pt.ReferenceAmountIn = ref.ReferenceAmountIn + pt.ReferenceAmountIn
	}

	return &models.PaymentTransactionReplacing{
		PendingTransaction:   *pt,
		ReferenceTransaction: ref,
	}, nil

}

func (n *nodeImpl) createTransactionWithFee(context context.Context, fee uint32, request *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {

	_, span := n.tracer.Start(context, "node-CreateTransaction "+n.GetAddress())
	defer span.End()

	log.Infof("CreateTransaction: Starting %s %d + %d = %d => %s ", request.SourceAddress, request.TotalIn, fee, request.TotalOut, n.GetAddress())

	//Verify fee
	if request.TotalIn-request.TotalOut != fee {
		return nil, errors.Errorf("Incorrect fee requested: %d != %d", request.TotalIn-request.TotalOut, fee)
	}

	span.SetAttributes(core.KeyValue{Key: "payment.source-address", Value: core.String(request.SourceAddress)})
	span.SetAttributes(core.KeyValue{Key: "payment.destination-address", Value: core.String(n.GetAddress())})
	span.SetAttributes(core.KeyValue{Key: "payment.amount-in", Value: core.Uint32(request.TotalIn)})
	span.SetAttributes(core.KeyValue{Key: "payment.amount-out", Value: core.Uint32(request.TotalOut)})
	pt := &models.PaymentTransaction{
		TransactionSourceAddress:  n.GetAddress(),
		ReferenceAmountIn:         request.TotalIn,
		AmountOut:                 request.TotalOut,
		PaymentSourceAddress:      request.SourceAddress,
		PaymentDestinationAddress: n.GetAddress(), //TODO CHECK IN MASTER
		ServiceSessionId:          request.ServiceSessionId,
	}
	tr, err := n.createTransactionWrapper(pt)
	if err != nil {
		//log.Fatal("Error creating transaction wrapper: " + err.Error())
		return nil, errors.Errorf("Error creating transaction wrapper: %v", err)
	}
	tr, err = n.rootClient.CreateTransaction(request, tr)
	if err != nil {
		return nil, err
	}
	tr.ToSpanAttributes(span, "create")
	return &models.CreateTransactionResponse{
		Transaction: tr,
	}, nil

}

func (n *nodeImpl) SignServiceTransaction(context context.Context, command *models.SignServiceTransactionCommand) (*models.SignServiceTransactionResponse, error) {
	_, span := n.tracer.Start(context, "node-SignServiceTransaction "+n.GetAddress())
	defer span.End()

	creditTransactionPayload := command.Transaction

	log.Infof("SignServiceTransaction: starting %s => %s ",
		creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransaction := creditTransactionPayload.PendingTransaction

	// Validate
	if creditTransaction.PaymentDestinationAddress != n.GetAddress() {
		destinationAddress := creditTransaction.PaymentDestinationAddress
		return nil, fmt.Errorf("transaction destination is incorrect: %s", destinationAddress)
	}

	signedCreditTransaction, err := n.rootClient.SignPaymentTransaction(&creditTransaction)

	if err != nil {
		return nil, errors.Errorf("Error writing transaction envelope: %v", err)
	}

	creditTransactionPayload.PendingTransaction = *signedCreditTransaction
	if err != nil {
		return nil, errors.Errorf("Error UpdateTransactionXDR %v", err)
	}
	log.Infof("SignServiceTransaction: done %s => %s ",
		creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransactionPayload.ToSpanAttributes(span, "credit")

	return &models.SignServiceTransactionResponse{
		Transaction: creditTransactionPayload,
	}, nil
}

func (n *nodeImpl) SignChainTransaction(context context.Context,
	command *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error) {

	_, span := n.tracer.Start(context, "node-SignChainTransaction "+n.GetAddress())
	defer span.End()
	credit := command.Credit
	debit := command.Debit
	creditTransaction := credit.PendingTransaction

	log.Infof("SignChainTransaction: started %s => %s ", creditTransaction.PaymentSourceAddress,
		creditTransaction.PaymentDestinationAddress)

	signedCreditTransaction, err := n.rootClient.SignPaymentTransaction(&creditTransaction)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return nil, err
	}
	credit.PendingTransaction = *signedCreditTransaction

	signedDebitTransaction, err := n.rootClient.SignPaymentTransaction(&debit.PendingTransaction)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return nil, err
	}

	debit.PendingTransaction = *signedDebitTransaction

	log.Infof("SignChainTransaction: done %s => %s ", credit.PendingTransaction.PaymentSourceAddress,
		credit.PendingTransaction.PaymentDestinationAddress)

	credit.ToSpanAttributes(span, "credit")
	debit.ToSpanAttributes(span, "debit")
	return &models.SignChainTransactionResponse{
		Credit: credit,
		Debit:  debit,
	}, nil
}

func (n *nodeImpl) commitTransaction(context context.Context, transaction *models.PaymentTransaction) error {

	log.Infof("CommitChainTransaction started %s => %s", transaction.PaymentSourceAddress,
		transaction.PaymentDestinationAddress)

	err := n.rootClient.VerifyTransaction(context, transaction)
	if err != nil {
		return fmt.Errorf("verify transaction error: %v", err)
	}
	if !n.accumulatingTransactionsMode {
		err := n.rootClient.SubmitTransaction(transaction)

		if err != nil {
			log.Error("Error submitting transaction: " + err.Error())
			return err
		}
		log.Infof("CommitChainTransaction finished %s => %s", transaction.PaymentSourceAddress,
			transaction.PaymentDestinationAddress)

		return nil
	}
	sequence, err := n.rootClient.GetTransactionSequenceNumber(transaction)
	if err != nil {
		return fmt.Errorf("GetTransactionSequenceNumber error : %v", err)
	}
	n.paymentRegistry.SaveTransaction(sequence, transaction)
	log.Infof("CommitChainTransaction finished %s => %s", transaction.PaymentSourceAddress,
		transaction.PaymentDestinationAddress)

	return nil

}

func (n *nodeImpl) CommitChainTransaction(context context.Context, command *models.CommitChainTransactionCommand) error {

	_, span := n.tracer.Start(context, "node-CommitChainTransaction "+n.GetAddress())
	defer span.End()
	err := n.commitTransaction(context, &command.Transaction.PendingTransaction)
	if err != nil {
		return err
	}
	command.Transaction.ToSpanAttributes(span, "single")
	return err
}

func (n *nodeImpl) CommitServiceTransaction(context context.Context, command *models.CommitServiceTransactionCommand) error {

	_, span := n.tracer.Start(context, "node-CommitServiceTransaction "+n.GetAddress())
	defer span.End()
	transactionWrapper := command.Transaction
	transaction := transactionWrapper.PendingTransaction
	paymentRequest := command.PaymentRequest
	log.Infof("CommitServiceTransaction started %s => %s", transaction.PaymentSourceAddress,
		transaction.PaymentDestinationAddress)

	err := n.commitTransaction(context, &transaction)

	if err != nil {
		return err
	}
	command.Transaction.ToSpanAttributes(span, "single")
	err = n.paymentRegistry.ReducePendingAmount(paymentRequest.ServiceSessionId, transaction.AmountOut)
	if err != nil {
		return err
	}
	log.Infof("CommitServiceTransaction finished %s => %s", transaction.PaymentSourceAddress,
		transaction.PaymentDestinationAddress)

	return nil
}

func (n *nodeImpl) GetTransactions() []*models.PaymentTransaction {
	trs := []*models.PaymentTransaction{}
	for _, item := range n.paymentRegistry.GetActiveTransactions() {
		trs = append(trs, &item.PaymentTransaction)
	}
	return trs
}

func (n *nodeImpl) GetTransaction(sessionId string) *models.PaymentTransaction {
	return n.paymentRegistry.GetTransactionBySessionId(sessionId)
}

func (n *nodeImpl) FlushTransactions(context context.Context) error {

	_, span := n.tracer.Start(context, "node-FlushTransactions "+n.GetAddress())
	defer span.End()

	log.Infof("FlushTransactions started")

	//TODO Sort transaction by sequence number and make sure to submit them only in sequence number order
	transactions := n.paymentRegistry.GetActiveTransactions()

	if len(transactions) == 0 {
		log.Info("FlushTransactions: No transactions to flush.")
		return nil
	}

	n.flushMux.Lock()
	defer n.flushMux.Unlock()

	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].Sequence < transactions[j].Sequence
	})
	//
	transactions, err := n.rootClient.RemoveTransactionsIfSequence(transactions)
	if err != nil {
		return err
	}
	if len(transactions) > 0 {

		err := n.rootClient.BumpSequenceIfNeed(transactions[0])
		if err != nil {
			return err
		}
		for _, t := range transactions {
			log.Infof("Submitting transaction for session %s", t.ServiceSessionId)
			err := n.rootClient.SubmitTransactionXDR(t.XDR)
			if err != nil {
				log.Errorf("Error in submit transaction (%v): %s", err, t.XDR)

				break
			}
			//TODO: Make the transaction removal more intellegent
			n.paymentRegistry.CompletePayment(t.PaymentSourceAddress, t.ServiceSessionId)
			//processedTransactions = append(processedTransactions, &t.PaymentTransaction)
		}
	}

	return nil
}

func (n *nodeImpl) ProcessResponse(ctx context.Context, response *models.UtilityResponse) error {
	paymentManager := n.paymentManagerRegestry.Get(response.SessionId)
	if paymentManager == nil {
		return fmt.Errorf("session unknown")
	}
	return paymentManager.ProcessResponse(ctx, response.NodeId, response.CommandId, response.CommandResponse)

}

func (n *nodeImpl) ProcessPayment(ctx context.Context, request *models.ProcessPaymentRequest) (*models.ProcessPaymentAccepted, error) {
	sessionId := request.PaymentRequest.ServiceSessionId
	if n.paymentManagerRegestry.Has(sessionId) {
		return nil, fmt.Errorf("duplicate session id")
	}
	paymentManager, err := n.paymentManagerRegestry.New(ctx, n, request)
	if err != nil {
		return nil, err
	}
	n.paymentManagerRegestry.Set(sessionId, paymentManager)

	err = paymentManager.Run(ctx, n.asyncMode)
	if n.asyncMode {
		return &models.ProcessPaymentAccepted{
			SessionId: sessionId,
		}, nil
	} else {
		return nil, err
	}

}

func (u *nodeImpl) CommandHandler(ctx context.Context, cmd *models.UtilityCommand) (models.OutCommandType, error) {

	switch body := cmd.CommandBody.(type) {
	case *models.CreateTransactionCommand:
		return u.CreateTransaction(ctx, body)

	case *models.SignServiceTransactionCommand:
		return u.SignServiceTransaction(ctx, body)

	case *models.SignChainTransactionCommand:
		return u.SignChainTransaction(ctx, body)

	case *models.CommitChainTransactionCommand:
		err := u.CommitChainTransaction(ctx, body)
		if err != nil {
			return nil, err
		}
		return &models.CommitServiceTransactionResponse{}, nil

	case *models.CommitServiceTransactionCommand:
		err := u.CommitServiceTransaction(ctx, body)
		if err != nil {
			return nil, err
		}
		return &models.CommitServiceTransactionResponse{}, nil
	default:
		return nil, fmt.Errorf("unknow command type: %v", body)
	}

}

func (u *nodeImpl) ProcessCommand(ctx context.Context, command *models.UtilityCommand) (models.OutCommandType, error) {
	if command.CallbackUrl != "" {
		callbacker := u.callbackerFactory(command)
		go func(callbacker CallBacker) {
			reply, err := u.CommandHandler(ctx, command)
			if err != nil {
				log.Fatalf("CommandHandler error: %v", err)
				return
			}
			err = callbacker.call(reply, err)
			if err != nil {
				log.Fatalf("Callback error: %v", err)
				return
			}
		}(callbacker)
		return nil, nil
	}
	reply, err := u.CommandHandler(ctx, command)

	if err != nil {
		return nil, fmt.Errorf("command submitted")
	} else {
		return reply, nil
	}

}

func (n *nodeImpl) GetBookHistory(commodity string, bins int, hours int) (*models.BookHistoryResponse, error) {
	err := n.db.Open()
	if err != nil {
		return nil, err
	}
	defer n.db.Close()
	stepDuration := time.Duration(hours) * time.Hour
	till := time.Now().Truncate(stepDuration).Add(stepDuration)
	from := till.Add(-stepDuration * time.Duration(bins))
	groups, err := n.db.SelectPaymentRequestGroup(commodity, stepDuration, from)
	if err != nil {
		return nil, err
	}
	return &models.BookHistoryResponse{
		Items: groups,
	}, nil
}

func (n *nodeImpl) GetActiveTransactionsAmount() (amount models.TransactionAmount) {

	for _, t := range n.paymentRegistry.GetActiveTransactions() {
		if err := n.rootClient.ValidateTimebounds(&t.PaymentTransaction); err == nil {
			amount += t.AmountOut
		}
	}
	return amount
}
func (n *nodeImpl) GetBookBalance() (*models.BookBalanceResponse, error) {
	amount := n.GetActiveTransactionsAmount()

	balance, err := n.rootClient.GetPPTokenBalance()
	if err != nil {
		return nil, err
	}
	timeStamp := time.Now()
	return &models.BookBalanceResponse{
		Balance:   models.PPtoken2MicroPP(amount) + balance,
		Timestamp: models.JsonTime(timeStamp),
	}, nil
}
