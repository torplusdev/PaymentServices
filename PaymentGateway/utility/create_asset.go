package utility

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"log"
)

// Source
const tokenCreatorSeed = "SAT3ZXAC5IQHF753DLROYVW5HRZGGFB2BHEXDWMDHCHE2URPSSDW3NY5"

// Issuing account
const issuerSeed = "SBMCAMFAYTXFIXBAOZJE5X2ZX4TJQI5X6P6NE5SHOEBHLHEMGKANRTOQ"

// Distribution account
const distributionSeed = "SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM"

func createAsset() {

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

		txCreateAccounts := txnbuild.Transaction{
			SourceAccount: &sourceAccountDetail,
			Operations:    []txnbuild.Operation{&createIssuerAccount, &createDistributionAccount},
			Timebounds:    txnbuild.NewTimeout(300),
			Network:       network.TestNetworkPassphrase,
		}

		txCreateAccounts.Build()
		txCreateAccounts.Sign(sourceKp)

		resp, err := client.SubmitTransaction(txCreateAccounts)

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
		Limit:         "100000",
	}

	txCreateTrustLine := txnbuild.Transaction{
		SourceAccount: &distributionAccountDetail,
		Operations:    []txnbuild.Operation{&changeTrust},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	xdr, err := txCreateTrustLine.BuildSignEncode(distributionKp)

	_ = xdr
	if err != nil {
		log.Print("Error signing transaction:")
	}

	resp, err := client.SubmitTransaction(txCreateTrustLine)

	createAssets := txnbuild.Payment{
		Destination:   distributionKp.Address(),
		Amount:        "10000",
		Asset:         tokenAsset,
		SourceAccount: &issuerAccountDetail,
	}

	txCreateAssets := txnbuild.Transaction{
		SourceAccount: &issuerAccountDetail,
		Operations:    []txnbuild.Operation{&createAssets},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txCreateAssets.Build()
	txCreateAssets.Sign(issuerKp)

	resp, err = client.SubmitTransaction(txCreateAssets)

	homedomain := "www.adwayser.com"

	// Asset creation: set home domain
	setOptionsSetHomedomain := txnbuild.SetOptions{
		HomeDomain:    &homedomain,
		SourceAccount: &issuerAccountDetail,
	}

	txSetOptionsSetHomedomain := txnbuild.Transaction{
		SourceAccount: &issuerAccountDetail,
		Operations:    []txnbuild.Operation{&setOptionsSetHomedomain},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	err = txSetOptionsSetHomedomain.Build()
	err = txSetOptionsSetHomedomain.Sign(issuerKp)

	resp, err = client.SubmitTransaction(txSetOptionsSetHomedomain)

	_ = resp
}

func main() {
	createAsset()
}
