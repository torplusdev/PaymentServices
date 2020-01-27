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

func (api RootApi) CreateGateway(gwAddress string){

	account, err := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID:gwAddress })

	if err  != nil {
		txSuccess, errCreate := api.client.Fund(gwAddress)

		if errCreate != nil {
			log.Fatal(err)
		}

		log.Printf("Account creation performed using transaction#:",txSuccess)
	}

	log.Printf("Account:",account)

	/*
	createAccountOp := txnbuild.CreateAccount{
		Destination: kp1.Address(),
		Amount:      "10",
	}

	client.Fund(pair.Address())


	// Get information about the account we just created
	accountRequest := horizonclient.AccountRequest{AccountID: pair.Address()}

	hAccount0, err := client.AccountDetail(accountRequest)

	if err != nil {
		log.Fatal(err)
	}

	// Generate a second randomly generated address
	kp1, err := keypair.Random()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Seed 1:", kp1.Seed())
	log.Println("Address 1:", kp1.Address())

	// Construct the operation
	createAccountOp := txnbuild.CreateAccount{
		Destination: kp1.Address(),
		Amount:      "10",
	}

	// Construct the transaction that will carry the operation
	tx := txnbuild.Transaction{
		SourceAccount: &hAccount0,
		Operations:    []txnbuild.Operation{&createAccountOp},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	// Sign the transaction, serialise it to XDR, and base 64 encode it
	txeBase64, err := tx.BuildSignEncode(pair)
	log.Println("Transaction base64: ", txeBase64)

	// Submit the transaction
	resp, err := client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError)
	}

	log.Println("\nTransaction response: ", resp)
*/
}

func getInitialAccountBalance() int {
	return 100
}

func (api RootApi) CreateUser(address string, seed string) error {

	_, err := api.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: address})

	accountData,_ := horizon.DefaultTestNetClient.LoadAccount(api.fullKeyPair.Address())

		if err == nil {
		log.Fatal("Account already exists")
	}


	createAccountOp := txnbuild.CreateAccount{
		Destination: address,
		Amount:      strconv.Itoa(getInitialAccountBalance()),
	}

	clientAccount := txnbuild.NewSimpleAccount(address,0)

	setOptionsAddRootSigner := txnbuild.SetOptions{
		Signer: &txnbuild.Signer{
			Address: api.fullKeyPair.Address(),
			Weight:  5,
		},
		SourceAccount: &clientAccount,
	}

	var masterWeight, thresholdLow, thresholdMed, thresholdHigh txnbuild.Threshold

	masterWeight = 5
	thresholdLow = 2
	thresholdMed = 3
	thresholdHigh = 4

	_ = masterWeight

	setOptionsChangeWeights := txnbuild.SetOptions{
		SourceAccount: &clientAccount,
		//MasterWeight:    &masterWeight,
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

	_ = payment
	_ = setOptionsChangeWeights
	_ = setOptionsAddRootSigner

	// Construct the transaction that will carry the operation
	tx := txnbuild.Transaction{
		SourceAccount: &accountData,
		Operations:    []txnbuild.Operation{&createAccountOp, &setOptionsChangeWeights, &payment},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	// Sign the transaction, serialise it to XDR, and base 64 encode it

	clientKey, _ := keypair.ParseFull(seed)

	tx.Build()
	tx.Sign(&api.fullKeyPair)


	strTrans,er1 := tx.Base64()

	if er1 != nil {

	}

	clientTrans,er2 := txnbuild.TransactionFromXDR(strTrans)

	if er2 != nil {

	}
	// Work around serialization bug (??): network passphrase isn't serialized
	clientTrans.Network = network.TestNetworkPassphrase

	clientTrans.Sign(clientKey)

//	txeBase64, err := tx.BuildSignEncode(&api.fullKeyPair )
//	clientTx,transErr := txnbuild.TransactionFromXDR(txeBase64)

	//bytes,_ := tx.TxEnvelope().MarshalBinary()



	//txnbuild.TransactionFromXDR(bytes)

	//clientTx,transErr := txnbuild.TransactionFromXDR(txeBase64)

	//if transErr != nil {
	//	log.Fatal("Error ")
	//}

	//clientTx.Sign(clientKey)
	//append(clientTx.TxEnvelope().Signatures, )

	// Submit the transaction
	//resp, err := api.client.SubmitTransactionXDR(txeBase64)
	resp, err := api.client.SubmitTransaction(clientTrans)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError,hError.Problem)
	}

	log.Println("\nTransaction response: ", resp)

	return nil
}