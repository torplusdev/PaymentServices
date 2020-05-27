package integration_tests

import (
	"github.com/stellar/go/keypair"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
)

func TestMultinodePayments(t *testing.T) {

	N := 3

	assert, ctx, span := testutils.InitTestCreateSpan(t,"TestMultinodePayments")
	defer span.End()

	sequencer := createSequencer(testSetup,assert,ctx)

	paymentAmount := 200.0

	nodes := make([]string,0)
	seeds := make([]string,0)

	port := 28085

	// Create N accounts for clients
	for i:=0; i<N;i++ {
		kp,_ := keypair.Random()

		testutils.CreateAndFundAccount(kp.Seed())
		nodes = append(nodes,kp.Address() )
		seeds = append(seeds,kp.Seed() )

		testSetup.StartUserNode(ctx,kp.Seed(),port + i)
	}

	// Get initial balances
	balancesPre := testutils.GetAccountBalances(seeds)

	for i:=0; i<N;i++ {
		testSetup.torMock.SetCircuitOrigin(nodes[i])
		result,_ := sequencer.performPayment(seeds[i], testutils.Service1Seed, paymentAmount)
		assert.Contains(result, "Payment processing completed")
	}

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances(seeds)

	paymentRoutingFees := float64(3*10)

	for i:=0; i<N;i++ {
		assert.InEpsilon(balancesPre[i] - paymentAmount - paymentRoutingFees,balancesPost[i],1E-6,"Incorrect user balance")
	}
}
