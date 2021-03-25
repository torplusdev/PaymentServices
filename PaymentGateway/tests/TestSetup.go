package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/serviceNode"
	"time"

	"github.com/stellar/go/keypair"
	"go.opentelemetry.io/otel/api/core"
	"google.golang.org/grpc/codes"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
)

type TestSetup struct {
	servers          []*http.Server
	nodes            map[string]*node.Node
	torMock          *TorMock
	torAddressPrefix string
}

const validityPeriod int64 = 1
const autoFlushPeriod time.Duration = 15*time.Minute

func (setup *TestSetup) ConfigureTor(port int) {
	setup.torMock = CreateTorMock(port)
	setup.torAddressPrefix = fmt.Sprintf("http://localhost:%d", port)
}

func (setup *TestSetup) Shutdown() {

	if setup.torMock != nil {
		setup.torMock.Shutdown()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, server := range setup.servers {
		server.Shutdown(ctx)
	}
}

func (setup *TestSetup) startNode(seed string, nodePort int, flushPeriod time.Duration, transactionValidity int64) {
	// TODO: eliminate cycle references
	srv,node, err := serviceNode.StartServiceNode(seed, nodePort, setup.torAddressPrefix, false, flushPeriod, transactionValidity)

	// Set default transaction validity to 1 min
	node.SetTransactionValiditySecs(60);

	 if err != nil {
	 	log.Fatal("Coudn't start node")
	 }

	 setup.nodes[seed] = node
	 setup.servers = append(setup.servers, srv)
}

func (setup *TestSetup) GetNode(seed string) *node.Node {

	return setup.nodes[seed]
}

// func (setup *TestSetup) ReplaceNode(seed string, nodeImplementation node.PPNode) {
// 	// TODO: add implementation
// 	//srv := setup.servers[seed]
// }

func (setup *TestSetup) StartServiceNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("service-node-start:%d %s", nodePort, seed))
	defer span.End()

	setup.startNode(seed, nodePort,autoFlushPeriod,validityPeriod)
	span.SetAttributes(core.KeyValue{
		Key:   "seed",
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK, seed+" Service Node started")
	kp, _ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(), nodePort)
	}
}

func (setup *TestSetup) StartTorNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("tor-node-start:%d %s", nodePort, seed))
	defer span.End()

	setup.startNode(seed, nodePort, autoFlushPeriod,validityPeriod)

	span.SetAttributes(core.KeyValue{
		Key:   core.Key("seed"),
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK, seed+" Tor Node started")

	kp, _ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterTorNode(kp.Address(), nodePort)
	}
}

func (setup *TestSetup) StartUserNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("user-node-start:%d %s", nodePort, seed))
	defer span.End()

	setup.startNode(seed, nodePort, autoFlushPeriod,validityPeriod)

	span.SetAttributes(core.KeyValue{
		Key:   core.Key("seed"),
		Value: core.String(seed),
	})

	span.SetStatus(codes.OK, "User Node started")
	kp, _ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(), nodePort)
		setup.torMock.SetCircuitOrigin(kp.Address())
	}

}

func (setup *TestSetup) CreatePaymentInfo(context context.Context, seed string, amount int) (common.PaymentRequest, error) {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "CreatePaymentInfo")

	span.SetAttributes(core.KeyValue{
		Key:   "seed",
		Value: core.String(seed)},
		core.KeyValue{
			Key:   "amount",
			Value: core.Int(amount),
		})
	defer span.End()

	kp, _ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	cpi := models.CreatePaymentInfo{
		ServiceType:   "ipfs",
		CommodityType: "data",
		Amount:        uint32(amount),
	}

	cpiBytes, err := json.Marshal(cpi)

	resp, err := common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/createPaymentInfo", port), bytes.NewReader(cpiBytes))

	if err != nil || resp.StatusCode != http.StatusOK {
		msg := err.Error()
		if resp.StatusCode != http.StatusOK {
			msg = fmt.Sprint(resp.StatusCode)
		}
		span.SetStatus(codes.Internal, msg)
		return common.PaymentRequest{}, err
	}

	dec := json.NewDecoder(resp.Body)

	var pr common.PaymentRequest

	err = dec.Decode(&pr)

	if err != nil {
		span.SetStatus(codes.Internal, err.Error())
		return common.PaymentRequest{}, err
	}

	span.SetStatus(codes.OK, "Payment Info created successfully")
	return pr, nil
}

func (setup *TestSetup) FlushTransactions(context context.Context) error {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "FlushTransactions")
	defer span.End()

	for k, v := range setup.torMock.GetNodes() {

		log.Printf("Flushing node %s (%d).\n ",k,v)

		resp, err := common.HttpGetWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/transactions/flush", v))

		defer resp.Body.Close()

		if err != nil || resp.StatusCode != http.StatusOK {
			msg := err.Error()
			if resp.StatusCode != http.StatusOK {
				msg = fmt.Sprint(resp.StatusCode)
			}
			span.SetStatus(codes.Internal, msg)
			return err
		}
		span.SetStatus(codes.OK, "FlushTransaction completed successfully")
	}

	return nil
}

type ProcessPaymentRequest struct {
	RouteAddresses []string
	CallbackUrl    string
	PaymentRequest string
}

func (setup *TestSetup) ProcessPayment(context context.Context, seed string, paymentRequest common.PaymentRequest) (string, error) {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "ProcessPayment")
	defer span.End()

	kp, _ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	prBytes, err := json.Marshal(paymentRequest)

	ppr := models.ProcessPaymentRequest{
		Route:             []models.RoutingNode{},
		CallbackUrl:       fmt.Sprintf("%s/api/command", setup.torAddressPrefix),
		StatusCallbackUrl: fmt.Sprintf("%s/api/paymentComplete", setup.torAddressPrefix),
		PaymentRequest:    string(prBytes),
		NodeId:            paymentRequest.Address,
	}

	nodes := setup.torMock.GetNodes()
	keys := make([]string, 0, len(nodes))

	for k := range nodes {
		keys = append(keys, k)
	}
	//append(setup.torMock.GetDefaultPaymentRoute(), paymentRequest.Address)
	for _, k := range setup.torMock.GetDefaultPaymentRoute() {

		ppr.Route = append(ppr.Route, models.RoutingNode{
			NodeId:  k,
			Address: k,
		})

	}

	pprBytes, err := json.Marshal(ppr)

	resp, err := common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/gateway/processPayment", port), bytes.NewReader(pprBytes))

	defer resp.Body.Close()

	//resp,err := http.Post(fmt.Sprintf("http://localhost:%d/api/gateway/processPayment", port),"application/json",bytes.NewReader(pprBytes))

	if err != nil || resp.StatusCode != http.StatusOK {

		var msg string

		if err != nil {
			msg = err.Error()
		}

		if resp.StatusCode != http.StatusOK {
			msg = fmt.Sprint(resp.StatusCode)
		}
		span.SetStatus(codes.Internal, msg)
		return "error", err
	}

	respByte, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		span.SetStatus(codes.Internal, err.Error())
		return "error", err
	}

	result := string(respByte)
	span.SetStatus(codes.OK, "ProcessPayment completed successfully")
	return result, nil
}

func (setup *TestSetup) SetDefaultPaymentRoute(route []string) {
	setup.torMock.SetDefaultRoute(route)
}

func CreateTestSetup() *TestSetup {

	setup := TestSetup{}
	setup.nodes = make(map[string]*node.Node )

	return &setup
}
