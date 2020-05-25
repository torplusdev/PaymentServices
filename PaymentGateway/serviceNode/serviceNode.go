package serviceNode

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/keypair"
	. "net/http"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/proxy"
	"paidpiper.com/payment-gateway/root"
	testutils "paidpiper.com/payment-gateway/tests"
)

func StartServiceNode(keySeed string, port int, torAddressPrefix string, asyncMode bool) (*Server,error) {
	tracer := common.CreateTracer("paidpiper/serviceNode")

	_, span := tracer.Start(context.Background(),"serviceNode-initialization")
	defer span.End()

	seed, err := keypair.ParseFull(keySeed)

	if err != nil {
		glog.Info("Error parsing node key: %v",err)
		return &Server{}, err
	}

	localNode := node.CreateNode(horizon.DefaultTestNetClient, seed.Address(), seed.Seed(),true)

	proxyNodeManager := proxy.New(localNode)

	priceList := make(map[string]map[string]commodity.Descriptor)

	priceList["ipfs"] = make(map[string]commodity.Descriptor)
	priceList["tor"] = make(map[string]commodity.Descriptor)

	priceList["ipfs"]["data"] = commodity.Descriptor{
		UnitPrice: 0.0000001,
		Asset:     "XLM",
	}

	priceList["tor"]["data"] = commodity.Descriptor{
		UnitPrice: 0.1,
		Asset:     "XLM",
	}

	commodityManager := commodity.New(priceList)

	rootApi := root.CreateRootApi(true)
	err = rootApi.CreateUser(seed.Address(), seed.Seed())

	if err != nil {
		glog.Info("Error creating user: %v",err)
		return &Server{},err
	}
	c := client.CreateClient(rootApi, seed.Seed(), proxyNodeManager)

	account, err := testutils.GetAccount(seed.Address())

	if err != nil {
		glog.Info("Error retrieving account data: %v",err)
		return &Server{},err
	}

	balance,_ := account.GetNativeBalance()
	fmt.Printf("Current balance for %v:%v",seed.Address(), balance)

	utilityController := controllers.NewUtilityController(
		localNode,
		commodityManager,
	)

	gatewayController := controllers.NewGatewayController(
		proxyNodeManager,
		c,
		seed,
		fmt.Sprintf("%s/api/command",torAddressPrefix),
		fmt.Sprintf("%s/api/paymentRoute/",torAddressPrefix),
		asyncMode,
	)

	router := mux.NewRouter()

	router.HandleFunc("/api/utility/createPaymentInfo", utilityController.CreatePaymentInfo).Methods("POST")
	router.HandleFunc("/api/utility/validatePayment", utilityController.ValidatePayment).Methods("POST")
	router.HandleFunc("/api/utility/transactions/flush", utilityController.FlushTransactions).Methods("GET")
	router.HandleFunc("/api/utility/transactions", utilityController.ListTransactions).Methods("GET")
	router.HandleFunc("/api/utility/stellarAddress", utilityController.GetStellarAddress).Methods("GET")
	router.HandleFunc("/api/utility/processCommand", utilityController.ProcessCommand).Methods("POST")
	router.HandleFunc("/api/gateway/processResponse", gatewayController.ProcessResponse).Methods("POST")
	router.HandleFunc("/api/gateway/processPayment", gatewayController.ProcessPayment).Methods("POST")

	server := &Server{
		Addr: fmt.Sprintf(":%d",port),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Warning("Error starting service node: %v",err)
		}
	}()

	return server,nil
}