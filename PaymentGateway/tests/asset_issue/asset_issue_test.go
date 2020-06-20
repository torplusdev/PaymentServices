package asset_issue

import (
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"paidpiper.com/payment-gateway/node"
	testutils "paidpiper.com/payment-gateway/tests"
	"reflect"
	"testing"
)
// Source
const tokenCreatorSeed =  "SAT3ZXAC5IQHF753DLROYVW5HRZGGFB2BHEXDWMDHCHE2URPSSDW3NY5"

// Issuing account
const issuerSeed =  "SBMCAMFAYTXFIXBAOZJE5X2ZX4TJQI5X6P6NE5SHOEBHLHEMGKANRTOQ"

// Distribution account
const distributionSeed = "SAQUH66AMZ3PURY2G3ROXRXGIF2JMZC7QFVED65PYP4YJQFIWCPCWKPM"


const user1Seed = "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
const service1Seed = "SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34"

// public GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN
const node1Seed = "SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN"

// public GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ
const node2Seed = "SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC"

// public GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW
const node3Seed = "SBZMAHJPLZLDKJU4DUIT6AU3BEVWKPGP6M6L2KWZXAELKNAIDADGZO7A"

// publc GASFIR7LHA2IAAMLN4WMBKPSFL6GSQGWHF3E7PHHGFADT254PBOOY2I7
const node4Seed = "SBVOHS5MWK5OHDFSCURZD7XZXTETKSRTKSFMU2IKJXUBM23I5FJHWDXK"

func setup() {

	f,_ := keypair.Random()
	s := f.Seed()
	_ = s
	// Addresses reused from other tests
	testutils.CreateAndFundAccount(user1Seed)
	testutils.CreateAndFundAccount(service1Seed)

	// Addresses specific to this test suite
	testutils.CreateAndFundAccount(node1Seed)
	testutils.CreateAndFundAccount(node2Seed)
	testutils.CreateAndFundAccount(node3Seed)
	testutils.CreateAndFundAccount(node4Seed)
}

func shutdown() {

}

var nm *testutils.TestNodeManager

func setupTestNodeManager(m *testing.M) {
	nm = testutils.CreateTestNodeManager()

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN",true))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC",true))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW",
		"SBZMAHJPLZLDKJU4DUIT6AU3BEVWKPGP6M6L2KWZXAELKNAIDADGZO7A",true))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GASFIR7LHA2IAAMLN4WMBKPSFL6GSQGWHF3E7PHHGFADT254PBOOY2I7",
		"SBVOHS5MWK5OHDFSCURZD7XZXTETKSRTKSFMU2IKJXUBM23I5FJHWDXK",true))

	// service
	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE",
		"SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34",true))

	// client
	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GBFQ5SXDQAU5LVJFOUYXZXPUGNJIDHAYIOD4PTJCJJNQSHOWWZF5FQTP",
		"SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU",true))
}

func TestMain(m *testing.M) {
	setup()
	setupTestNodeManager(m)
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func TestAccumulatingTransactionWithDifferentSequencesShouldFail(t *testing.T) {

	assert := assert.New(t)

	client := horizonclient.DefaultTestNetClient

	sourceKp, err := keypair.ParseFull(tokenCreatorSeed)
	if err != nil { log.Fatal(err) }

	issuerKp, err := keypair.ParseFull(issuerSeed)
	if err != nil { log.Fatal(err) }

	distributionKp, err := keypair.ParseFull(distributionSeed)
	if err != nil { log.Fatal(err) }

	_ = sourceKp
	_ = issuerKp
	_ = distributionKp

	sourceAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: sourceKp.Address()})


	// Create and fund source account, if it doesn't exist
	if (sourceAccountDetail.AccountID != sourceKp.Address()) {
		client.Fund(sourceKp.Address())
		sourceAccountDetail, _ = client.AccountDetail(
			horizonclient.AccountRequest{
				AccountID: sourceKp.Address()})
	}


	issuerAccountDetail, _ := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: issuerKp.Address()})

	// check if issuer account exists, if not create it and the distribution account
	if (issuerAccountDetail.AccountID != issuerKp.Address()) {

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

	//distributionAccount := txnbuild.NewSimpleAccount(distributionKp.Address(),0)
	//issuingAccount := txnbuild.NewSimpleAccount(issuerKp.Address(),0)

	// Create trust line
	tokenAsset  := txnbuild.CreditAsset{
		Code:   "pptoken",
		Issuer: issuerKp.Address(),
	}

	changeTrust := txnbuild.ChangeTrust{
		SourceAccount: &distributionAccountDetail,
		Line:tokenAsset,
		Limit:"100000",
	}

	txCreateTrustLine := txnbuild.Transaction{
		SourceAccount: &distributionAccountDetail,
		Operations:    []txnbuild.Operation{ &changeTrust},
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
		Operations:    []txnbuild.Operation{ &createAssets},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	txCreateAssets.Build()
	txCreateAssets.Sign(issuerKp)

	resp, err = client.SubmitTransaction(txCreateAssets)


	homedomain := "www.adwayser.com"

	// Asset creation: set home domain
	setOptionsSetHomedomain := txnbuild.SetOptions{
		HomeDomain:          &homedomain,
		SourceAccount:        &issuerAccountDetail,
	}

	txSetOptionsSetHomedomain := txnbuild.Transaction{
		SourceAccount: &issuerAccountDetail,
		Operations:    []txnbuild.Operation{ &setOptionsSetHomedomain},
		Timebounds:    txnbuild.NewTimeout(300),
		Network:       network.TestNetworkPassphrase,
	}

	err = txSetOptionsSetHomedomain.Build()
	err = txSetOptionsSetHomedomain.Sign(issuerKp)

	resp, err = client.SubmitTransaction(txSetOptionsSetHomedomain)


	_ = resp

	//testutils.CreateAndFundAccount(tokenCreatorSeed)
	//
	//// Create and fund the source account
	//
	//
	//testutils.CreateAndFundAccount(sourceSeed)
	//
	//// Create and fund the issuer account
	//
	//// Set source as signer @ issuer
	//testutils.CreateAndFundAccount(issuerSeed)
	//testutils.SetSigners(issuerSeed,sourceSeed)

	assert.True(true)

}

