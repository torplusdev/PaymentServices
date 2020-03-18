package payment_tests_rogue

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
)

type RogueNode struct {
	internalNode node.PPNode
}

func (r *RogueNode) AddPendingServicePayment(serviceSessionId string, amount common.TransactionAmount) {
	r.internalNode.AddPendingServicePayment(serviceSessionId,amount)
}

func (r *RogueNode) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	r.internalNode.SetAccumulatingTransactionsMode(accumulateTransactions)
}

func (r *RogueNode) CreatePaymentRequest(serviceSessionId string) (common.PaymentRequest, error) {
	return r.internalNode.CreatePaymentRequest(serviceSessionId)
}

func (r *RogueNode) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing,error) {
	transaction,err := r.internalNode.CreateTransaction(totalIn,fee,totalOut,sourceAddress)

	if err != nil {
		panic("unexpected error creating transaction")
	}

	payTrans := transaction.GetPaymentTransaction()
	refTrans := transaction.GetReferenceTransaction()

	if (refTrans == common.PaymentTransaction{}) {
		return transaction,err
	}

	payTransStellar,_ := txnbuild.TransactionFromXDR(payTrans.XDR)
	refTransStellar,_ := txnbuild.TransactionFromXDR(refTrans.XDR)

	paySequenceNumber,_ := payTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()
	refSequenceNumber,_ := refTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()

	if (paySequenceNumber != refSequenceNumber) {
		panic("sequence numbers are already different, unexpected")
	}

	op := payTransStellar.Operations[0]
	xdrOp, _ := op.BuildXDR()
	var payment *txnbuild.Payment

	switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment = &txnbuild.Payment{}
			err = payment.FromXDR(xdrOp)
			if err != nil {
				panic("error deserializing op xdr")
			}

		default:
			panic("unexpected operation type")

	}

	tx, err := build.Transaction(
		build.SourceAccount{payTransStellar.SourceAccount.GetAccountID()},
		build.AutoSequence{common.CreateStaticSequence(uint64(refSequenceNumber + 1))},
		build.Payment(
			build.SourceAccount{payment.SourceAccount.GetAccountID()},
			build.Destination{payment.Destination},
			build.NativeAmount{payment.Amount}	),
	)

	if err != nil {
		panic("unexpected error - transaction injection")
	}

	tx.Mutate(build.TestNetwork)

	txe, err := tx.Envelope()

	if err != nil {
		panic("unexpected error - envelope")
	}

	xdr, err := txe.Base64()

	transaction.UpdateTransactionXDR(xdr)

	return transaction,nil
}

func (r *RogueNode) SignTerminalTransactions(creditTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	return r.internalNode.SignTerminalTransactions(creditTransactionPayload)
}

func (r *RogueNode) SignChainTransactions(creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	return r.internalNode.SignChainTransactions(creditTransactionPayload,debitTransactionPayload)
}

func (r *RogueNode) CommitServiceTransaction(transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {
	return r.internalNode.CommitServiceTransaction(transaction,pr)
}

func (r *RogueNode) CommitPaymentTransaction(transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {
	return r.internalNode.CommitPaymentTransaction(transactionPayload)
}

func (r *RogueNode) GetAddress() string {
	return r.internalNode.GetAddress()
}

func CreateRogueNode_NonidenticalSequenceNumbers(client *horizon.Client,address string, seed string, accumulateTransactions bool) node.PPNode {

	node := node.CreateNode(client,address,seed,accumulateTransactions)

	rogueNode := RogueNode {
		internalNode:node,
	}

	return &rogueNode
}
