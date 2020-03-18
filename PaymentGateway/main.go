package PaymentGateway

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"paidpiper.com/payment-gateway/controllers"
	gw "paidpiper.com/payment-gateway/gatewayService"
	pb "paidpiper.com/payment-gateway/ppsidechannel"
	"paidpiper.com/payment-gateway/proxy"
	us "paidpiper.com/payment-gateway/utilityService"
)

func main() {

	key, _ := keypair.ParseFull("")

	proxyNodeManager := &proxy.NodeManager{}

	utilityService := &us.UtilityServiceImpl{}

	gatewayService := &gw.GatewayServiceImpl{
		NodeManager: proxyNodeManager,
		Seed: key,
	}

	utilityController := &controllers.UtilityController{
		Impl: utilityService,
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/utility/processCommand", utilityController.ProcessCommand).Methods("POST")
	router.HandleFunc("/api/utility/processResponse", proxyNodeManager.ProcessResponse).Methods("POST")
	router.HandleFunc("/api/gateway/processPayment", controllers.ProcessPayment).Methods("POST")

	err := http.ListenAndServe(":28080", router) //Launch the app, visit localhost:8000/api
	if err != nil {
		fmt.Print(err)
	}

	lis, err := net.Listen("tcp", ":28088")

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()


	pb.RegisterPPPaymentUtilityServicesServer(s, utilityService)
	pb.RegisterPPPaymentGatewayServer(s, gatewayService)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
