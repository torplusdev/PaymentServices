package special

import (
	"paidpiper.com/payment-gateway/utility"
	"testing"
)

func setup() {

}
//
//func TestSingleHopSingleChainPayment(t *testing.T) {
//
//	// Set route to empty
//	testSetup.SetDefaultPaymentRoute([]string{})
//
//	assert, ctx, span := InitTestCreateSpan(t, "TestSingleChainPayment")
//	defer span.End()
//
//	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed})
//	span.SetAttributes(core.KeyValue{
//		Key:   "userPreBalance",
//		Value: core.Float64(balancesPre[0])},
//		core.KeyValue{
//			Key:   "servicePreBalance",
//			Value: core.Float64(balancesPre[1])},
//	)
//	sequencer := CreateSequencer(testSetup, assert, ctx)
//	paymentAmount := 80e6
//
//	result, paymentRequest := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount)
//	assert.Contains(result, "Payment processing completed")
//
//	err := testSetup.FlushTransactions(ctx)
//	assert.NoError(err)
//
//	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed})
//	actualAmount := common.PPtoken2MicroPP(paymentRequest.Amount)
//
//	assert.InEpsilon(balancesPre[0]-actualAmount, balancesPost[0], 1E-6, "Incorrect user balance")
//	assert.InEpsilon(balancesPre[1]+actualAmount, balancesPost[1], 1E-6, "Incorrect service balance")
//}
//
//func TestSingleHopTwoChainPayments(t *testing.T) {
//
//	assert, ctx, span := InitTestCreateSpan(t, "TestTwoChainPayments")
//	defer span.End()
//
//	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})
//
//	sequencer := CreateSequencer(testSetup, assert, ctx)
//	paymentAmount1 := 300e6
//	paymentAmount2 := 600e6
//
//	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)
//	assert.Contains(result, "Payment processing completed")
//
//	result, pr2 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount2)
//
//	assert.Contains(result, "Payment processing completed")
//	err := testSetup.FlushTransactions(ctx)
//	assert.NoError(err)
//
//	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})
//
//	paymentAmount := float64(pr1.Amount) + float64(pr2.Amount)
//
//	assert.InEpsilon(balancesPre[0]-paymentAmount, balancesPost[0], 1E-6, "Incorrect user balance")
//	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1], 1E-6, "Incorrect service balance")
//}

func TestSimpleIssueTokens(t *testing.T) {
	utility.CreateAsset()
}

func TestCreateAsset(t *testing.T) {
	utility.UpdateAsset()
}

func TestSubmitBuyOffer(t *testing.T) {
	utility.SubmitBuyOffer()
}
