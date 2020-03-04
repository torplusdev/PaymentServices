package payment_tests_rogue

import (
	xid "github.com/rs/xid"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	"os"
	client "paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
	testutils "paidpiper.com/payment-gateway/tests"
	"paidpiper.com/payment-gateway/tests/mocks"
	"reflect"
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

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func createTestPayment(router common.PaymentRouter, paymentRequest common.PaymentRequest) ([]common.PaymentTransactionPayload,error) {

	route := router.CreatePaymentRoute(paymentRequest)

	transactions := make([]common.PaymentTransactionPayload,0, len(route))
	reverseAny(route)

	// Generate initial transaction
	for i, e := range route[0:len(route)-1] {

		var sourceAddress = route[i+1].Address
		stepNode := nm.GetNodeByAddress(e.Address)

		// Create and store transaction
		nodeTransaction := stepNode.CreateTransaction(paymentRequest.Amount, 0, paymentRequest.Amount, sourceAddress)
		transactions = append(transactions,nodeTransaction)
	}

	debitTransaction := transactions[0]

	serviceNode := nm.GetNodeByAddress(route[0].Address)
	serviceNode.SignTerminalTransactions(debitTransaction)

	for idx := 1; idx < len(transactions); idx++ {

		t := transactions[idx]
		stepNode := nm.GetNodeByAddress(t.GetPaymentDestinationAddress())
		creditTransaction := t

		stepNode.SignChainTransactions(creditTransaction,debitTransaction)
		debitTransaction = creditTransaction
	}

	fundingTransaction := transactions[len(transactions)-1]
	//address := route[len(transactions)-1].Address

	transaction := fundingTransaction.GetPaymentTransaction()

	t, _ := txnbuild.TransactionFromXDR(transaction.XDR)

	//op, _ := t.Operations[0].(*txnbuild.Payment)

	t.Network = transaction.StellarNetworkToken

	fullKeyPair,_ := keypair.ParseFull("SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU")

	_ = t.Sign(fullKeyPair)

	xdr,_ := t.Base64()


	_ = fundingTransaction.UpdateTransactionXDR(xdr)

	return transactions,nil
}

func TestAccumulatingTransactionWithDifferentSequencesShouldFail(t *testing.T) {

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

	nodes := mocks.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

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
	ok, err = client.FinalizePayment(nodes, transactions,pr1 )


	/*     ******                    Transaction 2			*************				*/
	guid2 := xid.New()

	// Add pending credit
	serviceNode.AddPendingServicePayment(guid2.String(),servicePayment)

	pr2,err := serviceNode.CreatePaymentRequest(guid2.String())

	// Initiate
	transactions,err = createTestPayment(nodes, pr2)

	// Verify
	ok, err = client.VerifyTransactions(nodes, pr2, transactions)
	assert.NoError(err)
	assert.True(ok && err == nil)

	// Commit
	ok, err = client.FinalizePayment(nodes, transactions, pr2)
	assert.True(ok && err == nil)

}
