package paymentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"paidpiper.com/payment-gateway/log"

	"github.com/gorilla/mux"
	boomserver "paidpiper.com/payment-gateway/boom/server"
	"paidpiper.com/payment-gateway/models"
)

type ppCallbackServer struct {
	server          *http.Server
	callbackHandler PPCallbackHandler
	router          *mux.Router
	metricsStore    *MetricsStore
	metricsHandler  http.Handler
}

func (p *ppCallbackServer) Start() {
	log.Infof("Start pp server %v", p.server.Addr)
	err := p.server.ListenAndServe()

	if err != nil {
		panic(err)
	}
}

func (p *ppCallbackServer) Shutdown(ctx context.Context) {
	err := p.server.Shutdown(ctx)

	if err != nil {
		log.Errorf("connection shutdown failed %s", err.Error())
	}
}

func (p *ppCallbackServer) SetPort(port int) {
	log.Infof("PP server port %v", port)
	p.server.Addr = fmt.Sprintf(":%d", port)
}

func NewServer() CallbackServer {
	router := mux.NewRouter()

	callbackServer := &ppCallbackServer{}
	AddHandlers(router, callbackServer)
	callbackServer.server = &http.Server{
		Addr:    ":30500",
		Handler: router,
	}

	return callbackServer

}
func (p *ppCallbackServer) SetMetricsSource(source MetricsSource) {
	p.metricsStore = NewMetricsStore(source)
	p.metricsHandler = p.metricsStore.Handler()
}
func AddHandlers(router *mux.Router, callbackServer *ppCallbackServer) {
	router.HandleFunc("/version", func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(rw, "dev_version")
	})
	router.HandleFunc("/metrics", func(rw http.ResponseWriter, r *http.Request) {
		if callbackServer.metricsHandler != nil {
			callbackServer.metricsHandler.ServeHTTP(rw, r)
		} else {
			rw.WriteHeader(http.StatusBadGateway)
		}

	})
	router.HandleFunc("/api/command", callbackServer.HandleProcessCommand).Methods("POST")
	router.HandleFunc("/api/commandResponse", callbackServer.HandleProcessCommandResponse).Methods("POST")
	router.HandleFunc("/api/paymentResponse", callbackServer.HandleProcessPaymentResponse).Methods("POST")
	boomserver.AddHandlers(router)
}
func (p *ppCallbackServer) SetCallbackHandler(cb PPCallbackHandler) {
	p.callbackHandler = cb
}
func (p *ppCallbackServer) HandleProcessCommand(w http.ResponseWriter, r *http.Request) {
	// Extract command request from the request and forward it to peer
	request := &models.ProcessCommand{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Errorf("ProcessCommand error: ", err)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Errorf("Error:%v", err)
		}
		return
	}
	log.Infof("Process command: SessionId: %v CommandType: %v NodeId: %v CommandId: %v",
		request.SessionId, request.CommandType, request.NodeId, request.CommandId)
	if p.callbackHandler != nil {
		err = p.callbackHandler.ProcessCommand(request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, err := io.WriteString(w, err.Error())
			if err != nil {
				log.Errorf("Error:%v", err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		fmt.Fprintln(w, "callbackHandler not found")
		w.WriteHeader(http.StatusBadGateway)
	}
}

func (p *ppCallbackServer) HandleProcessCommandResponse(w http.ResponseWriter, r *http.Request) {
	// Extract command response from the request and forward it to peer
	response := &models.UtilityResponse{}
	err := json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Errorf("ProcessCommandResponse error: ", err)

		if err != nil {
			log.Errorf("Error:%v", err)
		}
		return
	}
	log.Infof("Process response: SessionId: %v NodeId: %v CommandId: %v  CommandType: %v",
		response.SessionId, response.NodeId, response.CommandId, response.CommandType)
	if p.callbackHandler != nil {
		err = p.callbackHandler.ProcessCommandResponse(response)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, err := io.WriteString(w, err.Error())
			if err != nil {
				log.Errorf("Error:%v", err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		log.Error("Error callback handler is null SessionId: ", response.SessionId)
		fmt.Fprintln(w, "callbackHandler not found")
		w.WriteHeader(http.StatusBadGateway)
	}

}

func (p *ppCallbackServer) HandleProcessPaymentResponse(w http.ResponseWriter, r *http.Request) {
	request := &models.PaymentStatusResponseModel{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Errorf("ProcessPaymentResponse error: ", err)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Errorf("Error:%v", err)
		}
		return
	}
	log.Infof("Payment response: SessionId: %v Status: %v", request.SessionId, request.Status)
	if p.callbackHandler != nil {
		err = p.callbackHandler.ProcessPaymentResponse(request)
		if err != nil {

			w.WriteHeader(http.StatusNotFound)
			_, err := io.WriteString(w, err.Error())
			if err != nil {
				log.Errorf("Error:%v", err)
			}

			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		fmt.Fprintln(w, "callbackHandler not found")
		w.WriteHeader(http.StatusBadGateway)
	}

}
