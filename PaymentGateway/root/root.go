package root

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
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
	Verify(input []byte, sig []byte) error
	Sign(tr *txnbuild.Transaction) (*txnbuild.Transaction, error)
	SubmitTransaction(transaction *txnbuild.Transaction) (tx horizon.Transaction, err error)

	CreateTransaction(request *models.CreateTransactionCommand, tr *models.PaymentTransactionReplacing) (*models.PaymentTransactionReplacing, error)
	SetTransactionValiditySecs(transactionValiditySecs int64)
	SubmitTransactionXDR(xdr models.XDR) (horizon.Transaction, error)
	PaymentTransactionToStellar(trans *models.PaymentTransaction) (*txnbuild.Transaction, error)
}
type rootApiCore struct {
	client  *horizonclient.Client
	network string
}
type rootApi struct {
	rootApiCore
	fullKeyPair             keypair.Full
	rootAccount             *horizon.Account
	lastSequenceId          xdr.SequenceNumber
	sequenceMux             sync.Mutex
	transactionValiditySecs int64
}

const seed = "SAVD5NOJUVUJJIRFMPWSVIP4S6PXSEWAYWAG4WOALSSLKLVONW4YL3VT"

type RootApiFactory func(seed string, transactionValiditySecs int64) (RootApi, error)

func CreateRootApiFactory(useTestApi bool) RootApiFactory {
	return func(seed string, transactionValiditySecs int64) (RootApi, error) {
		if useTestApi {
			rc := &rootApiCore{
				client:  horizonclient.DefaultTestNetClient,
				network: network.TestNetworkPassphrase,
			}
			return createRootApi(rc, seed, transactionValiditySecs)
		}
		rc := &rootApiCore{
			client:  horizonclient.DefaultPublicNetClient,
			network: network.PublicNetworkPassphrase,
		}
		return createRootApi(rc, seed, transactionValiditySecs)
	}

}
func createRootApi(withCore *rootApiCore, seed string, transactionValiditySecs int64) (RootApi, error) {
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
func (api *rootApi) SetTransactionValiditySecs(transactionValiditySecs int64) {
	api.transactionValiditySecs = transactionValiditySecs
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
		log.Printf("Sequence number initialization: %d", seq)
		api.lastSequenceId = seq
	}
	var sequenceProvider int64
	// If this is the first transaction for the node+client pair and there's no reference transaction
	if tr.ReferenceTransaction == nil {
		api.sequenceMux.Lock()
		defer api.sequenceMux.Unlock()
		log.Printf("No reference transaction, assigning id %d and promoting", api.lastSequenceId)
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
		log.Printf("reference transaction found, assigning id %d", sequenceProvider)
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

	if err != nil {
		return nil, fmt.Errorf("error creating transaction: %v", err)
	}

	xdr, err := tx.Base64()

	if err != nil {
		return nil, fmt.Errorf("error serializing transaction: %v", err)
	}

	tr.PendingTransaction.XDR = models.NewXDR(xdr)

	tr.PendingTransaction.StellarNetworkToken = api.network
	tr.PendingTransaction.StellarNetworkToken = xdr
	log.Printf("CreateTransaction: Done %s => %s ", request.SourceAddress, api.GetAddress())

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
		return fmt.Errorf("balance of PPToken  is too low %f. Should be at least %d", balance, models.PPTokenMinAllowedBalance)
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
		return fmt.Errorf("error getting source account data: %v", err)
	}
	return nil
}
func (api *rootApi) Sign(tr *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	return tr.Sign(api.network, &api.fullKeyPair)

}
func (api *rootApi) Verify(input []byte, sig []byte) error {
	return api.fullKeyPair.Verify(input[:], sig)
}
func (api *rootApi) SubmitTransaction(transaction *txnbuild.Transaction) (tx horizon.Transaction, err error) {
	return api.client.SubmitTransaction(transaction)
}
func (api *rootApi) SubmitTransactionXDR(xdr models.XDR) (horizon.Transaction, error) {
	return api.client.SubmitTransactionXDR(xdr.String())
}
func (api *rootApi) GetAddress() string {
	return api.fullKeyPair.Address()
}
func (api *rootApi) PaymentTransactionToStellar(trans *models.PaymentTransaction) (*txnbuild.Transaction, error) {

	transactionWrapper, err := trans.XDR.TransactionFromXDR()

	if err != nil {
		return nil, fmt.Errorf("Error converting transaction from xdr: %s", err)
	}

	actualTransaction, result := transactionWrapper.Transaction()

	if !result {
		return nil, fmt.Errorf("Error converting transaction i from xdr (GenericTransaction): %v", result)
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
		return models.TransactionAmount(b), err
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
	log.Printf("Account creation performed using transaction#: %s", txSuccess.ResultXdr)
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

// 		log.Printf("Account creation performed using transaction#: %s", txSuccess.ResultXdr)
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

	log.Println("\nTransaction response: ", resp)

	return nil
}
