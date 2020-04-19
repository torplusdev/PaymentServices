package integration_tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/key"
	"os"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
	"time"
)

func setup() {

}

var testSetup *TestSetup

var tracerShutdown func()

func init() {

	tracerShutdown = testutils.InitGlobalTracer()

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

	tr := global.Tracer("TestInit")
	ctx,span := tr.Start(context.Background(),"NodeInitialization")

	testSetup.StartUserNode(ctx,testutils.User1Seed,28080)
	testSetup.StartTorNode(ctx, testutils.Node1Seed,28081)
	testSetup.StartTorNode(ctx, testutils.Node2Seed,28082)
	testSetup.StartTorNode(ctx, testutils.Node3Seed,28083)
	testSetup.StartServiceNode(ctx, testutils.Service1Seed,28084)

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

	assert := assert.New(t)

	tr := global.Tracer("test")

	ctx := correlation.NewContext(context.Background(),
		key.String("user", "test123"),
	)


	ctx,span := tr.Start(ctx,"TestEndToEndSinglePayment")
	defer span.End()


	paymentAmount := 300.0

	balancesPre := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})

	assert.Fail("oopsie")

	pr,err := testSetup.CreatePaymentInfo(ctx, testutils.Service1Seed,int(paymentAmount))
	assert.NoError(err)

	result, err := testSetup.ProcessPayment(ctx, testutils.User1Seed,pr)
	assert.NoError(err)
	assert.Contains(result,"Payment processing completed")


	err = testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := testutils.GetAccountBalances([]string {testutils.User1Seed,testutils.Service1Seed,testutils.Node1Seed,testutils.Node2Seed,testutils.Node3Seed})

	paymentRoutingFees := float64(3*10)

	assert.InEpsilon(balancesPre[0] - paymentAmount - paymentRoutingFees,balancesPost[0],1E-6,"Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount,balancesPost[1],1E-6,"Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount)/3

	assert.InEpsilon(balancesPre[2]+nodePaymentFee,balancesPost[2],1E-6,"Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee,balancesPost[3],1E-6,"Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee,balancesPost[4],1E-6,"Incorrect node3 balance")
}
