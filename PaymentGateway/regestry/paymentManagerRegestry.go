package regestry

import (
	"context"
	"fmt"
	"sync"

	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/log"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/proxy"
	"paidpiper.com/payment-gateway/torclient"
)

type PaymentManagerRegestry interface {
	New(ctx context.Context, source node.PPNode, request *models.ProcessPaymentRequest) (PaymentManager, error)
	Get(sessionId string) PaymentManager
	Has(sessionId string) bool
	Set(sessionId string, pm PaymentManager)
}

type paymentManagerRegestryImpl struct {
	mutex *sync.Mutex

	requestNodeManager   map[string]PaymentManager
	commodityManager     commodity.Manager
	torClient            torclient.TorClient
	serviceClient        client.ServiceClient
	commandClientFactory CommandClientFactory
}
type CommandClientFactory func(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler)

func NewPaymentManagerRegestry(
	commodityManager commodity.Manager,
	serviceClient client.ServiceClient,
	commandClientFactory CommandClientFactory,
	torClient torclient.TorClient) PaymentManagerRegestry {
	return &paymentManagerRegestryImpl{
		mutex:                &sync.Mutex{},
		requestNodeManager:   map[string]PaymentManager{},
		commodityManager:     commodityManager,
		serviceClient:        serviceClient,
		commandClientFactory: commandClientFactory,
		torClient:            torClient,
	}
}

func (g *paymentManagerRegestryImpl) New(ctx context.Context, source node.PPNode,
	request *models.ProcessPaymentRequest) (PaymentManager, error) {

	sessionId := request.PaymentRequest.ServiceSessionId
	routingNotes := request.Route

	paymentManager := NewPaymentManager(g.serviceClient, request)
	statusCallbacker := NewStatusCallbacker(request.StatusCallbackUrl)
	paymentManager.AddStatusCallbacker(statusCallbacker)
	localAdderss := source.GetAddress()
	err := paymentManager.AddSourceNode(localAdderss, source)
	if err != nil {
		return nil, err
	}

	commandCallbackUrl := request.CallbackUrl
	if len(routingNotes) == 0 {
		routeResponse, err := g.torClient.GetRoute(ctx, sessionId, request.NodeId.String(), request.PaymentRequest.Address)
		if err != nil {
			return nil, err
		}
		if len(routeResponse.Route) != 3 {
			err := fmt.Errorf("route len is not 3 (3!= %v)", len(routeResponse.Route))
			log.Error(err)
			return nil, err
		}
		routingNotes = routeResponse.Route
		commandCallbackUrl = routeResponse.CallbackUrl

		if routeResponse.StatusCallbackUrl != "" {
			statusCallbacker := NewStatusCallbacker(routeResponse.StatusCallbackUrl)
			paymentManager.AddStatusCallbacker(statusCallbacker)
		}
	}

	for i, rn := range routingNotes {
		nodeId := rn.NodeId
		log.Infof("Route %v %v", i, nodeId)
		commandClient, responseHandler := g.commandClientFactory(commandCallbackUrl, sessionId, nodeId)
		fee := g.commodityManager.GetProxyNodeFee()
		n := proxy.NewProxyNode(commandClient, responseHandler, rn.Address, fee)
		err := paymentManager.AddChainNode(rn.Address, rn.NodeId, n)
		if err != nil {
			return nil, err
		}
	}

	nodeId := request.NodeId.String()
	address := request.PaymentRequest.Address
	log.Infof("Route %v %v", "last", nodeId)
	commandClient, responseHandler := g.commandClientFactory(request.CallbackUrl, sessionId, nodeId)
	proxyNode := proxy.NewProxyNode(commandClient, responseHandler, address, 0)
	err = paymentManager.AddDestinationNode(address, nodeId, proxyNode)
	if err != nil {
		log.Infof("Destination node %v already exists in chain ", nodeId)
		return nil, err
	}
	return paymentManager, nil
}

func (g *paymentManagerRegestryImpl) Has(sessionId string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	_, ok := g.requestNodeManager[sessionId]
	return ok
}

func (g *paymentManagerRegestryImpl) Get(sessionId string) PaymentManager {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	item, ok := g.requestNodeManager[sessionId]
	if ok {
		return item
	}
	return nil
}

func (g *paymentManagerRegestryImpl) Set(sessionId string, pm PaymentManager) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.requestNodeManager[sessionId] = pm
}
