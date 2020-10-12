package controllers

import (
	"bytes"
	"encoding/json"
	"github.com/stellar/go/keypair"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/proxy"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/routing"
	"sync"
)

type GatewayController struct {
	localNode          node.PPNode
	transactionManager node.PPTransactionManager
	requestProvider    node.PPPaymentRequestProvider
	commodityManager   *commodity.Manager
	seed               *keypair.Full
	rootApi            *root.RootApi
	torCommandUrl      string
	torRouteUrl        string
	asyncMode          bool
	mutex              *sync.Mutex
	requestNodeManager map[string]*proxy.NodeManager
}

func NewGatewayController(node node.PPNode, tm node.PPTransactionManager, rp node.PPPaymentRequestProvider, commodityManager *commodity.Manager, seed *keypair.Full, rootApi *root.RootApi, torCommandUrl string, torRouteUrl string, asyncMode bool) *GatewayController {
	manager := &GatewayController{
		node,
		tm,
		rp,
		commodityManager,
		seed,
		rootApi,
		torCommandUrl,
		torRouteUrl,
		asyncMode,
		&sync.Mutex{},
		map[string]*proxy.NodeManager{},
	}

	return manager
}

func (g *GatewayController) GetNodeManager(sessionId string) (*proxy.NodeManager, bool) {
	g.mutex.Lock()

	defer g.mutex.Unlock()

	nodeManager, ok := g.requestNodeManager[sessionId]

	return nodeManager, ok
}

func (g *GatewayController) SetNodeManager(sessionId string, manager *proxy.NodeManager) {
	g.mutex.Lock()

	defer g.mutex.Unlock()

	g.requestNodeManager[sessionId] = manager
}

func (g *GatewayController) DeleteNodeManager(sessionId string) {
	g.mutex.Lock()

	defer g.mutex.Unlock()

	delete(g.requestNodeManager, sessionId)
}

func (g *GatewayController) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	nodeManager, ok := g.GetNodeManager(response.SessionId)

	if !ok {
		Respond(w, MessageWithStatus(http.StatusConflict, "Session unknown"))
		return
	}

	pNode := nodeManager.GetProxyNode(response.NodeId)

	if pNode == nil {
		Respond(w, MessageWithStatus(http.StatusConflict, "Node unknown"))
		return
	}

	pNode.ProcessResponse(response.CommandId, response.ResponseBody)
}

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "ProcessPayment")
	defer span.End()

	request := &models.ProcessPaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Bad request"))
		return
	}

	paymentRequest := &common.PaymentRequest{}

	err = json.Unmarshal([]byte(request.PaymentRequest), paymentRequest)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Unknown payment request"))
		return
	}

	log.Printf("Got ProcessPayment NodeId=%s, CallbackUrl=%s\n Request:%s", request.NodeId, request.CallbackUrl, request.PaymentRequest)

	_, ok := g.GetNodeManager(paymentRequest.ServiceSessionId)

	if ok {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Duplicate session id"))
		return
	}

	addr := make([]string, 0)

	addr = append(addr, g.seed.Address())

	commandCallbackUrl := request.CallbackUrl
	nodeStatusCallbackUrl := ""

	if request.Route == nil {
		resp, err := common.HttpGetWithContext(ctx, g.torRouteUrl+paymentRequest.ServiceSessionId)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Cant get payment route"))
			return
		}

		defer resp.Body.Close()

		routeResponse := &models.RouteResponse{}

		err = json.NewDecoder(resp.Body).Decode(routeResponse)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Cant get payment route"))
			return
		}

		request.Route = routeResponse.Route
		commandCallbackUrl = routeResponse.CallbackUrl
		nodeStatusCallbackUrl = routeResponse.StatusCallbackUrl
	}

	if commandCallbackUrl == "" {
		log.Printf("Callback url not provided for %s", paymentRequest.ServiceSessionId)

		commandCallbackUrl = g.torCommandUrl
	}

	nodeManager := proxy.New(g.localNode)

	for _, rn := range request.Route {
		addr = append(addr, rn.Address)

		err = nodeManager.AddNode(rn.Address, rn.NodeId, commandCallbackUrl, paymentRequest.ServiceSessionId)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Duplicate node id"))
			return
		}
	}

	// Create destination node
	addr = append(addr, paymentRequest.Address)

	err = nodeManager.AddNode(paymentRequest.Address, request.NodeId, request.CallbackUrl, paymentRequest.ServiceSessionId)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Duplicate node id"))
		return
	}

	future := make(chan ResponseMessage)

	g.SetNodeManager(paymentRequest.ServiceSessionId, nodeManager)

	go func(pr common.PaymentRequest, responseChannel chan<- ResponseMessage) {
		r := routing.CreatePaymentRouterStubFromAddresses(addr)

		c, err := client.CreateClient(g.rootApi, g.seed.Seed(), nodeManager, g.commodityManager)

		if err != nil {
			future <- MessageWithStatus(http.StatusBadRequest, "Client creation failed")

			log.Printf("Payment failed SessionId=%s", pr.ServiceSessionId)
			log.Print(err)

			return
		}

		if g.asyncMode {
			future <- MessageWithData(http.StatusCreated, &models.ProcessPaymentAccepted{
				SessionId: pr.ServiceSessionId,
			})
		}

		// Initiate
		transactions, err := c.InitiatePayment(ctx, r, pr)

		if err != nil {
			if !g.asyncMode {
				future <- MessageWithStatus(http.StatusBadRequest, "Init failed")
			}

			g.SendPaymentCallback(pr.ServiceSessionId, request.StatusCallbackUrl, nodeStatusCallbackUrl, 0)

			g.DeleteNodeManager(pr.ServiceSessionId)

			log.Printf("Payment failed SessionId=%s", pr.ServiceSessionId)
			log.Print(err)

			return
		}

		// Verify
		err = c.VerifyTransactions(ctx, r, pr, transactions)

		if err != nil {
			if !g.asyncMode {
				future <- MessageWithStatus(http.StatusBadRequest, "Verification failed")
			}

			g.SendPaymentCallback(pr.ServiceSessionId, request.StatusCallbackUrl, nodeStatusCallbackUrl, 0)

			g.DeleteNodeManager(pr.ServiceSessionId)

			log.Printf("Payment failed SessionId=%s", pr.ServiceSessionId)
			log.Print(err)

			return
		}

		// Commit
		err = c.FinalizePayment(ctx, r, transactions, pr)

		if err != nil {
			if !g.asyncMode {
				future <- MessageWithStatus(http.StatusBadRequest, "Finalize failed")
			}

			g.SendPaymentCallback(pr.ServiceSessionId, request.StatusCallbackUrl, nodeStatusCallbackUrl, 0)

			g.DeleteNodeManager(pr.ServiceSessionId)

			log.Printf("Payment failed SessionId=%s", pr.ServiceSessionId)
			log.Print(err)

			return
		}

		if !g.asyncMode {
			future <- MessageWithStatus(http.StatusOK, "Payment processing completed")
		}

		g.SendPaymentCallback(pr.ServiceSessionId, request.StatusCallbackUrl, nodeStatusCallbackUrl, 1)

		g.DeleteNodeManager(pr.ServiceSessionId)

		log.Printf("Payment completed SessionId=%s, ServiceRef=%s", pr.ServiceSessionId, pr.ServiceRef)
	}(*paymentRequest, future)

	Respond(w, future)
}

func (g *GatewayController) SendPaymentCallback(sessionId string, callbackUrl string, nodeStatusCallbackUrl string, status int) {
	if callbackUrl == "" && nodeStatusCallbackUrl == "" {
		return
	}

	values := map[string]interface{}{"SessionId": sessionId, "Status": status}

	jsonValue, _ := json.Marshal(values)

	if callbackUrl != "" {
		res, err := common.HttpPostWithoutContext(callbackUrl, bytes.NewBuffer(jsonValue))

		if err != nil {
			log.Print("Payment callback failed")
		} else {
			_ = res.Body.Close()
		}
	}

	if nodeStatusCallbackUrl != "" {
		res, err := common.HttpPostWithoutContext(nodeStatusCallbackUrl, bytes.NewBuffer(jsonValue))

		if err != nil {
			log.Print("Payment nodes callback failed")
		} else {
			_ = res.Body.Close()
		}
	}
}
