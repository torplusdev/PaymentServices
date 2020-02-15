package root

import (
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"log"
	"strconv"
)

type RootApi struct {
	client *horizonclient.Client
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
			AccountID:pair.Address() })


	api.rootAccount = rootAccountDetail

	if errAccount != nil {
		txSuccess, errCreate := api.client.Fund(pair.Address())

		if errCreate != nil {
			log.Fatal(err)
		}

		//TODO: Replace optimism with structured error handling
		rootAccountDetail, _ := api.client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID:pair.Address() })
		api.rootAccount = rootAccountDetail

		log.Printf("Account creation performed using transaction#:",txSuccess)
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

	accountData,_ := horizon.DefaultTestNetClient.LoadAccount(api.fullKeyPair.Address())

	if err == nil {
		return nil
		//log.Fatal("Account already exists")
	}


	createAccountOp := txnbuild.CreateAccount{
		Destination: address,
		Amount:      strconv.Itoa(getInitialAccountBalance()),
	}

	clientAccount := txnbuild.NewSimpleAccount(address,0)

	var masterWeight, thresholdLow, thresholdMed, thresholdHigh txnbuild.Threshold

	masterWeight = 0
	thresholdLow = 2
	thresholdMed = 3
	thresholdHigh = 4

	_ = masterWeight

	setOptionsChangeWeights := txnbuild.SetOptions{
		SourceAccount: &clientAccount,
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
		Destination:   address,
		Amount:        strconv.Itoa(getInitialAccountBalance()),
		//SourceAccount: &accountData,
		Asset:txnbuild.NativeAsset{},
	}

	// Construct the transaction that will carry the operation
	tx := txnbuild.Transaction{
		SourceAccount: &accountData,
		Operations:    []txnbuild.Operation{&createAccountOp, &setOptionsChangeWeights, &payment},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	clientKey, _ := keypair.ParseFull(seed)

	tx.Build()
	tx.Sign(&api.fullKeyPair)


	strTrans,er1 := tx.Base64()

	if er1 != nil {

	}

	clientTrans,er2 := txnbuild.TransactionFromXDR(strTrans)

	if er2 != nil {
		log.Fatal("Cannot deserialize transaction:",er2.Error())
	}
	// Work around serialization bug (??): network passphrase isn't serialized
	clientTrans.Network = network.TestNetworkPassphrase

	clientTrans.Sign(clientKey)

	resp, err := api.client.SubmitTransaction(clientTrans)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError,hError.Problem)
	}

	log.Println("\nTransaction response: ", resp)

	return nil
}