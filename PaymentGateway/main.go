package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"

	"paidpiper.com/payment-gateway/serviceNode"
)

func main() {
	s := os.Args[1]
	port := os.Args[2]

	//s := "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
	//port := 28080
	autoFlushPeriod := 15*time.Minute

	runtime.GOMAXPROCS(10)
	runtime.NumGoroutine()

	numericPort, err := strconv.Atoi(port)

	if err != nil {
		log.Panicf("Error parsing port number: %v", err.Error())
	}

	// Set up signal channel
	stop := make(chan os.Signal, 1)

	server, err := serviceNode.StartServiceNode(s, numericPort, "http://localhost:5900", true, autoFlushPeriod)

	if err != nil {
		log.Panicf("Error starting serviceNode: %v", err.Error())
	}

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Panicf("Error shutting down server: %v", err.Error())
	}
}
