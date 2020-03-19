package payment_tests_rogue

import (
	"errors"
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
	"paidpiper.com/payment-gateway/routing"
	testutils "paidpiper.com/payment-gateway/tests"
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

	nm.ReplaceNode("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		CreateRogueNode_NonidenticalSequenceNumbers(horizon.DefaultTestNetClient,
			"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
			"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC",false))

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed,nm)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)
	var servicePayment uint32 = 234

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
	ok, err = client.FinalizePayment(nodes, transactions,pr1 )


	/*     ******                    Transaction 2			*************				*/
	guid2 := xid.New()

	// Add pending credit
	serviceNode.AddPendingServicePayment(guid2.String(),servicePayment)

	pr2,err := serviceNode.CreatePaymentRequest(guid2.String())

	// Initiate
	transactions,err = client.InitiatePayment(nodes, pr2)

	for _,t := range transactions {

		ptr := t
		payTrans := ptr.GetPaymentTransaction()
		refTrans := ptr.GetReferenceTransaction()

		payTransStellar,_ := txnbuild.TransactionFromXDR(payTrans.XDR)
		refTransStellar,_ := txnbuild.TransactionFromXDR(refTrans.XDR)

		paySequenceNumber,_ := payTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()
		refSequenceNumber,_ := refTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()

		_ = paySequenceNumber
		_ = refSequenceNumber
	}


	// Verify
	ok, err = client.VerifyTransactions(nodes, pr2, transactions)
	var e *common.TransactionValidationError
	assert.True(errors.As(err,&e), e.Error())

}

func TestAccumulatingTransactionWithBadSignatureShouldFail(t *testing.T) {

	assert := assert.New(t)

	nm.ReplaceNode("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		CreateRogueNode_BadSignature(horizon.DefaultTestNetClient,
			"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
			"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN",false))

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed,nm)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)
	var servicePayment uint32 = 234

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
	ok, err = client.FinalizePayment(nodes, transactions,pr1 )
	assert.Error(err)
}
