package node

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/go-errors/errors"
	"github.com/rs/xid"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/horizon"
)

const nodeTransactionFee = 10

type serviceUsageCredit struct {
	amount  common.TransactionAmount
	updated time.Time
}

type Node struct {
	Address                      string
	secretSeed                   string
	horizon                      *horizon.Horizon
	accumulatingTransactionsMode bool
	transactionFee               common.TransactionAmount
	paymentRegistry              paymentRegistry
	//pendingPayment               map[string]serviceUsageCredit
	//activeTransactions           map[string]common.PaymentTransaction
	tracer         			trace.Tracer
	lastSequenceId 			xdr.SequenceNumber
	autoFlushPeriod 		time.Duration
	sequenceMux            	sync.Mutex
	flushMux			   	sync.Mutex
}

type PPNode interface {
	CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string, serviceSessionId string) (common.PaymentTransactionReplacing, error)
	SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error
	SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error
	CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (ok bool, err error)
	CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error)
	GetAddress() (string)
}

type PPTransactionManager interface {
	GetTransactions() []common.PaymentTransaction
	GetTransaction(sessionId string) common.PaymentTransaction
	FlushTransactions(context context.Context) (map[string]interface{}, error)
}

type PPPaymentRequestProvider interface {
	CreatePaymentRequest(context context.Context, amount common.TransactionAmount, asset string, serviceType string) (common.PaymentRequest, error)
}

func CreateNode(horizon *horizon.Horizon, address string, seed string, accumulateTransactions bool, autoFlushPeriodSeconds time.Duration) *Node {
	node := Node{
		Address:         address,
		secretSeed:      seed,
		horizon:         horizon,
		transactionFee:  nodeTransactionFee,
		paymentRegistry: createPaymentRegistry(address),
		//pendingPayment:               make(map[string]serviceUsageCredit),
		//activeTransactions:           make(map[string]common.PaymentTransaction),
		accumulatingTransactionsMode: accumulateTransactions,
		tracer:                       common.CreateTracer("node"),
		autoFlushPeriod: autoFlushPeriodSeconds,
		flushMux: sync.Mutex{},
		sequenceMux: sync.Mutex{},
	}

	go func() {
		if (node.autoFlushPeriod > 0) {
			for now := range time.Tick(node.autoFlushPeriod) {
				log.Debugf("Node %s autoflush tick: %s",address,now.String())
				_,err := node.FlushTransactions(context.Background())

				if err != nil {
					log.Errorf("Error during autoflush of node %s: %s",err.Error())
				}
			}
		} else {
			log.Debug("Node %s: autoflush disabled.")
		}
	}()

	return &node
}

type NodeManager interface {
	GetNodeByAddress(address string) PPNode
}

func (n *Node) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	n.accumulatingTransactionsMode = accumulateTransactions
}

//func (n *Node) GetPendingPayment(address string) (common.TransactionAmount, time.Time, error) {
//
//	if n.pendingPayment[address].updated.IsZero() {
//		return 0, time.Unix(0, 0), errors.Errorf("PaymentDestinationAddress not found: " + address)
//	}
//
//	return n.pendingPayment[address].amount, n.pendingPayment[address].updated, nil
//}
func (n *Node) CreatePaymentRequest(context context.Context, amount common.TransactionAmount, asset string, serviceType string) (common.PaymentRequest, error) {

	_, span := n.tracer.Start(context, "node-CreatePaymentRequest "+n.Address)
	defer span.End()

	serviceSessionId := xid.New().String()

	n.paymentRegistry.AddServiceUsage(serviceSessionId, amount)

	pr := common.PaymentRequest{
		ServiceSessionId: serviceSessionId,
		Address:          n.Address,
		Amount:           amount,
		Asset:            asset,
		ServiceRef:       serviceType}

	return pr, nil
}

func (n *Node) GetAddress() string {
	return n.Address
}

func (n *Node) createTransactionWrapper(internalTransaction common.PaymentTransaction) (common.PaymentTransactionReplacing, error) {

	return common.CreateReferenceTransaction(internalTransaction, n.paymentRegistry.getActiveTransaction(internalTransaction.PaymentSourceAddress))
}

func (n *Node) CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string, serviceSessionId string) (common.PaymentTransactionReplacing, error) {

	_, span := n.tracer.Start(context, "node-CreateTransaction "+n.Address)
	defer span.End()

	//Verify fee
	if totalIn-totalOut != fee {
		return common.PaymentTransactionReplacing{}, errors.Errorf("Incorrect fee requested: %d != %d", totalIn-totalOut, fee)
	}

	span.SetAttributes(core.KeyValue{Key: "payment.source-address", Value: core.String(sourceAddress)})
	span.SetAttributes(core.KeyValue{Key: "payment.destination-address", Value: core.String(n.Address)})
	span.SetAttributes(core.KeyValue{Key: "payment.amount-in", Value: core.Uint32(totalIn)})
	span.SetAttributes(core.KeyValue{Key: "payment.amount-out", Value: core.Uint32(totalOut)})

	transactionPayload, err := n.createTransactionWrapper(common.PaymentTransaction{
		TransactionSourceAddress:  n.Address,
		ReferenceAmountIn:         totalIn,
		AmountOut:                 totalOut,
		PaymentSourceAddress:      sourceAddress,
		PaymentDestinationAddress: n.Address,
		ServiceSessionId:          serviceSessionId,
	})

	var amount = transactionPayload.PendingTransaction.ReferenceAmountIn

	if err != nil {
		//log.Fatal("Error creating transaction wrapper: " + err.Error())
		return common.PaymentTransactionReplacing{}, errors.Errorf("Error creating transaction wrapper: %v", err)
	}


	//build.SequenceProvider
	var sequenceProvider int64

	sourceAccountDetail,err := n.horizon.GetAccount(sourceAddress)
	_ = sourceAccountDetail

	if err != nil {
		return common.PaymentTransactionReplacing{}, errors.Errorf("Error getting source account data: %s", err.Error())
	}



	// Uninitialized
	if n.lastSequenceId == 0 {

		account, err := n.horizon.GetAccount(n.Address)

		seq, err := account.GetSequenceNumber()
		//seq,err := n.horizon.GetAccount(n.Address).SequenceForAccount(n.Address)

		if err != nil {
			return common.PaymentTransactionReplacing{}, errors.Errorf("Error retrieving sequence number: %s", err.Error())
		}

		n.lastSequenceId = xdr.SequenceNumber(seq)
	}

	// If this is the first transaction for the node+client pair and there's no reference transaction
	if transactionPayload.GetReferenceTransaction() == (common.PaymentTransaction{}) {
		n.sequenceMux.Lock()
		defer n.sequenceMux.Unlock()
		sequenceProvider = int64(n.lastSequenceId) // build.AutoSequence{common.CreateStaticSequence(uint64(n.lastSequenceId - 1))}
		n.lastSequenceId = n.lastSequenceId + 1
	} else {
		referenceTransactionPayload := transactionPayload.GetReferenceTransaction()

		referenceTransactionWrapper, err := txnbuild.TransactionFromXDR(referenceTransactionPayload.XDR)

		if err != nil {
			return common.PaymentTransactionReplacing{}, errors.Errorf("Error deserializing XDR transaction: %s", err.Error())
		}

		referenceTransaction, result := referenceTransactionWrapper.Transaction()
		if !result {
			return common.PaymentTransactionReplacing{}, errors.Errorf("Error deserializing XDR transaction (GenericTransaction)")
		}

		account := referenceTransaction.SourceAccount()
		referenceSequenceNumber, err := account.GetSequenceNumber()

		sequenceProvider = referenceSequenceNumber-1 //build.AutoSequence{common.CreateStaticSequence(uint64(referenceSequenceNumber - 1))}
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: n.Address,
			Sequence:  sequenceProvider,
		},
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{&txnbuild.Payment{
			Destination: n.Address,
			Amount:      common.PPTokenToString(amount),
			Asset: txnbuild.CreditAsset{
				Code:   common.PPTokenAssetName,
				Issuer: common.PPTokenIssuerAddress,
			},
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: sourceAddress,
				Sequence:  0,
			},
		}},
		BaseFee:    200,
		Timebounds: txnbuild.NewTimeout(300),
	})

	//tx, err := build.Transaction(
	//	build.SourceAccount{n.Address},
	//	build.AutoSequence{sequenceProvider},
	//	build.Payment(
	//		build.SourceAccount{sourceAddress},
	//		build.Destination{n.Address},
	//		build.CreditAmount{
	//			Code:   ,
	//			Issuer: ,
	//			Amount: ,
	//		},
	//	),
	//)

	if err != nil {
		return common.PaymentTransactionReplacing{}, errors.Errorf("Error creating transaction: %v", err)
	}

	/*if n.client.URL == "https://horizon-testnet.stellar.org" {
		tx.Mutate(build.TestNetwork)
	} else {
		tx.Mutate(build.DefaultNetwork)
	}
	*/


	//err = n.horizon.AddTransactionToken(tx)
	//
	//if err != nil {
	//	return common.PaymentTransactionReplacing{}, errors.Errorf("Error adding transaction token: %v", err)
	//}

	xdr,err := tx.Base64()


	if err != nil {
		return common.PaymentTransactionReplacing{}, errors.Errorf("Error serializing transaction: %v", err)
	}

	transactionPayload.UpdateTransactionXDR(xdr)

	// TODO: This should be configurable via profile/strategy
	transactionPayload.UpdateStellarToken(network.TestNetworkPassphrase)

	transactionPayload.ToSpanAttributes(span, "credit")
	return transactionPayload, nil
}

func (n *Node) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-SignTerminalTransactions "+n.Address)
	defer span.End()

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()

	// Validate
	if creditTransaction.PaymentDestinationAddress != n.Address {
		return errors.Errorf("Transaction destination is incorrect: %s", creditTransaction.PaymentDestinationAddress)
	}

	kp, err := keypair.ParseFull(n.secretSeed)

	if err != nil {
		return errors.Errorf("Error parsing keypair: %v", err)
	}

	transactionWrapper, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)

	if err != nil {
		return errors.Errorf("Error parsing transaction: %v", err)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Transaction destination is incorrect (GenericTransaction)")
	}

	t,err = t.Sign(network.TestNetworkPassphrase, kp)

	if err != nil {
		return errors.Errorf("Failed to signed transaction: %v", err)
	}

	creditTransaction.XDR, err = t.Base64()

	if err != nil {
		return errors.Errorf("Error writing transaction envelope: %v", err)
	}

	creditTransactionPayload.UpdateTransactionXDR(creditTransaction.XDR)
	creditTransactionPayload.ToSpanAttributes(span, "credit")

	return nil
}

func (n *Node) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-SignChainTransactions "+n.Address)
	defer span.End()

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()
	debitTransaction := debitTransactionPayload.GetPaymentTransaction()

	kp, err := keypair.ParseFull(n.secretSeed)

	if err != nil {
		return errors.Errorf("Error parsing keypair: %v", err)
	}

	creditWrapper, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)

	credit, result := creditWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction (GenericTransaction)")
	}



	if err != nil {
		return errors.Errorf("Error parsing credit transaction: %v", err)
	}

	debitWrapper, err := txnbuild.TransactionFromXDR(debitTransaction.XDR)

	debit, result := debitWrapper.Transaction()

	if err != nil {
		return errors.Errorf("Error parsing debit transaction: %v", err)
	}

	credit,err = credit.Sign(creditTransaction.StellarNetworkToken,kp)

	if err != nil {
		log.Fatal("Failed to signed transaction")
		return err
	}

	debit,err = debit.Sign(debitTransaction.StellarNetworkToken,kp)

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

	creditTransactionPayload.ToSpanAttributes(span, "credit")
	debitTransactionPayload.ToSpanAttributes(span, "debit")
	return nil
}

func (n *Node) verifyTransactionSignatures(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {

	_, span := n.tracer.Start(context, "node-verifyTransactionSignatures "+n.Address)
	defer span.End()

	transaction := transactionPayload.GetPaymentTransaction()

	// Deserialize transactions
	transactionWrapper, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return false, errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t,result := transactionWrapper.Transaction()

	if !result {
		return false, errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	if t.SourceAccount().AccountID != n.Address {
		return false, errors.Errorf("Incorrect transaction source account")
	}
	//transaction.StellarNetworkToken

	var payerAccount string = ""
	for _, op := range t.Operations() {
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

	for _, signature := range t.Signatures() {
		from, err := keypair.ParseAddress(payerAccount)

		if err != nil {
			return false, errors.Errorf("Error in operation source address")
		}

		bytes, err := t.Hash(transaction.StellarNetworkToken)

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

	_, span := n.tracer.Start(context, "node-CommitPaymentTransaction "+n.Address)
	defer span.End()

	ok = false
	transaction := transactionPayload.GetPaymentTransaction()

	transactionWrapper, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return false, errors.Errorf("Error during transaction deser: %v", e)
	}

	t,result := transactionWrapper.Transaction()

	if !result {
		return false, errors.Errorf("Error during transaction deser: %v", e)
	}

	ok, err = n.verifyTransactionSignatures(context, transactionPayload)

	if !ok || err != nil {
		return false, err
	}

	if !n.accumulatingTransactionsMode {
		_, err := n.horizon.Client.SubmitTransaction(t)

		if err != nil {
			log.Error("Error submitting transaction: " + err.Error())
			return false, err
		}

		log.Debug("Transaction submitted: ")
	} else {
		n.paymentRegistry.saveTransaction(transaction.PaymentSourceAddress, transaction)
	}

	transactionPayload.ToSpanAttributes(span, "single")

	return true, nil
}

func (n *Node) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {

	_, span := n.tracer.Start(context, "node-CommitServiceTransaction "+n.Address)
	defer span.End()

	ok, err := n.CommitPaymentTransaction(context, transaction)

	if ok {
		err = n.paymentRegistry.reducePendingAmount(pr.ServiceSessionId, transaction.GetPaymentTransaction().AmountOut)
		return err == nil, err

	} else {
		return false, err
	}

	return true, nil
}

func (n *Node) GetTransactions() []common.PaymentTransaction {

	return n.paymentRegistry.getActiveTransactions()
}

func (n *Node) GetTransaction(sessionId string) common.PaymentTransaction {

	return n.paymentRegistry.getTransactionBySessionId(sessionId)
}

func (n *Node) FlushTransactions(context context.Context) (map[string]interface{}, error) {

	_, span := n.tracer.Start(context, "node-FlushTransactions "+n.Address)
	defer span.End()


	resultsMap := make(map[string]interface{})

	//TODO Sort transaction by sequence number and make sure to submit them only in sequence number order
	transactions := n.paymentRegistry.getActiveTransactions()

	n.flushMux.Lock()
	defer n.flushMux.Unlock()

	sort.Slice(transactions, func(i, j int) bool {
		transiWrapper, erri := txnbuild.TransactionFromXDR(transactions[i].XDR)

		if erri != nil {
			log.Errorf("Error converting transaction from xdr: %s", erri.Error())
		}

		transi, result := transiWrapper.Transaction()

		if !result {
			log.Errorf("Error converting transaction i from xdr (GenericTransaction)")
		}

		transjWrapper, errj := txnbuild.TransactionFromXDR(transactions[j].XDR)

		if errj != nil {
			log.Errorf("Error converting transaction from xdr: %s", errj.Error())
		}

		transj, result := transjWrapper.Transaction()

		if !result {
			log.Errorf("Error converting transaction j from xdr (GenericTransaction)")
		}

		account := transi.SourceAccount()
		seqi, erri := account.GetSequenceNumber()

		account = transj.SourceAccount()
		seqj, errj := account.GetSequenceNumber()

		if erri != nil {
			log.Errorf("Error getting sequence number transaction from xdr: %s", erri.Error())
		}

		if errj != nil {
			log.Errorf("Error converting transaction from xdr: %s", errj.Error())
		}

		return seqi < seqj
	})

	for a, t := range transactions {

		txSuccess, err := horizonclient.DefaultTestNetClient.SubmitTransactionXDR(t.XDR)

		resultsMap[t.TransactionSourceAddress] = txSuccess

		if err != nil {
			log.Errorf("Error submitting transaction for %v: %v", a, err)

			internalTransWrapper, err := txnbuild.TransactionFromXDR(t.XDR)

			if err != nil {
				log.Errorf("Error deserializing transaction for %v: %w", a, err)
			}

			internalTrans, result := internalTransWrapper.Transaction()

			if !result {
				log.Errorf("Error deserializing transaction (GenericTransaction)")
			}

			//n.verifyTransactionSignatures(context,t)

			account := internalTrans.SourceAccount()
			accountSeqNumber, _ := account.GetSequenceNumber()
			//transactionSeqNumber := &internalTrans.(*xdr.Transaction).SeqNum
			_ = accountSeqNumber

			_ = internalTrans
			resultsMap[t.TransactionSourceAddress] = err
		} else {
			n.paymentRegistry.completePayment(t.PaymentSourceAddress, t.ServiceSessionId)
		}

	}

	return resultsMap, nil
}
