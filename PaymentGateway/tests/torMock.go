package tests

import (
	"paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/node/local"
)

type TorMock struct {
	nodes        map[string]local.LocalPPNode
	torNodes     map[string]local.LocalPPNode
	defaultRoute []string

	originAddress string
}

// type torCommand struct {
// 	SessionID   string
// 	NodeId      string
// 	CommandId   string
// 	CommandType models.CommandType
// 	CommandBody []byte
// 	CallbackUrl string
// }

// func (t *torCommand) Type() models.CommandType {
// 	return 1
// }
// func respond(status int, w http.ResponseWriter, data map[string]interface{}) {
// 	w.WriteHeader(status)
// 	w.Header().Add("Content-Type", "application/json")

// 	if data != nil {
// 		err := json.NewEncoder(w).Encode(data)

// 		if err != nil {
// 			// Log
// 		}
// 	}
// }

// func respondObject(w http.ResponseWriter, data interface{}) {
// 	w.WriteHeader(200)
// 	w.Header().Add("Content-Type", "application/json")
// 	err := json.NewEncoder(w).Encode(data)

// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

// func spanFromRequest(r *http.Request, spanName string) (context.Context, trace.Span) {

// 	tracer := common.CreateTracer("paidpiper/tor-mock")
// 	attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

// 	r = r.WithContext(correlation.ContextWithMap(r.Context(), correlation.NewMap(correlation.MapUpdate{
// 		MultiKV: entries,
// 	})))

// 	ctx, span := tracer.Start(
// 		trace.ContextWithRemoteSpanContext(r.Context(), spanCtx),
// 		spanName,
// 		trace.WithAttributes(attrs...),
// 	)

// 	return ctx, span
// }

// func (tor *TorMock) processCommand(w http.ResponseWriter, req *http.Request) {

// 	ctx, span := spanFromRequest(req, "tor-processCommand")

// 	defer span.End()

// 	command := &torCommand{}

// 	port := tor.nodes[command.NodeId]

// 	commandType := command.CommandType

// 	utilityCmd := models.UtilityCommand{
// 		CommandCore: models.CommandCore{
// 			SessionId:   command.SessionID,
// 			NodeId:      "",
// 			CommandId:   command.CommandId,
// 			CommandType: models.CommandType(commandType),
// 		},
// 		CommandBody: command,
// 		CallbackUrl: command.CallbackUrl,
// 	}

// 	cmdBytes, err := json.Marshal(utilityCmd)
// 	if err != nil {
// 		w.WriteHeader(500)
// 	}

// 	response, err := common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/processCommand", port), bytes.NewReader(cmdBytes))

// 	//response,err := http.Post(fmt.Sprintf("http://localhost:%d/api/utility/processCommand",port),"application/json",bytes.NewReader(cmdBytes))

// 	respBytes, err := ioutil.ReadAll(response.Body)

// 	utilityResponse := models.UtilityResponse{
// 		CommandResponseCore: models.CommandResponseCore{
// 			CommandId: command.CommandId,
// 			SessionId: command.SessionID,

// 			NodeId: command.NodeId,
// 		},
// 		CommandResponse: respBytes,
// 	}

// 	responseBytes, err := json.Marshal(utilityResponse)

// 	originPort := tor.nodes[tor.originAddress]

// 	response, err = common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/gateway/processResponse", originPort), bytes.NewReader(responseBytes))
// 	//response,err = http.Post("http://localhost:28080/api/gateway/processResponse","application/json",bytes.NewReader(responseBytes))
// 	respBytes, err = ioutil.ReadAll(response.Body)

// 	w.WriteHeader(200)
// }

func (tor *TorMock) GetDefaultPaymentRoute() []string {
	return tor.defaultRoute
}

// func (tor *TorMock) paymentComplete(w http.ResponseWriter, req *http.Request) {

// 	_, span := spanFromRequest(req, "tor-paymentComplete")

// 	defer span.End()

// 	respond(200, w, nil)
// }

// func (tor *TorMock) processPaymentRoute(w http.ResponseWriter, req *http.Request) {

// 	_, span := spanFromRequest(req, "tor-processPaymentRoute")

// 	defer span.End()

// 	params := mux.Vars(req)

// 	node := params["nodeAddress"]

// 	response := models.RouteResponse{
// 		Route: []models.RoutingNode{},
// 	}
// 	_ = node

// 	for _, id := range tor.defaultRoute {

// 		response.Route = append(response.Route, models.RoutingNode{
// 			NodeId:  id,
// 			Address: id,
// 		})
// 	}

// 	respondObject(w, response)
// }

func (tor *TorMock) RegisterTorNode(node local.LocalPPNode) {
	address := node.GetAddress()
	tor.torNodes[address] = node
	tor.nodes[address] = node
}

func (tor *TorMock) RegisterNode(node local.LocalPPNode) {
	tor.nodes[node.GetAddress()] = node
}

func (tor *TorMock) GetNodes() map[string]local.LocalPPNode {
	return tor.nodes
}

func (tor *TorMock) GetNodeByAddress(address string) local.LocalPPNode {
	return tor.nodes[address]
}

func (tor *TorMock) SetDefaultRoute(route []string) {
	for _, node := range route {
		if _, ok := tor.nodes[node]; !ok {
			log.Printf("Error in route setup, node %s not in nodes", node)
		}
	}

	tor.defaultRoute = route
}

func (tor *TorMock) SetCircuitOrigin(address string) {
	tor.originAddress = address
}

func CreateTorMock(torPort int) *TorMock {

	tor := TorMock{
		nodes:    make(map[string]local.LocalPPNode),
		torNodes: make(map[string]local.LocalPPNode),
	}

	//router := mux.NewRouter()

	// router.HandleFunc("/api/command", tor.processCommand).Methods("POST")
	// router.HandleFunc("/api/paymentComplete", tor.paymentComplete).Methods("POST")
	// router.HandleFunc("/api/paymentRoute/{nodeAddress}", tor.processPaymentRoute).Methods("GET")

	// server := &http.Server{
	// 	Addr:    fmt.Sprintf(":%d", torPort),
	// 	Handler: router,
	// }

	// go func() {
	// 	if err := server.ListenAndServe(); err != nil {
	// 		if err.Error() != "http: Server closed" {
	// 			glog.Fatalf("Error starting tor mock: %v", err)
	// 		}
	// 	}
	// }()

	return &tor
}
