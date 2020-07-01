package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"io/ioutil"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"time"
)

type TorMock struct {
	server       *http.Server
	nodes        map[string]int
	torNodes     map[string]int
	defaultRoute []string

	originAddress string
}

type torCommand struct {
	SessionID   string
	NodeId      string
	CommandId   string
	CommandType int
	CommandBody []byte
	CallbackUrl string
}

func respond(status int, w http.ResponseWriter, data map[string]interface{}) {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")

	if data != nil {
		err := json.NewEncoder(w).Encode(data)

		if err != nil {
			// Log
		}
	}
}

func respondObject(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)

	if err != nil {
		// Log
	}
}

func spanFromRequest(r *http.Request, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/tor-mock")
	attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

	r = r.WithContext(correlation.ContextWithMap(r.Context(), correlation.NewMap(correlation.MapUpdate{
		MultiKV: entries,
	})))

	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(r.Context(), spanCtx),
		spanName,
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

func (tor *TorMock) processCommand(w http.ResponseWriter, req *http.Request) {

	ctx, span := spanFromRequest(req, "tor-processCommand")

	defer span.End()

	command := &torCommand{}
	err := json.NewDecoder(req.Body).Decode(command)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	port := tor.nodes[command.NodeId]

	commandType := command.CommandType

	if err != nil {
		w.WriteHeader(500)
		return
	}

	utilityCmd := models.UtilityCommand{
		SessionId:   command.SessionID,
		CommandId:   command.CommandId,
		CommandBody: command.CommandBody,
		CommandType: commandType,
		CallbackUrl: command.CallbackUrl,
	}

	cmdBytes, err := json.Marshal(utilityCmd)

	response, err := common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/processCommand", port), bytes.NewReader(cmdBytes))

	//response,err := http.Post(fmt.Sprintf("http://localhost:%d/api/utility/processCommand",port),"application/json",bytes.NewReader(cmdBytes))

	respBytes, err := ioutil.ReadAll(response.Body)

	utilityResponse := models.UtilityResponse{
		CommandId:    command.CommandId,
		SessionId:    command.SessionID,
		ResponseBody: respBytes,
		NodeId:       command.NodeId,
	}

	responseBytes, err := json.Marshal(utilityResponse)

	originPort := tor.nodes[tor.originAddress]

	response, err = common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/gateway/processResponse", originPort), bytes.NewReader(responseBytes))
	//response,err = http.Post("http://localhost:28080/api/gateway/processResponse","application/json",bytes.NewReader(responseBytes))
	respBytes, err = ioutil.ReadAll(response.Body)

	w.WriteHeader(200)
}

func (tor *TorMock) GetDefaultPaymentRoute() []string {
	return tor.defaultRoute
}

func (tor *TorMock) paymentComplete(w http.ResponseWriter, req *http.Request) {

	_, span := spanFromRequest(req, "tor-paymentComplete")

	defer span.End()

	respond(200, w, nil)
}

func (tor *TorMock) processPaymentRoute(w http.ResponseWriter, req *http.Request) {

	_, span := spanFromRequest(req, "tor-processPaymentRoute")

	defer span.End()

	params := mux.Vars(req)

	node := params["nodeAddress"]

	response := models.RouteResponse{
		Route: []models.RoutingNode{},
	}
	_ = node

	for _, id := range tor.defaultRoute {

		response.Route = append(response.Route, models.RoutingNode{
			NodeId:  id,
			Address: id,
		})
	}

	respondObject(w, response)
}

func (tor *TorMock) Shutdown() {

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tor.server.Shutdown(ctx)
}

func (tor *TorMock) RegisterTorNode(address string, port int) {
	tor.torNodes[address] = port
	tor.nodes[address] = port
}

func (tor *TorMock) RegisterNode(address string, port int) {
	tor.nodes[address] = port
}

func (tor *TorMock) GetNodePort(address string) int {
	return tor.nodes[address]
}

func (tor *TorMock) GetNodes() map[string]int {
	return tor.nodes
}

func (tor *TorMock) SetDefaultRoute(route []string) {
	for _, node := range route {
		if _, ok := tor.nodes[node]; !ok {
			log.Fatalf("Error in route setup, node %s not in nodes", node)
		}
	}

	tor.defaultRoute = route
}

func (tor *TorMock) SetCircuitOrigin(address string) {
	tor.originAddress = address
}

func CreateTorMock(torPort int) *TorMock {

	tor := TorMock{
		nodes:    make(map[string]int),
		torNodes: make(map[string]int),
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/command", tor.processCommand).Methods("POST")
	router.HandleFunc("/api/paymentComplete", tor.paymentComplete).Methods("POST")
	router.HandleFunc("/api/paymentRoute/{nodeAddress}", tor.processPaymentRoute).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", torPort),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err.Error() != "http: Server closed" {
				glog.Fatalf("Error starting tor mock: %v", err)
			}
		}
	}()

	tor.server = server

	return &tor
}
