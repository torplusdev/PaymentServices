package utility

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"log"
	"paidpiper.com/payment-gateway/common"
)

// Source
const tokenCreatorSeed = "SAT3ZXAC5IQHF753DLROYVW5HRZGGFB2BHEXDWMDHCHE2URPSSDW3NY5"

// Issuing account
const issuerSeed = "SBMCAMFAYTXFIXBAOZJE5X2ZX4TJQI5X6P6NE5SHOEBHLHEMGKANRTOQ"

// Distribution account
const distributionSeed = "SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM"

func CreateAsset() {

	client := horizonclient.DefaultTestNetClient

	sourceKp, err := keypair.ParseFull(tokenCreatorSeed)
	if err != nil {
		log.Fatal(err)
	}

	issuerKp, err := keypair.ParseFull(issuerSeed)
	if err != nil {
		log.Fatal(err)
	}

	distributionKp, err := keypair.ParseFull(distributionSeed)
	if err != nil {
		log.Fatal(err)
	}

	_ = sourceKp
	_ = issuerKp
	_ = distributionKp

	sourceAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: sourceKp.Address()})

	// Create and fund source account, if it doesn't exist
	if sourceAccountDetail.AccountID != sourceKp.Address() {
		client.Fund(sourceKp.Address())
		sourceAccountDetail, _ = client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID: sourceKp.Address()})
	}

	issuerAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: issuerKp.Address()})

	// check if issuer account exists, if not create it and the distribution account
	if issuerAccountDetail.AccountID != issuerKp.Address() {

		createIssuerAccount := txnbuild.CreateAccount{
			SourceAccount: &sourceAccountDetail,
			Destination:   issuerKp.Address(),
			Amount:        "100",
		}

		createDistributionAccount := txnbuild.CreateAccount{
			SourceAccount: &sourceAccountDetail,
			Destination:   distributionKp.Address(),
			Amount:        "100",
		}

		txCreateAccounts,err := txnbuild.NewTransaction(
			txnbuild.TransactionParams{
			SourceAccount: &sourceAccountDetail,
			Operations:    []txnbuild.Operation{&createIssuerAccount, &createDistributionAccount},
			Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
				IncrementSequenceNum: true,
				BaseFee:              200,
		})

		txCreateAccounts.Sign(network.TestNetworkPassphrase,sourceKp)

		signedTransaction, err := txCreateAccounts.Sign(network.TestNetworkPassphrase, sourceKp)

		resp, err := client.SubmitTransaction(signedTransaction)

		_ = resp
		_ = err
	}

	distributionAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: distributionKp.Address()})

	// Create trust line
	tokenAsset := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: issuerKp.Address(),
	}

	changeTrust := txnbuild.ChangeTrust{
		SourceAccount: &distributionAccountDetail,
		Line:          tokenAsset,
	}

	txCreateTrustLine, err := txnbuild.NewTransaction( txnbuild.TransactionParams{
		SourceAccount: &distributionAccountDetail,
		Operations:    []txnbuild.Operation{&changeTrust},
		Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
		IncrementSequenceNum: true,
		BaseFee:              200,
	})

	signedTransaction, err := txCreateTrustLine.Sign(network.TestNetworkPassphrase, distributionKp)

	if err != nil {
		log.Print("Error signing transaction:")
	}

	resp, err := client.SubmitTransaction(signedTransaction)

	createAssets := txnbuild.Payment{
		Destination:   distributionKp.Address(),
		Amount:        "10000",
		Asset:         tokenAsset,
		SourceAccount: &issuerAccountDetail,
	}

	txCreateAssets,err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &issuerAccountDetail,
		Operations:    []txnbuild.Operation{&createAssets},
		Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
		IncrementSequenceNum: true,
		BaseFee:              200,
	})


	signedTransaction, err = txCreateAssets.Sign(network.TestNetworkPassphrase, issuerKp)

	resp, err = client.SubmitTransaction(signedTransaction)

	homedomain := "www.adwayser.com"

	// Asset creation: set home domain
	setOptionsSetHomedomain := txnbuild.SetOptions{
		HomeDomain:    &homedomain,
		SourceAccount: &issuerAccountDetail,
	}

	txSetOptionsSetHomedomain,err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &issuerAccountDetail,
		Operations:    []txnbuild.Operation{&setOptionsSetHomedomain},
		Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
		IncrementSequenceNum: true,
		BaseFee:              200,
	})

	signedTransaction, err = txSetOptionsSetHomedomain.Sign(network.TestNetworkPassphrase, issuerKp)

	resp, err = client.SubmitTransaction(signedTransaction)

	_ = resp
}


func SubmitBuyOffer() {
	client := horizonclient.DefaultTestNetClient

	sourceKp, err := keypair.ParseFull(tokenCreatorSeed)
	if err != nil {
		log.Fatal(err)
	}

	issuerKp, err := keypair.ParseFull(issuerSeed)
	if err != nil {
		log.Fatal(err)
	}

	distributionKp, err := keypair.ParseFull(distributionSeed)
	if err != nil {
		log.Fatal(err)
	}

	_ = sourceKp

	distributionAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: distributionKp.Address()})

	// Create trust line
	tokenAsset := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: issuerKp.Address(),
	}

	manageBuyOffer := txnbuild.ManageBuyOffer{
		Selling: txnbuild.NativeAsset{},
		Buying:  tokenAsset,
		Amount:  "1000000",
		Price: "0.000001",
		OfferID: 0,
		SourceAccount: &distributionAccountDetail,
	}

	txBuyOffer, err := txnbuild.NewTransaction( txnbuild.TransactionParams{
		SourceAccount: &distributionAccountDetail,
		Operations:    []txnbuild.Operation{&manageBuyOffer},
		Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
		IncrementSequenceNum: true,
		BaseFee:              200,
	})

	signedTransaction, err := txBuyOffer.Sign(network.TestNetworkPassphrase, distributionKp)

	resp, err := client.SubmitTransaction(signedTransaction)

	_ = resp

}

func UpdateAsset() {

	client := horizonclient.DefaultTestNetClient

	sourceKp, err := keypair.ParseFull(tokenCreatorSeed)
	if err != nil {
		log.Fatal(err)
	}

	issuerKp, err := keypair.ParseFull(issuerSeed)
	if err != nil {
		log.Fatal(err)
	}

	distributionKp, err := keypair.ParseFull(distributionSeed)
	if err != nil {
		log.Fatal(err)
	}

	_ = sourceKp
	_ = issuerKp
	_ = distributionKp

	sourceAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: sourceKp.Address()})

	// Create and fund source account, if it doesn't exist
	if sourceAccountDetail.AccountID != sourceKp.Address() {
		client.Fund(sourceKp.Address())
		sourceAccountDetail, _ = client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID: sourceKp.Address()})
	}

	issuerAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: issuerKp.Address()})

	// Create trust line
	tokenAsset := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: issuerKp.Address(),
	}

	createAssets := txnbuild.Payment{
		Destination:   distributionKp.Address(),
		Amount:        "90000000",
		Asset:         tokenAsset,
	}


	txCreateAssets,err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &issuerAccountDetail,
		BaseFee: 200,
		IncrementSequenceNum: true,
		Operations:    []txnbuild.Operation{&createAssets},
		Timebounds:    txnbuild.NewTimeout(common.StellarImmediateOperationTimeoutSec),
	})

	if err != nil {
		log.Fatal(err)
	}

	txCreateAssets,_ = txCreateAssets.Sign(network.TestNetworkPassphrase, issuerKp)
	xdr,_ := txCreateAssets.Base64()
	_ = xdr


	_, err = client.SubmitTransaction(txCreateAssets)
}
