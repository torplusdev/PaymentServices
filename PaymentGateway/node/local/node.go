package local

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stellar/go/protocols/horizon"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/local/paymentregestry"
	"paidpiper.com/payment-gateway/regestry"
	"paidpiper.com/payment-gateway/root"

	"github.com/go-errors/errors"
	"github.com/rs/xid"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
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
	ProcessCommand(ctx context.Context, command *models.UtilityCommand) (int, error)
}

type nodeImpl struct {
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

	node := &nodeImpl{
		rootClient:                   rootClient,
		transactionFee:               nodeTransactionFee,
		paymentRegistry:              paymentregestry.New(),
		commodityManager:             commodity.New(), //TODO REMOVE
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
	_, span := n.tracer.Start(ctx, "node-CreatePaymentRequest "+n.GetAddress())
	defer span.End()
	paymentRequest, err := n.commodityManager.Calculate(request)
	if err != nil {
		return nil, errors.Errorf("invalid commodity")
	}
	sessionId := xid.New().String()
	pr := &models.PaymentRequest{
		Address:          n.GetAddress(),
		ServiceSessionId: sessionId,
		Amount:           request.Amount,
		Asset:            paymentRequest.Asset,
		ServiceRef:       paymentRequest.ServiceRef,
	}
	return n.registerPaymentRequest(ctx, pr)
}
func (n *nodeImpl) registerPaymentRequest(ctx context.Context, request *models.PaymentRequest) (*models.PaymentRequest, error) {

	log.Infof("CreatePaymentRequest: Starting %d  %s/%s ", request.Amount, request.Asset, request.ServiceRef)

	n.paymentRegistry.AddServiceUsage(request.ServiceSessionId, request.Amount)
	return request, nil
}

func (n *nodeImpl) CreateTransaction(context context.Context, command *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {
	fee := command.TotalIn - command.TotalOut
	return n.createTransactionWithFee(context, fee, command)
}

func (n *nodeImpl) ValidatePayment(ctx context.Context, request *models.ValidatePaymentRequest) (*models.ValidatePaymentResponse, error) {
	return n.commodityManager.ReverseCalculate(request.ServiceType, request.CommodityType, request.PaymentRequest.Amount, request.PaymentRequest.Asset)

}

//func (n *nodeImpl) GetPendingPayment(address string) (models.TransactionAmount, time.Time, error) {
//
//	if n.pendingPayment[address].updated.IsZero() {
//		return 0, time.Unix(0, 0), errors.Errorf("PaymentDestinationAddress not found: " + address)
//	}
//
//	return n.pendingPayment[address].amount, n.pendingPayment[address].updated, nil
//}

func (n *nodeImpl) GetAddress() string {
	return n.rootClient.GetAddress()
}
func (n *nodeImpl) GetAccount() (*horizon.Account, error) {
	return n.rootClient.GetAccount()
}

func (n *nodeImpl) createTransactionWrapper(internalTransaction *models.PaymentTransaction) (*models.PaymentTransactionReplacing, error) {
	refTransaction := n.paymentRegistry.GetActiveTransaction(internalTransaction.PaymentSourceAddress)
	return n.CreateReferenceTransaction(internalTransaction, refTransaction)
}

func (n *nodeImpl) CreateReferenceTransaction(pt *models.PaymentTransaction, ref *models.PaymentTransaction) (*models.PaymentTransactionReplacing, error) {
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
	//TODO CHECK

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

	tr, err := n.createTransactionWrapper(&models.PaymentTransaction{
		TransactionSourceAddress:  n.GetAddress(),
		ReferenceAmountIn:         request.TotalIn,
		AmountOut:                 request.TotalOut,
		PaymentSourceAddress:      request.SourceAddress,
		PaymentDestinationAddress: n.GetAddress(),
		ServiceSessionId:          request.ServiceSessionId,
	})
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

	creditTransactionPayload := command.Transaction
	_, span := n.tracer.Start(context, "node-SignServiceTransaction "+n.GetAddress())
	defer span.End()

	log.Infof("SignServiceTransaction: starting %s => %s ", creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransaction := creditTransactionPayload.PendingTransaction

	// Validate
	if creditTransaction.PaymentDestinationAddress != n.GetAddress() {
		return nil, errors.Errorf("Transaction destination is incorrect: %s", creditTransaction.PaymentDestinationAddress)
	}

	transactionWrapper, err := creditTransaction.XDR.TransactionFromXDR()

	if err != nil {
		return nil, errors.Errorf("Error parsing transaction: %v", err)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return nil, errors.Errorf("Transaction destination is incorrect (GenericTransaction)")
	}

	t, err = n.rootClient.Sign(t)

	if err != nil {
		return nil, errors.Errorf("Failed to signed transaction: %v", err)
	}
	str, err := t.Base64()
	if err != nil {
		return nil, err
	}
	creditTransaction.XDR = models.NewXDR(str)

	if err != nil {
		return nil, errors.Errorf("Error writing transaction envelope: %v", err)
	}
	creditTransactionPayload.PendingTransaction.XDR = creditTransaction.XDR
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
	creditTransactionPayload := command.Credit
	debitTransactionPayload := command.Debit
	log.Infof("SignChainTransaction: started %s => %s ", creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransaction := creditTransactionPayload.PendingTransaction
	debitTransaction := debitTransactionPayload.PendingTransaction

	creditWrapper, err := creditTransaction.XDR.TransactionFromXDR()

	if err != nil {
		return nil, errors.Errorf("Error building transaction from XDR: %v", err)
	}

	credit, result := creditWrapper.Transaction()

	if !result {
		return nil, errors.Errorf("Error deserializing transaction (GenericTransaction)")
	}

	if err != nil {
		return nil, errors.Errorf("Error parsing credit transaction: %v", err)
	}

	debitWrapper, err := debitTransaction.XDR.TransactionFromXDR()

	debit, _ := debitWrapper.Transaction()

	if err != nil {
		return nil, errors.Errorf("Error parsing debit transaction: %v", err)
	}

	credit, err = n.rootClient.Sign(credit)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return nil, err
	}

	debit, err = n.rootClient.Sign(debit)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return nil, err
	}

	str, err := credit.Base64()
	if err != nil {
		return nil, err
	}
	creditTransaction.XDR = models.NewXDR(str)
	if err != nil {
		log.Fatal("Error writing credit transaction envelope: " + err.Error())
		return nil, err
	}

	creditTransactionPayload.PendingTransaction.XDR = creditTransaction.XDR

	str, err = debit.Base64()
	if err != nil {
		return nil, err
	}
	debitTransaction.XDR = models.NewXDR(str)
	if err != nil {
		log.Fatal("Error writing debit transaction envelope: " + err.Error())
		return nil, err
	}

	debitTransactionPayload.PendingTransaction.XDR = debitTransaction.XDR

	log.Infof("SignChainTransaction: done %s => %s ", creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransactionPayload.ToSpanAttributes(span, "credit")
	debitTransactionPayload.ToSpanAttributes(span, "debit")
	return &models.SignChainTransactionResponse{
		Credit: creditTransactionPayload,
		Debit:  debitTransactionPayload,
	}, nil
}

func (n *nodeImpl) verifyTransactionSequence(context context.Context, transactionPayload *models.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-verifyTransactionSequence"+n.GetAddress())
	defer span.End()

	transaction := transactionPayload.PendingTransaction

	// Deserialize transactions
	transactionWrapper, e := transaction.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: %v", e)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	nodeAccount, err := n.GetAccount()
	if err != nil {
		return errors.Errorf("error reading account: %v", err)
	}

	currentSequence, err := nodeAccount.GetSequenceNumber()
	if err != nil {
		return fmt.Errorf("error getting sequence: %v", err)
	}

	account := t.SourceAccount()
	transactionSequence, err := account.GetSequenceNumber()
	if err != nil {
		return err
	}
	if transactionSequence <= currentSequence {
		log.Warnf("Incorrect sequence detected, current account is at %d, transaction is %d", currentSequence, transactionSequence)
		return errors.Errorf("incorrect sequence detected")
	}

	log.Infof("verifyTransactionSequence finished successfully - account#:%d transaction#:%d", currentSequence, transactionSequence)
	return nil
}

func (n *nodeImpl) verifyTransactionSignatures(context context.Context, transactionPayload *models.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-verifyTransactionSignatures "+n.GetAddress())
	defer span.End()

	log.Infof("verifyTransactionSignatures started %s => %s", transactionPayload.PendingTransaction.PaymentSourceAddress,
		transactionPayload.PendingTransaction.PaymentDestinationAddress)

	transaction := transactionPayload.PendingTransaction

	// Deserialize transactions
	transactionWrapper, e := transaction.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	if t.SourceAccount().AccountID != n.GetAddress() {
		return errors.Errorf("Incorrect transaction source account")
	}
	//transaction.StellarNetworkToken

	var payerAccount string = ""
	for _, op := range t.Operations() {
		xdrOp, _ := op.BuildXDR()

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment := &txnbuild.Payment{}

			err := payment.FromXDR(xdrOp)

			if err != nil {
				return errors.Errorf("Error converting operation")
			}

			payerAccount = payment.SourceAccount
		default:
			return errors.Errorf("Unexpected operation during verification")
		}
	}

	payerVerified := false
	sourceVerified := false

	for _, signature := range t.Signatures() {
		from, err := keypair.ParseAddress(payerAccount)

		if err != nil {
			return errors.Errorf("Error in operation source address")
		}

		bytes, err := t.Hash(transaction.StellarNetworkToken)

		if err != nil {
			return errors.Errorf("Error during tx hashing")
		}

		err = from.Verify(bytes[:], signature.Signature)

		if err == nil {
			payerVerified = true
		}

		err = n.rootClient.Verify(bytes[:], signature.Signature)

		if err == nil {
			sourceVerified = true
		}
	}

	if !payerVerified {
		return errors.Errorf("Error validating payer signature")
	}

	if !sourceVerified {
		return errors.Errorf("Error validating source signature")
	}

	log.Infof("verifyTransactionSequence finished successfully")

	//TODO: Validate timebounds

	return nil
}
func (n *nodeImpl) commitTransaction(context context.Context, tr *models.PaymentTransactionReplacing) error {
	log.Infof("CommitChainTransaction started %s => %s", tr.PendingTransaction.PaymentSourceAddress,
		tr.PendingTransaction.PaymentDestinationAddress)

	transaction := tr.PendingTransaction

	transactionWrapper, err := transaction.XDR.TransactionFromXDR()

	if err != nil {
		return fmt.Errorf("error during transaction deser: %v", err)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return fmt.Errorf("error during transaction")
	}

	err = n.verifyTransactionSequence(context, tr)

	if err != nil {
		log.Warn("Transaction verification failed (sequence)")
		return err
	}

	err = n.verifyTransactionSignatures(context, tr)

	if err != nil {
		log.Warn("Transaction verification failed (signatures)")
		return err
	}

	if !n.accumulatingTransactionsMode {
		_, err := n.rootClient.SubmitTransaction(t)

		if err != nil {
			log.Error("Error submitting transaction: " + err.Error())
			return err
		}

		log.Debug("Transaction submitted: ")
	} else {
		n.paymentRegistry.SaveTransaction(transaction.PaymentSourceAddress, &transaction)
	}

	log.Infof("CommitChainTransaction finished %s => %s", tr.PendingTransaction.PaymentSourceAddress,
		tr.PendingTransaction.PaymentDestinationAddress)

	return nil
}
func (n *nodeImpl) CommitChainTransaction(context context.Context, command *models.CommitChainTransactionCommand) error {

	_, span := n.tracer.Start(context, "node-CommitChainTransaction "+n.GetAddress())
	defer span.End()
	err := n.commitTransaction(context, command.Transaction)
	if err != nil {
		return err
	}
	command.Transaction.ToSpanAttributes(span, "single")
	return err
}

func (n *nodeImpl) CommitServiceTransaction(context context.Context, command *models.CommitServiceTransactionCommand) error {

	_, span := n.tracer.Start(context, "node-CommitServiceTransaction "+n.GetAddress())
	defer span.End()
	transaction := command.Transaction
	log.Infof("CommitServiceTransaction started %s => %s", transaction.PendingTransaction.PaymentSourceAddress,
		transaction.PendingTransaction.PaymentDestinationAddress)

	err := n.commitTransaction(context, command.Transaction)

	if err != nil {
		return err
	}
	command.Transaction.ToSpanAttributes(span, "single")
	err = n.paymentRegistry.ReducePendingAmount(command.PaymentRequest.ServiceSessionId, transaction.PendingTransaction.AmountOut)

	log.Infof("CommitServiceTransaction finished %s => %s", transaction.PendingTransaction.PaymentSourceAddress,
		transaction.PendingTransaction.PaymentDestinationAddress)

	return err
}

func (n *nodeImpl) GetTransactions() []*models.PaymentTransaction {
	return n.paymentRegistry.GetActiveTransactions()
}

func (n *nodeImpl) GetTransaction(sessionId string) *models.PaymentTransaction {
	return n.paymentRegistry.GetTransactionBySessionId(sessionId)
}

// Find takes a slice and looks for an element in it. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func Find(slice []*models.PaymentTransaction, val *models.PaymentTransaction) (int, bool) {
	for i, item := range slice {

		if cmp.Equal(item, val) {
			return i, true
		}
	}
	return -1, false
}

//TODO TO REGESTRY
func (n *nodeImpl) FlushTransactions(context context.Context) error {

	_, span := n.tracer.Start(context, "node-FlushTransactions "+n.GetAddress())
	defer span.End()

	log.Infof("FlushTransactions started")

	resultsMap := make(map[string]interface{})

	//TODO Sort transaction by sequence number and make sure to submit them only in sequence number order
	transactions := n.paymentRegistry.GetActiveTransactions()

	if len(transactions) == 0 {
		log.Info("FlushTransactions: No transactions to flush.")
		return nil
	}

	n.flushMux.Lock()
	defer n.flushMux.Unlock()

	sort.Slice(transactions, func(i, j int) bool {
		transi, erri := n.rootClient.PaymentTransactionToStellar(transactions[i])
		transj, errj := n.rootClient.PaymentTransactionToStellar(transactions[j])
		if erri != nil {
			log.Errorf("Error converting transaction 1: %s", erri.Error())
		}
		if errj != nil {
			log.Errorf("Error converting transaction 2: %s", errj.Error())
		}

		account := transi.SourceAccount()
		seqi, erri := account.GetSequenceNumber()

		account = transj.SourceAccount()
		seqj, errj := account.GetSequenceNumber()

		if erri != nil {
			log.Errorf("Error getting sequence number transaction from xdr: %s", erri.Error())
		}
		if errj != nil {
			log.Errorf("Error converting transaction from xdr: %s", errj.Error())
		}

		return seqi < seqj
	})

	var (
		nodeAccount *horizon.Account
		err         error
	)

	if nodeAccount, err = n.GetAccount(); err != nil {
		return errors.Errorf("Error gettings account details: %v", err)
	}

	firstTransaction, err := n.rootClient.PaymentTransactionToStellar(transactions[0])

	if err != nil {
		return errors.Errorf("Can't get first transaction from wrapper: %s", err.Error())
	}

	// Handle unfulfilled transactions, if needed
	currentSequence, err := nodeAccount.GetSequenceNumber()

	if err != nil {
		return errors.Errorf("Error reading sequence: %v", err)
	}

	transactionToRemove := 0

	// Filter out missed transactions

	if firstTransaction.SourceAccount().Sequence <= currentSequence {
		for _, t := range transactions {

			innerTransaction, err := n.rootClient.PaymentTransactionToStellar(t)

			if err != nil {
				log.Warn("Problematic transaction detected, couldn't convert from XDR - removing.")

				transactionToRemove = transactionToRemove + 1
				continue
			}

			if innerTransaction.SourceAccount().Sequence <= currentSequence {
				log.Warnf("Problematic transaction detected  -bad sequence %d <= %d- removing.", innerTransaction.SourceAccount().Sequence, currentSequence)
				transactionToRemove = transactionToRemove + 1
			}
		}

		if transactionToRemove > 0 {
			log.Warnf("Bad first transactions were detected (%d) and removed.", transactionToRemove)

			transactions = transactions[transactionToRemove:]

			if len(transactions) == 0 {
				log.Warnf("No further transactions to process after removing %d transactions", transactionToRemove)
				return nil
			}
		}
	}

	bumper := func(current int64, bumpTo int64) error {
		tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: n.GetAddress(),
				Sequence:  current,
			},
			IncrementSequenceNum: true,
			BaseFee:              config.StellarImmediateOperationBaseFee,
			Timebounds:           txnbuild.NewTimeout(config.StellarImmediateOperationTimeoutSec),
			Operations: []txnbuild.Operation{
				&txnbuild.BumpSequence{
					BumpTo:        bumpTo,
					SourceAccount: nodeAccount.AccountID,
				},
			},
		})

		if err != nil {
			return errors.Errorf("Error creating seq bump tx: %v", err)
		}

		tx, err = n.rootClient.Sign(tx)

		if err != nil {
			return errors.Errorf("Error signing seq bump tx: %v", err)
		}

		_, err = n.rootClient.SubmitTransaction(tx)

		if err != nil {
			xdr, _ := tx.Base64()
			log.Errorf("Error in seq bump transaction: %s" + xdr)
			return errors.Errorf("Error submitting seq bump tx: %v", err)
		}

		return nil
	}

	var processedTransactions []*models.PaymentTransaction

	for len(transactions) > 0 {

		currentSequence, err = nodeAccount.GetSequenceNumber()

		if err != nil {
			return errors.Errorf("Error reading sequence: %v", err)
		}

		firstTransaction, err = n.rootClient.PaymentTransactionToStellar(transactions[0])

		if err != nil {
			return errors.Errorf("Can't get first transaction from wrapper: %s", err.Error())
		}

		// Bump if needed
		if firstTransaction.SourceAccount().Sequence > currentSequence+1 {
			log.Warnf("Sequence bump needed: %d", firstTransaction.SourceAccount().Sequence-(currentSequence+1))

			err := bumper(currentSequence, firstTransaction.SourceAccount().Sequence-1)

			if err != nil {
				return errors.Errorf("Error during sequence bump: %s", err)
			}
		}

		for a, t := range transactions {
			log.Infof("Submitting transaction for session %s", t.ServiceSessionId)
			txSuccess, transactionError := n.rootClient.SubmitTransactionXDR(t.XDR)
			resultsMap[t.PaymentSourceAddress] = txSuccess

			if transactionError != nil {

				log.Errorf("Error in submit transaction (%s): %s", transactionError.Error(), t.XDR)

				if stellarError, ok := transactionError.(*horizonclient.Error); ok {

					resultCodes, innerErr := stellarError.ResultCodes()

					if innerErr != nil {
						log.Errorf("Error unwrapping stellar errors: %v", innerErr.Error())
					} else {
						log.Errorf("Stellar error details - transaction error: %s", resultCodes.TransactionCode)

						for _, operror := range resultCodes.OperationCodes {
							log.Errorf("Stellar error details - operation error: %s", operror)
						}
					}
				} else {
					log.Errorf("Couldn't parse error as stellar: " + transactionError.Error())
				}

				internalTrans, err := n.rootClient.PaymentTransactionToStellar(t)

				if err != nil {
					log.Errorf("Error deserializing transaction for %d: %s", a, err)
				}

				//n.verifyTransactionSignatures(context,t)

				account := internalTrans.SourceAccount()
				accountSeqNumber, _ := account.GetSequenceNumber()
				//transactionSeqNumber := &internalTrans.(*xdr.Transaction).SeqNum
				_ = accountSeqNumber

				resultsMap[t.PaymentSourceAddress] = transactionError

				transactions = append(transactions[:a], transactions[a+1:]...)
				break
			}
			//TODO: Make the transaction removal more intellegent
			n.paymentRegistry.CompletePayment(t.PaymentSourceAddress, t.ServiceSessionId)
			processedTransactions = append(processedTransactions, t)

		}

		var left_transactions []*models.PaymentTransaction

		for _, x := range transactions {
			_, found := Find(processedTransactions, x)

			if !found {
				left_transactions = append(left_transactions, x)
			}
		}

		transactions = left_transactions
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

// //TODO REMOVE
// func (n *nodeImpl) CreatePaymentManager(ctx context.Context, request *models.ProcessPaymentRequest) (regestry.PaymentManager, error) {
// 	log.Infof("Got ProcessPayment NodeId=%s, CallbackUrl=%s\n Request:%v", request.NodeId, request.CallbackUrl, request.PaymentRequest)
// 	return n.paymentManagerRegestry.New(ctx, request)
// }
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

func (u *nodeImpl) ProcessCommand(ctx context.Context, command *models.UtilityCommand) (int, error) {
	if command.CallbackUrl != "" {
		callbacker := u.callbackerFactory(command)
		go func(callbacker CallBacker) {
			reply, err := u.CommandHandler(ctx, command)
			if err != nil {
				log.Fatalf("CommandHandler error: %v", err)
				return
			}
			if reply != nil {
				err := callbacker.call(reply)
				if err != nil {
					log.Fatalf("Callback error: %v", err)
					return
				}
			}
		}(callbacker)
		return http.StatusCreated, nil
	}
	_, err := u.CommandHandler(ctx, command)

	if err != nil {
		return http.StatusConflict, common.Error(http.StatusConflict, "command submitted")
	} else {
		return http.StatusOK, nil
	}

}
