package client

import (
	"context"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-errors/errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"go.opentelemetry.io/otel/api/trace"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
)

type Client struct {
	client           *horizonclient.Client
	fullKeyPair      *keypair.Full
	account          horizon.Account
	nodeManager      node.NodeManager
	commodityManager *commodity.Manager
	tracer           trace.Tracer
}

func CreateClient(rootApi *root.RootApi, clientSeed string, nm node.NodeManager, commodityManager *commodity.Manager) *Client {

	client := Client{
		nodeManager:      nm,
		commodityManager: commodityManager,
		tracer:           common.CreateTracer("client"),
	}

	// Initialization
	apiClient := rootApi.GetClient()
	pair, err := keypair.ParseFull(clientSeed)

	if err != nil {
		log.Fatal(err)
	}

	gwAccountDetail, errAccount := apiClient.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	if errAccount != nil {
		log.Fatal("Client account doesnt exist")
	} else {
		client.client = apiClient
		client.fullKeyPair = pair
		client.account = gwAccountDetail
	}

	return &client
}

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func (client *Client) GetCommodityManager() *commodity.Manager {
	return client.commodityManager
}

func (client *Client) SignInitialTransactions(context context.Context, fundingTransactionPayload *common.PaymentTransactionReplacing, expectedDestination string, expectedAmount common.TransactionAmount) error {

	_, span := client.tracer.Start(context, "client-SignInitialTransactions")
	defer span.End()

	transaction := fundingTransactionPayload.GetPaymentTransaction()

	transactionWrapper, err := txnbuild.TransactionFromXDR(transaction.XDR)

	if err != nil {
		return errors.Errorf("transaction parse error: %v", err)
	}

	innerTransaction, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("transaction parse error (GenericTransaction) ")
	}

	if len(innerTransaction.Operations()) != 1 {
		return errors.Errorf("Transaction shall have only a single payment operation")
	}

	op, ok := innerTransaction.Operations()[0].(*txnbuild.Payment)

	if !ok {
		return errors.Errorf("Error in payment operation format")
	}

	if op.SourceAccount.GetAccountID() != client.fullKeyPair.Address() || op.Destination != expectedDestination {
		return errors.Errorf("Transaction op addresses are incorrect")
	}

	floatAmount, err := strconv.ParseFloat(op.Amount, 32)
	amount := uint64(floatAmount)

	// Add amount from previous transaction
	expectedAmount = expectedAmount + fundingTransactionPayload.ReferenceTransaction.ReferenceAmountIn

	if err != nil || amount != uint64(expectedAmount) {
		return errors.Errorf("Transaction amount is incorrect")
	}



	resultTransaction,err := innerTransaction.Sign(transaction.StellarNetworkToken, client.fullKeyPair)

	if err != nil {
		return errors.Errorf("Failed to sign transaction")
	}

	xdr, err := resultTransaction.Base64()

	if err != nil {
		return errors.Errorf("Error converting transaction to binary xdr: %v", err)
	}

	err = fundingTransactionPayload.UpdateTransactionXDR(xdr)

	if err != nil {
		return errors.Errorf("Error writing transaction envelope: %v", err)
	}

	return nil
}

func (client *Client) VerifyTransactions(context context.Context, router common.PaymentRouter, paymentRequest common.PaymentRequest, transactions []common.PaymentTransactionReplacing) (bool, error) {

	_, span := client.tracer.Start(context, "client-VerifyTransactions")
	defer span.End()

	ok := false

	for _, t := range transactions {
		e := t.Validate()

		if e != nil {
			log.Print("Error validating transaction: " + e.Error())
			return false, e
		}

		trans, e := txnbuild.TransactionFromXDR(t.GetPaymentTransaction().XDR)

		if e != nil {
			log.Print("Error deserializing xdr: " + e.Error())
			return false, e
		}
		_ = trans
	}

	// stub
	ok = true

	return ok, nil
}

func (client *Client) InitiatePayment(context context.Context, router common.PaymentRouter, paymentRequest common.PaymentRequest) ([]common.PaymentTransactionReplacing, error) {

	ctx, span := client.tracer.Start(context, "client-InitiatePayment")
	defer span.End()

	route := router.CreatePaymentRoute(paymentRequest)

	//validate route extremities
	if strings.Compare(route[0].Address, client.fullKeyPair.Address()) != 0 {
		log.Print("Bad routing: Incorrect starting address ", route[0].Address, " != ", client.fullKeyPair.Address())
		return nil, errors.New("Incorrect starting address")
	}

	if strings.Compare(route[len(route)-1].Address, paymentRequest.Address) != 0 {
		log.Print("Bad routing: Incorrect destination address")
		return nil, errors.New("Incorrect destination address")
	}

	// TODO: Move out to external validation sequence
	/*
		accountDetail, errAccount := client.client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID:client.fullKeyPair.Address() })

		if errAccount != nil {
			log.Print("Error retrieving account data: ", errAccount.Error())
			return nil,errors.Errorf("Account validation error","")
		}


		balance := accountDetail.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)

		numericBalance, err := strconv.ParseFloat(balance,32)

		if err != nil {
			log.Print("Error parsing account balance: ", err.Error())
			return nil, errors.Errorf("Account balance parse error","")
		}

		if paymentRequest.Amount > uint32(numericBalance) {
			log.Print("Insufficient client balance: ")
			return nil, errors.Errorf("Client has insufficient account balance","")
		}
	*/

	var totalFee common.TransactionAmount = 0

	transactions := make([]common.PaymentTransactionReplacing, 0, len(route))

	//Iterating in reverse order
	reverseAny(route)

	// Generate initial transaction
	for i, e := range route[0 : len(route)-1] {

		var sourceAddress = route[i+1].Address
		stepNode := client.nodeManager.GetNodeByAddress(e.Address)

		var transactionFee = e.Fee

		// We don't let service node to have transaction fees
		if e == route[0] {
			transactionFee = 0
		}

		// Create and store transaction
		nodeTransaction, err := stepNode.CreateTransaction(ctx, paymentRequest.Amount+totalFee+transactionFee, transactionFee, paymentRequest.Amount+totalFee, sourceAddress, paymentRequest.ServiceSessionId)

		if err != nil {
			log.Print("Error creating transaction for node " + sourceAddress + " : " + err.Error())
			return nil, errors.Errorf("Error creating transaction for node %v: %v", sourceAddress, err)
		}

		transactions = append(transactions, nodeTransaction)

		// Accumulate fees
		totalFee = totalFee + transactionFee
	}
	/*
		// Add initial client-originated funding transaction
		clientNode := node.GetNodeApi(route[len(route)-1].PaymentDestinationAddress,route[len(route)-1].Seed)
		clientTransaction := clientNode.CreateTransaction(0, 0, paymentRequest.Amount + totalFee, sourceAddress)
		clientTransaction.Seed = e.Seed
		transactions = append(transactions,clientTransaction)
	*/

	for _, t := range transactions {
		log.Printf("Transaction detail: %s ==> %s", t.PendingTransaction.PaymentSourceAddress, t.PendingTransaction.PaymentDestinationAddress)
		if t.PendingTransaction.PaymentSourceAddress == t.PendingTransaction.PaymentDestinationAddress {
			log.Print("Error")
		}
	}

	// initialize debit with service transaction
	debitTransaction := &transactions[0]

	// Signing terminal transaction
	serviceNode := client.nodeManager.GetNodeByAddress(route[0].Address)

	err := serviceNode.SignTerminalTransactions(ctx, debitTransaction)

	if err != nil {
		log.Print("Error signing terminal transaction ( node " + route[0].Address + ") : " + err.Error())
		return nil, errors.Errorf("Error signing terminal transaction (%v): %v", debitTransaction, err)
	}

	// Consecutive signing process
	for idx := 1; idx < len(transactions); idx++ {

		t := &transactions[idx]

		stepNode := client.nodeManager.GetNodeByAddress(t.GetPaymentDestinationAddress())
		creditTransaction := t

		err  = stepNode.SignChainTransactions(ctx, creditTransaction, debitTransaction)

		if err != nil {
			log.Print("Error signing transaction ( node " + t.GetPaymentDestinationAddress() + ") : " + err.Error())
			return nil, errors.Errorf("Error signing transaction (%v): %w", debitTransaction, err)
		}

		debitTransaction = creditTransaction
	}

	for _, t := range transactions {
		if t.PendingTransaction.PaymentSourceAddress == t.PendingTransaction.PaymentDestinationAddress {
			log.Print("Error")
		}
	}

	err = client.SignInitialTransactions(ctx, &transactions[len(transactions)-1], route[len(transactions)-1].Address, paymentRequest.Amount+totalFee)

	if err != nil {
		log.Print("Error in transaction: " + err.Error())
		return nil, errors.New("Error signing initial transaction has insufficient account balance")
	}

	// At this point all transactions are signed by all parties

	for _, t := range transactions {
		if t.PendingTransaction.PaymentSourceAddress == t.PendingTransaction.PaymentDestinationAddress {
			log.Print("Error")
		}
	}

	return transactions, nil
}

func (client *Client) FinalizePayment(context context.Context, router common.PaymentRouter, transactions []common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {

	ctx, span := client.tracer.Start(context, "client-FinalizePayment")
	defer span.End()

	ok := true

	// TODO: Refactor to minimize possible mid-chain errors
	for _, t := range transactions {
		trans := t.GetPaymentTransaction()
		paymentNode, err := router.GetNodeByAddress(trans.PaymentDestinationAddress)

		if err != nil {
			log.Print("Error retrieving node object: " + err.Error())
			return false, errors.Errorf("Error retrieving node object %s", err.Error())
		}

		_ = paymentNode

		stepNode := client.nodeManager.GetNodeByAddress(paymentNode.Address)

		var res = false

		// If this is a payment to the requesting node
		if trans.PaymentDestinationAddress == pr.Address {
			res, err = stepNode.CommitServiceTransaction(ctx, &t, pr)
		} else {
			res, err = stepNode.CommitPaymentTransaction(ctx, &t)
		}

		if err != nil {
			log.Print("Error committing transaction: " + err.Error())
			return false, errors.Errorf("Error committing transaction %s", err.Error())
		}

		ok = ok && res
	}

	return ok, nil
}
