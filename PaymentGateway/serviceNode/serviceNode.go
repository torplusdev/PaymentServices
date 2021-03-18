package serviceNode

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/node/local"
)

type loggableWriter struct {
	mux.Router
}

func (r *loggableWriter) Handle(path string, handler http.Handler) *mux.Route {
	return r.Router.Handle(path, handlers.LoggingHandler(log.Writer(), handler))
}
func RunHttpServer(config *config.Configuration) (func(), error) {
	local, err := local.FromConfig(config)
	if err != nil {
		return nil, err
	}
	server := HttpLocalNode(local, config.Port)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}, nil
}
func HttpLocalNode(localNode local.LocalPPNode, port int) *http.Server {

	utilityController := controllers.NewHttpUtilityController(localNode)

	gatewayController := controllers.NewHttpGatewayController(localNode)

	router := &loggableWriter{
		Router: *mux.NewRouter(),
	}

	router.Handle("/api/utility/createPaymentInfo", http.HandlerFunc(utilityController.HttpNewPaymentRequest)).Methods("POST")
	router.Handle("/api/utility/validatePayment", http.HandlerFunc(utilityController.HttpValidatePayment)).Methods("POST")
	router.Handle("/api/utility/transactions/flush", http.HandlerFunc(utilityController.HttpFlushTransactions)).Methods("GET")
	router.Handle("/api/utility/transactions", http.HandlerFunc(utilityController.ListTransactions)).Methods("GET")
	router.Handle("/api/utility/transaction/{sessionId}", http.HandlerFunc(utilityController.HttpGetTransaction)).Methods("GET")
	router.Handle("/api/utility/stellarAddress", http.HandlerFunc(utilityController.HttpGetStellarAddress)).Methods("GET")
	router.Handle("/api/utility/processCommand", http.HandlerFunc(utilityController.HttpProcessCommand)).Methods("POST")
	router.Handle("/api/utility/balance", http.HandlerFunc(utilityController.HttpGetBalance)).Methods("GET")

	router.Handle("/api/gateway/processResponse", http.HandlerFunc(gatewayController.HttpProcessResponse)).Methods("POST")
	router.Handle("/api/gateway/processPayment", http.HandlerFunc(gatewayController.HttpProcessPayment)).Methods("POST")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handlers.RecoveryHandler()(router),
	}

	server.SetKeepAlivesEnabled(false)

	go func() { //TODO DONE
		if err := server.ListenAndServe(); err != nil {
			glog.Warningf("Error starting service node: %s", err)
		}
	}()
	return server
}
