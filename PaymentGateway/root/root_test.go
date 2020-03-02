package root

import (
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"os"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
)

const gw1Seed = "SBZIQ67KEAM3T5M6VQBVKAPBXCL5GMIEPYSJZHDRVDBMQRUYLQGWRJWO"
const user1Seed = "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
const service1Seed = "SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34"


func setup() {
	testutils.CreateAndFundAccount(gw1Seed)
	testutils.CreateAndFundAccount(user1Seed)
	testutils.CreateAndFundAccount(service1Seed)
}

func shutdown() {

}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestUserAccountCreation(t *testing.T) {
	assert := assert.New(t)
	k,_ := keypair.Random()

	rootApi := CreateRootApi(true)

	/**** *User Creation ***/

	rootApi.CreateUser(k.Address(),k.Seed())

	rootAccountDetail, errAccount := testutils.GetAccount(k.Address())

	if errAccount != nil {
		t.Errorf("Account should exist")
	}

	if rootAccountDetail.AccountID !=  k.Address() {
		t.Errorf("Account should have correct address")
	}

	/**** *User cannot perform payment with only his signature ***/
	destination,_ := keypair.Parse(gw1Seed)

	tx, err := build.Transaction(
		build.TestNetwork,
		build.SourceAccount{k.Seed()},
		build.AutoSequence{horizon.DefaultTestNetClient},
		build.Payment(
			build.Destination{destination.Address()},
			build.NativeAmount{"1"},
		),
	)

	assert.Nil(err)

	txe,err := (tx.Sign(k.Seed()));  assert.Nil(err)

	txStr,err := txe.Base64(); 	assert.Nil(err)

	_, err = horizonclient.DefaultTestNetClient.SubmitTransactionXDR(txStr)
	for k,v := range err.(*horizonclient.Error).Problem.Extras["result_codes"].(map[string]interface{}) {
		assert.True(k == "transaction" && v == "tx_bad_auth","User signature is insufficient for executing payment orders")
	}

	/**** *Root can perform payment with only his signature ***/

	tx, err = build.Transaction(
		build.TestNetwork,
		build.SourceAccount{k.Seed()},
		build.AutoSequence{horizon.DefaultTestNetClient},
		build.Payment(
			build.Destination{destination.Address()},
			build.NativeAmount{"1"},
		),
	)

	assert.Nil(err)

	//TODO: Remove explicit root seed
	txe,err = (tx.Sign("SAVD5NOJUVUJJIRFMPWSVIP4S6PXSEWAYWAG4WOALSSLKLVONW4YL3VT"));  assert.Nil(err)

	txStr,err = txe.Base64(); 	assert.Nil(err)

	_, err = horizonclient.DefaultTestNetClient.SubmitTransactionXDR(txStr)
	assert.Nil(err)
}