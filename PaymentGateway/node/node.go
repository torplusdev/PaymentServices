package node

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"paidpiper.com/payment-gateway/common"
	"strconv"
)

const nodeTransactionFee = 10

type Node struct {
	Address string
	secretSeed string
	client horizon.Client
	transactionFee common.TransactionAmount
	pendingPayment map[string]float64
}

func CreateNode(address string, seed string) *Node {

	node := Node {
		Address:address,
		secretSeed:seed,
		client:*horizon.DefaultTestNetClient,
		transactionFee:nodeTransactionFee,
	}

	return &node
}

type NodeManager interface {
	GetNodeByAddress(address string) *Node
}

func (n *Node) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) common.PaymentTransaction {
	transaction := common.PaymentTransaction{
		TransactionSource:n.Address,
		ReferenceAmountIn:totalIn,
		AmountOut:totalOut,
		Address:n.Address,
	}

	//Verify fee
	if (totalIn - totalOut != fee) {
		log.Fatal("Incorrect fee requested")
	}

	var amount = totalIn

	tx, err := build.Transaction(
		build.SourceAccount{n.Address},
		build.AutoSequence{&n.client},
		build.Payment(
			build.SourceAccount{sourceAddress},
			build.Destination{n.Address},
			build.NativeAmount{strconv.FormatUint(uint64(amount),10)},
		),
	)

	if (err != nil) {
		log.Fatal("Error creating transaction: " + err.Error())
	}
	if (n.client.URL == "https://horizon-testnet.stellar.org") {
		tx.Mutate(build.TestNetwork)
	} else {
		tx.Mutate(build.DefaultNetwork)
	}

	txe,err := tx.Envelope()

	if (err != nil) {
		log.Fatal("Error generating transaction envelope: " + err.Error())
	}

	transaction.XDR,err = txe.Base64()

	if (err != nil) {
		log.Fatal("Error serializing transaction: " + err.Error())
	}

	// TODO: This should be configurable via profile/strategy
	transaction.Network = build.TestNetwork.Passphrase

	return transaction
}


func (n *Node) SignTerminalTransactions(creditTransaction *common.PaymentTransaction) *errors.Error {

	// Validate
	if (creditTransaction.Address != n.Address) {
		log.Fatal("Transaction destination is incorrect ", creditTransaction.Address)
		return errors.Errorf("Transaction destination error","")
	}

	kp,err := keypair.ParseFull(n.secretSeed)

	if (err != nil) {
		log.Fatal("Error parsing keypair: ", err.Error())
		return errors.Errorf("Runtime key error","")
	}

	t, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)
	t.Network = creditTransaction.Network

	if (err != nil) {
		log.Fatal("Error parsing transaction: ", err.Error())
		return errors.Errorf("Transaction parse error","")
	}

	err = t.Sign(kp)

	if (err != nil) {
		log.Fatal("Failed to signed transaction")
	}

	creditTransaction.XDR,err = t.Base64()

	if (err != nil) {
		log.Fatal("Error writing transaction envelope: " + err.Error())
		return errors.Errorf("Transaction envelope error","")
	}

	return nil

}

func (n *Node) SignChainTransactions(creditTransaction *common.PaymentTransaction, debitTransaction *common.PaymentTransaction) *errors.Error {

	kp,err := keypair.ParseFull(n.secretSeed)

	if (err != nil) {
		log.Fatal("Error parsing keypair: ", err.Error())
		return errors.Errorf("Runtime key error","")
	}

	credit, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)
	credit.Network = creditTransaction.Network

	if (err != nil) {
		log.Fatal("Error parsing credit transaction: ", err.Error())
		return errors.Errorf("Transaction parse error","")
	}

	debit, err  := txnbuild.TransactionFromXDR(debitTransaction.XDR)
	debit.Network = debitTransaction.Network

	if (err != nil) {
		log.Fatal("Error parsing debit transaction: ", err.Error())
		return errors.Errorf("Transaction parse error","")
	}

	err = credit.Sign(kp)

	if (err != nil) {
		log.Fatal("Failed to signed transaction")
	}


	err = debit.Sign(kp)

	if (err != nil) {
		log.Fatal("Failed to signed transaction")
	}

	creditTransaction.XDR,err = credit.Base64()

	if (err != nil) {
		log.Fatal("Error writing credit transaction envelope: " + err.Error())
		return errors.Errorf("Transaction envelope error","")
	}

	debitTransaction.XDR,err = debit.Base64()

	if (err != nil) {
		log.Fatal("Error writing debit transaction envelope: " + err.Error())
		return errors.Errorf("Transaction envelope error","")
	}

	return nil
}

func (n *Node) CommitPaymentTransaction(transaction common.PaymentTransaction) (ok bool,err error) {

	ok = false

	t,e := txnbuild.TransactionFromXDR(transaction.XDR)

	if (e!= nil) {
		log.Error("Error during transaction deser: " + e.Error())
	}
	_ = t

	res, err := n.client.SubmitTransaction(transaction.XDR)

	if (err != nil) {
		log.Error("Error submitting transaction: " + err.Error())
	}

	ok = true
	log.Debug("Transaction submitted: " + res.Result)

	return
}
