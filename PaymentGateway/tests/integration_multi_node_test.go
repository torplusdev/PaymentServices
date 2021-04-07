package tests

import (
	"testing"

	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/models"
	. "paidpiper.com/payment-gateway/tests/util"
)

func TestMultinodePayments(t *testing.T) {

	N := 3

	assert, ctx, span := InitTestCreateSpan(t, "TestMultinodePayments")
	defer span.End()

	sequencer := CreateSequencer(testSetup, assert, ctx)

	var commodityAmount uint32 = 200e6

	nodes := make([]string, 0)
	seeds := make([]string, 0)

	// Create N accounts for clients
	for i := 0; i < N; i++ {
		kp, _ := keypair.Random()

		CreateAndFundAccount(kp.Seed(), Node)
		nodes = append(nodes, kp.Address())
		seeds = append(seeds, kp.Seed())

		testSetup.StartUserNode(ctx, kp.Seed())
	}

	// Get initial balances
	balancesPre := GetAccountBalances(seeds)
	var amount models.TransactionAmount = 0

	for i := 0; i < N; i++ {
		testSetup.torMock.SetCircuitOrigin(nodes[i])

		result, pr, err := sequencer.PerformPayment(seeds[i], Service1Seed, commodityAmount)
		assert.NoError(err)
		amount = pr.Amount
		assert.Contains(result, "Payment processing completed")
	}

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := GetAccountBalances(seeds)

	paymentRoutingFees := float64(3 * 10)

	for i := 0; i < N; i++ {
		assert.InEpsilon(balancesPre[i]-float64(amount)-paymentRoutingFees, balancesPost[i], 1e-6, "Incorrect user balance")
	}
}
