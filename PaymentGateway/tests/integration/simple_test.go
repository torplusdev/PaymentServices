package integration_tests

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
	testutils.CreateAndFundAccount(testutils.Node2Seed)
	testutils.CreateAndFundAccount(testutils.Node3Seed)
	testutils.CreateAndFundAccount(testutils.Node4Seed)

	testSetup = CreateTestSetup()
	torPort := 57842


	testSetup.ConfigureTor(torPort)

	tr := CreateTracer("TestInit")

	ctx,span := tr.Start(context.Background(),"NodeInitialization")

	testSetup.StartUserNode(ctx,testutils.User1Seed,28080)
	testSetup.StartTorNode(ctx, testutils.Node1Seed,28081)
	testSetup.StartTorNode(ctx, testutils.Node2Seed,28082)
	testSetup.StartTorNode(ctx, testutils.Node3Seed,28083)
	testSetup.StartServiceNode(ctx, testutils.Service1Seed,28084)
	span.SetStatus(codes.OK,"All Nodes Stared Up" )

	testSetup.SetDefaultPaymentRoute([]string {
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW"})

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


	balancesPre := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})
	span.SetAttributes(core.KeyValue{
		Key:   "userPreBalance",
		Value: core.Float64(balancesPre[0]) },
		core.KeyValue{
			Key: "servicePreBalance",
			Value: core.Float64(balancesPre[1]) },
		core.KeyValue{
		Key: "node1PreBalance",
		Value: core.Float64(balancesPre[2])	},
		core.KeyValue{
			Key: "node2PreBalance",
			Value: core.Float64(balancesPre[3])	},
		core.KeyValue{
			Key: "node3PreBalance",
			Value: core.Float64(balancesPre[4])	},
		)
	sequencer := createSequencer(testSetup,assert,ctx)
	paymentAmount := 300.0

	result := sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount)
	assert.Contains(result,"Payment processing completed")

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})

	paymentRoutingFees := float64(3*10)

	assert.InEpsilon(balancesPre[0] - paymentAmount - paymentRoutingFees,balancesPost[0],1E-6,"Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount,balancesPost[1],1E-6,"Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount)/3
	span.SetAttributes(core.KeyValue{
		Key:   "userPostBalance",
		Value: core.Float64(balancesPost[0]) },
		core.KeyValue{
			Key: "servicePostBalance",
			Value: core.Float64(balancesPost[1]) },
		core.KeyValue{
			Key: "node1PostBalance",
			Value: core.Float64(balancesPost[2])	},
		core.KeyValue{
			Key: "node2PostBalance",
			Value: core.Float64(balancesPost[3])	},
		core.KeyValue{
			Key: "node3PostBalance",
			Value: core.Float64(balancesPost[4])	},
		core.KeyValue{
			Key: "paymentAmount",
			Value: core.Float64(paymentAmount)	},
		core.KeyValue{
			Key: "paymentRoutingFees",
			Value: core.Float64(paymentRoutingFees)	},
		core.KeyValue{
			Key: "nodePaymentFee",
			Value: core.Float64(nodePaymentFee)	},
	)
	assert.InEpsilon(balancesPre[2]+nodePaymentFee,balancesPost[2],1E-6,"Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee,balancesPost[3],1E-6,"Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee,balancesPost[4],1E-6,"Incorrect node3 balance")

}

func TestTwoChainPayments(t *testing.T) {

	assert, ctx, span := testutils.InitTestCreateSpan(t,"TestTwoChainPayments")
	defer span.End()

	balancesPre := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})
	span.SetAttributes(core.KeyValue{
		Key:   "userPreBalance",
		Value: core.Float64(balancesPre[0]) },
		core.KeyValue{
			Key: "servicePreBalance",
			Value: core.Float64(balancesPre[1]) },
		core.KeyValue{
			Key: "node1PreBalance",
			Value: core.Float64(balancesPre[2])	},
		core.KeyValue{
			Key: "node2PreBalance",
			Value: core.Float64(balancesPre[3])	},
		core.KeyValue{
			Key: "node3PreBalance",
			Value: core.Float64(balancesPre[4])	},
	)

	sequencer := createSequencer(testSetup,assert,ctx)
	paymentAmount1 := 300.0
	paymentAmount2 := 600.0

	sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount1)
	result := sequencer.performPayment(testutils.User1Seed, testutils.Service1Seed, paymentAmount2)

	assert.Contains(result,"Payment processing completed")
	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})

	paymentAmount := paymentAmount1+paymentAmount2

	paymentRoutingFees := float64(3*10) * 2

	assert.InEpsilon(balancesPre[0] - paymentAmount - paymentRoutingFees,balancesPost[0],1E-6,"Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount,balancesPost[1],1E-6,"Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount)/3
	span.SetAttributes(core.KeyValue{
		Key:   "userPostBalance",
		Value: core.Float64(balancesPost[0]) },
		core.KeyValue{
			Key: "servicePostBalance",
			Value: core.Float64(balancesPost[1]) },
		core.KeyValue{
			Key: "node1PostBalance",
			Value: core.Float64(balancesPost[2])	},
		core.KeyValue{
			Key: "node2PostBalance",
			Value: core.Float64(balancesPost[3])	},
		core.KeyValue{
			Key: "node3PostBalance",
			Value: core.Float64(balancesPost[4])	},
		core.KeyValue{
			Key: "paymentAmount",
			Value: core.Float64(paymentAmount)	},
		core.KeyValue{
			Key: "paymentRoutingFees",
			Value: core.Float64(paymentRoutingFees)	},
		core.KeyValue{
			Key: "nodePaymentFee",
			Value: core.Float64(nodePaymentFee)	},
	)
	assert.InEpsilon(balancesPre[2]+nodePaymentFee,balancesPost[2],1E-6,"Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee,balancesPost[3],1E-6,"Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee,balancesPost[4],1E-6,"Incorrect node3 balance")
}
