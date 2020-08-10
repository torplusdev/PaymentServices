package serviceNode

import (
	"context"
	"fmt"
	. "net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/horizon"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
)

func StartServiceNode(keySeed string, port int, torAddressPrefix string, asyncMode bool) (*Server, error) {
	tracer := common.CreateTracer("paidpiper/serviceNode")

	_, span := tracer.Start(context.Background(), "serviceNode-initialization")
	defer span.End()

	seed, err := keypair.ParseFull(keySeed)

	if err != nil {
		glog.Infof("Error parsing node key: %s", err)
		return nil, err
	}

	horizon := horizon.NewHorizon()

	localNode := node.CreateNode(horizon, seed.Address(), seed.Seed(), true)

	priceList := make(map[string]map[string]commodity.Descriptor)

	priceList["ipfs"] = make(map[string]commodity.Descriptor)
	priceList["tor"] = make(map[string]commodity.Descriptor)
	priceList["http"] = make(map[string]commodity.Descriptor)

	priceList["ipfs"]["data"] = commodity.Descriptor{
		UnitPrice: 0.0000001,
		Asset:     common.PPTokenAssetName,
	}

	priceList["tor"]["data"] = commodity.Descriptor{
		UnitPrice: 0.1,
		Asset:     common.PPTokenAssetName,
	}

	priceList["http"]["attention"] = commodity.Descriptor{
		UnitPrice: 0.01,
		Asset:     common.PPTokenAssetName,
	}

	commodityManager := commodity.New(priceList)

	rootApi := root.CreateRootApi(true)
	err = rootApi.CreateUser(seed.Address(), seed.Seed())

	if err != nil {
		glog.Infof("Error creating user: %s", err)
		return nil, err
	}

	balance, err := horizon.GetBalance(seed.Address())

	if err != nil {
		glog.Infof("Error retrieving account data: %s", err)
		return nil, err
	}

	fmt.Printf("Current balance for %v:%v", seed.Address(), balance)

	utilityController := controllers.NewUtilityController(
		localNode,
		localNode,
		localNode,
		commodityManager,
	)

	gatewayController := controllers.NewGatewayController(
		localNode,
		localNode,
		localNode,
		commodityManager,
		seed,
		rootApi,
		fmt.Sprintf("%s/api/command", torAddressPrefix),
		fmt.Sprintf("%s/api/paymentRoute/", torAddressPrefix),
		asyncMode,
	)

	router := mux.NewRouter()

	router.HandleFunc("/api/utility/createPaymentInfo", utilityController.CreatePaymentInfo).Methods("POST")
	router.HandleFunc("/api/utility/validatePayment", utilityController.ValidatePayment).Methods("POST")
	router.HandleFunc("/api/utility/transactions/flush", utilityController.FlushTransactions).Methods("GET")
	router.HandleFunc("/api/utility/transactions", utilityController.ListTransactions).Methods("GET")
	router.HandleFunc("/api/utility/transaction/{sessionId}", utilityController.GetTransaction).Methods("GET")
	router.HandleFunc("/api/utility/stellarAddress", utilityController.GetStellarAddress).Methods("GET")
	router.HandleFunc("/api/utility/processCommand", utilityController.ProcessCommand).Methods("POST")
	router.HandleFunc("/api/gateway/processResponse", gatewayController.ProcessResponse).Methods("POST")
	router.HandleFunc("/api/gateway/processPayment", gatewayController.ProcessPayment).Methods("POST")

	server := &Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	server.SetKeepAlivesEnabled(false)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Warningf("Error starting service node: %s", err)
		}
	}()

	return server, nil
}
