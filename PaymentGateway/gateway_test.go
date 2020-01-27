package gateway

import (
	"github.com/stellar/go/keypair"
	"os"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
)
import "paidpiper.com/payment-gateway/root"

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


func TestCreateUserAccount(t *testing.T) {

	k,_ := keypair.Random()

	rootApi := root.CreateRootApi(true)

	rootApi.CreateUser(k.Address(),k.Seed())

	rootAccountDetail, errAccount := testutils.GetAccount(k.Address())

	if errAccount != nil {
		t.Errorf("Account should exist")
	}

	if rootAccountDetail.AccountID !=  k.Address() {
		t.Errorf("Account should have correct address")
	}
}