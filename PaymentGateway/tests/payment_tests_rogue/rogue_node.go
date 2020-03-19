package payment_tests_rogue

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
)

type RogueNode struct {
	internalNode node.PPNode
	transactionCreationFunction func(*RogueNode,common.TransactionAmount,common.TransactionAmount,common.TransactionAmount, string) (common.PaymentTransactionReplacing,error)
	signChainTransactionsFunction func(*RogueNode, *common.PaymentTransactionReplacing, *common.PaymentTransactionReplacing) *errors.Error
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

	return r.transactionCreationFunction(r, totalIn, fee, totalOut, sourceAddress)
}

func createTransactionCorrect(r *RogueNode,totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing,error) {
	return r.internalNode.CreateTransaction(totalIn, fee, totalOut, sourceAddress)
}

func (r *RogueNode) SignTerminalTransactions(creditTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	return r.internalNode.SignTerminalTransactions(creditTransactionPayload)
}

func (r *RogueNode) SignChainTransactions(creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	return r.signChainTransactionsFunction(r,creditTransactionPayload,debitTransactionPayload)
}

func signChainTransactionsNoError(r *RogueNode,creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
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

func createTransactionIncorrectSequence(r *RogueNode,totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing,error) {
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

func signChainTransactionsBadSignature(r *RogueNode,creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()
	debitTransaction := debitTransactionPayload.GetPaymentTransaction()

	kp, err := keypair.Random()

	credit, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)
	credit.Network = creditTransaction.StellarNetworkToken

	if err != nil {
		return errors.New("Transaction deser error")
	}

	debit, err := txnbuild.TransactionFromXDR(debitTransaction.XDR)
	debit.Network = debitTransaction.StellarNetworkToken

	if err != nil {
		return errors.New("Transaction parse error")
	}

	err = credit.Sign(kp)

	if err != nil {
		return errors.New("Failed to sign transaction")
	}

	err = debit.Sign(kp)

	if err != nil {
		return errors.New("Failed to sign transaction")
	}

	creditTransaction.XDR, err = credit.Base64()

	if err != nil {
		return errors.New("Transaction envelope error")
	}

	creditTransactionPayload.UpdateTransactionXDR(creditTransaction.XDR)

	debitTransaction.XDR, err = debit.Base64()

	if err != nil {
		return errors.New("Transaction envelope error")
	}

	debitTransactionPayload.UpdateTransactionXDR(debitTransaction.XDR)

	return nil
}

func CreateRogueNode_NonidenticalSequenceNumbers(client *horizon.Client,address string, seed string, accumulateTransactions bool) node.PPNode {

	node := node.CreateNode(client,address,seed,accumulateTransactions)

	rogueNode := RogueNode {
		internalNode:node,
		transactionCreationFunction:createTransactionIncorrectSequence,
		signChainTransactionsFunction:signChainTransactionsNoError,
	}

	return &rogueNode
}

func CreateRogueNode_BadSignature(client *horizon.Client,address string, seed string, accumulateTransactions bool) node.PPNode {

	node := node.CreateNode(client,address,seed,accumulateTransactions)

	rogueNode := RogueNode {
		internalNode:node,
		transactionCreationFunction:createTransactionCorrect,
		signChainTransactionsFunction:signChainTransactionsBadSignature,
	}

	return &rogueNode
}
