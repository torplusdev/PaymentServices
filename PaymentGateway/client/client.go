package client

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"log"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
	"reflect"
	"strconv"
	"strings"
)

type Client struct {
	client *horizonclient.Client
	fullKeyPair *keypair.Full
	account horizon.Account
}

func CreateClient(rootApi *root.RootApi, cliendSeed string) *Client {

	client := Client{}

	// Initialization
	apiClient := rootApi.GetClient()
	pair, err := keypair.ParseFull(cliendSeed)

	if err != nil {
		log.Fatal(err)
	}

	gwAccountDetail, errAccount := apiClient.AccountDetail(
		horizonclient.AccountRequest{
			AccountID:pair.Address() })

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

func (client *Client) SignInitialTransactions(fundingTransaction *common.PaymentTransaction, expectedDestination string, expectedAmount common.TransactionAmount) *errors.Error {


	t, err := txnbuild.TransactionFromXDR(fundingTransaction.XDR)

	if (err != nil) {
		log.Fatal("Error parsing transaction: ", err.Error())
		return errors.Errorf("Transaction parse error","")
	}

	if len(t.Operations) != 1  {
		log.Fatal("Transaction shall have only a single payment operation")
	}

	op, ok := t.Operations[0].(*txnbuild.Payment)

	if !ok {
		log.Fatal("Transaction shall have only a single payment operation")
	}

	if op.SourceAccount.GetAccountID() != client.fullKeyPair.Address() || op.Destination != expectedDestination {
		log.Fatal("Transaction op addresses are incorrect")
	}

	floatAmount,err := strconv.ParseFloat(op.Amount,32)
	amount := uint64(floatAmount)

	if err!=nil || amount != uint64(expectedAmount) {
		log.Fatal("Transaction amount is incorrect")
	}

	t.Network = fundingTransaction.Network

	err = t.Sign(client.fullKeyPair)

	if (err != nil) {
		log.Fatal("Failed to signed transaction")
	}
	fundingTransaction.XDR,err = t.Base64()

	if (err != nil) {
		log.Fatal("Error writing transaction envelope: " + err.Error())
		return errors.Errorf("Transaction envelope error","")
	}

	return nil

}

func (client *Client) VerifyTransactions(router common.PaymentRouter, paymentRequest common.PaymentRequest, transactions *[]common.PaymentTransaction) (bool,error) {

	ok := false

	for _,t := range *transactions {
		trans, e := txnbuild.TransactionFromXDR(t.XDR)

		if (e != nil) {
			log.Fatal("Error deser xdr: " + e.Error())

		}

		_ = trans
	}

	// stub
	ok = true

	return ok,nil
}

func (client *Client) InitiatePayment(router common.PaymentRouter, paymentRequest common.PaymentRequest) *[]common.PaymentTransaction {

	route := router.CreatePaymentRoute(paymentRequest)

	//validate route extremities

	if (strings.Compare(route[0].Address, client.fullKeyPair.Address()) != 0) {
		log.Fatal("Bad routing: Incorrect starting address ",route[0].Address," != ", client.fullKeyPair.Address())
	}

	if (strings.Compare(route[len(route)-1].Address, paymentRequest.Address) != 0) {
		log.Fatal("Bad routing: Incorrect destination address")
	}

	var totalFee common.TransactionAmount = 0


	transactions := make([]common.PaymentTransaction,0, len(route))

	//Iterating in reverse order
	reverseAny(route)


	// Generate initial transaction
	for i, e := range route[0:len(route)-1] {

		var sourceAddress = route[i+1].Address
		stepNode := node.GetNodeApi(e.Address, e.Seed)

		var transactionFee = e.Fee

		// We don't let service node to have transaction fees
		if e == route[0] {
			transactionFee = 0
		}

		// Create and store transaction
		nodeTransaction := stepNode.CreateTransaction(paymentRequest.Amount + totalFee + transactionFee, transactionFee, paymentRequest.Amount + totalFee, sourceAddress)
		nodeTransaction.Seed = e.Seed
		transactions = append(transactions,nodeTransaction)

		// Accumulate fees
		totalFee = totalFee +  transactionFee
	}
/*
	// Add initial client-originated funding transaction
	clientNode := node.GetNodeApi(route[len(route)-1].Address,route[len(route)-1].Seed)
	clientTransaction := clientNode.CreateTransaction(0, 0, paymentRequest.Amount + totalFee, sourceAddress)
	clientTransaction.Seed = e.Seed
	transactions = append(transactions,clientTransaction)
*/

	// initialize debit with service transaction
	debitTransaction := &transactions[0]

	// Signing terminal transaction
	serviceNode := node.GetNodeApi(route[0].Address,route[0].Seed)
	serviceNode.SignTerminalTransactions(debitTransaction)

	// Consecutive signing process
	for idx := 1; idx < len(transactions); idx++ {

		t := &transactions[idx]
		//TODO: Remove seed initialization
		stepNode := node.GetNodeApi(t.Address, t.Seed)
		creditTransaction := t

		stepNode.SignChainTransactions(creditTransaction,debitTransaction)

		debitTransaction = creditTransaction

	}

	err := client.SignInitialTransactions(&transactions[len(transactions)-1],route[len(transactions)-1].Address,paymentRequest.Amount + totalFee)

	if (err != nil) {
		log.Fatal("Error in transaction: " + err.Error())
	}

	// At this point all transactions are signed by all parties

	return &transactions
}

func (client *Client) FinalizePayment(router common.PaymentRouter, transactions *[]common.PaymentTransaction) (bool,error) {

	ok := false

	// TODO: Refactor to minimize possible mid-chain errors
	for _,t := range *transactions {
		paymentNode,err := router.GetNodeByAddress(t.Address)

		if (err!= nil) {
			log.Fatal("Error retrieving node object: " + err.Error())
		}

		_ = paymentNode


		stepNode := node.GetNodeApi(paymentNode.Address,paymentNode.Seed)

		stepNode.CommitPaymentTransaction(t)

	}

	return ok,nil
}