package root

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"log"
	"strconv"
)

type RootApi struct {
	client      *horizonclient.Client
	rootAccount horizon.Account
	fullKeyPair keypair.Full
}

const seed = "SAVD5NOJUVUJJIRFMPWSVIP4S6PXSEWAYWAG4WOALSSLKLVONW4YL3VT"

func CreateRootApi(useTestApi bool) *RootApi {
	rootApi := RootApi{}

	if useTestApi {
		rootApi.client = horizonclient.DefaultTestNetClient
	} else {
		rootApi.client = horizonclient.DefaultPublicNetClient
	}

	rootApi.initialize()

	return &rootApi
}

func (api *RootApi) initialize() {
	//pair, err  := keypair.Random()
	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	api.fullKeyPair = *pair

	rootAccountDetail, errAccount := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: pair.Address()})

	api.rootAccount = rootAccountDetail

	if errAccount != nil {
		txSuccess, errCreate := api.client.Fund(pair.Address())

		if errCreate != nil {
			log.Fatal(err)
		}

		//TODO: Replace optimism with structured error handling
		rootAccountDetail, _ := api.client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID: pair.Address()})
		api.rootAccount = rootAccountDetail

		log.Printf("Account creation performed using transaction#: %s", txSuccess.ResultXdr)
	}

}

func getInitialAccountBalance() int {
	return 100
}

func (api RootApi) GetClient() *horizonclient.Client {

	return api.client
}

func (api RootApi) CreateUser(address string, seed string) error {

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
		SourceAccount:   &clientAccount,
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
		Timebounds: 		  txnbuild.NewTimeout(300),
	})

	clientKey, _ := keypair.ParseFull(seed)

	tx.Sign(network.TestNetworkPassphrase,&api.fullKeyPair)

	strTrans, er1 := tx.Base64()

	if er1 != nil {

	}

	clientTransWrapper, er2 := txnbuild.TransactionFromXDR(strTrans)

	if er2 != nil {
		log.Fatal("Cannot deserialize transaction:", er2.Error())
	}

	clientTrans, result := clientTransWrapper.Transaction()

	if !result {
		log.Fatal("Cannot deserialize transaction (GenericTransaction):", er2.Error())
	}
	
	clientTrans.Sign(network.TestNetworkPassphrase,clientKey)

	resp, err := api.client.SubmitTransaction(clientTrans)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError, hError.Problem)
	}

	log.Println("\nTransaction response: ", resp)

	return nil
}
