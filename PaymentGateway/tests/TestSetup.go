package tests

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/stellar/go/keypair"
	"go.opentelemetry.io/otel/api/core"
	"google.golang.org/grpc/codes"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local"
	"paidpiper.com/payment-gateway/node/proxy"
)

type TestSetup struct {
	nodes            map[string]local.LocalPPNode
	torMock          *TorMock
	torAddressPrefix string
}

const validityPeriod int64 = 1
const autoFlushPeriod time.Duration = 15 * time.Minute

func (setup *TestSetup) ConfigureTor(port int) {
	setup.torMock = CreateTorMock(port)
	setup.torAddressPrefix = fmt.Sprintf("http://localhost:%d", port)
}

func (setup *TestSetup) startNode(seed string, lushPeriod time.Duration, transactionValidity int64) (local.LocalPPNode, error) {
	// TODO: eliminate cycle references

	cfg := config.DefaultCfg()
	cfg.RootApiConfig.Seed = seed

	node, err := local.FromConfigWithClientFactory(cfg, setup.ClientFactory)
	if err != nil {
		log.Fatal("Coudn't start node")
		return nil, err
	}
	return node, nil
}

func (setup *TestSetup) ClientFactory(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler) {
	return setup.torMock.GetNodeByAddress(nodeId), nil
}

func (setup *TestSetup) GetNode(seed string) local.LocalPPNode {
	return setup.nodes[seed]
}

// func (setup *TestSetup) ReplaceNode(seed string, nodeImplementation node.PPNode) {
// 	// TODO: add implementation
// 	//srv := setup.servers[seed]
// }

func (setup *TestSetup) StartServiceNode(ctx context.Context, seed string) error {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("service-node-start: %s", seed))
	defer span.End()

	node, err := setup.startNode(seed, autoFlushPeriod, validityPeriod)
	if err != nil {
		return err
	}

	span.SetAttributes(core.KeyValue{
		Key:   "seed",
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK, seed+" Service Node started")

	if setup.torMock != nil {
		setup.torMock.RegisterNode(node)
		return nil
	}
	return fmt.Errorf("tor mock not inited")

}

func (setup *TestSetup) StartTorNode(ctx context.Context, seed string) error {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("tor-node-start: %s", seed))
	defer span.End()

	node, err := setup.startNode(seed, autoFlushPeriod, validityPeriod)
	if err != nil {
		return nil
	}
	span.SetAttributes(core.KeyValue{
		Key:   core.Key("seed"),
		Value: core.String(seed),
	})
	span.SetStatus(codes.OK, seed+" Tor Node started")

	if setup.torMock != nil {
		setup.torMock.RegisterTorNode(node)
	}
	return nil
}

func (setup *TestSetup) StartUserNode(ctx context.Context, seed string) error {

	tr := common.CreateTracer("TestInit")
	_, span := tr.Start(ctx, fmt.Sprintf("user-node-start: %s", seed))
	defer span.End()

	node, err := setup.startNode(seed, autoFlushPeriod, validityPeriod)
	if err != nil {
		return err
	}
	span.SetAttributes(core.KeyValue{
		Key:   core.Key("seed"),
		Value: core.String(seed),
	})

	span.SetStatus(codes.OK, "User Node started")

	if setup.torMock != nil {
		setup.torMock.RegisterNode(node)
		setup.torMock.SetCircuitOrigin(node.GetAddress())
	}
	return nil
}

func (setup *TestSetup) NewPaymentRequest(context context.Context, seed string, commodity uint32) (*models.PaymentRequest, error) {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "CreatePaymentInfo")

	span.SetAttributes(core.KeyValue{
		Key:   "seed",
		Value: core.String(seed)},
		core.KeyValue{
			Key:   "commodity",
			Value: core.Uint32(commodity),
		})
	defer span.End()

	kp, _ := keypair.ParseFull(seed)

	node := setup.torMock.GetNodeByAddress(kp.Address())

	cpi := &models.CreatePaymentInfo{
		ServiceType:   "ipfs",
		CommodityType: "data",
		Amount:        commodity,
	}
	return node.NewPaymentRequest(ctx, cpi)

}

func (setup *TestSetup) FlushTransactions(context context.Context) error {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "FlushTransactions")
	defer span.End()

	for k, v := range setup.torMock.GetNodes() {
		log.Printf("Flushing node %s.\n ", k)
		err := v.FlushTransactions(ctx)
		if err != nil {
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

func (setup *TestSetup) ProcessPayment(context context.Context, seed string, paymentRequest *models.PaymentRequest) (*models.ProcessPaymentAccepted, error) {

	tr := common.CreateTracer("test")
	ctx, span := tr.Start(context, "ProcessPayment")
	defer span.End()

	kp, _ := keypair.ParseFull(seed)

	node := setup.torMock.GetNodeByAddress(kp.Address())

	ppr := models.ProcessPaymentRequest{
		Route: []models.RoutingNode{},
		// CallbackUrl:       fmt.Sprintf("%s/api/command", setup.torAddressPrefix),
		// StatusCallbackUrl: fmt.Sprintf("%s/api/paymentComplete", setup.torAddressPrefix),
		PaymentRequest: paymentRequest,
		NodeId:         models.PeerID(paymentRequest.Address),
	}

	//nodes := setup.torMock.GetNodes()
	// keys := make([]string, 0, len(nodes))

	// for k := range nodes {
	// 	keys = append(keys, k)
	// }
	//append(setup.torMock.GetDefaultPaymentRoute(), paymentRequest.Address)
	for _, k := range setup.torMock.GetDefaultPaymentRoute() {

		ppr.Route = append(ppr.Route, models.RoutingNode{
			NodeId:  k,
			Address: k,
		})

	}

	resp, err := node.ProcessPayment(ctx, &ppr)
	if err != nil {
		return nil, err
	}

	if err != nil {

		var msg string

		if err != nil {
			msg = err.Error()
		}

		span.SetStatus(codes.Internal, msg)
		return nil, err
	}

	span.SetStatus(codes.OK, "ProcessPayment completed successfully")
	return resp, nil
}

func (setup *TestSetup) SetDefaultPaymentRoute(route []string) {
	setup.torMock.SetDefaultRoute(route)
}

func CreateTestSetup() *TestSetup {

	setup := TestSetup{}
	setup.nodes = map[string]local.LocalPPNode{}

	return &setup
}
