package tests

import (
	"context"

	"github.com/go-errors/errors"
	"github.com/rs/xid"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/regestry"
	"paidpiper.com/payment-gateway/torclient"

	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/local"
	"paidpiper.com/payment-gateway/node/proxy"
	"paidpiper.com/payment-gateway/root"
)

type RogueNode struct {
	node.PPNode
	transactionCreationFunction  func(*RogueNode, context.Context, *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error)
	SignChainTransactionFunction func(r *RogueNode, context context.Context, cmd *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error)
}

func (r *RogueNode) GetFee() uint32 {
	return 0
}

// func (r *RogueNode) AddPendingServicePayment(context context.Context, serviceSessionId string, amount models.TransactionAmount) {
// 	r.internalNode.AddPendingServicePayment(context, serviceSessionId, amount)
// }

//
//func (r *RogueNode) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
//	r.internalNode.SetAccumulatingTransactionsMode(accumulateTransactions)
//}

func (r *RogueNode) SignChainTransaction(ctx context.Context, command *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error) {
	if r.SignChainTransactionFunction != nil {
		return r.SignChainTransactionFunction(r, ctx, command)

	}
	return r.PPNode.SignChainTransaction(ctx, command)
}

func (r *RogueNode) CreateTransaction(context context.Context, request *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {

	return r.transactionCreationFunction(r, context, request)
}

//TODO REMOVE IF TEST IS CORRECT
func createTransactionCorrect(r *RogueNode, context context.Context, request *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {

	request.ServiceSessionId = xid.New().String()

	return r.CreateTransaction(context, request)
}

func createTransactionIncorrectSequence(r *RogueNode, context context.Context, request *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {
	res, err := r.CreateTransaction(context, request)

	if err != nil {
		panic("unexpected error creating transaction")
	}

	payTrans := res.Transaction.PendingTransaction
	refTrans := res.Transaction.ReferenceTransaction

	if refTrans == nil {
		return res, err
	}

	payTransStellarWrapper, _ := payTrans.XDR.TransactionFromXDR()
	payTransStellar, _ := payTransStellarWrapper.Transaction()

	refTransStellarWrapper, _ := refTrans.XDR.TransactionFromXDR()
	refTransStellar, _ := refTransStellarWrapper.Transaction()

	account := payTransStellar.SourceAccount()
	paySequenceNumber, _ := account.GetSequenceNumber()
	account = refTransStellar.SourceAccount()
	refSequenceNumber, _ := account.GetSequenceNumber()

	if paySequenceNumber != refSequenceNumber {
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

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: sourceAccount.AccountID,
			Sequence:  sourceAccount.Sequence,
		},
		BaseFee:    200,
		Timebounds: txnbuild.NewTimeout(600),
		Operations: []txnbuild.Operation{&txnbuild.Payment{
			Destination: payment.Destination,
			Amount:      payment.Amount,
			Asset: txnbuild.CreditAsset{
				Code:   models.PPTokenAssetName,
				Issuer: models.PPTokenIssuerAddress,
			},
			SourceAccount: payment.SourceAccount,
		},
		},
	})
	//	build.SourceAccount{payTransStellar.SourceAccount.GetAccountID()},
	//	build.AutoSequence{models.CreateStaticSequence(uint64(refSequenceNumber + 1))},
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

	xdr, _ := tx.Base64()

	res.Transaction.PendingTransaction.XDR = (models.NewXDR(xdr))

	return &models.CreateTransactionResponse{
		Transaction: res.Transaction,
	}, nil
}

func SignChainTransactionBadSignature(r *RogueNode, context context.Context, cmd *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error) {

	creditTransaction := cmd.Credit.PendingTransaction
	debitTransaction := cmd.Debit.PendingTransaction

	kp, _ := keypair.Random()

	creditWrapper, err := creditTransaction.XDR.TransactionFromXDR()

	credit, _ := creditWrapper.Transaction()

	if err != nil {
		return nil, errors.New("Transaction deser error")
	}

	debitWrapper, err := debitTransaction.XDR.TransactionFromXDR()
	debit, _ := debitWrapper.Transaction()

	if err != nil {
		return nil, errors.New("Transaction parse error")
	}

	credit, err = credit.Sign(creditTransaction.StellarNetworkToken, kp)

	if err != nil {
		return nil, errors.New("Failed to sign transaction")
	}

	debit, err = debit.Sign(debitTransaction.StellarNetworkToken, kp)

	if err != nil {
		return nil, errors.New("Failed to sign transaction")
	}

	base64, err := credit.Base64()
	if err != nil {
		return nil, err
	}
	creditTransaction.XDR = models.NewXDR(base64)
	if err != nil {
		return nil, errors.New("Transaction envelope error")
	}

	cmd.Credit.PendingTransaction.XDR = creditTransaction.XDR

	base64, err = debit.Base64()
	if err != nil {
		return nil, err
	}
	debitTransaction.XDR = models.NewXDR(base64)

	if err != nil {
		return nil, errors.New("Transaction envelope error")
	}

	cmd.Debit.PendingTransaction.XDR = debitTransaction.XDR

	return &models.SignChainTransactionResponse{
		Credit: cmd.Credit,
		Debit:  cmd.Debit,
	}, nil
}

func CreateRogueNode_NonidenticalSequenceNumbers(address string, seed string, accumulateTransactions bool) (node.PPNode, error) {

	rootClient, err := root.CreateRootApiFactory(true)(seed, 600)
	if err != nil {
		return nil, err
	}
	commandClientFactory := func(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler) {
		return nil, nil
	}
	commodityManager := commodity.New()
	paymentManager := regestry.NewPaymentManagerRegestry(
		commodityManager,
		client.New(rootClient),
		commandClientFactory,
		torclient.NewTorClient(""),
	)
	factory := func(cmd *models.UtilityCommand) local.CallBacker {
		return nil
	}
	node, _ := local.New(rootClient, paymentManager, factory, config.NodeConfig{
		AutoFlushPeriod:        0,
		AsyncMode:              true,
		AccumulateTransactions: accumulateTransactions,
	})

	rogueNode := RogueNode{
		PPNode:                      node,
		transactionCreationFunction: createTransactionIncorrectSequence,
	}

	return &rogueNode, nil
}

func CreateRogueNode_BadSignature(address string, seed string, accumulateTransactions bool) (node.PPNode, error) {

	//	horizon := horizon.NewHorizon()
	rootClient, err := root.CreateRootApiFactory(true)(seed, 600)
	if err != nil {
		return nil, err
	}

	commodityManager := commodity.New()
	commandClientFactory := func(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler) {
		return nil, nil
	}
	paymentManager := regestry.NewPaymentManagerRegestry(
		commodityManager,
		client.New(rootClient),
		commandClientFactory,
		torclient.NewTorClient(""),
	)
	factory := func(cmd *models.UtilityCommand) local.CallBacker {
		return nil
	}
	node, _ := local.New(rootClient, paymentManager, factory, config.NodeConfig{
		AutoFlushPeriod:        0,
		AsyncMode:              true,
		AccumulateTransactions: accumulateTransactions,
	})

	rogueNode := RogueNode{
		PPNode:                       node,
		transactionCreationFunction:  createTransactionCorrect,
		SignChainTransactionFunction: SignChainTransactionBadSignature,
	}

	return &rogueNode, nil
}
