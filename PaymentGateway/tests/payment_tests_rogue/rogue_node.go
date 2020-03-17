package payment_tests_rogue

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/clients/horizon"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
)

type RogueNode struct {
	internalNode node.PPNode
}

func (r RogueNode) AddPendingServicePayment(serviceSessionId string, amount common.TransactionAmount) {
	r.internalNode.AddPendingServicePayment(serviceSessionId,amount)
}

func (r RogueNode) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	r.internalNode.SetAccumulatingTransactionsMode(accumulateTransactions)
}

func (r RogueNode) CreatePaymentRequest(serviceSessionId string) (common.PaymentRequest, error) {
	return r.internalNode.CreatePaymentRequest(serviceSessionId)
}

func (r RogueNode) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) common.PaymentTransactionPayload {
	return r.internalNode.CreateTransaction(totalIn,fee,totalOut,sourceAddress)
}

func (r RogueNode) SignTerminalTransactions(creditTransactionPayload common.PaymentTransactionPayload) *errors.Error {
	return r.internalNode.SignTerminalTransactions(creditTransactionPayload)
}

func (r RogueNode) SignChainTransactions(creditTransactionPayload common.PaymentTransactionPayload, debitTransactionPayload common.PaymentTransactionPayload) *errors.Error {
	return r.internalNode.SignChainTransactions(creditTransactionPayload,debitTransactionPayload)
}

func (r RogueNode) CommitServiceTransaction(transaction common.PaymentTransactionPayload, pr common.PaymentRequest) (bool, error) {
	return r.internalNode.CommitServiceTransaction(transaction,pr)
}

func (r RogueNode) CommitPaymentTransaction(transactionPayload common.PaymentTransactionPayload) (ok bool, err error) {
	return r.internalNode.CommitPaymentTransaction(transactionPayload)
}

func (r RogueNode) GetAddress() string {
	return r.internalNode.GetAddress()
}

func CreateRogueNode_NonidenticalSequenceNumbers(client *horizon.Client,address string, seed string, accumulateTransactions bool) node.PPNode {

	node := node.CreateNode(client,address,seed,accumulateTransactions)

	rogueNode := RogueNode {
		internalNode:node,
	}

	return &rogueNode
}
