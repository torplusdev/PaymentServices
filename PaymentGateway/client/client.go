package client

import (
	"context"
	"fmt"
	"strconv"

	"paidpiper.com/payment-gateway/log"

	"github.com/go-errors/errors"

	"github.com/stellar/go/txnbuild"
	"go.opentelemetry.io/otel/api/trace"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"

	"paidpiper.com/payment-gateway/root"
)

type NodeChain interface {
	GetNodeByAddress(address string) node.PPNode
	Validate(from string, to string) error
	GetAllNodes() []node.PPNode
	GetDestinationNode() node.PPNode
}

//TODO TO STAGE MODEL
type ServiceClient interface {
	InitiatePayment(context.Context, NodeChain, *models.PaymentRequest) ([]*models.PaymentTransactionReplacing, error)
	VerifyTransactions(context.Context, []*models.PaymentTransactionReplacing) error
	FinalizePayment(context.Context, NodeChain, *models.PaymentRequest, []*models.PaymentTransactionReplacing) error
}
type serviceClient struct {
	root.RootApi
	tracer trace.Tracer
}

func New(rootApi root.RootApi) ServiceClient {
	client := &serviceClient{
		RootApi: rootApi,
		tracer:  common.CreateTracer("client"),
	}
	return client
}

func (client *serviceClient) signInitialTransactions(context context.Context,
	tr *models.PaymentTransactionReplacing,
	expectedDestination string, expectedAmount models.TransactionAmount) (*models.PaymentTransactionReplacing, error) {

	_, span := client.tracer.Start(context, "client-SignInitialTransactions")
	defer span.End()
	transaction := &tr.PendingTransaction
	log.Infof("SignInitialTransactions: Starting %s %d => %s ",
		transaction.PaymentSourceAddress,
		transaction.AmountOut,
		transaction.PaymentDestinationAddress)

	transactionWrapper, err := transaction.XDR.TransactionFromXDR()

	if err != nil {
		return nil, fmt.Errorf("transaction parse error: %v", err)
	}

	innerTransaction, ok := transactionWrapper.Transaction()

	if !ok {
		return nil, fmt.Errorf("transaction parse error (GenericTransaction) ")
	}

	if len(innerTransaction.Operations()) != 1 {
		return nil, fmt.Errorf("transaction shall have only a single payment operation")
	}

	op, ok := innerTransaction.Operations()[0].(*txnbuild.Payment)

	if !ok {
		return nil, fmt.Errorf("error in payment operation format")
	}

	localAddress := client.GetAddress()
	if op.SourceAccount != localAddress {
		return nil, fmt.Errorf("source account is invalid")
	}
	if op.Destination != expectedDestination {
		return nil, fmt.Errorf("destination account is invalid")
	}
	floatAmount, err := strconv.ParseFloat(op.Amount, 32)
	if err != nil {
		return nil, fmt.Errorf("call ParseFloat error: %v", err)
	}
	amount := models.MicroPPToken2PPtoken(floatAmount)
	if tr.ReferenceTransaction != nil {
		expectedAmount = expectedAmount + tr.ReferenceTransaction.ReferenceAmountIn
	}

	if amount != expectedAmount {
		return nil, fmt.Errorf("transaction amount is incorrect: expected %d, received %d", amount, expectedAmount)
	}

	resultTransaction, err := client.Sign(innerTransaction)

	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction")
	}

	xdr, err := resultTransaction.Base64()

	if err != nil {
		return nil, fmt.Errorf("error converting transaction to binary xdr: %v", err)
	}

	tr.PendingTransaction.XDR = models.NewXDR(xdr)

	if err != nil {
		return nil, fmt.Errorf("error writing transaction envelope: %v", err)
	}

	log.Infof("SignInitialTransactions: Finished %s %d => %s ",
		transaction.PaymentSourceAddress,
		transaction.AmountOut,
		transaction.PaymentDestinationAddress)

	return tr, nil
}

func (client *serviceClient) VerifyTransactions(context context.Context, trs []*models.PaymentTransactionReplacing) error {

	_, span := client.tracer.Start(context, "client-VerifyTransactions")
	defer span.End()

	//log.Printf("VerifyTransactions: started   %d => %s ", paymentRequest.Amount, paymentRequest.Address)

	for _, t := range trs {
		e := t.Validate()

		if e != nil {
			return fmt.Errorf("error validating transaction: %s", e)
		}

		log.Infof("VerifyTransactions: Validated %s => %s ",
			t.PendingTransaction.PaymentSourceAddress,
			t.PendingTransaction.PaymentDestinationAddress)
	}
	return nil
}

type TransactionsCollection struct {
	transactions []*models.PaymentTransactionReplacing
	totalFee     uint32
}

func (client *serviceClient) signTransactions(ctx context.Context, paymentRequest *models.PaymentRequest, nodeCollection NodeChain, trs *TransactionsCollection) ([]*models.PaymentTransactionReplacing, error) {
	transactions := trs.transactions

	signedTransactions := []*models.PaymentTransactionReplacing{}
	serviceNode := nodeCollection.GetDestinationNode()
	// Signing terminal transaction
	serviceNodeAddress := serviceNode.GetAddress()

	log.Infof("InitiatePayment: SignServiceTransaction (%s) ", serviceNodeAddress)

	// initialize debit with service transaction
	debitTransaction := transactions[0]
	command := &models.SignServiceTransactionCommand{Transaction: debitTransaction}

	signedDebitTransactionResponse, err := serviceNode.SignServiceTransaction(ctx, command)

	if err != nil {
		log.Errorf("Error signing terminal transaction ( node %s) : %v ", serviceNodeAddress, err)
		return nil, errors.Errorf("Error signing terminal transaction (%v): %v", debitTransaction, err)
	}

	signedDebitTransaction := signedDebitTransactionResponse.Transaction

	// Consecutive signing process
	for idx := 1; idx < len(transactions); idx++ {
		creditTransaction := transactions[idx]
		destAddress := creditTransaction.PendingTransaction.PaymentDestinationAddress
		log.Infof("InitiatePayment: Sign chain  (%s) ", destAddress)
		stepNode := nodeCollection.GetNodeByAddress(destAddress)
		if stepNode == nil {
			return nil, errors.Errorf("Error: couldn't find a chain step node with address %s", destAddress)
		}

		cmd := &models.SignChainTransactionCommand{
			Credit: creditTransaction,
			Debit:  signedDebitTransaction,
		}
		signedTransaction, err := stepNode.SignChainTransaction(ctx, cmd)

		if err != nil {
			log.Errorf("Error signing transaction ( node " + destAddress + ") : " + err.Error())
			return nil, errors.Errorf("Error signing transaction (%v): %w", debitTransaction, err)
		}
		signedTransactions = append(signedTransactions, signedTransaction.Debit)

		signedDebitTransaction = signedTransaction.Credit
	}
	totalFee := trs.totalFee
	log.Infof("InitiatePayment: SignInitial   %d => %s ", paymentRequest.Amount+totalFee, transactions[len(transactions)-1].PendingTransaction.PaymentDestinationAddress)
	nodes := nodeCollection.GetAllNodes()

	//firstTransaction := signedDebitTransaction[len(transactions)-1]
	selfTr, err := client.signInitialTransactions(ctx,
		signedDebitTransaction,
		nodes[1].GetAddress(),
		paymentRequest.Amount+totalFee)
	if err != nil {
		log.Errorf("error in transaction: %v", err)
		return nil, fmt.Errorf("sign initial transactions error: %v", err)
	}
	signedTransactions = append(signedTransactions, selfTr)

	return signedTransactions, nil
}

func (client *serviceClient) createTransactions(ctx context.Context, paymentRequest *models.PaymentRequest, nodeCollection NodeChain) (*TransactionsCollection, error) {
	var totalFee models.TransactionAmount = 0
	nodes := nodeCollection.GetAllNodes()
	payChainLen := len(nodes)
	transactions := make([]*models.PaymentTransactionReplacing, 0, payChainLen-1)
	lastNodeIndex := payChainLen - 1
	// Generate initial transaction
	for i := lastNodeIndex; i > 0; i-- {
		sourceNode := nodes[i-1]
		destNode := nodes[i]
		sourceAddress := sourceNode.GetAddress()
		destAddress := destNode.GetAddress()
		transactionFee := destNode.GetFee()
		log.Infof("InitiatePayment: Creating transaction %s => %s", sourceAddress, destAddress)
		request := &models.CreateTransactionCommand{
			TotalIn:          paymentRequest.Amount + totalFee + transactionFee,
			TotalOut:         paymentRequest.Amount + totalFee,
			SourceAddress:    sourceAddress,
			ServiceSessionId: paymentRequest.ServiceSessionId,
		}

		// Create and store transaction
		nodeTransaction, err := destNode.CreateTransaction(ctx, request)

		if err != nil {
			return nil, errors.Errorf("error creating transaction for node %v: %v", sourceAddress, err)
		}
		tr := nodeTransaction.Transaction
		err = tr.PendingTransaction.Validate()
		if err != nil {
			return nil, err
		}

		log.Infof("InitiatePayment: Transaction created  %s %d => %s", nodeTransaction.Transaction.PendingTransaction.PaymentSourceAddress,
			nodeTransaction.Transaction.PendingTransaction.AmountOut,
			nodeTransaction.Transaction.PendingTransaction.PaymentDestinationAddress)

		transactions = append(transactions, nodeTransaction.Transaction)
		// Accumulate fees
		totalFee = totalFee + transactionFee
	}

	return &TransactionsCollection{
		transactions,
		totalFee,
	}, nil
}

func (client *serviceClient) InitiatePayment(context context.Context,
	nodeCollection NodeChain,
	paymentRequest *models.PaymentRequest) ([]*models.PaymentTransactionReplacing, error) {

	ctx, span := client.tracer.Start(context, "client-InitiatePayment")
	defer span.End()

	// TODO: Move out to external validation sequence

	err := nodeCollection.Validate(client.GetAddress(), paymentRequest.Address)
	if err != nil {
		return nil, err
	}
	balance, err := client.GetMicroPPTokenBalance()
	if err != nil {
		return nil, err
	}
	if paymentRequest.Amount > uint32(balance) {
		log.Infof("insufficient client balance: %v", balance)
		return nil, errors.Errorf("client has insufficient account balance =%v", balance)
	}

	//Iterating in reverse order

	trs, err := client.createTransactions(ctx, paymentRequest, nodeCollection)
	if err != nil {
		return nil, fmt.Errorf("create transactions error:%v", err)
	}

	singedTransaction, err := client.signTransactions(ctx, paymentRequest, nodeCollection, trs)
	if err != nil {
		return nil, fmt.Errorf("signTransactions error: %v", err)
	}

	for _, t := range singedTransaction { // Move to http layer
		if t.PendingTransaction.PaymentSourceAddress == t.PendingTransaction.PaymentDestinationAddress {
			return nil, errors.Errorf("Error invalid transaction chain, address targets itself %s.", t.PendingTransaction.PaymentSourceAddress)
		}
	}
	log.Info("Payment Complete Success: ", paymentRequest.ServiceSessionId)
	return singedTransaction, nil
}

func (client *serviceClient) FinalizePayment(context context.Context,
	nodeManager NodeChain,
	pr *models.PaymentRequest,
	transactions []*models.PaymentTransactionReplacing) error {

	ctx, span := client.tracer.Start(context, "client-FinalizePayment")
	defer span.End()

	log.Infof("Started FinalizePayment (%s) %d => %s", pr.ServiceRef, pr.Amount, pr.Address)

	// TODO: Refactor to minimize possible mid-chain errors
	for _, tr := range transactions {
		trans := tr.PendingTransaction
		paymentNode := nodeManager.GetNodeByAddress(trans.PaymentDestinationAddress)
		if paymentNode == nil {
			log.Print("Error retrieving node object: ")
			return errors.Errorf("error retrieving node object ")
		}

		// If this is a payment to the requesting node
		if trans.PaymentDestinationAddress == pr.Address {
			log.Infof("Requesting CommitServiceTransaction (%s) => %s", tr.PendingTransaction.ServiceSessionId, tr.PendingTransaction.PaymentDestinationAddress)
			err := paymentNode.CommitServiceTransaction(ctx, &models.CommitServiceTransactionCommand{
				Transaction:    tr,
				PaymentRequest: pr,
			})
			if err != nil {
				return fmt.Errorf("error committing transaction: %v", err)
			}
			continue
		}
		log.Infof("Requesting CommitChainTransaction (%s) => %s", tr.PendingTransaction.ServiceSessionId, tr.PendingTransaction.PaymentDestinationAddress)
		err := paymentNode.CommitChainTransaction(ctx, &models.CommitChainTransactionCommand{
			Transaction: tr,
		})
		if err != nil {
			return fmt.Errorf("error committing transaction: %v", err)
		}

	}

	return nil
}
