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
	os.Exit(code)
}

func TestEndToEndSinglePayment(t *testing.T) {

	assert := assert.New(t)

	tr := global.Tracer("test")

	ctx := correlation.NewContext(context.Background(),
		key.String("user", "test123"),
	)


	ctx,span := tr.Start(ctx,"TestEndToEndSinglePayment")
	defer span.End()


	userBalancePre := testutils.GetAccountBalance(testutils.User1Seed)
	serviceBalancePre := testutils.GetAccountBalance(testutils.Service1Seed)

	pr,err := testSetup.CreatePaymentInfo(ctx, testutils.Service1Seed,10)
	assert.NoError(err)

	result, err := testSetup.ProcessPayment(ctx, testutils.User1Seed,pr)
	assert.NoError(err)
	assert.Contains(result,"Payment processing completed")


	err = testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	userBalancePost := testutils.GetAccountBalance(testutils.User1Seed)
	serviceBalancePost := testutils.GetAccountBalance(testutils.Service1Seed)


	if (!assert.InEpsilon(userBalancePre-10,userBalancePost,1E-6,"Incorrect user balance")) {
		t.Fail()
	}

	if (!assert.InEpsilon(serviceBalancePre+10,serviceBalancePost,1E-6,"Incorrect service balance")) {
		t.Fail()
	}
}
