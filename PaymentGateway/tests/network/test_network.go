package network

import (
	"context"

	"github.com/rs/xid"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/local"
	"paidpiper.com/payment-gateway/node/proxy"
	"paidpiper.com/payment-gateway/root"
)

type TestNetwork interface {
	//Get
}
type testNetwork struct {
	nodesByNodeID map[string]node.PPNode
	seedNodeIds   map[string]string
	clientSeed    string
	chainSeeds    []string
	serviceSeed   string
}

func New(clientSeed string, chainSeeds []string, serviceSeed string) (TestNetwork, error) {
	net := &testNetwork{
		nodesByNodeID: map[string]node.PPNode{},
		clientSeed:    clientSeed,
		chainSeeds:    chainSeeds,
		serviceSeed:   serviceSeed,
	}
	for _, seed := range append([]string{clientSeed, serviceSeed}, chainSeeds...) {
		nodeId := xid.New().String()
		node, err := net.createNodeWith(seed)
		if err != nil {
			return nil, err
		}
		net.nodesByNodeID[nodeId] = node

	}
	return net, nil
}

func (net *testNetwork) createNodeWith(seed string) (local.LocalPPNode, error) {
	transactionValiditySecs := config.DefaultCfg().RootApiConfig.TransactionValiditySecs

	rootClient, err := root.CreateRootApiFactory(true)(seed, transactionValiditySecs)
	if err != nil {
		return nil, err
	}
	localNode, err := local.LocalHost(config.DefaultCfg(), rootClient, net, net.CommandClientFactory)
	if err != nil {
		return nil, err
	}
	return localNode, nil
}

func (net *testNetwork) GetRoute(ctx context.Context, sessionId string) (*models.RouteResponse, error) {
	route := []models.RoutingNode{}
	for _, seed := range net.chainSeeds {
		nodeId := net.seedNodeIds[seed]
		route = append(route, models.RoutingNode{
			NodeId:  nodeId,
			Address: net.nodesByNodeID[nodeId].GetAddress(),
		})
	}
	return &models.RouteResponse{
		CallbackUrl:       "",
		StatusCallbackUrl: "",
		Route:             route,
	}, nil
}

func (net *testNetwork) CommandClientFactory(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler) {
	return net.nodesByNodeID[nodeId], nil //ResponseHandler is nil because withou http layer
}
