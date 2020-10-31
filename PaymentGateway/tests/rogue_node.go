package tests

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/rs/xid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/horizon"
	"paidpiper.com/payment-gateway/node"
)

type RogueNode struct {
	internalNode                  node.PPNode
	transactionCreationFunction   func(*RogueNode, context.Context, common.TransactionAmount, common.TransactionAmount, common.TransactionAmount, string) (common.PaymentTransactionReplacing, error)
	signChainTransactionsFunction func(*RogueNode, context.Context, *common.PaymentTransactionReplacing, *common.PaymentTransactionReplacing) error
}

// func (r *RogueNode) AddPendingServicePayment(context context.Context, serviceSessionId string, amount common.TransactionAmount) {
// 	r.internalNode.AddPendingServicePayment(context, serviceSessionId, amount)
// }

//
//func (r *RogueNode) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
//	r.internalNode.SetAccumulatingTransactionsMode(accumulateTransactions)
//}

func (r *RogueNode) CreatePaymentRequest(context context.Context, serviceSessionId string) (common.PaymentRequest, error) {

	return r.CreatePaymentRequest(context, serviceSessionId)
}

func (r *RogueNode) CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string, serviceSessionId string) (common.PaymentTransactionReplacing, error) {

	return r.transactionCreationFunction(r, context, totalIn, fee, totalOut, sourceAddress)
}

func createTransactionCorrect(r *RogueNode, context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {
	return r.internalNode.CreateTransaction(context, totalIn, fee, totalOut, sourceAddress, xid.New().String())
}

func (r *RogueNode) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {
	return r.internalNode.SignTerminalTransactions(context, creditTransactionPayload)
}

func (r *RogueNode) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {
	return r.signChainTransactionsFunction(r, context, creditTransactionPayload, debitTransactionPayload)
}

func signChainTransactionsNoError(r *RogueNode, context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {
	return r.internalNode.SignChainTransactions(context, creditTransactionPayload, debitTransactionPayload)
}

func (r *RogueNode) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) error {
	return r.internalNode.CommitServiceTransaction(context, transaction, pr)
}

func (r *RogueNode) CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error {
	return r.internalNode.CommitPaymentTransaction(context, transactionPayload)
}

func (r *RogueNode) GetAddress() string {
	return ""
	//return r.internalNode.GetAddress()
}

func createTransactionIncorrectSequence(r *RogueNode, context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {
	transaction, err := r.internalNode.CreateTransaction(context, totalIn, fee, totalOut, sourceAddress, xid.New().String())

	if err != nil {
		panic("unexpected error creating transaction")
	}

	payTrans := transaction.GetPaymentTransaction()
	refTrans := transaction.GetReferenceTransaction()

	if (refTrans == common.PaymentTransaction{}) {
		return transaction, err
	}

	payTransStellarWrapper,_ := txnbuild.TransactionFromXDR(payTrans.XDR)
	payTransStellar,_ := payTransStellarWrapper.Transaction()

	refTransStellarWrapper,_ := txnbuild.TransactionFromXDR(refTrans.XDR)
	refTransStellar,_ := refTransStellarWrapper.Transaction()

	account := payTransStellar.SourceAccount()
	paySequenceNumber,_ := account.GetSequenceNumber()
	account = refTransStellar.SourceAccount()
	refSequenceNumber,_ := account.GetSequenceNumber()

	if (paySequenceNumber != refSequenceNumber) {
		panic("sequence numbers are already different, unexpected")
	}

	op := payTransStellar.Operations()[0]
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

	sourceAccount := payTransStellar.SourceAccount()

	seq,_ := payment.SourceAccount.GetSequenceNumber()

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: sourceAccount.AccountID,
			Sequence:  sourceAccount.Sequence,
		},
		BaseFee: 200,
		Timebounds: txnbuild.NewTimeout(600),
		Operations: []txnbuild.Operation{&txnbuild.Payment{
			Destination: payment.Destination,
			Amount:      payment.Amount,
			Asset: txnbuild.CreditAsset{
				Code:   common.PPTokenAssetName,
				Issuer: common.PPTokenIssuerAddress,
			},
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: payment.SourceAccount.GetAccountID(),
				Sequence:  seq,
			},
		},
		},
	})
	//	build.SourceAccount{payTransStellar.SourceAccount.GetAccountID()},
	//	build.AutoSequence{common.CreateStaticSequence(uint64(refSequenceNumber + 1))},
	//	build.Payment(
	//		build.SourceAccount{payment.SourceAccount.GetAccountID()},
	//		build.Destination{payment.Destination},
	//		build.NativeAmount{payment.Amount}	),
	//)

	if err != nil {
		panic("unexpected error - transaction injection")
	}

	//tx.Mutate(build.TestNetwork)


	//txe, err := tx.TxEnvelope()
	//
	//if err != nil {
	//	panic("unexpected error - envelope")
	//}

	xdr, err := tx.Base64()

	transaction.UpdateTransactionXDR(xdr)

	return transaction, nil
}

func signChainTransactionsBadSignature(r *RogueNode, context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()
	debitTransaction := debitTransactionPayload.GetPaymentTransaction()

	kp, err := keypair.Random()

	creditWrapper, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)

	credit, _ := creditWrapper.Transaction()

	if err != nil {
		return errors.New("Transaction deser error")
	}

	debitWrapper, err := txnbuild.TransactionFromXDR(debitTransaction.XDR)
	debit,_ := debitWrapper.Transaction()

	if err != nil {
		return errors.New("Transaction parse error")
	}

	credit,err = credit.Sign(creditTransaction.StellarNetworkToken,kp)

	if err != nil {
		return errors.New("Failed to sign transaction")
	}

	debit,err = debit.Sign(debitTransaction.StellarNetworkToken, kp)

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

func CreateRogueNode_NonidenticalSequenceNumbers(address string, seed string, accumulateTransactions bool) node.PPNode {

	horizon := horizon.NewHorizon()

	node,_ := node.CreateNode(horizon, address, seed, accumulateTransactions,0,600)

	rogueNode := RogueNode{
		internalNode:                  node,
		transactionCreationFunction:   createTransactionIncorrectSequence,
		signChainTransactionsFunction: signChainTransactionsNoError,
	}

	return &rogueNode
}

func CreateRogueNode_BadSignature(address string, seed string, accumulateTransactions bool) node.PPNode {

	horizon := horizon.NewHorizon()

	node,_ := node.CreateNode(horizon, address, seed, accumulateTransactions,0,600)

	rogueNode := RogueNode{
		internalNode:                  node,
		transactionCreationFunction:   createTransactionCorrect,
		signChainTransactionsFunction: signChainTransactionsBadSignature,
	}

	return &rogueNode
}
