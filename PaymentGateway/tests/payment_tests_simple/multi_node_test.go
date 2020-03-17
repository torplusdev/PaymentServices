package payment_tests_simple

import (
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
	"log"
	"math"
	"os"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/routing"
	"strconv"

	xid "github.com/rs/xid"
	client "paidpiper.com/payment-gateway/client"
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

var nm *testutils.TestNodeManager

func setupTestNodeManager(m *testing.M) {
	nm = testutils.CreateTestNodeManager()

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN",false))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC",false))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW",
		"SBZMAHJPLZLDKJU4DUIT6AU3BEVWKPGP6M6L2KWZXAELKNAIDADGZO7A",false))

	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GASFIR7LHA2IAAMLN4WMBKPSFL6GSQGWHF3E7PHHGFADT254PBOOY2I7",
		"SBVOHS5MWK5OHDFSCURZD7XZXTETKSRTKSFMU2IKJXUBM23I5FJHWDXK",false))

	// service
	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE",
		"SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34",false))

	// client
	nm.AddNode(node.CreateNode(horizon.DefaultTestNetClient,
		"GBFQ5SXDQAU5LVJFOUYXZXPUGNJIDHAYIOD4PTJCJJNQSHOWWZF5FQTP",
		"SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU",false))
}

func TestMain(m *testing.M) {
	setup()
	setupTestNodeManager(m)
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestSingleE2EPaymentNoAccumulation(t *testing.T) {

	// Initialization
	assert := assert.New(t)
	keyUser, _ := keypair.ParseFull(user1Seed)

	keyService, _ := keypair.ParseFull(service1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed, nm)
	assert.NotNil(client)

	accPre, err := testutils.GetAccount(keyService.Address())

	assert.NoError(err)

	var servicePayment uint32 = 300

	//Service
	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

	guid := xid.New()

	// Add pending credit
	serviceNode.AddPendingServicePayment(guid.String(),servicePayment)

	// Client
	pr,err := serviceNode.CreatePaymentRequest(guid.String())

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

	// Initiate
	transactions,err := client.InitiatePayment(nodes, pr)
	assert.NotNil(transactions)

	// Verify
	ok, err := client.VerifyTransactions(nodes, pr, transactions)
	assert.NoError(err)
	assert.True(ok && err == nil)

	// Commit
	ok, err = client.FinalizePayment(nodes, transactions,pr)

	for _, t := range transactions {
		log.Print(testutils.Print(t.GetPaymentTransaction()))
	}

	assert.True(ok && err == nil)

	accPost, err := testutils.GetAccount(keyService.Address())
	assert.NoError(err)

	strPre, _ := accPre.GetNativeBalance()
	strPost, _ := accPost.GetNativeBalance()

	iPost, _ := strconv.ParseFloat(strPost, 64)
	iPre, _ := strconv.ParseFloat(strPre, 64)

	delta := iPost - iPre

	assert.True(math.Abs(delta - float64(servicePayment)) < 1e-3)
}

func TestPaymentByClientWithInsufficientBalanceFails(t *testing.T) {

	// Initialization
	assert := assert.New(t)

	keyUser, _ := keypair.ParseFull(user1Seed)
	keyService, _ := keypair.ParseFull(service1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed,nm)
	assert.NotNil(client)

	accPre, err := testutils.GetAccount(keyService.Address())

	assert.NoError(err)

	currentBalance,err := strconv.ParseFloat(accPre.Balances[0].Balance,32)
	assert.NoError(err)

	paymentAmount := common.TransactionAmount(currentBalance+100)

	pr := common.PaymentRequest{
		Address:    keyService.Address(),
		Amount:     paymentAmount,
		Asset:      "XLM",
		ServiceRef: "test"}

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

	transactions,err := client.InitiatePayment(nodes, pr)
	assert.Error(err,"Client has insufficient account balance")
	assert.Nil(transactions)
}

func TestPaymentsChainWithAccumulation(t *testing.T) {

	// Initialization
	assert := assert.New(t)

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed,nm)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)
	var servicePayment uint32 = 123

	//Service
	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})


	/*     ******                    Transaction 1			***********				*/
	guid1 := xid.New()

	// Add pending credit
	serviceNode.AddPendingServicePayment(guid1.String(),servicePayment)
	pr1,err := serviceNode.CreatePaymentRequest(guid1.String())

	// Initiate
	transactions,err := client.InitiatePayment(nodes, pr1)
	assert.NotNil(transactions)

	// Verify
	ok, err := client.VerifyTransactions(nodes, pr1, transactions)
	assert.NoError(err)
	assert.True(ok && err == nil)

	// Commit
	ok, err = client.FinalizePayment(nodes, transactions, pr1)

	/*     ******                    Transaction 2			*************				*/
	guid2 := xid.New()

	// Add pending credit
	serviceNode.AddPendingServicePayment(guid2.String(),servicePayment)

	pr2,err := serviceNode.CreatePaymentRequest(guid2.String())

	// Initiate
	transactions,err = client.InitiatePayment(nodes, pr2)
	assert.NotNil(transactions)

	// Verify
	ok, err = client.VerifyTransactions(nodes, pr2, transactions)
	assert.NoError(err)
	assert.True(ok && err == nil)

	// Commit
	ok, err = client.FinalizePayment(nodes, transactions, pr2)
	assert.True(ok && err == nil)

}