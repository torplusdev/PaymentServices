package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stellar/go/keypair"
	"go.opentelemetry.io/otel/api/core"
	"google.golang.org/grpc/codes"
	"io/ioutil"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/serviceNode"
	testutils "paidpiper.com/payment-gateway/tests"
	"time"
)

type TestSetup struct {
	servers []*http.Server
	torMock *testutils.TorMock
	torAddressPrefix string
}

func (setup *TestSetup) ConfigureTor(port int) {
	setup.torMock = testutils.CreateTorMock(port)
	setup.torAddressPrefix = fmt.Sprintf("http://localhost:%d",port)
}

func (setup *TestSetup) Shutdown() {

	if setup.torMock != nil {
		setup.torMock.Shutdown()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _,server := range setup.servers{
		server.Shutdown(ctx)
	}
}

func (setup *TestSetup) startNode(seed string, nodePort int) {
	srv,err := serviceNode.StartServiceNode(seed, nodePort, setup.torAddressPrefix,false)

	if err!=nil {
		log.Fatal("Coudn't start node")
	}
	setup.servers = append(setup.servers,srv)
}

func (setup *TestSetup) StartServiceNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_,span := tr.Start(ctx,fmt.Sprintf("service-node-start:%d %s",nodePort,seed))
	defer span.End()

	setup.startNode(seed,nodePort)
	span.SetAttributes(core.KeyValue{
		Key:  "seed",
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK,seed + " Service Node started")
	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(),nodePort)
	}
}

func (setup *TestSetup) StartTorNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_,span := tr.Start(ctx,fmt.Sprintf("tor-node-start:%d %s",nodePort,seed))
	defer span.End()

	setup.startNode(seed,nodePort)

	span.SetAttributes(core.KeyValue{
		Key: core.Key("seed"),
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK,seed + " Tor Node started")

	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterTorNode(kp.Address(),nodePort)
	}
}

func (setup *TestSetup) StartUserNode(ctx context.Context, seed string, nodePort int) {

	tr := common.CreateTracer("TestInit")
	_,span := tr.Start(ctx,fmt.Sprintf("user-node-start:%d %s",nodePort,seed))
	defer span.End()

	setup.startNode(seed,nodePort)

	span.SetAttributes(core.KeyValue{
		Key: core.Key("seed"),
		Value: core.String(seed),
	})

	span.SetStatus(codes.OK,"User Node started")
	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(),nodePort)
		setup.torMock.SetCircuitOrigin(kp.Address())
	}

}



func (setup *TestSetup) CreatePaymentInfo(context context.Context,seed string, amount int) (common.PaymentRequest,error) {

	tr := common.CreateTracer("test")
	ctx,span :=tr.Start(context,"CreatePaymentInfo")

	span.SetAttributes(core.KeyValue{
		Key:   "seed",
		Value: core.String(seed) },
		core.KeyValue{
			Key: "amount",
			Value: core.Int(amount),
		})
	defer span.End()

	kp,_ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	cpi := models.CreatePaymentInfo{
		ServiceType:   "test",
		CommodityType: "ipfs",
		Amount:        uint32(amount),
	}

	cpiBytes,err := json.Marshal(cpi)

	resp,err := common.HttpPostWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/createPaymentInfo", port),bytes.NewReader(cpiBytes))

	if err != nil || resp.StatusCode != http.StatusOK {
		msg := err.Error()
		if resp.StatusCode != http.StatusOK{
			msg = string(resp.StatusCode)
		}
		span.SetStatus(codes.Internal,msg)
		return common.PaymentRequest{}, err
	}

	dec := json.NewDecoder(resp.Body)

	var pr common.PaymentRequest

	err = dec.Decode(&pr)

	if err != nil  {
		span.SetStatus(codes.Internal,err.Error())
		return common.PaymentRequest{}, err
	}

	span.SetStatus(codes.OK,"Payment Info created successfully")
	return pr,nil
}

func (setup *TestSetup) FlushTransactions(context context.Context) error {

	tr := common.CreateTracer("test")
	ctx,span :=tr.Start(context,"FlushTransactions")
	defer span.End()

	for _,v := range setup.torMock.GetNodes() {

		resp,err := common.HttpGetWithContext(ctx, fmt.Sprintf("http://localhost:%d/api/utility/flushTransactions", v))

		if err != nil || resp.StatusCode != http.StatusOK {
			msg:= err.Error()
			if resp.StatusCode != http.StatusOK{
				msg = string(resp.StatusCode)
			}
			span.SetStatus(codes.Internal,msg)
			return err
		}
		span.SetStatus(codes.OK,"FlushTransaction completed successfully")
	}

	return nil
}

type ProcessPaymentRequest struct {
	RouteAddresses       []string
	CallbackUrl			 string
	PaymentRequest		 string
}

func (setup *TestSetup) ProcessPayment(context context.Context, seed string,paymentRequest common.PaymentRequest) (string, error) {

	tr := common.CreateTracer("test")
	ctx,span :=tr.Start(context,"ProcessPayment")
	defer span.End()

	kp,_ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	prBytes,err := json.Marshal(paymentRequest)

	ppr := models.ProcessPaymentRequest{
		Route:          []models.RoutingNode{},
		CallbackUrl:    "",
		PaymentRequest: string(prBytes),
		NodeId:         paymentRequest.Address,
	}

	nodes := setup.torMock.GetNodes()
	keys := make([]string, 0, len(nodes))

	for k := range nodes {
		keys = append(keys, k)
	}
	//append(setup.torMock.GetDefaultPaymentRoute(), paymentRequest.Address)
	for _,k := range setup.torMock.GetDefaultPaymentRoute() {

		ppr.Route = append(ppr.Route, models.RoutingNode{
			NodeId:  k,
			Address: k,
		})

	}


	pprBytes,err := json.Marshal(ppr)

	resp,err := common.HttpPostWithContext(ctx,fmt.Sprintf("http://localhost:%d/api/gateway/processPayment", port), bytes.NewReader(pprBytes))

	//resp,err := http.Post(fmt.Sprintf("http://localhost:%d/api/gateway/processPayment", port),"application/json",bytes.NewReader(pprBytes))

	if err != nil || resp.StatusCode != http.StatusOK {
		msg:=err.Error()
		if resp.StatusCode != http.StatusOK {
			msg = string(resp.StatusCode)
		}
		span.SetStatus(codes.Internal,msg)
		return "error", err
	}

	respByte, err := ioutil.ReadAll(resp.Body)

	if err != nil  {
		span.SetStatus(codes.Internal,err.Error())
		return "error", err
	}

	result := string(respByte)
	span.SetStatus(codes.OK,"ProcessPayment completed successfully")
	return result, nil
}

func (setup *TestSetup) SetDefaultPaymentRoute(route []string) {
	setup.torMock.SetDefaultRoute(route)
}


func CreateTestSetup() *TestSetup {

	setup := TestSetup{}

	return &setup
}