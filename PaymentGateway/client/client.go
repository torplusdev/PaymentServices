package client

import (
	"context"
	"fmt"
	"log"

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
	expectedDestination string, expectedAmount models.TransactionAmount) error {

	_, span := client.tracer.Start(context, "client-SignInitialTransactions")
	defer span.End()

	transaction := &tr.PendingTransaction
	log.Printf("SignInitialTransactions: Starting %s %d => %s ",
		transaction.PaymentSourceAddress,
		transaction.AmountOut,
		transaction.PaymentDestinationAddress)

	transactionWrapper, err := transaction.XDR.TransactionFromXDR()

	if err != nil {
		return errors.Errorf("transaction parse error: %v", err)
	}

	innerTransaction, ok := transactionWrapper.Transaction()

	if !ok {
		return errors.Errorf("transaction parse error (GenericTransaction) ")
	}

	if len(innerTransaction.Operations()) != 1 {
		return errors.Errorf("Transaction shall have only a single payment operation")
	}

	op, ok := innerTransaction.Operations()[0].(*txnbuild.Payment)

	if !ok {
		return errors.Errorf("Error in payment operation format")
	}

	//TODO to up
	localAddress := client.GetAddress()
	if op.SourceAccount != localAddress || op.Destination != expectedDestination {
		return errors.Errorf("Transaction op addresses are incorrect")
	}

	amount, err := client.GetMicroPPTokenBalance()

	expectedAmount = expectedAmount + tr.ReferenceTransaction.ReferenceAmountIn

	if err != nil || amount != expectedAmount {
		return errors.Errorf("Transaction amount is incorrect: expected %d, received %d", amount, expectedAmount)
	}

	resultTransaction, err := client.Sign(innerTransaction)

	if err != nil {
		return errors.Errorf("Failed to sign transaction")
	}

	xdr, err := resultTransaction.Base64()

	if err != nil {
		return errors.Errorf("Error converting transaction to binary xdr: %v", err)
	}

	tr.PendingTransaction.XDR = models.NewXDR(xdr)

	if err != nil {
		return errors.Errorf("Error writing transaction envelope: %v", err)
	}

	log.Printf("SignInitialTransactions: Finished %s %d => %s ",
		transaction.PaymentSourceAddress,
		transaction.AmountOut,
		transaction.PaymentDestinationAddress)

	return nil
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

		log.Printf("VerifyTransactions: Validated %s => %s ",
			t.PendingTransaction.PaymentSourceAddress,
			t.PendingTransaction.PaymentDestinationAddress)
	}

	// stub
	return nil
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
		log.Printf("insufficient client balance: %v", balance)
		return nil, errors.Errorf("client has insufficient account balance =%v", balance)
	}

	var totalFee models.TransactionAmount = 0

	//Iterating in reverse order

	nodes := nodeCollection.GetAllNodes()
	payChainLen := len(nodes)
	transactions := make([]*models.PaymentTransactionReplacing, 0, payChainLen-1)
	lastNodeIndex := payChainLen - 1
	// Generate initial transaction
	for i := lastNodeIndex; i > 1; i-- {
		//	for i, pr := range route[0 : len(route)-1] {

		sourceNode := nodes[i-1]
		destinationNode := nodes[i]
		sourceAddress := sourceNode.GetAddress()
		destinationAddress := destinationNode.GetAddress()
		transactionFee := sourceNode.GetFee()
		log.Printf("InitiatePayment: Creating transaction %s => %s", sourceAddress, destinationAddress)
		// We don't let service node to have transaction fees
		if i == lastNodeIndex {
			transactionFee = 0
		}
		request := &models.CreateTransactionCommand{
			TotalIn:          paymentRequest.Amount + totalFee + transactionFee,
			TotalOut:         paymentRequest.Amount + totalFee,
			SourceAddress:    sourceAddress,
			ServiceSessionId: paymentRequest.ServiceSessionId,
		}

		// Create and store transaction
		nodeTransaction, err := destinationNode.CreateTransaction(ctx, request)

		if err != nil {
			return nil, errors.Errorf("error creating transaction for node %v: %v", sourceAddress, err)
		}
		tr := nodeTransaction.Transaction
		err = tr.PendingTransaction.Validate()
		if err != nil {
			return nil, err
		}

		log.Printf("InitiatePayment: Transaction created  %s %d => %s", nodeTransaction.Transaction.PendingTransaction.PaymentSourceAddress,
			nodeTransaction.Transaction.PendingTransaction.AmountOut,
			nodeTransaction.Transaction.PendingTransaction.PaymentDestinationAddress)

		transactions = append(transactions, nodeTransaction.Transaction)
		// Accumulate fees
		totalFee = totalFee + transactionFee

	}

	// initialize debit with service transaction
	debitTransaction := transactions[0]
	serviceNode := nodeCollection.GetDestinationNode()
	// Signing terminal transaction
	serviceNodeAddress := serviceNode.GetAddress()

	log.Printf("InitiatePayment: SignServiceTransaction (%s) ", serviceNodeAddress)

	command := &models.SignServiceTransactionCommand{Transaction: debitTransaction}

	_, err = serviceNode.SignServiceTransaction(ctx, command)
	if err != nil {
		log.Print("Error signing terminal transaction ( node " + serviceNodeAddress + ") : " + err.Error())
		return nil, errors.Errorf("Error signing terminal transaction (%v): %v", debitTransaction, err)
	}

	// Consecutive signing process
	for idx := 1; idx < len(transactions); idx++ {
		creditTransaction := transactions[idx]
		destAddress := creditTransaction.PendingTransaction.PaymentDestinationAddress
		log.Printf("InitiatePayment: Sign chain  (%s) ", destAddress)
		stepNode := nodeCollection.GetNodeByAddress(destAddress)
		if stepNode == nil {
			return nil, errors.Errorf("Error: couldn't find a chain step node with address %s", destAddress)
		}

		cmd := &models.SignChainTransactionCommand{
			Credit: creditTransaction,
			Debit:  debitTransaction,
		}
		_, err = stepNode.SignChainTransaction(ctx, cmd)

		if err != nil {
			log.Print("Error signing transaction ( node " + destAddress + ") : " + err.Error())
			return nil, errors.Errorf("Error signing transaction (%v): %w", debitTransaction, err)
		}

		debitTransaction = creditTransaction
	}

	// for _, t := range transactions { I THINK IT IS NOT NEED //TUMARSAL
	// 	if t.PendingTransaction.PaymentSourceAddress == t.PendingTransaction.PaymentDestinationAddress {
	// 		return nil, errors.Errorf("Error invalid transaction chain, address targets itself %s.", t.PendingTransaction.PaymentSourceAddress)
	// 	}
	// }

	log.Printf("InitiatePayment: SignInitial   %d => %s ", paymentRequest.Amount+totalFee, transactions[len(transactions)-1].PendingTransaction.PaymentDestinationAddress)

	//правлиьно ли не понял что написано
	// err = client.signInitialTransactions(ctx,
	// 	transactions[len(transactions)-1],
	// 	route[len(transactions)-1].Address,
	// 	paymentRequest.Amount+totalFee)
	firstTransaction := transactions[len(transactions)-1]
	err = client.signInitialTransactions(ctx,
		firstTransaction,
		serviceNodeAddress,
		paymentRequest.Amount+totalFee)
	if err != nil {
		log.Print("Error in transaction: " + err.Error())
		return nil, errors.New("Error signing initial transaction has insufficient account balance")
	}

	return transactions, nil
}

func (client *serviceClient) FinalizePayment(context context.Context,
	nodeManager NodeChain,
	pr *models.PaymentRequest,
	transactions []*models.PaymentTransactionReplacing) error {

	ctx, span := client.tracer.Start(context, "client-FinalizePayment")
	defer span.End()

	log.Printf("Started FinalizePayment (%s) %d => %s", pr.ServiceRef, pr.Amount, pr.Address)

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
			log.Printf("Requesting CommitServiceTransaction (%s) => %s", tr.PendingTransaction.ServiceSessionId, tr.PendingTransaction.PaymentDestinationAddress)
			err := paymentNode.CommitServiceTransaction(ctx, &models.CommitServiceTransactionCommand{
				Transaction:    tr,
				PaymentRequest: pr,
			})
			if err != nil {
				return fmt.Errorf("error committing transaction %s", err)
			}
			continue
		}
		log.Printf("Requesting CommitChainTransaction (%s) => %s", tr.PendingTransaction.ServiceSessionId, tr.PendingTransaction.PaymentDestinationAddress)
		err := paymentNode.CommitChainTransaction(ctx, &models.CommitChainTransactionCommand{
			Transaction: tr,
		})
		if err != nil {
			return fmt.Errorf("error committing transaction %s", err)
		}

	}

	return nil
}
