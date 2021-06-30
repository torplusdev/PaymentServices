package main

import (
	"os"
	"runtime"

	"paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/version"

	"paidpiper.com/payment-gateway/serviceNode"
)

func main() {

	stop := make(chan os.Signal, 1)
	log.Info("payment_gateway %v, built %v ", version.Version(), version.BuildDate())
	config, err := config.ParseConfig()
	log.Info("Port: ", config.Port)
	if err != nil {
		log.Errorf("get config error: %v", err)
		<-stop
		return
	}

	tracerShutdownFunc := common.InitGlobalTracer(config.JaegerConfig)
	defer tracerShutdownFunc()
	runtime.GOMAXPROCS(config.MaxConcurrency)
	runtime.NumGoroutine()
	serverShutdown, err := serviceNode.RunHttpServer(config)
	if err != nil {
		log.Panicf("Error starting serviceNode: %v", err)
	} else {
		defer serverShutdown()
	}
	<-stop
}
