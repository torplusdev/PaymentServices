package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	"net/http"
	"os"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/proxy"
)

func main() {
	s := os.Args[1]
	port := os.Args[2]

	//s := "SC33EAUSEMMVSN4L3BJFFR732JLASR4AQY7HBRGA6BVKAPJL5S4OZWLU"
	//port := 28080

	seed, err := keypair.ParseFull(s)

	if err != nil {
		fmt.Print(err)
		return
	}

	localNode := node.CreateNode(horizon.DefaultTestNetClient, seed.Address(), seed.Seed(),true)

	proxyNodeManager := proxy.New(localNode)

	utilityController := &controllers.UtilityController {
		Node: localNode,
	}

	gatewayController := controllers.New(
		proxyNodeManager,
		seed,
		"http://localhost:57842/api/command",
		"http://localhost:57842/api/paymentRoute/",
	)

	router := mux.NewRouter()

	router.HandleFunc("/api/utility/createPaymentInfo/{amount}", utilityController.CreatePaymentInfo).Methods("GET")
	router.HandleFunc("/api/utility/stellarAddress", utilityController.GetStellarAddress).Methods("GET")
	router.HandleFunc("/api/utility/processCommand", utilityController.ProcessCommand).Methods("POST")
	router.HandleFunc("/api/gateway/processResponse", gatewayController.ProcessResponse).Methods("POST")
	router.HandleFunc("/api/gateway/processPayment", gatewayController.ProcessPayment).Methods("POST")

	err = http.ListenAndServe(":" + port, router) //Launch the app, visit localhost:8000/api

	if err != nil {
		fmt.Print(err)
	}
}
