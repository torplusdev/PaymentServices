package single_hop_tests

import (
	"context"
	"go.opentelemetry.io/otel/api/core"
	"google.golang.org/grpc/codes"
	"os"
	. "paidpiper.com/payment-gateway/common"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
	"time"
)

func setup() {

}

var testSetup *TestSetup

var tracerShutdown func()

func init() {

	traceProvider, shutdownFunc := testutils.InitGlobalTracer()
	InitializeTracer(traceProvider)

	tracerShutdown = shutdownFunc

	// Addresses reused from other tests
	testutils.CreateAndFundAccount(testutils.User1Seed)
	testutils.CreateAndFundAccount(testutils.Service1Seed)

	// Addresses specific to this test suite
	testutils.CreateAndFundAccount(testutils.Node1Seed)

	testSetup = CreateTestSetup()
	torPort := 57842

	testSetup.ConfigureTor(torPort)

	tr := CreateTracer("TestInit")

	ctx,span := tr.Start(context.Background(),"NodeInitialization")

	testSetup.StartUserNode(ctx,testutils.User1Seed,28080)
	testSetup.StartTorNode(ctx, testutils.Node1Seed,28081)
	testSetup.StartServiceNode(ctx, testutils.Service1Seed,28084)
	span.SetStatus(codes.OK,"All Nodes Stared Up" )
	
	testSetup.SetDefaultPaymentRoute([]string {})

	span.End()

	// Wait for everything to start up
	time.Sleep(2 * time.Second)

}

func shutdown() {
	testSetup.Shutdown()
	tracerShutdown()
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	_ = code
	os.Exit(code)
}

func TestSingleChainPayment(t *testing.T) {

	assert, ctx, span := testutils.InitTestCreateSpan(t,"TestSingleChainPayment")
	defer span.End()

	balancesPre := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed})
	span.SetAttributes(core.KeyValue{
		Key:   "userPreBalance",
		Value: core.Float64(balancesPre[0]) },
		core.KeyValue{
			Key: "servicePreBalance",
			Value: core.Float64(balancesPre[1]) },
		)
	sequencer := createSequencer(testSetup,assert,ctx)
	paymentAmount := 30e6

	result, paymentRequest := sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount)
	assert.Contains(result,"Payment processing completed")

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed})
	actualAmount := float64(paymentRequest.Amount)

	assert.InEpsilon(balancesPre[0] - actualAmount,balancesPost[0],1E-6,"Incorrect user balance")
	assert.InEpsilon(balancesPre[1] + actualAmount,balancesPost[1],1E-6,"Incorrect service balance")
}

func TestTwoChainPayments(t *testing.T) {

	assert, ctx, span := testutils.InitTestCreateSpan(t,"TestTwoChainPayments")
	defer span.End()

	balancesPre := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed})

	sequencer := createSequencer(testSetup,assert,ctx)
	paymentAmount1 := 300e6
	paymentAmount2 := 600e6

	result, pr1 := sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount1)
	assert.Contains(result,"Payment processing completed")

	result, pr2 := sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount2)

	assert.Contains(result,"Payment processing completed")
	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})

	paymentAmount := float64(pr1.Amount) + float64(pr2.Amount)

	assert.InEpsilon(balancesPre[0] - paymentAmount, balancesPost[0],1E-6,"Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1],1E-6,"Incorrect service balance")
}