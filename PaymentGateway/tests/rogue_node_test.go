package tests

import (
	"testing"

	"paidpiper.com/payment-gateway/root"
)

func TestSeed(t *testing.T) {
	seed := "SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC"
	clientItem, err := root.CreateRootApiFactory(true)(seed, 600)
	if err != nil {
		t.Error(err)
	}
	address := clientItem.GetAddress()
	if address != "GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ" {
		t.Error("Not valid address from seed")
	}
}

// func TestAccumulatingTransactionWithDifferentSequencesShouldFail(t *testing.T) {

// 	assert := assert.New(t)
// 	myNode, err := CreateRogueNode_NonidenticalSequenceNumbers("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
// 		"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC", false)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	nm.ReplaceNode("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ", myNode)

// 	rootApi, err := root.CreateRootApiFactory(true)(user1Seed, 600)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	rootApi.CreateUser()

// 	nm.SetAccumulatingTransactionsMode(true)
// 	//regestry.FromSeeds()
// 	//Service
// 	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

// 	//route := prouter.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

// 	client := client.New(rootApi)
// 	assert.NotNil(client)

// 	/*     ******                    Transaction 1			***********				*/

// 	paymentRequestProvider := serviceNode.(local.LocalPPNode)

// 	// Add pending credit
// 	//serviceNode.AddPendingServicePayment(guid1.String(),servicePayment)
// 	pib := &models.PaymentRequstBase{
// 		Amount:     100e6,
// 		Asset:      "data",
// 		ServiceRef: "ipfs",
// 	}
// 	pr1, err := paymentRequestProvider.CreatePaymentRequest(context.Background(), pib)

// 	// Initiate
// 	transactions, err := client.InitiatePayment(context.Background(), route, pr1)
// 	assert.NotNil(transactions)

// 	// Verify
// 	err = client.VerifyTransactions(context.Background(), transactions)
// 	assert.NoError(err)
// 	// Commit
// 	err = client.FinalizePayment(context.Background(), route, pr1, transactions)

// 	pr2, err := paymentRequestProvider.CreatePaymentRequest(context.Background(), pib)

// 	// Initiate
// 	transactions, err = client.InitiatePayment(context.Background(), route, pr2)

// 	for _, t := range transactions {

// 		ptr := t
// 		payTrans := ptr.PendingTransaction
// 		refTrans := ptr.ReferenceTransaction

// 		payTransStellarWrapper, _ := payTrans.XDR.TransactionFromXDR()
// 		payTransStellar, _ := payTransStellarWrapper.Transaction()

// 		refTransStellarWrapper, _ := refTrans.XDR.TransactionFromXDR()
// 		refTransStellar, _ := refTransStellarWrapper.Transaction()

// 		account := payTransStellar.SourceAccount()
// 		paySequenceNumber, _ := account.GetSequenceNumber()

// 		account = refTransStellar.SourceAccount()
// 		refSequenceNumber, _ := account.GetSequenceNumber()

// 		_ = paySequenceNumber
// 		_ = refSequenceNumber
// 	}

// 	// Verify
// 	err = client.VerifyTransactions(context.Background(), transactions)
// 	//var e *models.TransactionValidationError
// 	//assert.EqualError(err, e.Err.Error())
// 	//assert.True(errors.As(err,&e), e.Error())
// 	//TODO fix
// }

// func TestAccumulatingTransactionWithBadSignatureShouldFail(t *testing.T) {

// 	assert := assert.New(t)
// 	myNode, err := CreateRogueNode_BadSignature("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
// 		"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN", false)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	nm.ReplaceNode("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN", myNode)

// 	keyUser, _ := keypair.ParseFull(user1Seed)

// 	rootApi, err := root.CreateRootApiFactory(true)(keyUser.Seed(), 600)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	rootApi.CreateUser()

// 	nm.SetAccumulatingTransactionsMode(true)
// 	//var servicePayment uint32 = 234

// 	//Service
// 	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

// 	nodes := prouter.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

// 	client := client.New(rootApi)

// 	paymentRequestProvider := serviceNode.(local.LocalPPNode)

// 	/*     ******                    Transaction 1			***********				*/
// 	//guid1 := xid.New()
// 	pib := &models.PaymentRequstBase{
// 		Amount:     100e6,
// 		Asset:      "data",
// 		ServiceRef: "ipfs",
// 	}
// 	// Add pending credit
// 	//serviceNode.AddPendingServicePayment(guid1.String(),servicePayment)
// 	pr1, err := paymentRequestProvider.CreatePaymentRequest(context.Background(), pib)

// 	// Initiate
// 	transactions, err := client.InitiatePayment(context.Background(), nodes, pr1)
// 	assert.NotNil(transactions)

// 	// Verify
// 	err = client.VerifyTransactions(context.Background(), transactions)
// 	assert.NoError(err)

// 	// Commit
// 	err = client.FinalizePayment(context.Background(), nodes, pr1, transactions)
// 	assert.Error(err)
// }
