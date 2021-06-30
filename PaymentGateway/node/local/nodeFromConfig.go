package local

import (
	"context"
	"fmt"

	"paidpiper.com/payment-gateway/log"

	"github.com/golang/glog"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/config"
	"paidpiper.com/payment-gateway/node/proxy"
	"paidpiper.com/payment-gateway/regestry"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/torclient"
)

// func NewNodeStartConfig(config *common.Configuration) *NodeStartConfig {
// 	return &NodeStartConfig{
// 		RootApiCfg:        config.RootApiConfig,
// 		TorAddressPrefix:  "http://localhost:5817",
// 		AsyncMode:         true,
// 		AutoFlushDuration: config.AutoFlushPeriod,
// 	}
// }

// type NodeStartConfig struct {
// 	RootApiCfg        RootApiConfig
// 	TorAddressPrefix  string
// 	AsyncMode         bool
// 	AutoFlushDuration time.Duration
// }

func FromConfig(config *config.Configuration) (LocalPPNode, error) {
	//cfg := NewNodeStartConfig(config)
	clientFactory := TorClientFactory(config.TorAddressPrefix)
	return FromConfigWithClientFactory(config, clientFactory)
}

func FromConfigWithClientFactory(config *config.Configuration, clientFactory regestry.CommandClientFactory) (LocalPPNode, error) {
	tracer := common.CreateTracer("paidpiper/serviceNode")

	_, span := tracer.Start(context.Background(), "serviceNode-initialization")
	defer span.End()

	rootClient, err := CreateRootApi(config.RootApiConfig)
	if err != nil {
		return nil, err
	}
	torClient := TorRouteBuilder(config.TorAddressPrefix)

	return LocalHost(config, rootClient, torClient, clientFactory)
}

//TODO FIX
func LocalHost(config *config.Configuration, rootClient root.RootApi,
	torClient torclient.TorClient,
	commandClientFactory regestry.CommandClientFactory) (LocalPPNode, error) {
	commodityManager := commodity.New()
	paymentRegestry := regestry.NewPaymentManagerRegestry(
		commodityManager,
		client.New(rootClient),
		commandClientFactory,
		torClient)
	localNode, err := New(rootClient, paymentRegestry,
		newCallbacker,
		config.NodeConfig)

	if err != nil {
		glog.Infof("Error creating Node object: %s", err)
		return nil, err
	}

	return localNode, nil
}

func CreateRootApi(cfg config.RootApiConfig) (root.RootApi, error) {
	clientFactory := root.CreateRootApiFactory(cfg.UseTestApi)
	rootClient, err := clientFactory(cfg.Seed, cfg.TransactionValiditySecs)
	if err != nil {
		return nil, err
	}
	// Account validation
	err = rootClient.ValidateForPPNode()
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("client creation failed")
	}
	balance, err := rootClient.GetMicroPPTokenBalance()

	if err != nil {
		glog.Infof("Error retrieving account data: %s", err)
		return nil, err
	}
	log.Infof("Current balance for %v:%v\n", rootClient.GetAddress(), balance)
	return rootClient, nil
}

func TorClientFactory(host string) regestry.CommandClientFactory {
	defaultUrl := fmt.Sprintf("%s/api/command", host)
	return func(url string, sessionId string, nodeId string) (proxy.CommandClient, proxy.CommandResponseHandler) {
		if url == "" {
			log.Tracef("Callback url not provided for %s", sessionId)
			url = defaultUrl
		}
		return proxy.NewCommandClient(url, sessionId, nodeId)
	}
}

func TorRouteBuilder(host string) torclient.TorClient {
	if host == "" {
		host = "http://localhost:5817"
	}
	torRouteUrl := fmt.Sprintf("%s/api/paymentRoute/", host)
	torClient := torclient.NewTorClient(torRouteUrl)
	return torClient
}
