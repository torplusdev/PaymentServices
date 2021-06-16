package paymentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	boomserver "paidpiper.com/payment-gateway/boom/server"
	"paidpiper.com/payment-gateway/models"
)

type PPCallbackServer struct {
	server *http.Server
	PPCallback
}

func (p *PPCallbackServer) Start() {
	err := p.server.ListenAndServe()

	if err != nil {
		panic(err)
	}
}

func (p *PPCallbackServer) Shutdown(ctx context.Context) {
	err := p.server.Shutdown(ctx)

	if err != nil {
		log.Fatalf("connection shutdown failed %s", err.Error())
	}
}

func NewServer(commandListenPort int, ppcallback *PPCallback) CallbackHandler {
	router := mux.NewRouter()

	callbackServer := &PPCallbackServer{
		PPCallback: *ppcallback,
	}

	router.HandleFunc("/api/command", callbackServer.HandleProcessCommand).Methods("POST")
	router.HandleFunc("/api/commandResponse", callbackServer.HandleProcessCommandResponse).Methods("POST")
	router.HandleFunc("/api/paymentResponse", callbackServer.HandleProcessPaymentResponse).Methods("POST")
	boomserver.AddHandlers(router)
	callbackServer.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", commandListenPort),
		Handler: router,
	}

	return callbackServer
}

func (p *PPCallbackServer) HandleProcessCommand(w http.ResponseWriter, r *http.Request) {
	// Extract command request from the request and forward it to peer
	request := &models.ProcessCommand{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}
		return
	}
	err = p.ProcessCommand(request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *PPCallbackServer) HandleProcessCommandResponse(w http.ResponseWriter, r *http.Request) {
	// Extract command response from the request and forward it to peer
	response := &models.ProcessCommandResponse{}
	err := json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}
		return
	}
	err = p.ProcessCommandResponse(response)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (p *PPCallbackServer) HandleProcessPaymentResponse(w http.ResponseWriter, r *http.Request) {
	request := &models.PaymentStatusResponseModel{}
	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}
		return
	}
	err = p.ProcessPaymentResponse(request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err := io.WriteString(w, err.Error())
		if err != nil {
			log.Fatalf("Error:%v", err)
		}

		return
	}
	w.WriteHeader(http.StatusOK)
}
