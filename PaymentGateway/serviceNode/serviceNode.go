package serviceNode

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"paidpiper.com/payment-gateway/log"

	webUiConfig "paidpiper.com/provider-service/config"
	webUiServer "paidpiper.com/provider-service/server"

	"github.com/go-chi/chi/v5"
	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/controllers"
	chi_server "paidpiper.com/payment-gateway/http/server"
	"paidpiper.com/payment-gateway/node/local"
	"paidpiper.com/payment-gateway/serviceNode/uiclient"
	"paidpiper.com/payment-gateway/version"
)

type loggableWriter struct {
	mux.Router
	cors bool
}

func (r *loggableWriter) Handle(path string, handler http.Handler) *mux.Route {
	if !r.cors {
		h := handlers.LoggingHandler(log.Writer(), handler)
		return r.Router.Handle(path, h)
	} else {
		h := handlers.LoggingHandler(log.Writer(), handler)
		return r.Router.Handle(path, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Set("Access-Control-Allow-Origin", "*")
			h.ServeHTTP(rw, r)
		}))
	}
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
			log.Errorf("Error shutting down server: %v", err)
		}
	}, nil
}

func HttpLocalNode(localNode local.LocalPPNode, port int) *http.Server {

	utilityController := controllers.NewHttpUtilityController(localNode)

	gatewayController := controllers.NewHttpGatewayController(localNode)

	resolverController := controllers.NewResolverController()

	router := &loggableWriter{
		Router: *mux.NewRouter(),
	}

	router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, version.Version())
	})
	router.Handle("/api/utility/createPaymentInfo", http.HandlerFunc(utilityController.HttpNewPaymentRequest)).Methods("POST")
	router.Handle("/api/utility/validatePayment", http.HandlerFunc(utilityController.HttpValidatePayment)).Methods("POST")
	router.Handle("/api/utility/transactions/flush", http.HandlerFunc(utilityController.HttpFlushTransactions)).Methods("GET")
	router.Handle("/api/utility/transactions/info", http.HandlerFunc(utilityController.HttpGetTransactionInfo)).Methods("GET")
	router.Handle("/api/utility/transactions", http.HandlerFunc(utilityController.ListTransactions)).Methods("GET")
	router.Handle("/api/utility/transaction/{sessionId}", http.HandlerFunc(utilityController.HttpGetTransaction)).Methods("GET")
	router.Handle("/api/utility/stellarAddress", http.HandlerFunc(utilityController.HttpGetStellarAddress)).Methods("GET")
	router.Handle("/api/utility/processCommand", http.HandlerFunc(utilityController.HttpProcessCommand)).Methods("POST")

	router.Handle("/api/utility/balance", http.HandlerFunc(utilityController.HttpGetBalance)).Methods("GET")

	router.Handle("/api/book/history/{commodity}/{hours}/{bins}", http.HandlerFunc(utilityController.HttpBookHistory)).Methods("GET")
	router.Handle("/api/book/transactions/{direction}/{limits}", http.HandlerFunc(utilityController.HttpBookTransactions)).Methods("GET")
	router.Handle("/api/book/transactionChart/{hours}", http.HandlerFunc(utilityController.HttpBookTransactionsGroup)).Methods("GET")
	// router.Handle("/api/book/transactionHistory/{bins}", http.HandlerFunc(utilityController.HttpBookTransactions)).Methods("GET")

	router.Handle("/api/book/balance", http.HandlerFunc(utilityController.HttpBookBalance)).Methods("GET")
	router.Handle("/api/gateway/processResponse", http.HandlerFunc(gatewayController.HttpProcessResponse)).Methods("POST")
	router.Handle("/api/gateway/processPayment", http.HandlerFunc(gatewayController.HttpProcessPayment)).Methods("POST")

	router.Handle("/api/resolver/setupResolving", http.HandlerFunc(resolverController.SetupResolving)).Methods("GET")
	router.Handle("/api/resolver/resolve", http.HandlerFunc(resolverController.DoResolve)).Methods("POST")

	uiServer, err := webUiServer.New(webUiConfig.InitConfig(), uiclient.New(localNode))
	if err != nil {
		log.Error("start web ui server error")
	}
	var baseRouter chi.Router = chi.NewRouter()
	baseRouter.NotFound(uiServer.ServeHTTP)
	clOptions := chi_server.ChiServerOptions{
		BaseRouter: baseRouter,
	}
	chiHandler := chi_server.HandlerWithOptions(utilityController, clOptions)
	router.NotFoundHandler = http.HandlerFunc(chiHandler.ServeHTTP)
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
	fmt.Println("Server is ready!")
	return server
}
