package tests

import (
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/common"
	"testing"
)

func TestMultinodePayments(t *testing.T) {

	N := 3

	assert, ctx, span := InitTestCreateSpan(t,"TestMultinodePayments")
	defer span.End()

	sequencer := CreateSequencer(testSetup,assert,ctx)

	paymentAmount := 200e6

	nodes := make([]string,0)
	seeds := make([]string,0)

	port := 28085

	// Create N accounts for clients
	for i:=0; i<N;i++ {
		kp,_ := keypair.Random()

		CreateAndFundAccount(kp.Seed(),Node)
		nodes = append(nodes,kp.Address() )
		seeds = append(seeds,kp.Seed() )

		testSetup.StartUserNode(ctx,kp.Seed(),port + i)
	}

	// Get initial balances
	balancesPre := GetAccountBalances(seeds)
	var amount common.TransactionAmount = 0

	for i:=0; i<N;i++ {
		testSetup.torMock.SetCircuitOrigin(nodes[i])

		result,pr := sequencer.PerformPayment(seeds[i], Service1Seed, paymentAmount)
		amount = pr.Amount
		assert.Contains(result, "Payment processing completed")
	}

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := GetAccountBalances(seeds)

	paymentRoutingFees := float64(3*10)

	for i:=0; i<N;i++ {
		assert.InEpsilon(balancesPre[i] - float64(amount) - paymentRoutingFees,balancesPost[i],1E-6,"Incorrect user balance")
	}
}

