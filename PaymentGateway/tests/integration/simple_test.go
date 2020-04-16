package integration_tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/global"
	"os"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
	"time"
)

func setup() {

}

var testSetup *TestSetup

func init() {

	testutils.InitGlobalTracer()

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

	testSetup.StartUserNode(testutils.User1Seed,28080)
	testSetup.StartTorNode(testutils.Node1Seed,28081)
	testSetup.StartTorNode(testutils.Node2Seed,28082)
	testSetup.StartTorNode(testutils.Node3Seed,28083)
	testSetup.StartServiceNode(testutils.Service1Seed,28084)

	span.End()

	// Wait for everything to start up
	time.Sleep(2 * time.Second)

}

func shutdown() {

}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestSimple(t *testing.T) {

	assert := assert.New(t)

	userBalancePre := testutils.GetAccountBalance(testutils.User1Seed)
	serviceBalancePre := testutils.GetAccountBalance(testutils.Service1Seed)

	pr,err := testSetup.CreatePaymentInfo(testutils.Service1Seed,10)
	assert.NoError(err)

	result, err := testSetup.ProcessPayment(testutils.User1Seed,pr)
	assert.NoError(err)
	assert.Contains(result,"Payment processing completed")


	err = testSetup.FlushTransactions()
	assert.NoError(err)

	userBalancePost := testutils.GetAccountBalance(testutils.User1Seed)
	serviceBalancePost := testutils.GetAccountBalance(testutils.Service1Seed)

	v := testutils.AreBalancesEqual(userBalancePre-10, userBalancePost)
	assert.True(v, "Incorrect user balance")
	assert.True(testutils.AreBalancesEqual(serviceBalancePre+10, serviceBalancePost),"Incorrect service balance")


}
