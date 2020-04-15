package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"paidpiper.com/payment-gateway/models"
	"strconv"
	"time"
)

type TorMock struct {
	server *http.Server
	nodes  map[string]int
	torNodes map[string]int
}

type torCommand struct {
	CommandBody string
	CommandId string
	CommandType string
	NodeId string
}

func respond(status int, w http.ResponseWriter, data map[string]interface{}) {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)

	if err != nil {
		// Log
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
func (tor *TorMock) processCommand(w http.ResponseWriter, req *http.Request) {
	command := &torCommand{}
	err := json.NewDecoder(req.Body).Decode(command)

	if err != nil {
		w.WriteHeader(500)
		return
	}

	port := tor.nodes[command.NodeId]

	commandType,err := strconv.Atoi(command.CommandType)

	if (err != nil) {
		w.WriteHeader(500)
		return
	}

	utilityCmd := models.UtilityCommand {
		CommandBody: command.CommandBody,
		CommandType: commandType,
	}

	cmdBytes,err := json.Marshal(utilityCmd)

	response,err := http.Post(fmt.Sprintf("http://localhost:%d/api/utility/processCommand",port),"application/json",bytes.NewReader(cmdBytes))
	respBytes, err := ioutil.ReadAll(response.Body)

	utilityResponse := models.UtilityResponse{
		CommandId:    command.CommandId,
		ResponseBody: string(respBytes),
		NodeId:       command.NodeId,
	}

	responseBytes,err := json.Marshal(utilityResponse)

	response,err = http.Post("http://localhost:28080/api/gateway/processResponse","application/json",bytes.NewReader(responseBytes))
	respBytes, err = ioutil.ReadAll(response.Body)

	w.WriteHeader(200)
}

func (tor *TorMock) processPaymentRoute(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)

	node := params["nodeAddress"]

	response := models.RouteResponse{
		RouteAddresses: []string{},
	}
	_ = node

	for k,_ := range tor.torNodes {
		response.RouteAddresses = append(response.RouteAddresses,k )
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

func (tor *TorMock) RegisterNode( address string, port int) {
	tor.nodes[address] = port
}

func CreateTorMock(torPort int)  (*TorMock) {

	tor := TorMock{
		nodes: make(map[string]int),
		torNodes: make(map[string]int),
	}

	router := mux.NewRouter()

	router.HandleFunc("/api/command", tor.processCommand).Methods("POST")
	router.HandleFunc("/api/paymentRoute/{nodeAddress}", tor.processPaymentRoute).Methods("GET")

	server := &http.Server{
		Addr: fmt.Sprintf(":%d",torPort),
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			glog.Fatal("Error starting tor mock: %v",err)
		}
	}()

	tor.server = server

	return &tor
}