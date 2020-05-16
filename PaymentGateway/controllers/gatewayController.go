package controllers

import (
	"encoding/json"
	"github.com/stellar/go/keypair"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/proxy"
	"paidpiper.com/payment-gateway/routing"
)

type GatewayController struct {
	nodeManager 	*proxy.NodeManager
	client 			*client.Client
	seed			*keypair.Full
	torCommandUrl	string
	torRouteUrl		string
	asyncMode 		bool
}

func NewGatewayController(nodeManager *proxy.NodeManager, client *client.Client, seed *keypair.Full, torCommandUrl string, torRouteUrl string, asyncMode bool) *GatewayController {
	manager := &GatewayController {
		nodeManager,
		client,
		seed,
		torCommandUrl,
		torRouteUrl,
		asyncMode,
	}

	return manager
}

func (g *GatewayController) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Invalid request"))
		return
	}

	pNode := g.nodeManager.GetProxyNode(response.NodeId)

	pNode.ProcessResponse(response.CommandId, response.ResponseBody)
}

func (g *GatewayController) ValidatePayment(w http.ResponseWriter, r *http.Request) {
	// TODO: implement validation by table
}

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r,"ProcessPayment")
	defer span.End()

	request := &models.ProcessPaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Bad request"))
		return
	}

	paymentRequest := &common.PaymentRequest{}

	err = json.Unmarshal([]byte(request.PaymentRequest), paymentRequest)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError, "Unknown payment request"))
		return
	}

	addr := make([]string, 0)

	addr = append(addr, g.seed.Address())

	/*
	if len(request.Route) == 0 {
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
	 */

	for _, rn := range request.Route {
		addr = append(addr, rn.Address)

		// TODO: introduce node Id into route
		g.nodeManager.AddNode(rn.Address, rn.NodeId, g.torCommandUrl)
	}

	// Create destination node
	addr = append(addr, paymentRequest.Address)

	url := request.CallbackUrl

	if url == "" {
		url = g.torCommandUrl
	}

	g.nodeManager.AddNode(paymentRequest.Address, request.NodeId, url)

	router := routing.CreatePaymentRouterStubFromAddresses(addr)

	future := make (chan ResponseMessage)

	go func(c *client.Client, r common.PaymentRouter, pr common.PaymentRequest, responseChannel chan<- ResponseMessage) {
		if g.asyncMode {
			future <- MessageWithStatus(http.StatusCreated,"Payment in process")
		}
		// Initiate
		transactions, err := c.InitiatePayment(ctx, r, pr)

		if err != nil {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusInternalServerError,"Init failed") }

			return
		}

		// Verify
		ok, err := c.VerifyTransactions(ctx, r, pr, transactions)

		if !ok {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusInternalServerError,"Verification failed") }
			return
		}

		// Commit
		ok, err = c.FinalizePayment(ctx, r, transactions, pr)

		if !ok {
			if !g.asyncMode { future <- MessageWithStatus(http.StatusInternalServerError,"Finalize failed") }
			return
		}

		if !g.asyncMode { future <- MessageWithStatus(http.StatusOK,"Payment processing completed") }

		log.Print("Payment completed")
	}(g.client, router, *paymentRequest, future)

	Respond(w, future)
}


