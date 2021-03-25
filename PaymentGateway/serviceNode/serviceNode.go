package serviceNode

import (
	"context"
	"fmt"
	"log"
	. "net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/horizon"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
)

func StartServiceNode(keySeed string, port int, torAddressPrefix string, asyncMode bool, autoFlushDuration time.Duration, transactionValiditySecs int64) (*Server,*node.Node, error) {
	tracer := common.CreateTracer("paidpiper/serviceNode")

	_, span := tracer.Start(context.Background(), "serviceNode-initialization")
	defer span.End()

	seed, err := keypair.ParseFull(keySeed)

	if err != nil {
		glog.Infof("Error parsing node key: %s", err)
		return nil, nil,  err
	}

	horizon := horizon.NewHorizon()

	localNode,err := node.CreateNode(horizon, seed.Address(), seed.Seed(), true, autoFlushDuration, transactionValiditySecs)

	if err != nil {
		glog.Infof("Error creating Node object: %s", err)
		return nil, nil, err
	}

	priceList := make(map[string]map[string]commodity.Descriptor)

	priceList["ipfs"] = make(map[string]commodity.Descriptor)
	priceList["tor"] = make(map[string]commodity.Descriptor)
	priceList["http"] = make(map[string]commodity.Descriptor)

	priceList["ipfs"]["data"] = commodity.Descriptor{
		UnitPrice: 0.00000002,
		Asset:     common.PPTokenAssetName,
	}

	priceList["tor"]["data"] = commodity.Descriptor{
		UnitPrice: 0.1,
		Asset:     common.PPTokenAssetName,
	}

	priceList["http"]["attention"] = commodity.Descriptor{
		UnitPrice: 0.1,
		Asset:     common.PPTokenAssetName,
	}

	commodityManager := commodity.New(priceList)

	rootApi := root.CreateRootApi(true)
	err = rootApi.CreateUser(seed.Address(), seed.Seed())

	if err != nil {
		glog.Infof("Error creating user: %s", err)
		return nil,nil, err
	}

	balance, err := horizon.GetBalance(seed.Address())

	if err != nil {
		glog.Infof("Error retrieving account data: %s", err)
		return nil,nil,  err
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

	router.Handle("/api/utility/createPaymentInfo", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.CreatePaymentInfo))).Methods("POST")
	router.Handle("/api/utility/validatePayment", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.ValidatePayment))).Methods("POST")
	router.Handle("/api/utility/transactions/flush", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.FlushTransactions))).Methods("GET")
	router.Handle("/api/utility/transactions", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.ListTransactions))).Methods("GET")
	router.Handle("/api/utility/transaction/{sessionId}", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.GetTransaction))).Methods("GET")
	router.Handle("/api/utility/stellarAddress", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.GetStellarAddress))).Methods("GET")
	router.Handle("/api/utility/balance", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.GetBalance))).Methods("GET")
	router.Handle("/api/utility/processCommand", handlers.LoggingHandler(log.Writer(), HandlerFunc(utilityController.ProcessCommand))).Methods("POST")
	router.Handle("/api/gateway/processResponse", handlers.LoggingHandler(log.Writer(), HandlerFunc(gatewayController.ProcessResponse))).Methods("POST")
	router.Handle("/api/gateway/processPayment", handlers.LoggingHandler(log.Writer(), HandlerFunc(gatewayController.ProcessPayment))).Methods("POST")

	server := &Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handlers.RecoveryHandler()(router),
	}

	server.SetKeepAlivesEnabled(false)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Warningf("Error starting service node: %s", err)
		}
	}()

	return server,localNode, nil
}
