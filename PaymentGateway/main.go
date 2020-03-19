package PaymentGateway

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"net/http"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/proxy"
)

func main() {
	seed, err := keypair.ParseFull("")

	if err != nil {
		fmt.Print(err)
		return
	}

	proxyNodeManager := &proxy.NodeManager{}

	utilityController := &controllers.UtilityController {
		Node: &node.Node{
			Address: seed.Address(),
		},
	}

	gatewayController := &controllers.GatewayController{
		NodeManager: proxyNodeManager,
		Seed:        seed,
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/utility/processCommand", utilityController.ProcessCommand).Methods("POST")
	router.HandleFunc("/api/utility/processResponse", proxyNodeManager.ProcessResponse).Methods("POST")
	router.HandleFunc("/api/gateway/processPayment", gatewayController.ProcessPayment).Methods("POST")

	err = http.ListenAndServe(":28080", router) //Launch the app, visit localhost:8000/api

	if err != nil {
		fmt.Print(err)
	}
}
