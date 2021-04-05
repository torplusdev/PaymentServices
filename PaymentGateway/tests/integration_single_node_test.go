package tests

import (
	"context"
	"github.com/stellar/go/keypair"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
	"google.golang.org/grpc/codes"
	"os"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/utility"
	"testing"
	"time"
)

var testSetup *TestSetup

var tracerShutdown func()

func seed2addr(seed string) string {
	kp,_ := keypair.ParseFull(seed)
	return kp.Address()
}

func init() {

	traceProvider, shutdownFunc := InitGlobalTracer()
	common.InitializeTracer(traceProvider)

	tracerShutdown = shutdownFunc

	// Addresses reused from other tests
	CreateAndFundAccount(User1Seed,Client)
	CreateAndFundAccount(Service1Seed,Node)

	// Addresses specific to this test suite
	CreateAndFundAccount(Node1Seed,Node)
	CreateAndFundAccount(Node2Seed,Node)
	CreateAndFundAccount(Node3Seed,Node)
	CreateAndFundAccount(Node4Seed,Node)

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
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed),
	})

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

func getPreBalances(span trace.Span)  []float64 {
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
	return balancesPre;
}

func getPostBalances(span trace.Span) []float64 {

	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

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
	)

	return balancesPost
}

func TestMain(m *testing.M) {
	code := m.Run()
	shutdown()
	_ = code
	os.Exit(code)
}

func TestSingleChainPayment(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed),
	})

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

	totalPaidFees := common.PPTokenToNumeric(paymentRoutingFees)
	totalReceivedService := common.PPTokenToNumeric(paymentAmount)

	assert.InEpsilon(balancesPre[0]-totalPaidFees-totalReceivedService, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+totalReceivedService, balancesPost[1], 1E-6, "Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - totalReceivedService) / 3

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

	assert.InEpsilon(balancesPre[2]+ nodePaymentFee, balancesPost[2], 1E-6, "Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+ nodePaymentFee, balancesPost[3], 1E-6, "Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+ nodePaymentFee, balancesPost[4], 1E-6, "Incorrect node3 balance")
}

func TestSinglePaymentAutoFlush(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed),
	})

	assert, ctx, span := InitTestCreateSpan(t, "TestSingleChainPayment")
	defer span.End()

	//TODO: Properly expose autoflush configuration and set it to 1min
	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	testSetup.GetNode(User1Seed).SetAutoFlush(1*time.Minute);
	testSetup.GetNode(Service1Seed).SetAutoFlush(1*time.Minute);
	testSetup.GetNode(Node1Seed).SetAutoFlush(1*time.Minute);
	testSetup.GetNode(Node2Seed).SetAutoFlush(1*time.Minute);
	testSetup.GetNode(Node3Seed).SetAutoFlush(1*time.Minute);

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

	time.Sleep(65*time.Second)

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
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed)})

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

func TestPaymentAfterwoChainPayments(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed)})

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
	paymentAmount3 := 200e6

	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)
	result, pr2 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount2)

	assert.Contains(result, "Payment processing completed")
	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	result, pr3 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount3)

	assert.Contains(result, "Payment processing completed")
	err = testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	// Take the actual converted amount in XLM
	paymentAmount1 = float64(pr1.Amount)
	paymentAmount2 = float64(pr2.Amount)
	paymentAmount3 = float64(pr3.Amount)

	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	paymentAmount := paymentAmount1 + paymentAmount2 + paymentAmount3

	paymentRoutingFees := float64(3*10) * 3

	totalPaidFees := common.PPTokenToNumeric(paymentRoutingFees)
	totalReceivedService := common.PPTokenToNumeric(paymentAmount)

	assert.InEpsilon(balancesPre[0]-totalReceivedService-totalPaidFees, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+totalReceivedService, balancesPost[1], 1E-6, "Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - totalReceivedService) / 3

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

func TestPaymentAfterUnfulfilledPayment(t *testing.T) {

	testSetup.SetDefaultPaymentRoute([]string{
		seed2addr(Node1Seed),
		seed2addr(Node2Seed),
		seed2addr(Node3Seed)})

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
	paymentAmount3 := 200e6

	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)


	assert.Contains(result, "Payment processing completed")
	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	result, pr3 := sequencer.PerformPayment(User1Seed, Node4Seed, paymentAmount3)

	assert.Contains(result, "Payment processing completed")
	err = testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	// Take the actual converted amount in XLM
	paymentAmount1 = float64(pr1.Amount)
	paymentAmount3 = float64(pr3.Amount)

	balancesPost := GetAccountBalances([]string{User1Seed, Service1Seed, Node1Seed, Node2Seed, Node3Seed})

	paymentAmount := paymentAmount1 + paymentAmount2 + paymentAmount3

	paymentRoutingFees := float64(3*10) * 3

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

func TestPaymentsToDifferentAddresses(t *testing.T) {

	assert, ctx, span := InitTestCreateSpan(t, "TestPaymentsToDifferentAddresses")
	defer span.End()

	balancesPre := getPreBalances(span)

	sequencer := CreateSequencer(testSetup, assert, ctx)
	paymentAmount1 := 300e6
	paymentAmount2 := 200e6

	testSetup.GetNode(Service1Seed).SetTransactionValiditySecs(600);
	testSetup.GetNode(User1Seed).SetTransactionValiditySecs(600);

	testSetup.SetDefaultPaymentRoute([]string{ seed2addr(Node2Seed)})

	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)

	assert.Contains(result, "Payment processing completed")

	testSetup.SetDefaultPaymentRoute([]string{ seed2addr(Node1Seed)})

	result, pr2 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount2)

	assert.Contains(result, "Payment processing completed")

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	// Take the actual converted amount in XLM
	paymentAmount1 = float64(pr1.Amount)
	paymentAmount2 = float64(pr2.Amount)

	balancesPost := getPostBalances(span)

	paymentAmount := paymentAmount1 + paymentAmount2

	paymentRoutingFees := float64(3*10) * 3

	assert.InEpsilon(balancesPre[0]-paymentAmount-paymentRoutingFees, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1], 1E-6, "Incorrect service balance")

	nodePaymentFee := (balancesPre[0] - balancesPost[0] - paymentAmount) / 3

	assert.InEpsilon(balancesPre[2]+nodePaymentFee, balancesPost[2], 1E-6, "Incorrect node1 balance")
	assert.InEpsilon(balancesPre[3]+nodePaymentFee, balancesPost[3], 1E-6, "Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4]+nodePaymentFee, balancesPost[4], 1E-6, "Incorrect node3 balance")
}

func TestIncorrectTransactionsAreDiscardedByFlush(t *testing.T) {

	assert, ctx, span := InitTestCreateSpan(t, "TestPaymentsToDifferentAddresses")
	defer span.End()

	balancesPre := getPreBalances(span)

	sequencer := CreateSequencer(testSetup, assert, ctx)
	paymentAmount1 := 300e6
	paymentAmount2 := 200e6

	testSetup.GetNode(Service1Seed).SetTransactionValiditySecs(1);
	testSetup.GetNode(User1Seed).SetTransactionValiditySecs(1);

	testSetup.SetDefaultPaymentRoute([]string{ seed2addr(Node2Seed)})

	result, pr1 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount1)

	assert.Contains(result, "Payment processing completed")

	testSetup.GetNode(Service1Seed).SetTransactionValiditySecs(600);
	testSetup.GetNode(User1Seed).SetTransactionValiditySecs(600);
	testSetup.GetNode(Node1Seed).SetTransactionValiditySecs(600);

	testSetup.SetDefaultPaymentRoute([]string{ seed2addr(Node1Seed)})

	result, pr2 := sequencer.PerformPayment(User1Seed, Service1Seed, paymentAmount2)

	assert.Contains(result, "Payment processing completed")

	err := testSetup.FlushTransactions(ctx)
	assert.NoError(err)

	// Take the actual converted amount in XLM
	paymentAmount1 = float64(pr1.Amount)
	paymentAmount2 = float64(pr2.Amount)

	balancesPost := getPostBalances(span)

	paymentAmount := paymentAmount2

	paymentRoutingFees := float64(10) * 1

	assert.InEpsilon(balancesPre[0]-paymentAmount-paymentRoutingFees, balancesPost[0], 1E-6, "Incorrect user balance")
	assert.InEpsilon(balancesPre[1]+paymentAmount, balancesPost[1], 1E-6, "Incorrect service balance")
	assert.InEpsilon(balancesPre[2]+paymentRoutingFees, balancesPost[2], 1E-6, "Incorrect node1 balance")
	// Node 2 and 3 shouldn't change
	assert.InEpsilon(balancesPre[3], balancesPost[3], 1E-6, "Incorrect node2 balance")
	assert.InEpsilon(balancesPre[4], balancesPost[4], 1E-6, "Incorrect node3 balance")
}


func TestCheckAccountFunds(t *testing.T) {

	nodes := []string{"GBVIMI2NAJJ3TO5YSYKUAZXZCPJNX7MDTMLXI62KKU73V7AHTIKDKUOP",
		"GCND2GZ2XUCXZ6URJWWD7PYJZUGPJHLMLQ5IJ6UEJM44VGZVNYH3LCB4",
		"GB6ESPMHPSOJICYBQI2HNWNAUZWSU757CKHOYPDKQGWAFK3R4Z3INUDC",
		"GAYAPB5WDZJ5OF4PFKUBWRPGYZKU4647DHVKLNFHN35DORH6H7F7N7VQ",
		"GAFDLNCWMIBDSMQZ7DLR44OITBS627E3S7HZLL57OS6IL7SUMCUUDUIV",
		"GCUORHKYF424MXVPK6TRDXC77RSPTNGAQ2B3XNIMF732RVO75FGURONN",
		"GAAE7TA2EJLRRLYPVL3YPJ3TOTSYGJW7AYAIVY257COQ37UHCOWHPJIU",
		"GCZB7HGSGWQDXPZ7IJBTV63WC7KW7RKZ4AZEUHYPHHTVJZZ2EZQ7E5W6",
		"GAOIW2EK6ATYKKAUUGZH3ZNXG4CC7YIWOEHSPZHR4GYCMZQMRVSRCESD",
		"GDM7URTN2RWOL34JZSAO26J4RKEZANRMCID4T6FKO72CROR3BTMYXRRX",
		"GANKXDSXTFCSGSSZN33A2MQLRAJGTWS56SMXIQRIS2R63ZH4FW5L4KHZ",
		"GDSSD7PKPJVVWYN2Z5ACYH42BFBRIIB3657NFDCYGOEYOWX4DF4FLOYW",
		"GC45CU5MOJZXU5DQY2QND57TRR5FRSF45DAL7OOM7JFWQNPAJEE2BO3C",
		"GARGQG2RJ5UIJRWTFP6E4PYBD2JXAKX2N5DJ3TQACQN2BTUBYFXHIJ4O",
		"GADCWDBQZ2VXWEUAMSMYYFYUXEEQYVGAMOPTAYQUCZXYPIG65CFVKSW5",
		"GDMXKGYUVGF3LIUFAN4CY6XCBPBDBRZKUXFEDK4TRZ6KBOLW5L7PIJF6",
		"GCMK6EF25IURCHEOYCSUNX57HYJWRQS5QARISIZOVPLF5OBZS3746D4L",
		"GAYAWF6IHQ5NE3JVM7ACHUK5YVHTB3VPS23H6JXRQABBMQGMAFSF6RLR",
		"GCK5JSIF3CCFF56233UNJUFZPWZQOXOKLBISAPVFR3UDPVI63GJ43OQV",
		"GAV2JMNJ3BJTZPYP7QPQ6ACWIOVR55LX6QGBQLSIV6XX6LZ5ZWXERJYX",
		"GBMK6CBUUY2N4HU577RQXV5PC5DFLATBV3DUMA2ACK4JQT7R6OJZYR2Z",
		"GC4GZW2TZ76RIQTIJI4MPLJKSRMVMJC5QQ5VITWNLBQ2J55KZ7MILLBD",
		"GAEHHXGMJ43XOB2P2NG26RZW2QQJT27E2MMKEYZWO3LZN46WA7CUKW3J",
		"GDRLSV5RNN6JHIJSLJ4LS6WKU7ISROW6ODUH56TOKTFHHWZUYOODC5QB",
		"GBZLGVKFYBDSZHGZ27F63IXRGU5EHITIQFYL7MS3P3XKAJ6WJ5JAG2IJ",
		"GDA2BBUSVHPFQXRMJNPWBOENFZRYTTO2Y7N3CMZWZ5BNCDAAMSMK6AHB",
		"GBZKCWYZQT26IP3OSEKK3HN3OIV5ZYPGKENVY63RZIHUVE6E3GUXPO43",
		"GD746PMXZKOJZMK7T74AR74NOLRPQJJAAEBDGQTLEYALZEK7OANS2DEH",
	}

	//seed := "SD3GOZWPM22EV2M3TSBPTOY5R5GHNLDHQFVJBEBONPXUB7KYDLI5K63C"

	for _, node_seed := range nodes {
		UpdateAccountLimits(node_seed, 10000)
	}
}



func TestIssueTokens(t *testing.T) {
	utility.UpdateAsset()
}