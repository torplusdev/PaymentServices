package tests

import (
	"github.com/stellar/go/keypair"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
	. "paidpiper.com/payment-gateway/tests/util"
)

func seed2addr(seed string) string {
	kp, _ := keypair.ParseFull(seed)
	return kp.Address()
}

func getPreBalances(span trace.Span) []float64 {
	balancesPre := GetAccountBalances([]string{User1Seed, Service1Seed,
		Node1Seed, Node2Seed, Node3Seed})

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
	return balancesPre
}

func setPreBalances(span trace.Span, balancesPre []float64) {

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
}

func getPostBalances(span trace.Span) []float64 {

	balancesPost := GetAccountBalances([]string{User1Seed,
		Service1Seed,
		Node1Seed,
		Node2Seed,
		Node3Seed})

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

func setPostBalances(span trace.Span, balancesPost []float64, paymentAmount float64, paymentRoutingFees float64, nodePaymentFee float64) {
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
}
