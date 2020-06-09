package controllers

import (
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
)

type GatewayController struct {
	localNode 			*node.Node
	commodityManager	*commodity.Manager
	seed				*keypair.Full
	rootApi 			*root.RootApi
	torCommandUrl		string
	torRouteUrl			string
	asyncMode 			bool
	requestNodeManager	map[string]*proxy.NodeManager
}

func NewGatewayController(localNode *node.Node, commodityManager *commodity.Manager, seed *keypair.Full, rootApi *root.RootApi, torCommandUrl string, torRouteUrl string, asyncMode bool) *GatewayController {
	manager := &GatewayController {
		localNode,
		commodityManager,
		seed,
		rootApi,
		torCommandUrl,
		torRouteUrl,
		asyncMode,
		map[string]*proxy.NodeManager{},
	}

	return manager
}

func (g *GatewayController) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	nodeManager, ok := g.requestNodeManager[response.SessionId]

	if !ok {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Session unknown"))
		return
	}

	pNode := nodeManager.GetProxyNode(response.NodeId)

	pNode.ProcessResponse(response.CommandId, response.ResponseBody)
}

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r,"ProcessPayment")
	defer span.End()

	nodeManager := proxy.New(g.localNode)

	c := client.CreateClient(g.rootApi, g.seed.Seed(), nodeManager, g.commodityManager)

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

	_, ok := g.requestNodeManager[paymentRequest.ServiceSessionId]

	if ok {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Duplicate session id"))
		return
	}

	addr := make([]string, 0)

	addr = append(addr, g.seed.Address())

	if request.Route == nil {
		resp, err := common.HttpGetWithContext(ctx, g.torRouteUrl + paymentRequest.Address)
		//resp, err := http.Get(g.torRouteUrl + paymentRequest.Address)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Cant get payment route"))
			return
		}

		routeResponse := &models.RouteResponse{}

		err = json.NewDecoder(resp.Body).Decode(routeResponse)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Cant get payment route"))
			return
		}

		request.Route  = routeResponse.Route
	}

	for _, rn := range request.Route {
		addr = append(addr, rn.Address)

		err = nodeManager.AddNode(rn.Address, rn.NodeId, g.torCommandUrl, paymentRequest.ServiceSessionId)

		if err != nil {
			Respond(w, MessageWithStatus(http.StatusInternalServerError, "Duplicate node id"))
			return
		}
	}

	// Create destination node
	addr = append(addr, paymentRequest.Address)

	commandCallbackUrl := request.CallbackUrl

	if commandCallbackUrl == "" {
		commandCallbackUrl = g.torCommandUrl
	}

	err = nodeManager.AddNode(paymentRequest.Address, request.NodeId, commandCallbackUrl, paymentRequest.ServiceSessionId)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Duplicate node id"))
		return
	}

	router := routing.CreatePaymentRouterStubFromAddresses(addr)

	future := make (chan ResponseMessage)

	g.requestNodeManager[paymentRequest.ServiceSessionId] = nodeManager

	go func(c *client.Client, r common.PaymentRouter, pr common.PaymentRequest, responseChannel chan<- ResponseMessage) {
		if g.asyncMode {
			future <- MessageWithStatus(http.StatusCreated,"Payment in process")
		}

		// Initiate
		transactions, err := c.InitiatePayment(ctx, r, pr)

		if err != nil {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusBadRequest,"Init failed") }

			delete(g.requestNodeManager, pr.ServiceSessionId)

			return
		}

		// Verify
		ok, err := c.VerifyTransactions(ctx, r, pr, transactions)

		if !ok {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusBadRequest,"Verification failed") }

			delete(g.requestNodeManager, pr.ServiceSessionId)

			return
		}

		// Commit
		ok, err = c.FinalizePayment(ctx, r, transactions, pr)

		if !ok {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusBadRequest,"Finalize failed") }

			delete(g.requestNodeManager, pr.ServiceSessionId)

			return
		}

		if !g.asyncMode { future <- MessageWithStatus(http.StatusOK,"Payment processing completed") }

		delete(g.requestNodeManager, pr.ServiceSessionId)
		log.Print("Payment completed")
	}(c, router, *paymentRequest, future)

	Respond(w, future)
}


