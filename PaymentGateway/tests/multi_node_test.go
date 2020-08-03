package tests

const user1Seed = "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
const service1Seed = "SBBNHWCWUFLM4YXTF36WUZP4A354S75BQGFGUMSAPCBTN645TERJAC34"
const service1Port = 28084

// public GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN
const node1Seed = "SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN"

// public GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ
const node2Seed = "SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC"

// public GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW
const node3Seed = "SBZMAHJPLZLDKJU4DUIT6AU3BEVWKPGP6M6L2KWZXAELKNAIDADGZO7A"

// publc GASFIR7LHA2IAAMLN4WMBKPSFL6GSQGWHF3E7PHHGFADT254PBOOY2I7
const node4Seed = "SBVOHS5MWK5OHDFSCURZD7XZXTETKSRTKSFMU2IKJXUBM23I5FJHWDXK"

var nm *TestNodeManager

/*
func setupTestNodeManager(m *testing.M) {
	nm = CreateTestNodeManager()

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
*/

// func TestPaymentByClientWithInsufficientBalanceFails(t *testing.T) {

// 	// Initialization
// 	assert := assert.New(t)

// 	keyUser, _ := keypair.ParseFull(user1Seed)
// 	keyService, _ := keypair.ParseFull(service1Seed)

// 	rootApi := root.CreateRootApi(true)
// 	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

// 	var client = client.CreateClient(rootApi, user1Seed, nm, nil)
// 	assert.NotNil(client)

// 	accPre, err := GetAccount(keyService.Address())

// 	assert.NoError(err)

// 	currentBalance, err := strconv.ParseFloat(accPre.Balances[0].Balance, 32)
// 	assert.NoError(err)

// 	paymentAmount := common.TransactionAmount(currentBalance + 100/common.PPTokenUnitPrice)

// 	pr := common.PaymentRequest{
// 		Address:    keyService.Address(),
// 		Amount:     paymentAmount,
// 		Asset:      "XLM",
// 		ServiceRef: "test"}

// 	router := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

// 	transactions, err := client.InitiatePayment(context.Background(), router, pr)
// 	assert.Error(err, "Client has insufficient account balance")
// 	assert.Nil(transactions)
// }

/*
func TestPaymentsChainWithAccumulation(t *testing.T) {

	// Initialization
	assert := assert.New(t)

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client = client.CreateClient(rootApi, user1Seed, nm, nil)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)
	var servicePayment uint32 = 123e6

	//Service
	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})


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
*/
