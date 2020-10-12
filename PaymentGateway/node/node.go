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
	CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) error
	CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error
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

	log.SetLevel(log.InfoLevel)

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

	log.Infof("CreatePaymentRequest: Starting %d  %s/%s ",amount,asset,serviceType)

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

	log.Infof("CreateTransaction: Starting %s %d + %d = %d => %s ",sourceAddress,totalIn,fee,totalOut,n.Address)

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

		if err != nil {
			return common.PaymentTransactionReplacing{}, errors.Errorf("Error getting horizon account: %s", err.Error())
		}

		seq, err := account.GetSequenceNumber()

		log.Infof("Sequence number initialization: %d",seq)
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
		log.Infof("No reference transaction, assigning id %d and promoting",n.lastSequenceId)
		sequenceProvider = int64(n.lastSequenceId) + 7 // build.AutoSequence{common.CreateStaticSequence(uint64(n.lastSequenceId - 1))}
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
		log.Infof("Reference transaction found, assigning id %d",sequenceProvider)
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
		Timebounds: txnbuild.NewTimeout(common.TransactionTimeoutSeconds),
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

	log.Infof("CreateTransaction: Done %s => %s ",sourceAddress,n.Address)

	transactionPayload.ToSpanAttributes(span, "credit")
	return transactionPayload, nil
}

func (n *Node) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-SignTerminalTransactions "+n.Address)
	defer span.End()

	log.Infof("SignTerminalTransactions: starting %s => %s ",creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

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

	log.Infof("SignTerminalTransactions: done %s => %s ",creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransactionPayload.ToSpanAttributes(span, "credit")

	return nil
}

func (n *Node) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-SignChainTransactions "+n.Address)
	defer span.End()

	log.Infof("SignChainTransactions: started %s => %s ",creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransaction := creditTransactionPayload.GetPaymentTransaction()
	debitTransaction := debitTransactionPayload.GetPaymentTransaction()

	kp, err := keypair.ParseFull(n.secretSeed)

	if err != nil {
		return errors.Errorf("Error parsing keypair: %v", err)
	}

	creditWrapper, err := txnbuild.TransactionFromXDR(creditTransaction.XDR)

	if err != nil {
		return errors.Errorf("Error building transaction from XDR: %v", err)
	}

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

	log.Infof("SignChainTransactions: done %s => %s ",creditTransactionPayload.PendingTransaction.PaymentSourceAddress,
		creditTransactionPayload.PendingTransaction.PaymentDestinationAddress)

	creditTransactionPayload.ToSpanAttributes(span, "credit")
	debitTransactionPayload.ToSpanAttributes(span, "debit")
	return nil
}

func (n *Node) verifyTransactionSequence(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-verifyTransactionSequence"+n.Address)
	defer span.End()

	transaction := transactionPayload.GetPaymentTransaction()

	// Deserialize transactions
	transactionWrapper, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t,result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	nodeAccount, err := n.horizon.GetAccount(n.Address)
	if err != nil {
		return errors.Errorf("Error reading account: " + err.Error())
	}

	currentSequence,err := nodeAccount.GetSequenceNumber()
	if err != nil {
		return errors.Errorf("Error getting sequence: " + err.Error())
	}

	account := t.SourceAccount()
	transactionSequence, err := account.GetSequenceNumber()

	if transactionSequence <= currentSequence {
		log.Warn("Incorrect sequence detected, current account is at %d, transaction is %d",currentSequence,transactionSequence)
		return errors.Errorf("incorrect sequence detected")
	}

	log.Infof("verifyTransactionSequence finished successfully - account#:%d transaction#:%d",currentSequence,transactionSequence)
	return nil
}

func (n *Node) verifyTransactionSignatures(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-verifyTransactionSignatures "+n.Address)
	defer span.End()

	log.Infof("verifyTransactionSignatures started %s => %s", transactionPayload.PendingTransaction.PaymentSourceAddress,
		transactionPayload.PendingTransaction.PaymentDestinationAddress)

	transaction := transactionPayload.GetPaymentTransaction()

	// Deserialize transactions
	transactionWrapper, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t,result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	if t.SourceAccount().AccountID != n.Address {
		return errors.Errorf("Incorrect transaction source account")
	}
	//transaction.StellarNetworkToken

	var payerAccount string = ""
	for _, op := range t.Operations() {
		xdrOp, _ := op.BuildXDR()

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment := &txnbuild.Payment{}

			err := payment.FromXDR(xdrOp)

			if err != nil {
				return errors.Errorf("Error converting operation")
			}

			payerAccount = payment.SourceAccount.GetAccountID()
		default:
			return errors.Errorf("Unexpected operation during verification")
		}
	}

	payerVerified := false
	sourceVerified := false

	for _, signature := range t.Signatures() {
		from, err := keypair.ParseAddress(payerAccount)

		if err != nil {
			return errors.Errorf("Error in operation source address")
		}

		bytes, err := t.Hash(transaction.StellarNetworkToken)

		if err != nil {
			return errors.Errorf("Error during tx hashing")
		}

		err = from.Verify(bytes[:], signature.Signature)

		if err == nil {
			payerVerified = true
		}

		own, err := keypair.ParseFull(n.secretSeed)
		if err != nil {
			return errors.Errorf("Error creating key")
		}

		err = own.Verify(bytes[:], signature.Signature)

		if err == nil {
			sourceVerified = true
		}
	}

	if !payerVerified {
		return errors.Errorf("Error validating payer signature")
	}

	if !sourceVerified {
		return errors.Errorf("Error validating source signature")
	}

	log.Infof("verifyTransactionSequence finished successfully")

	//TODO: Validate timebounds

	return nil
}

func (n *Node) CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error {

	_, span := n.tracer.Start(context, "node-CommitPaymentTransaction "+n.Address)
	defer span.End()

	log.Infof("CommitPaymentTransaction started %s => %s", transactionPayload.PendingTransaction.PaymentSourceAddress,
		transactionPayload.PendingTransaction.PaymentDestinationAddress)

	transaction := transactionPayload.GetPaymentTransaction()

	transactionWrapper, e := txnbuild.TransactionFromXDR(transaction.XDR)

	if e != nil {
		return errors.Errorf("Error during transaction deser: %v", e)
	}

	t,result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error during transaction deser: %v", e)
	}

	e = n.verifyTransactionSequence(context,transactionPayload)

	if e != nil {
		log.Warn("Transaction verification failed (sequence)")
		return e
	}

	e = n.verifyTransactionSignatures(context, transactionPayload)

	if e != nil {
		log.Warn("Transaction verification failed (signatures)")
		return e
	}

	if !n.accumulatingTransactionsMode {
		_, err := n.horizon.Client.SubmitTransaction(t)

		if err != nil {
			log.Error("Error submitting transaction: " + err.Error())
			return err
		}

		log.Debug("Transaction submitted: ")
	} else {
		n.paymentRegistry.saveTransaction(transaction.PaymentSourceAddress, transaction)
	}

	log.Infof("CommitPaymentTransaction finished %s => %s", transactionPayload.PendingTransaction.PaymentSourceAddress,
		transactionPayload.PendingTransaction.PaymentDestinationAddress)

	transactionPayload.ToSpanAttributes(span, "single")

	return nil
}

func (n *Node) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) error {

	_, span := n.tracer.Start(context, "node-CommitServiceTransaction "+n.Address)
	defer span.End()

	log.Infof("CommitServiceTransaction started %s => %s", transaction.PendingTransaction.PaymentSourceAddress,
		transaction.PendingTransaction.PaymentDestinationAddress)

	err := n.CommitPaymentTransaction(context, transaction)

	if err != nil {
		return err
	}

	err = n.paymentRegistry.reducePendingAmount(pr.ServiceSessionId, transaction.GetPaymentTransaction().AmountOut)

	log.Infof("CommitServiceTransaction finished %s => %s", transaction.PendingTransaction.PaymentSourceAddress,
		transaction.PendingTransaction.PaymentDestinationAddress)

	return err
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

	log.Infof("FlushTransactions started")

	resultsMap := make(map[string]interface{})

	//TODO Sort transaction by sequence number and make sure to submit them only in sequence number order
	transactions := n.paymentRegistry.getActiveTransactions()

	if len(transactions) == 0 {
		log.Info("FlushTransactions: No transactions to flush.")
		return resultsMap,nil
	}

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

	nodeAccount, err := n.horizon.GetAccount(n.Address)

	if err != nil {
		return resultsMap, errors.Errorf("Error gettings account details: %v", err)
	}

	if err != nil {
		return resultsMap, errors.Errorf("Error getting horizon account: %s", err.Error())
	}

	wrapper, err := txnbuild.TransactionFromXDR(transactions[0].XDR)

	if err != nil {
		return resultsMap, errors.Errorf("Error converting first transaction from xdr: %v", err)
	}

	firstTransaction,ok := wrapper.Transaction()

	if !ok {
		return resultsMap, errors.Errorf("Can't get first transaction from wrapper")
	}

	wrapper, err = txnbuild.TransactionFromXDR(transactions[len(transactions)-1].XDR)

	lastTransaction,ok := wrapper.Transaction()
	_ = lastTransaction

	if !ok {
		return resultsMap, errors.Errorf("Can't get last transaction from wrapper")
	}

	if err != nil {
		return resultsMap, errors.Errorf("Error converting last transaction from xdr: %v", err)
	}

	// Handle unfulfilled transactions, if needed
	currentSequence, err := nodeAccount.GetSequenceNumber()

	if err != nil {
		return resultsMap, errors.Errorf("Error reading sequence: %v", err)
	}

	transactionToRemove := 0

	// Filter out missed transactions

	if firstTransaction.SourceAccount().Sequence <= currentSequence {
		for _, t := range transactions {

			wrapper, err = txnbuild.TransactionFromXDR(t.XDR)

			if err != nil {
				log.Warn("Problematic transaction detected, couldn't convert from XDR - removing.")

				transactionToRemove = transactionToRemove + 1
				continue
			}

			innerTransaction,ok := wrapper.Transaction()

			if !ok {
				log.Warn("Problematic transaction detected, couldn't extract inner transaction - removing.")

				transactionToRemove = transactionToRemove + 1
				continue
			}

			if innerTransaction.SourceAccount().Sequence <= currentSequence {
				log.Warnf("Problematic transaction detected  -bad sequence %d <= %d- removing.",innerTransaction.SourceAccount().Sequence,currentSequence)
				transactionToRemove = transactionToRemove + 1
			}
		}

		if transactionToRemove > 0 {
			log.Warnf("Bad transactions detected (%d) and where removed.", transactionToRemove)

			transactions = transactions[transactionToRemove:]

			wrapper, err := txnbuild.TransactionFromXDR(transactions[0].XDR)

			if err != nil {
				return resultsMap, errors.Errorf("Error converting first transaction from xdr: %v", err)
			}

			firstTransaction,ok = wrapper.Transaction()

			if !ok {
				return resultsMap, errors.Errorf("Can't get first transaction from wrapper")
			}
		}
	}

	// Bump if needed
	if firstTransaction.SourceAccount().Sequence > currentSequence+1 {
		log.Warnf("Sequence bump needed, unfullfilled transactions detected: %d", firstTransaction.SourceAccount().Sequence-(currentSequence+1))

		tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount: &txnbuild.SimpleAccount{
				AccountID: n.Address,
				Sequence:  currentSequence,
			},
			IncrementSequenceNum: true,
			BaseFee: 200,
			Timebounds: txnbuild.NewTimeout(300),
			Operations: []txnbuild.Operation{&txnbuild.BumpSequence{
				BumpTo:        firstTransaction.SourceAccount().Sequence - 1,
				SourceAccount: &nodeAccount,
			}}})

		if err != nil {
			return resultsMap, errors.Errorf("Error creating seq bump tx: %v", err)
		}

		kp,err := keypair.ParseFull(n.secretSeed)

		if err != nil {
			return resultsMap, errors.Errorf("Error getting key: %v", err)
		}

		tx,err = tx.Sign(network.TestNetworkPassphrase,kp)

		if err != nil {
			return resultsMap, errors.Errorf("Error signing seq bump tx: %v", err)
		}

		_, err = horizonclient.DefaultTestNetClient.SubmitTransaction(tx)

		if err != nil {
			xdr,_ := tx.Base64()
			log.Errorf("Error in seq bump transaction: %s" + xdr)
			return resultsMap, errors.Errorf("Error submitting seq bump tx: %v", err)
		}
	}

	for a, t := range transactions {

		log.Infof("Submitting transaction for session %s",t.ServiceSessionId)
		txSuccess, err := horizonclient.DefaultTestNetClient.SubmitTransactionXDR(t.XDR)

		if err != nil {

			log.Errorf("Error in submit transaction: %s" + t.XDR)

			stellarError, ok := err.(horizonclient.Error)
			if ok {
				resultCodes, innerErr := stellarError.ResultCodes()

				if innerErr != nil {
					log.Errorf("Error unwrapping stellar errors: %v", innerErr.Error())
				}

				log.Errorf("Stellar error details - transaction error: %s", resultCodes.TransactionCode)

				for _, operror := range resultCodes.OperationCodes {
					log.Errorf("Stellar error details - operation error: %s", operror)
				}
			} else {
				log.Errorf("Couldn't parse error as stellar: " + err.Error())
			}
		}

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
