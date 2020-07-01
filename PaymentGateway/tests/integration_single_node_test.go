package tests

import (
	"context"
	"go.opentelemetry.io/otel/api/core"
	"google.golang.org/grpc/codes"
	"os"
	"paidpiper.com/payment-gateway/common"
	"testing"
	"time"
)

var testSetup *TestSetup

var tracerShutdown func()

func init() {

	traceProvider, shutdownFunc := InitGlobalTracer()
	common.InitializeTracer(traceProvider)

	tracerShutdown = shutdownFunc

	// Addresses reused from other tests
	CreateAndFundAccount(User1Seed)
	CreateAndFundAccount(Service1Seed)

	// Addresses specific to this test suite
	CreateAndFundAccount(Node1Seed)
	CreateAndFundAccount(Node2Seed)
	CreateAndFundAccount(Node3Seed)
	CreateAndFundAccount(Node4Seed)

	testSetup = CreateTestSetup()
	torPort := 57842

	testSetup.ConfigureTor(torPort)

	tr := common.CreateTracer("TestInit")

	ctx, span := tr.Start(context.Background(), "NodeInitialization")

	testSetup.StartUserNode(ctx, User1Seed, 28080)
	testSetup.StartTorNode(ctx, Node1Seed, 28081)
	testSetup.StartTorNode(ctx, Node2Seed, 28082)
	testSetup.StartTorNode(ctx, Node3Seed, 28083)

	testSetup.StartServiceNode(ctx, Service1Seed, 28084)
	span.SetStatus(codes.OK, "All Nodes Stared Up")

	testSetup.SetDefaultPaymentRoute([]string{
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW"})

	span.End()

	// Wait for everything to start up
	time.Sleep(2 * time.Second)

}

func shutdown() {
	testSetup.Shutdown()

	if tracerShutdown != nil {
		tracerShutdown()
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	shutdown()
	_ = code
	os.Exit(code)
}

func TestSingleChainPayment(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW"})

	assert, ctx, span := InitTestCreateSpan(t, "TestSingleChainPayment")
	defer span.End()

	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	span.SetAttributes(core.KeyValue{
		Key:   "userPreBalance",
		Value: core.Float64(balancesPre[0])},
		core.KeyValue{
			Key:   "servicePreBalance",
			Value: core.Float64(balancesPre[1])},
		core.KeyValue{
			Key:   "node1PreBalance",
			Value: core.Float64(balancesPre[2])},
		core.KeyValue{
			Key:   "node2PreBalance",
			Value: core.Float64(balancesPre[3])},
		core.KeyValue{
			Key:   "node3PreBalance",
			Value: core.Float64(balancesPre[4])},
	)
	sequencer := CreateSequencer(testSetup, assert, ctx)
	// 100 MB
	paymentAmount := 1001e6

	result, pr := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount)
	assert.Contains(result, "Payment processing completed")

	paymentAmount = float64(pr.Amount)

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	paymentRoutingFees := float64(3 * 10)

	assert.InEpsilon(balancesPre[0]-paymentAmount-paymentRoutingFees, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1], 1E-6, "Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount) / 3
	span.SetAttributes(core.KeyValue{
		Key:   "userPostBalance",
		Value: core.Float64(balancesPost[0])},
		core.KeyValue{
			Key:   "servicePostBalance",
			Value: core.Float64(balancesPost[1])},
		core.KeyValue{
			Key:   "node1PostBalance",
			Value: core.Float64(balancesPost[2])},
		core.KeyValue{
			Key:   "node2PostBalance",
			Value: core.Float64(balancesPost[3])},
		core.KeyValue{
			Key:   "node3PostBalance",
			Value: core.Float64(balancesPost[4])},
		core.KeyValue{
			Key:   "paymentAmount",
			Value: core.Float64(paymentAmount)},
		core.KeyValue{
			Key:   "paymentRoutingFees",
			Value: core.Float64(paymentRoutingFees)},
		core.KeyValue{
			Key:   "nodePaymentFee",
			Value: core.Float64(nodePaymentFee)},
	)
	assert.InEpsilon(balancesPre[2]+nodePaymentFee, balancesPost[2], 1E-6, "Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee, balancesPost[3], 1E-6, "Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee, balancesPost[4], 1E-6, "Incorrect node3 balance")

}

func TestTwoChainPayments(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		"GDRQ2GFDIXSPOBOICRJUEVQ3JIZJOWW7BXV2VSIN4AR6H6SD32YER4LN",
		"GD523N6LHPRQS3JMCXJDEF3ZENTSJLRUDUF2CU6GZTNGFWJXSF3VNDJJ",
		"GB3IKDN72HFZSLY3SYE5YWULA5HG32AAKEDJTG6J6X2YKITHBDDT2PIW"})

	assert, ctx, span := InitTestCreateSpan(t, "TestTwoChainPayments")
	defer span.End()

	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})
	span.SetAttributes(core.KeyValue{
		Key:   "userPreBalance",
		Value: core.Float64(balancesPre[0])},
		core.KeyValue{
			Key:   "servicePreBalance",
			Value: core.Float64(balancesPre[1])},
		core.KeyValue{
			Key:   "node1PreBalance",
			Value: core.Float64(balancesPre[2])},
		core.KeyValue{
			Key:   "node2PreBalance",
			Value: core.Float64(balancesPre[3])},
		core.KeyValue{
			Key:   "node3PreBalance",
			Value: core.Float64(balancesPre[4])},
	)

	sequencer := CreateSequencer(testSetup, assert, ctx)
	paymentAmount1 := 300e6
	paymentAmount2 := 600e6

	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)
	result, pr2 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount2)

	assert.Contains(result, "Payment processing completed")
	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	// Take the actual converted amount in XLM
	paymentAmount1 = float64(pr1.Amount)
	paymentAmount2 = float64(pr2.Amount)

	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	paymentAmount := paymentAmount1 + paymentAmount2

	paymentRoutingFees := float64(3*10) * 2

	assert.InEpsilon(balancesPre[0]-paymentAmount-paymentRoutingFees, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1], 1E-6, "Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount) / 3
	span.SetAttributes(core.KeyValue{
		Key:   "userPostBalance",
		Value: core.Float64(balancesPost[0])},
		core.KeyValue{
			Key:   "servicePostBalance",
			Value: core.Float64(balancesPost[1])},
		core.KeyValue{
			Key:   "node1PostBalance",
			Value: core.Float64(balancesPost[2])},
		core.KeyValue{
			Key:   "node2PostBalance",
			Value: core.Float64(balancesPost[3])},
		core.KeyValue{
			Key:   "node3PostBalance",
			Value: core.Float64(balancesPost[4])},
		core.KeyValue{
			Key:   "paymentAmount",
			Value: core.Float64(paymentAmount)},
		core.KeyValue{
			Key:   "paymentRoutingFees",
			Value: core.Float64(paymentRoutingFees)},
		core.KeyValue{
			Key:   "nodePaymentFee",
			Value: core.Float64(nodePaymentFee)},
	)
	assert.InEpsilon(balancesPre[2]+nodePaymentFee, balancesPost[2], 1E-6, "Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee, balancesPost[3], 1E-6, "Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee, balancesPost[4], 1E-6, "Incorrect node3 balance")
}
