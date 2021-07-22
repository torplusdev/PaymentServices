package root

import (
	"context"
	"fmt"

	"strconv"
	"sync"

	"github.com/go-errors/errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
)

type RootApi interface {
	CreateUser() error
	ValidateForPPNode() error
	CheckSourceAddress(address string) error
	GetAddress() string
	GetAccount() (*horizon.Account, error)
	GetSequenceNumber() (xdr.SequenceNumber, error)
	GetMicroPPTokenBalance() (models.TransactionAmount, error)
	GetPPTokenBalance() (float64, error)
	VerifyTransaction(context.Context, *models.PaymentTransaction) error
	ValidateTimebounds(*models.PaymentTransaction) error
	SignPaymentTransaction(tr *models.PaymentTransaction) (*models.PaymentTransaction, error)
	SignXDR(tr models.XDR) (models.XDR, error)
	Sign(tr *txnbuild.Transaction) (*txnbuild.Transaction, error)
	SubmitTransaction(transaction *models.PaymentTransaction) error
	SubmitTransactionOld(transaction *txnbuild.Transaction) error
	GetTransactionSequenceNumber(transaction *models.PaymentTransaction) (int64, error)
	CreateTransaction(request *models.CreateTransactionCommand, tr *models.PaymentTransactionReplacing) (*models.PaymentTransactionReplacing, error)
	SetTransactionValiditySecs(transactionValiditySecs int64)
	SubmitTransactionXDR(xdr models.XDR) error
	PaymentTransactionToStellar(trans *models.PaymentTransaction) (*txnbuild.Transaction, error)
	RemoveTransactionsIfSequence(transactions []*models.PaymentTransactionWithSequence) ([]*models.PaymentTransactionWithSequence, error)
	BumpSequenceIfNeed(transaction *models.PaymentTransactionWithSequence) error
	ValidateSignarureCount(xdr models.XDR, count int) error
}

type rootApiCore struct {
	client       *horizonclient.Client
	networkToken string
}
type rootApi struct {
	rootApiCore
	fullKeyPair             keypair.Full
	rootAccount             *horizon.Account
	lastSequenceId          xdr.SequenceNumber
	sequenceMux             sync.Mutex
	transactionValiditySecs int64
}

type RootApiFactory func(seed string, transactionValiditySecs int64) (RootApi, error)

func createTestRootApi(seed string, transactionValiditySecs int64) (RootApi, error) {
	rc := &rootApiCore{
		client:       horizonclient.DefaultTestNetClient,
		networkToken: network.TestNetworkPassphrase,
	}
	r, err := createRootApi(rc, seed, transactionValiditySecs)
	return r, err
}

func createPublicRootApi(seed string, transactionValiditySecs int64) (RootApi, error) {
	rc := &rootApiCore{
		client:       horizonclient.DefaultPublicNetClient,
		networkToken: network.PublicNetworkPassphrase,
	}
	r, err := createRootApi(rc, seed, transactionValiditySecs)
	return r, err
}

func CreateRootApiFactory(useTestApi bool) RootApiFactory {
	if useTestApi {
		return createTestRootApi
	} else {
		return createPublicRootApi
	}

}

func createRootApi(withCore *rootApiCore, seed string, transactionValiditySecs int64) (*rootApi, error) {
	fullKeyPair, err := keypair.ParseFull(seed)
	if err != nil {
		log.Panicf("Error parsing node key: %s", err)

	}
	rootApi := &rootApi{
		rootApiCore:             *withCore,
		fullKeyPair:             *fullKeyPair,
		rootAccount:             nil,
		lastSequenceId:          0,
		sequenceMux:             sync.Mutex{},
		transactionValiditySecs: transactionValiditySecs,
	}

	err = rootApi.initialize()
	if err != nil {
		return nil, err
	}
	return rootApi, err
}

func (api *rootApi) ValidateSignarureCount(xdr models.XDR, count int) error {
	transactionWrapper, e := xdr.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	signatures := t.Signatures()
	if len(signatures) != count {
		return fmt.Errorf("signatures count invalid")
	}
	return nil
}

func (api *rootApi) SetTransactionValiditySecs(transactionValiditySecs int64) {
	api.transactionValiditySecs = transactionValiditySecs
}

func (api *rootApi) GetTransactionSequenceNumber(transaction *models.PaymentTransaction) (int64, error) {

	nodeAccount, err := api.PaymentTransactionToStellar(transaction)
	if err != nil {
		return 0, errors.Errorf("error reading account: %v", err)
	}

	return nodeAccount.SourceAccount().Sequence, nil

}

func (api *rootApi) CreateTransaction(request *models.CreateTransactionCommand, tr *models.PaymentTransactionReplacing) (*models.PaymentTransactionReplacing, error) {

	var amount = tr.PendingTransaction.ReferenceAmountIn

	err := api.CheckSourceAddress(request.SourceAddress)
	if err != nil {
		return nil, err
	}

	// Uninitialized
	if api.lastSequenceId == 0 {
		seq, err := api.GetSequenceNumber()
		if err != nil {
			return nil, err
		}
		log.Infof("Sequence number initialization: %d", seq)
		api.lastSequenceId = seq
	}
	var sequenceProvider int64
	// If this is the first transaction for the node+client pair and there's no reference transaction
	if tr.ReferenceTransaction == nil {
		api.sequenceMux.Lock()
		defer api.sequenceMux.Unlock()
		log.Infof("No reference transaction, assigning id %d and promoting", api.lastSequenceId)
		sequenceProvider = int64(api.lastSequenceId)
		api.lastSequenceId = api.lastSequenceId + 1
	} else {
		referenceTransactionPayload := tr.ReferenceTransaction

		referenceTransactionWrapper, err := referenceTransactionPayload.XDR.TransactionFromXDR()

		if err != nil {
			return nil, fmt.Errorf("error deserializing XDR transaction: %s", err)
		}

		referenceTransaction, result := referenceTransactionWrapper.Transaction()
		if !result {
			return nil, fmt.Errorf("error deserializing XDR transaction (GenericTransaction)")
		}

		account := referenceTransaction.SourceAccount()
		referenceSequenceNumber, err := account.GetSequenceNumber()
		if err != nil {
			return nil, err
		}
		sequenceProvider = referenceSequenceNumber - 1
		log.Infof("reference transaction found, assigning id %d", sequenceProvider)
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: api.GetAddress(),
			Sequence:  sequenceProvider,
		},
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{&txnbuild.Payment{
			Destination: api.GetAddress(),
			Amount:      models.PPTokenToString(amount),
			Asset: txnbuild.CreditAsset{
				Code:   models.PPTokenAssetName,
				Issuer: models.PPTokenIssuerAddress,
			},
			SourceAccount: request.SourceAddress,
		}},
		BaseFee:    200,
		Timebounds: txnbuild.NewTimeout(api.transactionValiditySecs),
	})
	//tx.Timebounds().
	if err != nil {
		return nil, fmt.Errorf("error creating transaction: %v", err)
	}

	xdr, err := tx.Base64()

	if err != nil {
		return nil, fmt.Errorf("error serializing transaction: %v", err)
	}

	tr.PendingTransaction.XDR = models.NewXDR(xdr)
	tr.PendingTransaction.StellarNetworkToken = api.networkToken

	log.Infof("CreateTransaction: Done %s => %s ", request.SourceAddress, api.GetAddress())

	//tr.ToSpanAttributes(span, "credit")//TODO CREATE SPAN

	err = tr.PendingTransaction.Validate()
	if err != nil {
		return nil, err
	}
	return tr, nil
}

func (api *rootApi) ValidateForPPNode() error {
	balance, err := api.GetMicroPPTokenBalance()
	if err != nil {
		return nil
	}
	if balance < models.PPTokenMinAllowedBalance {
		log.Info("balance of PPToken  is too low %d. Should be at least %d", balance, models.PPTokenMinAllowedBalance)
	}
	address := api.GetAddress()
	nodeAccountDetail, err := api.GetAccount()

	if err != nil {
		return fmt.Errorf("client account doesnt exist: %s ", err.Error())
	}
	signerMap := nodeAccountDetail.SignerSummary()
	masterWeight := signerMap[address]

	if masterWeight < int32(nodeAccountDetail.Thresholds.MedThreshold) {
		return fmt.Errorf("error in client account: master weight (%d) should be at least at medium threshold (%d) ",
			masterWeight, nodeAccountDetail.Thresholds.MedThreshold)
	}
	return nil

}

func (api *rootApi) CheckSourceAddress(a string) error {
	_, err := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: a})
	if err != nil {
		return fmt.Errorf("error getting source account data: %v, value: %v", err, a)
	}
	return nil
}

func (api *rootApi) SignPaymentTransaction(tr *models.PaymentTransaction) (*models.PaymentTransaction, error) {
	signedXDR, err := api.SignXDR(tr.XDR)
	if err != nil {
		return nil, err
	}

	return &models.PaymentTransaction{
		TransactionSourceAddress:  tr.TransactionSourceAddress,
		ReferenceAmountIn:         tr.ReferenceAmountIn,
		AmountOut:                 tr.AmountOut,
		XDR:                       signedXDR,
		PaymentSourceAddress:      tr.PaymentSourceAddress,
		PaymentDestinationAddress: tr.PaymentDestinationAddress,
		StellarNetworkToken:       tr.StellarNetworkToken,
		ServiceSessionId:          tr.ServiceSessionId,
	}, nil
}

func (api *rootApi) SignXDR(xdr models.XDR) (models.XDR, error) {
	transactionWrapper, err := xdr.TransactionFromXDR()

	if err != nil {
		return nil, fmt.Errorf("error parsing transaction: %v", err)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return nil, fmt.Errorf("transaction destination is incorrect (GenericTransaction)")
	}

	signedTransaction, err := api.Sign(t)

	if err != nil {
		return nil, fmt.Errorf("failed to signed transaction: %v", err)
	}
	str, err := signedTransaction.Base64()
	if err != nil {
		return nil, err
	}
	signedXdr := models.NewXDR(str)
	if signedXdr.Equals(xdr) {
		return nil, fmt.Errorf("singed xdr eq to xdr %v=%v", xdr, signedXdr)
	}
	return signedXdr, nil
}

func (api *rootApi) Sign(tr *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	return tr.Sign(api.networkToken, &api.fullKeyPair)
}

func (api *rootApi) VerifyTransaction(context context.Context, transaction *models.PaymentTransaction) error {

	err := api.verifyTransactionSequence(context, transaction)

	if err != nil {
		log.Warn("Transaction verification failed (sequence)")
		return err
	}

	err = api.verifyTransactionSignatures(context, transaction)

	if err != nil {
		log.Warn("Transaction verification failed (signatures)")
		return err
	}
	return nil
}

func (n *rootApi) ValidateTimebounds(transaction *models.PaymentTransaction) error {
	transactionWrapper, e := transaction.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: %v", e)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}
	tb := t.Timebounds()
	return tb.Validate()
}

func (n *rootApi) verifyTransactionSequence(context context.Context, transaction *models.PaymentTransaction) error {

	// Deserialize transactions
	transactionWrapper, e := transaction.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: %v", e)
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	nodeAccount, err := n.GetAccount()
	if err != nil {
		return errors.Errorf("error reading account: %v", err)
	}

	currentSequence, err := nodeAccount.GetSequenceNumber()
	if err != nil {
		return fmt.Errorf("error getting sequence: %v", err)
	}

	account := t.SourceAccount()
	transactionSequence, err := account.GetSequenceNumber()
	if err != nil {
		return err
	}
	if transactionSequence <= currentSequence {
		log.Warnf("Incorrect sequence detected, current account is at %d, transaction is %d", currentSequence, transactionSequence)
		return errors.Errorf("incorrect sequence detected")
	}

	log.Infof("verifyTransactionSequence finished successfully - account#:%d transaction#:%d", currentSequence, transactionSequence)
	return nil
}

func (n *rootApi) verifyTransactionSignatures(context context.Context, transaction *models.PaymentTransaction) error {

	log.Infof("verifyTransactionSignatures started %s => %s", transaction.PaymentSourceAddress,
		transaction.PaymentDestinationAddress)

	//transaction := transactionPayload.PendingTransaction

	// Deserialize transactions
	transactionWrapper, e := transaction.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: " + e.Error())
	}

	t, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}

	if t.SourceAccount().AccountID != n.GetAddress() {
		return errors.Errorf("Incorrect transaction source account")
	}
	operations := t.Operations()
	var payerAccount string = ""
	for _, op := range operations {
		xdrOp, _ := op.BuildXDR()

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:
			payment := &txnbuild.Payment{}

			err := payment.FromXDR(xdrOp)

			if err != nil {
				return errors.Errorf("Error converting operation")
			}

			payerAccount = payment.SourceAccount
		default:
			return errors.Errorf("Unexpected operation during verification")
		}
	}

	payerVerified := false
	sourceVerified := false
	signatures := t.Signatures()
	log.Info("Signatures count:", len(signatures))
	for _, signature := range signatures {
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
			continue
		}
		err = n.verifySignature(bytes[:], signature.Signature)

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

func (api *rootApi) verifySignature(input []byte, sig []byte) error {
	return api.fullKeyPair.Verify(input[:], sig)
}

func (api *rootApi) SubmitTransactionOld(transaction *txnbuild.Transaction) error {
	_, err := api.client.SubmitTransaction(transaction)
	return err
}

func (api *rootApi) SubmitTransaction(t *models.PaymentTransaction) error {

	// Deserialize transactions
	transactionWrapper, e := t.XDR.TransactionFromXDR()

	if e != nil {
		return errors.Errorf("Error deserializing transaction from XDR: %v", e)
	}

	tr, result := transactionWrapper.Transaction()

	if !result {
		return errors.Errorf("Error deserializing transaction from XDR (GenericTransaction)")
	}
	_, err := api.client.SubmitTransaction(tr)
	if err != nil {
		return err
	}
	return nil
}

func (api *rootApi) SubmitTransactionXDR(xdr models.XDR) error {
	_, err := api.client.SubmitTransactionXDR(xdr.String())
	if err != nil {
		if stellarError, ok := err.(*horizonclient.Error); ok {

			resultCodes, innerErr := stellarError.ResultCodes()

			if innerErr != nil {
				return fmt.Errorf("error unwrapping stellar errors: %v", innerErr)

			} else {
				var err error

				for _, operror := range resultCodes.OperationCodes {
					if err != nil {
						err = fmt.Errorf("%v: Stellar error details - operation error: %s", err, operror)
					} else {
						err = fmt.Errorf("stellar error details - operation error: %s", operror)
					}

				}
				return fmt.Errorf("stellar error details - transaction error: %s :%v", resultCodes.TransactionCode, err)
			}
		} else {
			return fmt.Errorf("couldn't parse error as stellar: %v", err)

		}

	}
	return err
}

func (api *rootApi) GetAddress() string {
	return api.fullKeyPair.Address()
}

func (api *rootApi) RemoveTransactionsIfSequence(transactions []*models.PaymentTransactionWithSequence) ([]*models.PaymentTransactionWithSequence, error) {

	var (
		nodeAccount *horizon.Account
		err         error
	)

	if nodeAccount, err = api.GetAccount(); err != nil {
		return nil, errors.Errorf("Error gettings account details: %v", err)
	}

	// Handle unfulfilled transactions, if needed
	currentSequence, err := nodeAccount.GetSequenceNumber()

	if err != nil {
		return nil, errors.Errorf("Error reading sequence: %v", err)
	}

	transactionToRemove := 0

	// Filter out missed transactions

	if transactions[0].Sequence <= currentSequence {
		for _, t := range transactions {

			if err != nil {
				log.Warn("Problematic transaction detected, couldn't convert from XDR - removing.")

				transactionToRemove = transactionToRemove + 1
				continue
			}

			if t.Sequence <= currentSequence {
				log.Warnf("Problematic transaction detected  -bad sequence %d <= %d- removing.", t.Sequence, currentSequence)
				transactionToRemove = transactionToRemove + 1
			}
		}

		if transactionToRemove > 0 {
			log.Warnf("Bad first transactions were detected (%d) and removed.", transactionToRemove)

			transactions = transactions[transactionToRemove:]

			if len(transactions) == 0 {
				log.Warnf("No further transactions to process after removing %d transactions", transactionToRemove)
				return transactions, nil
			}
		}
	}
	return transactions, nil
}

func (api *rootApi) BumpSequenceIfNeed(transaction *models.PaymentTransactionWithSequence) error {
	var (
		nodeAccount *horizon.Account
		err         error
	)

	if nodeAccount, err = api.GetAccount(); err != nil {
		return errors.Errorf("Error gettings account details: %v", err)
	}
	currentSequence, err := nodeAccount.GetSequenceNumber()

	if err != nil {
		return errors.Errorf("Error reading sequence: %v", err)
	}
	if transaction.Sequence > currentSequence+1 {
		log.Warnf("Sequence bump needed: %d", transaction.Sequence-(currentSequence+1))

		err := api.BumpSequence(currentSequence, transaction.Sequence-1)

		if err != nil {
			return errors.Errorf("Error during sequence bump: %s", err)
		}
	}
	return nil
}

func (api *rootApi) BumpSequence(current int64, bumpTo int64) error {
	nodeAccount, err := api.GetAccount()
	if err != nil {
		return err
	}
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: api.GetAddress(),
			Sequence:  current,
		},
		IncrementSequenceNum: true,
		BaseFee:              config.StellarImmediateOperationBaseFee,
		Timebounds:           txnbuild.NewTimeout(config.StellarImmediateOperationTimeoutSec),
		Operations: []txnbuild.Operation{
			&txnbuild.BumpSequence{
				BumpTo:        bumpTo,
				SourceAccount: nodeAccount.AccountID,
			},
		},
	})

	if err != nil {
		return errors.Errorf("Error creating seq bump tx: %v", err)
	}

	tx, err = api.Sign(tx)

	if err != nil {
		return errors.Errorf("Error signing seq bump tx: %v", err)
	}

	err = api.SubmitTransactionOld(tx)

	if err != nil {
		xdr, _ := tx.Base64()
		log.Errorf("Error in seq bump transaction: %s" + xdr)
		return errors.Errorf("Error submitting seq bump tx: %v", err)
	}

	return nil
}

func (api *rootApi) PaymentTransactionToStellar(trans *models.PaymentTransaction) (*txnbuild.Transaction, error) {

	transactionWrapper, err := trans.XDR.TransactionFromXDR()

	if err != nil {
		return nil, fmt.Errorf("error converting transaction from xdr: %s", err)
	}

	actualTransaction, result := transactionWrapper.Transaction()

	if !result {
		return nil, fmt.Errorf("error converting transaction i from xdr (GenericTransaction): %v", result)
	}

	return actualTransaction, nil
}

func (api *rootApi) GetSequenceNumber() (xdr.SequenceNumber, error) {
	account, err := api.GetAccount()

	if err != nil {
		return xdr.SequenceNumber(0), fmt.Errorf("error getting horizon account: %s", err)
	}

	seq, err := account.GetSequenceNumber()

	if err != nil {
		return xdr.SequenceNumber(0), fmt.Errorf("error retrieving sequence number: %s", err)
	}

	return xdr.SequenceNumber(seq), nil
}

func (api *rootApi) GetAccount() (*horizon.Account, error) {
	address := api.GetAddress()
	acc, err := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: address,
		})
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

func (api *rootApi) GetMicroPPTokenBalance() (models.TransactionAmount, error) {
	b, err := api.GetPPTokenBalance()
	if err != nil {
		return models.TransactionAmount(0), err
	}
	return models.MicroPPToken2PPtoken(b), nil
}

func (api *rootApi) GetPPTokenBalance() (float64, error) {
	account, err := api.GetAccount()
	if err != nil {
		return 0, err
	}

	balance := account.GetCreditBalance(models.PPTokenAssetName, models.PPTokenIssuerAddress)
	nbalance, err := strconv.ParseFloat(balance, 32)
	if err != nil {
		return 0, err
	}
	return nbalance, nil
}

func (api *rootApi) initialize() error {

	rootAccount, err := api.GetAccount()

	if err == nil {
		api.rootAccount = rootAccount
		return nil
	}

	txSuccess, err := api.client.Fund(api.fullKeyPair.Address())

	if err != nil {
		log.Fatal(err)
	}

	//TODO: Replace optimism with structured error handling
	rootAccount, err = api.GetAccount()
	if err != nil {
		return err
	}
	api.rootAccount = rootAccount
	log.Infof("Account creation performed using transaction#: %s", txSuccess.ResultXdr)
	return nil
}

// func (api *rootApi) initialize() {
// 	//pair, err  := keypair.Random()
// 	pair, err := keypair.ParseFull(seed)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	api.fullKeyPair = *pair

// 	rootAccountDetail, errAccount := api.client.AccountDetail(
// 		horizonclient.AccountRequest{
// 			AccountID: pair.Address()})

// 	api.rootAccount = rootAccountDetail

// 	if errAccount != nil {
// 		txSuccess, errCreate := api.client.Fund(pair.Address())

// 		if errCreate != nil {
// 			log.Fatal(err)
// 		}

// 		//TODO: Replace optimism with structured error handling
// 		rootAccountDetail, _ := api.client.AccountDetail(
// 			horizonclient.AccountRequest{
// 				AccountID: pair.Address()})
// 		api.rootAccount = rootAccountDetail

// 		log.Infof("Account creation performed using transaction#: %s", txSuccess.ResultXdr)
// 	}

// }

func getInitialAccountBalance() int {
	return 100
}

func (api *rootApi) CreateUser() error {
	address := api.fullKeyPair.Address()
	_, err := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: address})

	accountData, _ := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: api.fullKeyPair.Address()})

	if err == nil {
		return nil
		//log.Fatal("Account already exists")
	}

	createAccountOp := txnbuild.CreateAccount{
		Destination: address,
		Amount:      strconv.Itoa(getInitialAccountBalance()),
	}

	clientAccount := txnbuild.NewSimpleAccount(address, 0)

	var masterWeight, thresholdLow, thresholdMed, thresholdHigh txnbuild.Threshold

	masterWeight = 0
	thresholdLow = 2
	thresholdMed = 3
	thresholdHigh = 4

	_ = masterWeight

	setOptionsChangeWeights := txnbuild.SetOptions{
		SourceAccount:   clientAccount.AccountID,
		MasterWeight:    &masterWeight,
		LowThreshold:    &thresholdLow,
		MediumThreshold: &thresholdMed,
		HighThreshold:   &thresholdHigh,
		Signer: &txnbuild.Signer{
			Address: api.fullKeyPair.Address(),
			Weight:  6,
		},
	}

	payment := txnbuild.Payment{
		Destination: address,
		Amount:      strconv.Itoa(getInitialAccountBalance()),
		//SourceAccount: &accountData,
		Asset: txnbuild.NativeAsset{},
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &accountData,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{&createAccountOp, &setOptionsChangeWeights, &payment},
		Timebounds:           txnbuild.NewTimeout(300),
	})
	if err != nil {
		return err
	}
	//TODO is send Sign
	tx, err = tx.Sign(network.TestNetworkPassphrase, &api.fullKeyPair)
	if err != nil {
		return err
	}
	strTrans, err := tx.Base64()

	if err != nil {
		return err
	}

	clientTransWrapper, er2 := txnbuild.TransactionFromXDR(strTrans)

	if er2 != nil {
		log.Fatal("Cannot deserialize transaction:", er2.Error())
	}

	clientTrans, result := clientTransWrapper.Transaction()

	if !result {
		log.Fatal("Cannot deserialize transaction (GenericTransaction):", er2.Error())
	}
	//TODO SHULD SEND SIGNED?
	clientTrans, err = clientTrans.Sign(network.TestNetworkPassphrase, &api.fullKeyPair)
	if err != nil {
		return err
	}
	resp, err := api.client.SubmitTransaction(clientTrans)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError, hError.Problem)
	}

	log.Infof("\nTransaction response: ", resp)

	return nil
}
