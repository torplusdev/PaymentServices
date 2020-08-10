package tests

import (
	"reflect"
)

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

// func TestAccumulatingTransactionWithDifferentSequencesShouldFail(t *testing.T) {
// 	assert := assert.New(t)
// 	testSetup.StartServiceNode(context.TODO(), service1Seed, service1Port)
// 	nm.ReplaceNode("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
// 		CreateRogueNode_NonidenticalSequenceNumbers("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
// 			"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC", false))

// 	keyUser, _ := keypair.ParseFull(user1Seed)

// 	rootApi := root.CreateRootApi(true)
// 	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

// 	var client = client.CreateClient(rootApi, user1Seed, nm, nil)
// 	assert.NotNil(client)

// 	nm.SetAccumulatingTransactionsMode(true)
// 	var servicePayment uint32 = 234

// 	//Service
// 	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

// 	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

// 	/*     ******                    Transaction 1			***********				*/
// 	guid1 := xid.New()

// 	// Add pending credit
// 	serviceNode.AddPendingServicePayment(guid1.String(), servicePayment)
// 	pr1, err := serviceNode.CreatePaymentRequest(guid1.String())

// 	// Initiate
// 	transactions, err := client.InitiatePayment(context.TODO(), nodes, pr1)
// 	assert.NotNil(transactions)

// 	// Verify
// 	ok, err := client.VerifyTransactions(context.TODO(), nodes, pr1, transactions)
// 	assert.NoError(err)
// 	assert.True(ok && err == nil)

// 	// Commit
// 	ok, err = client.FinalizePayment(context.TODO(), nodes, transactions, pr1)

// 	/*     ******                    Transaction 2			*************				*/
// 	guid2 := xid.New()

// 	// Add pending credit
// 	serviceNode.AddPendingServicePayment(guid2.String(), servicePayment)

// 	pr2, err := serviceNode.CreatePaymentRequest(guid2.String())

// 	// Initiate
// 	transactions, err = client.InitiatePayment(context.TODO(), nodes, pr2)

// 	for _, t := range transactions {

// 		ptr := t
// 		payTrans := ptr.GetPaymentTransaction()
// 		refTrans := ptr.GetReferenceTransaction()

// 		payTransStellar, _ := txnbuild.TransactionFromXDR(payTrans.XDR)
// 		refTransStellar, _ := txnbuild.TransactionFromXDR(refTrans.XDR)

// 		paySequenceNumber, _ := payTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()
// 		refSequenceNumber, _ := refTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()

// 		_ = paySequenceNumber
// 		_ = refSequenceNumber
// 	}

// 	// Verify
// 	ok, err = client.VerifyTransactions(context.TODO(), nodes, pr2, transactions)
// 	var e *common.TransactionValidationError
// 	assert.True(errors.As(err, &e), e.Error())

// }

// func TestAccumulatingTransactionWithBadSignatureShouldFail(t *testing.T) {

// 	assert := assert.New(t)

// 	nm.ReplaceNode("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
// 		CreateRogueNode_BadSignature(horizon.DefaultTestNetClient,
// 			"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
// 			"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN", false))

// 	keyUser, _ := keypair.ParseFull(user1Seed)

// 	rootApi := root.CreateRootApi(true)
// 	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

// 	var client = client.CreateClient(rootApi, user1Seed, nm, nil)
// 	assert.NotNil(client)

// 	nm.SetAccumulatingTransactionsMode(true)
// 	var servicePayment uint32 = 234

// 	//Service
// 	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

// 	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

// 	/*     ******                    Transaction 1			***********				*/
// 	guid1 := xid.New()

// 	// Add pending credit
// 	serviceNode.AddPendingServicePayment(guid1.String(), servicePayment)
// 	pr1, err := serviceNode.CreatePaymentRequest(guid1.String())

// 	// Initiate
// 	transactions, err := client.InitiatePayment(context.TODO(), nodes, pr1)
// 	assert.NotNil(transactions)

// 	// Verify
// 	ok, err := client.VerifyTransactions(context.TODO(), nodes, pr1, transactions)
// 	assert.NoError(err)
// 	assert.True(ok && err == nil)

// 	// Commit
// 	ok, err = client.FinalizePayment(context.TODO(), nodes, transactions, pr1)
// 	assert.Error(err)
// }
