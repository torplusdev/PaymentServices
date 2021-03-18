package regestry

import (
	"context"
	"fmt"
	"log"

	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/node/proxy"
)

type PaymentManager interface {
	AddSourceNode(address string, node node.PPNode) error
	AddChainNode(address, nodeId string, node node.PPNode) error
	AddDestinationNode(address, nodeId string, node node.PPNode) error

	Run(ctx context.Context, async bool) error
	Complete(msg *models.PaymentStatusResponseModel)
	ProcessResponse(context context.Context, nodeId string, commandId string, response []byte) error
	AddStatusCallbackUrl(url string)
}

func NewPaymentManager(serviceClient client.ServiceClient,
	request *models.ProcessPaymentRequest,
	statusCallbacker StatusCallbacker) PaymentManager {
	return &paymentManager{
		client:           serviceClient,
		nodes:            NewNodeManager(),
		ch:               make(chan *models.PaymentStatusResponseModel),
		request:          request,
		nodesByNodeId:    map[string]proxy.ProxyNode{},
		statusCallbacker: statusCallbacker,
	}
}

type paymentManager struct {
	request          *models.ProcessPaymentRequest
	client           client.ServiceClient
	nodes            NodeManager
	ch               chan *models.PaymentStatusResponseModel
	nodesByNodeId    map[string]proxy.ProxyNode
	statusCallbacker StatusCallbacker
}

func (pm *paymentManager) AddSourceNode(address string, node node.PPNode) error {
	return pm.nodes.AddSourceNode(address, node)
}
func (pm *paymentManager) AddChainNode(address, nodeId string, node node.PPNode) error {
	return pm.nodes.AddChainNode(address, node)
}
func (pm *paymentManager) AddDestinationNode(address, nodeId string, node node.PPNode) error {
	err := pm.nodes.AddDestinationNode(address, node)
	return err
}

func (pm *paymentManager) paymentProcess(ctx context.Context) error {
	request := pm.request
	sessionId := request.PaymentRequest.ServiceSessionId

	// Initiate
	transactions, err := pm.client.InitiatePayment(ctx, pm.nodes, request.PaymentRequest)

	if err != nil {
		log.Printf("Payment failed SessionId=%s", sessionId)
		log.Print(err)

		return fmt.Errorf("initiate payment failed")
	}

	// Verify
	err = pm.client.VerifyTransactions(ctx, transactions)

	if err != nil {
		log.Printf("Payment failed SessionId=%s", sessionId)
		log.Print(err)
		return fmt.Errorf("verification failed")
	}

	// Commit
	err = pm.client.FinalizePayment(ctx, pm.nodes, request.PaymentRequest, transactions)

	if err != nil {

		log.Printf("payment failed SessionId=%s", sessionId)
		log.Print(err)

		return fmt.Errorf("finalize failed")
	}

	log.Printf("Payment completed SessionId=%s, ServiceRef=%s", sessionId, request.PaymentRequest.ServiceRef)
	return nil
}

func (pm *paymentManager) runSync(ctx context.Context) error {
	err := pm.paymentProcess(ctx)
	if err != nil {
		return err
	}
	msg := pm.Wait()
	pm.statusCallbacker.Complete(msg)
	return nil
}
func (pm *paymentManager) Run(ctx context.Context, async bool) error {
	if async {
		go func(pm *paymentManager) {
			err := pm.runSync(context.Background())
			if err != nil {
				log.Fatalf("Error paymentProcess %v", err)
			}
		}(pm)
		return nil
	}
	return pm.runSync(ctx)
}
func (pm *paymentManager) Wait() *models.PaymentStatusResponseModel {
	return <-pm.ch
}
func (pm *paymentManager) Complete(msg *models.PaymentStatusResponseModel) {
	if pm.ch != nil {
		pm.ch <- msg
	}
}
func (pm *paymentManager) ProcessResponse(context context.Context, nodeId string, commandId string, response []byte) error {
	proxyNode, ok := pm.nodesByNodeId[nodeId]
	if !ok {
		return fmt.Errorf("proxynode not found")
	}
	return proxyNode.ProcessResponse(context, commandId, response)
}
func (pm *paymentManager) AddStatusCallbackUrl(url string) {
	pm.statusCallbacker.AddUrl(url)
}

type StatusCallbacker interface {
	AddUrl(url string)
	Complete(msg *models.PaymentStatusResponseModel)
}

func NewStatusCallbacker() StatusCallbacker {
	return &statusCallbacker{}
}

type statusCallbacker struct {
	urls []string
}

func (scb *statusCallbacker) AddUrl(url string) {
	if url == "" {
		return
	}
	scb.urls = append(scb.urls, url)
}
func (scb *statusCallbacker) Complete(msg *models.PaymentStatusResponseModel) {
	go func() {
		for _, url := range scb.urls {
			scb.sendPaymentCallback(url, msg)
		}
	}()
}
func (scb *statusCallbacker) sendPaymentCallback(callbackUrl string, msg *models.PaymentStatusResponseModel) {
	if callbackUrl == "" {
		return
	}
	err := common.HttpPaymentStatus(callbackUrl, msg)
	if err != nil {
		log.Print("Payment callback failed")

	}
}
