package root

import (
	"os"
	"testing"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	txnbuild "github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"paidpiper.com/payment-gateway/models"
	testutils "paidpiper.com/payment-gateway/tests/util"
)

const gw1Seed = "SBZIQ67KEAM3T5M6VQBVKAPBXCL5GMIEPYSJZHDRVDBMQRUYLQGWRJWO"

const user1Seed = "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
const service1Seed = "SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34"

func setup() {
	testutils.CreateAndFundAccount(gw1Seed, testutils.Node)
	testutils.CreateAndFundAccount(user1Seed, testutils.Node)
	testutils.CreateAndFundAccount(service1Seed, testutils.Node)
}

func shutdown() {

}

const seed = "SAVD5NOJUVUJJIRFMPWSVIP4S6PXSEWAYWAG4WOALSSLKLVONW4YL3VT"

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestUserAccountCreation(t *testing.T) {
	assert := assert.New(t)
	k, _ := keypair.Random()

	/**** *User Creation ***/

	client, err := createTestRootApi(k.Seed(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	assert.NoError(client.CreateUser())
	rootAccountDetail, errAccount := testutils.GetAccount(k.Address())

	if errAccount != nil {
		t.Errorf("Account should exist")
	}

	if rootAccountDetail.AccountID != k.Address() {
		t.Errorf("Account should have correct address")
	}

	/**** *User cannot perform payment with only his signature ***/
	destination, _ := keypair.Parse(gw1Seed)

	op := &txnbuild.Payment{
		SourceAccount: k.Address(),
		Amount:        "1",
		Destination:   destination.Address(),
		Asset: txnbuild.CreditAsset{
			Code:   models.PPTokenAssetName,
			Issuer: models.PPTokenIssuerAddress,
		},
	}
	tp := txnbuild.TransactionParams{
		SourceAccount:        &txnbuild.SimpleAccount{},
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{op},
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewTimeout(300),
	}
	tx, err := txnbuild.NewTransaction(tp)
	assert.Nil(err)

	txe, err := client.Sign(tx)
	assert.Nil(err)

	txStr, err := txe.Base64()
	assert.Nil(err)

	_, err = horizonclient.DefaultTestNetClient.SubmitTransactionXDR(txStr)
	for k, v := range err.(*horizonclient.Error).Problem.Extras["result_codes"].(map[string]interface{}) {
		assert.True(k == "transaction" && v == "tx_bad_auth", "User signature is insufficient for executing payment orders")
	}

	op = &txnbuild.Payment{
		Destination: destination.Address(),
		Amount:      "1",
		Asset: txnbuild.CreditAsset{
			Code:   models.PPTokenAssetName,
			Issuer: models.PPTokenIssuerAddress,
		},
		SourceAccount: k.Address(),
		//	Asset         Asset
		//	SourceAccount string
	}
	tp = txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{
			AccountID: k.Address(),
			Sequence:  1, //TODO TODO ??
		},
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{op},
		BaseFee:              txnbuild.MinBaseFee,
		Timebounds:           txnbuild.NewTimeout(300),
	}
	tx, err = txnbuild.NewTransaction(tp)
	assert.Nil(err)

	//TODO: Remove explicit root seed
	mc, err := createTestRootApi("SAVD5NOJUVUJJIRFMPWSVIP4S6PXSEWAYWAG4WOALSSLKLVONW4YL3VT", 600)
	if err != nil {
		assert.Nil(err)
	}
	txe, err = mc.Sign(tx)
	assert.Nil(err)

	txStr, err = txe.Base64()
	assert.Nil(err)

	_, err = horizonclient.DefaultTestNetClient.SubmitTransactionXDR(txStr)
	assert.Nil(err)
}
