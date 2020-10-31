package tests

import (
	"context"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stretchr/testify/assert"
	client "paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/routing"
	"reflect"
	"testing"
)

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
		CreateRogueNode_NonidenticalSequenceNumbers("GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
			"SDK7QBPKP5M7SCU7XZVWAIUJW2I2SM4PQJMWH5PSCMAI7WF3A4HRHVVC",false))

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client,_ = client.CreateClient(rootApi, user1Seed, nm, nil)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)

	//Service
	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

	/*     ******                    Transaction 1			***********				*/
	//guid1 := xid.New()

	paymentRequestProvider := serviceNode.(node.PPPaymentRequestProvider)

	// Add pending credit
	//serviceNode.AddPendingServicePayment(guid1.String(),servicePayment)
	pr1,err := paymentRequestProvider.CreatePaymentRequest(context.Background(),100e6,"data","ipfs")

	// Initiate
	transactions,err := client.InitiatePayment(context.Background(),nodes, pr1)
	assert.NotNil(transactions)

	// Verify
	err = client.VerifyTransactions(context.Background(),nodes, pr1, transactions)
	assert.NoError(err)
	// Commit
	err = client.FinalizePayment(context.Background(),nodes, transactions,pr1 )

	/*     ******                    Transaction 2			*************				*/
	//guid2 := xid.New()

	// Add pending credit
	//serviceNode.AddPendingServicePayment(guid2.String(),servicePayment)

	pr2,err := paymentRequestProvider.CreatePaymentRequest(context.Background(),100e6,"data","ipfs")

	// Initiate
	transactions,err = client.InitiatePayment(context.Background(),nodes, pr2)

	for _,t := range transactions {

		ptr := t
		payTrans := ptr.GetPaymentTransaction()
		refTrans := ptr.GetReferenceTransaction()

		payTransStellarWrapper,_ := txnbuild.TransactionFromXDR(payTrans.XDR)
		payTransStellar,_ := payTransStellarWrapper.Transaction()

		refTransStellarWrapper,_ := txnbuild.TransactionFromXDR(refTrans.XDR)
		refTransStellar,_ := refTransStellarWrapper.Transaction()

		account := payTransStellar.SourceAccount()
		paySequenceNumber,_ := account.GetSequenceNumber()

		account = refTransStellar.SourceAccount()
		refSequenceNumber,_ := account.GetSequenceNumber()

		_ = paySequenceNumber
		_ = refSequenceNumber
	}


	// Verify
	err = client.VerifyTransactions(context.Background(),nodes, pr2, transactions)
	var e *common.TransactionValidationError
	assert.EqualError(err,e.Err.Error())
	//assert.True(errors.As(err,&e), e.Error())

}

func TestAccumulatingTransactionWithBadSignatureShouldFail(t *testing.T) {

	assert := assert.New(t)

	nm.ReplaceNode("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		CreateRogueNode_BadSignature("GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
			"SCEV4AU2G4NYAW76P46EVM77N5TL2NLW2IYO5TJSLB6S4OBBJQ62ZVJN",false))

	keyUser, _ := keypair.ParseFull(user1Seed)

	rootApi := root.CreateRootApi(true)
	rootApi.CreateUser(keyUser.Address(), keyUser.Seed())

	var client,_ = client.CreateClient(rootApi, user1Seed, nm, nil)
	assert.NotNil(client)

	nm.SetAccumulatingTransactionsMode(true)
	//var servicePayment uint32 = 234

	//Service
	serviceNode := nm.GetNodeByAddress("GCCGR53VEHVQ2R6KISWXT4HYFS2UUM36OVRTECH2G6OVEULBX3CJCOGE")

	nodes := routing.CreatePaymentRouterStubFromAddresses([]string{user1Seed, node1Seed, node2Seed, node3Seed, service1Seed})

	paymentRequestProvider := serviceNode.(node.PPPaymentRequestProvider)

	/*     ******                    Transaction 1			***********				*/
	//guid1 := xid.New()

	// Add pending credit
	//serviceNode.AddPendingServicePayment(guid1.String(),servicePayment)
	pr1,err := paymentRequestProvider.CreatePaymentRequest(context.Background(),100e6,"data","ipfs")

	// Initiate
	transactions,err := client.InitiatePayment(context.Background(), nodes, pr1)
	assert.NotNil(transactions)

	// Verify
	err = client.VerifyTransactions(context.Background(), nodes, pr1, transactions)
	assert.NoError(err)

	// Commit
	err = client.FinalizePayment(context.Background(), nodes, transactions,pr1 )
	assert.Error(err)
}
