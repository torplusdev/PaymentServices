package node

import (
	"context"
	"github.com/go-errors/errors"
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"paidpiper.com/payment-gateway/common"
	"strconv"
	"time"
)

const nodeTransactionFee = 10

type serviceUsageCredit struct {
	amount  common.TransactionAmount
	updated time.Time
}

type Node struct {
	Address                      string
	secretSeed                   string
	client                       *horizon.Client
	accumulatingTransactionsMode bool
	transactionFee               common.TransactionAmount
	pendingPayment               map[string]serviceUsageCredit
	activeTransactions           map[string]common.PaymentTransaction
	tracer 						 trace.Tracer
}

type PPNode interface {
	CreateTransaction(context context.Context,totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error)
	SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error
	SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error
	CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (ok bool, err error)
	CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error)
}

func CreateNode(client *horizon.Client, address string, seed string, accumulateTransactions bool) *Node {
	node := Node{
		Address:                      address,
		secretSeed:                   seed,
		client:                       client,
		transactionFee:               nodeTransactionFee,
		pendingPayment:               make(map[string]serviceUsageCredit),
		activeTransactions:           make(map[string]common.PaymentTransaction),
		accumulatingTransactionsMode: accumulateTransactions,
		tracer:						  global.Tracer("node"),
	}

	return &node
}

type NodeManager interface {
	GetNodeByAddress(address string) PPNode
}

func (n *Node) AddPendingServicePayment(context context.Context, serviceSessionId string, amount common.TransactionAmount) error {

	_,span :=n.tracer.Start(context,"node-AddPendingServicePayment " + n.Address)
	defer span.End()

	if n.pendingPayment[serviceSessionId].updated.IsZero() {
		n.pendingPayment[serviceSessionId] = serviceUsageCredit{
			amount:  amount,
			updated: time.Now(),
		}
	} else {
		n.pendingPayment[serviceSessionId] = serviceUsageCredit{
			amount:  n.pendingPayment[serviceSessionId].amount + amount,
			updated: time.Now(),
		}
	}

	return nil
}

func (n *Node) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	n.accumulatingTransactionsMode = accumulateTransactions
}

func (n *Node) GetPendingPayment(address string) (common.TransactionAmount, time.Time, error) {

	if n.pendingPayment[address].updated.IsZero() {
		return 0, time.Unix(0, 0), errors.Errorf("PaymentDestinationAddress not found: " + address)
	}

	return n.pendingPayment[address].amount, n.pendingPayment[address].updated, nil
}

func (n *Node) CreatePaymentRequest(context context.Context, serviceSessionId string) (common.PaymentRequest, error) {

	_,span :=n.tracer.Start(context,"node-CreatePaymentRequest "+ n.Address)
	defer span.End()

	if n.pendingPayment[serviceSessionId].updated.IsZero() {
		return common.PaymentRequest{}, nil
	} else {
		pr := common.PaymentRequest{
			ServiceSessionId: serviceSessionId,
			Address:          n.Address,
			Amount:           n.pendingPayment[serviceSessionId].amount,
			Asset:            "XLM",
			ServiceRef:       "test"}

		return pr, nil
	}
}

func (n *Node) GetAddress() string {
	return n.Address
}

func (n *Node) createTransactionWrapper(internalTransaction common.PaymentTransaction) (common.PaymentTransactionReplacing, error) {
	return common.CreateReferenceTransaction(internalTransaction, n.activeTransactions[internalTransaction.PaymentSourceAddress])
}


func (n *Node) CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {

	_,span :=n.tracer.Start(context,"node-CreateTransaction " + n.Address)
	defer span.End()

	//Verify fee
	if totalIn-totalOut != fee {
		log.Fatal("Incorrect fee requested")
	}

	var amount = totalIn

	transactionPayload, err := n.createTransactionWrapper(common.PaymentTransaction {
		TransactionSourceAddress:  n.Address,
		ReferenceAmountIn:         totalIn,
		AmountOut:                 totalOut,
		PaymentSourceAddress:      sourceAddress,
		PaymentDestinationAddress: n.Address,
	})

	if err != nil {
		log.Fatal("Error creating transaction wrapper: " + err.Error())
		return common.PaymentTransactionReplacing{}, err
	}

	var sequenceProvider build.SequenceProvider

	// If this is the first transaction for the node+client pair and there's no reference transaction
	if transactionPayload.GetReferenceTransaction() == (common.PaymentTransaction{}) {
		sequenceProvider = build.AutoSequence{n.client}

	} else {
		referenceTransactionPayload := transactionPayload.GetReferenceTransaction()

		referenceTransaction,err := txnbuild.TransactionFromXDR(referenceTransactionPayload.XDR)

		if err != nil {
			return common.PaymentTransactionReplacing{}, errors.Errorf("Error deserializing XDR transaction: %s",err.Error())
		}

		referenceSequenceNumber, err := referenceTransaction.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()

		_ = referenceSequenceNumber
		sequenceProvider = build.AutoSequence{common.CreateStaticSequence(uint64(referenceSequenceNumber - 1))}
	}

	tx, err := build.Transaction(
		build.SourceAccount{n.Address},
		build.AutoSequence{sequenceProvider},
		build.Payment(
			build.SourceAccount{sourceAddress},
			build.Destination{n.Address},
			build.NativeAmount{strconv.FormatUint(uint64(amount), 10)},
		),
	)

	if err != nil {
		log.Fatal("Error creating transaction: " + err.Error())
		return common.PaymentTransactionReplacing{}, err
	}
	if n.client.URL == "https://horizon-testnet.stellar.org" {
		tx.Mutate(build.TestNetwork)
	} else {
		tx.Mutate(build.DefaultNetwork)
	}

	txe, err := tx.Envelope()

	if err != nil {
		log.Fatal("Error generating transaction envelope: " + err.Error())
		return common.PaymentTransactionReplacing{}, err
	}

	xdr, err := txe.Base64()

	if err != nil {
		log.Fatal("Error serializing transaction: " + err.Error())
		return common.PaymentTransactionReplacing{}, err
	}

	transactionPayload.UpdateTransactionXDR(xdr)

	// TODO: This should be configurable via profile/strategy
	transactionPayload.UpdateStellarToken(build.TestNetwork.Passphrase)

	return transactionPayload,nil
}

func (n *Node) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {

	_,span :=n.tracer.Start(context,"node-SignTerminalTransactions " + n.Address)
	defer span.End()

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()

	// Validate
	if creditTransaction.PaymentDestinationAddress != n.Address {
		log.Fatal("Transaction destination is incorrect ", creditTransaction.PaymentDestinationAddress)
		return errors.New("Transaction destination error")
	}

	kp, err := keypair.ParseFull(n.secretSeed)

	if err != nil {
		log.Fatal("Error parsing keypair: ", err.Error())
		return err
	}

	t, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)
	t.Network = creditTransaction.StellarNetworkToken

	if err != nil {
		log.Fatal("Error parsing transaction: ", err.Error())
		return err
	}

	err = t.Sign(kp)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return err
	}

	creditTransaction.XDR, err = t.Base64()

	if err != nil {
		log.Fatal("Error writing transaction envelope: " + err.Error())
		return err
	}

	creditTransactionPayload.UpdateTransactionXDR(creditTransaction.XDR)

	return nil
}

func (n *Node) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	_,span :=n.tracer.Start(context,"node-SignChainTransactions " + n.Address)
	defer span.End()

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()
	debitTransaction := debitTransactionPayload.GetPaymentTransaction()

	kp, err := keypair.ParseFull(n.secretSeed)

	if err != nil {
		log.Fatal("Error parsing keypair: ", err.Error())
		return err
	}

	credit, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)
	credit.Network = creditTransaction.StellarNetworkToken

	if err != nil {
		log.Fatal("Error parsing credit transaction: ", err.Error())
		return err
	}

	debit, err := txnbuild.TransactionFromXDR(debitTransaction.XDR)
	debit.Network = debitTransaction.StellarNetworkToken

	if err != nil {
		log.Fatal("Error parsing debit transaction: ", err.Error())
		return err
	}

	err = credit.Sign(kp)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return err
	}

	err = debit.Sign(kp)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return err
	}

	creditTransaction.XDR, err = credit.Base64()

	if err != nil {
		log.Fatal("Error writing credit transaction envelope: " + err.Error())
		return err
	}

	creditTransactionPayload.UpdateTransactionXDR(creditTransaction.XDR)

	debitTransaction.XDR, err = debit.Base64()

	if err != nil {
		log.Fatal("Error writing debit transaction envelope: " + err.Error())
		return err
	}

	debitTransactionPayload.UpdateTransactionXDR(debitTransaction.XDR)

	return nil
}

func (n *Node) verifyTransactionSignatures(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {

	_,span :=n.tracer.Start(context,"node-verifyTransactionSignatures " + n.Address )
	defer span.End()


	transaction := transactionPayload.GetPaymentTransaction()

	// Deserialize transactions
	t, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return false, errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	if t.SourceAccount.GetAccountID() != n.Address {
		return false, errors.Errorf("Incorrect transaction source account")
	}

	t.Network = transaction.StellarNetworkToken

	var payerAccount string = ""
	for _, op := range t.Operations {
		xdrOp, _ := op.BuildXDR()

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment := &txnbuild.Payment{}

			err = payment.FromXDR(xdrOp)

			if err != nil {
				return false, errors.Errorf("Error converting operation")
			}

			payerAccount = payment.SourceAccount.GetAccountID()
		default:
			return false, errors.Errorf("Unexpected operation during verification")
		}
	}

	payerVerified := false
	sourceVerified := false

	for _, signature := range t.TxEnvelope().Signatures {
		from, err := keypair.ParseAddress(payerAccount)

		if err != nil {
			return false, errors.Errorf("Error in operation source address")
		}

		bytes, err := t.Hash()

		if err != nil {
			return false, errors.Errorf("Error during tx hashing")
		}

		err = from.Verify(bytes[:], signature.Signature)

		if err == nil {
			payerVerified = true
		}

		own, err := keypair.ParseFull(n.secretSeed)
		if err != nil {
			return false, errors.Errorf("Error creating key")
		}

		err = own.Verify(bytes[:], signature.Signature)

		if err == nil {
			sourceVerified = true
		}
	}

	if !payerVerified {
		return false, errors.Errorf("Error validating payer signature")
	}

	if !sourceVerified {
		return false, errors.Errorf("Error validating source signature")
	}

	//TODO: Validate timebounds

	return true, nil
}

func (n *Node) CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {

	_,span :=n.tracer.Start(context,"node-CommitPaymentTransaction "  + n.Address)
	defer span.End()

	ok = false
	transaction := transactionPayload.GetPaymentTransaction()

	t, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		log.Error("Error during transaction deser: " + e.Error())
	}
	_ = t

	ok, err = n.verifyTransactionSignatures(context, transactionPayload)

	if !ok || err != nil {
		return false, err
	}

	if !n.accumulatingTransactionsMode {
		res, err := n.client.SubmitTransaction(transaction.XDR)

		if err != nil {
			log.Error("Error submitting transaction: " + err.Error())
			return false, err
		}

		log.Debug("Transaction submitted: " + res.Result)
	} else {
		// Save transaction
		n.activeTransactions[transaction.PaymentSourceAddress] = *transaction
	}

	return true, nil
}

func (n *Node) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {

	_,span :=n.tracer.Start(context,"node-CommitServiceTransaction "  + n.Address)
	defer span.End()

	n.pendingPayment[pr.ServiceSessionId] = serviceUsageCredit{
		amount:  n.pendingPayment[pr.ServiceSessionId].amount - transaction.GetPaymentTransaction().AmountOut,
		updated: time.Now(),
	}

	n.CommitPaymentTransaction(context, transaction)

	return true, nil
}

func (n *Node) FlushTransactions(context context.Context) (map[string]interface{},error) {

	_,span :=n.tracer.Start(context,"node-FlushTransactions " + n.Address)
	defer span.End()

	resultsMap := make(map[string]interface{})

	for a,t := range n.activeTransactions {

		txSuccess,err := horizonclient.DefaultTestNetClient.SubmitTransactionXDR(t.XDR)

		resultsMap[a] = txSuccess.TransactionSuccessToString()

		if err != nil {
			log.Errorf("Error submitting transaction for %v: %w",a,err)
			resultsMap[a] =  err
		}
	}

	return resultsMap,nil
}
