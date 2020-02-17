package payment_tests

import (
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"log"
	"math"
	"os"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/tests/mocks"
	"strconv"

	client2 "paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/root"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
)

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

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestSingleE2EPayment(t *testing.T) {

	// Initialization
	assert := assert.New(t)
	keyUser, _ := keypair.ParseFull(user1Seed)

	keyService, _ := keypair.ParseFull(service1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client2.CreateClient(rootApi, user1Seed)
	assert.NotNil(client)

	accPre, err := testutils.GetAccount(keyService.Address())

	assert.NoError(err)

	pr := common.PaymentRequest{
		Address:    keyService.Address(),
		Amount:     100,
		Asset:      "XLM",
		ServiceRef: "test"}

	nodes := mocks.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

	// Initiate
	transactions := client.InitiatePayment(nodes, pr)
	assert.NotNil(transactions)

	// Verify
	ok, err := client.VerifyTransactions(nodes, pr, transactions)
	assert.True(ok && err == nil)

	// Commit
	ok, err = client.FinalizePayment(nodes, transactions)

	for _, t := range *transactions {
		log.Print(testutils.Print(t))
	}

	assert.True(ok && err == nil)

	accPost, err := testutils.GetAccount(keyService.Address())
	assert.NoError(err)

	strPre, _ := accPre.GetNativeBalance()
	strPost, _ := accPost.GetNativeBalance()

	iPost, _ := strconv.ParseFloat(strPost, 64)
	iPre, _ := strconv.ParseFloat(strPre, 64)

	delta := iPost - iPre

	assert.True(math.Abs(delta-100) < 1e-3)

}
