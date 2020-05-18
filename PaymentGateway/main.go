package main

import (
	"context"
	"log"
	"os"
	"paidpiper.com/payment-gateway/serviceNode"
	"runtime"
	"strconv"
	"time"
)

var(
	configuration Configuration
	configFilePath string
)
func main() {
	if len(os.Args) > 1 {
		configFilePath = os.Args[1]
	} else {
		configFilePath = "conf.json"
	}
	GetConfig(configFilePath)
	s := configuration.Seed
	port := configuration.Port
	torAddressPrefix := configuration.TorAddressPrefix
	//s := "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
	//port := 28080

	runtime.GOMAXPROCS(10)
	runtime.NumGoroutine()

	numericPort, err := strconv.Atoi(port)

	if err != nil {
		log.Panicf("Error parsing port number: %v",err.Error())
	}

	// Set up signal channel
	stop := make(chan os.Signal, 1)

	server,err := serviceNode.StartServiceNode(s,numericPort,torAddressPrefix, true)

	if err != nil {
		log.Panicf("Error starting serviceNode: %v",err.Error())
	}

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Panicf("Error shutting down server: %v",err.Error())
	}
}
