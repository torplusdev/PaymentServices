package controllers

import (
	"encoding/json"
	"github.com/stellar/go/keypair"
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
}

func New(nodeManager *proxy.NodeManager, client *client.Client, seed *keypair.Full, torCommandUrl string, torRouteUrl string) *GatewayController {
	manager := &GatewayController {
		nodeManager,
		client,
		seed,
		torCommandUrl,
		torRouteUrl,
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

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {
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

	if len(request.RouteAddresses) == 0 {
		resp, err := http.Get(g.torRouteUrl + paymentRequest.Address)

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

		request.RouteAddresses = routeResponse.RouteAddresses
	}

	for _, a := range request.RouteAddresses {
		addr = append(addr, a)

		n := proxy.NewProxy(a, g.torCommandUrl, a)

		g.nodeManager.AddNode(a, n)
	}

	// Create destination node
	addr = append(addr, paymentRequest.Address)

	url := request.CallbackUrl

	if url == "" {
		url = g.torCommandUrl
	}

	n := proxy.NewProxy(paymentRequest.Address, url, request.RequestReference)

	g.nodeManager.AddNode(paymentRequest.Address, n)

	router := routing.CreatePaymentRouterStubFromAddresses(addr)

	future := make (chan ResponseMessage)
	returnAsyncImmediately := false

	go func(c *client.Client, r common.PaymentRouter, pr common.PaymentRequest) {

		if returnAsyncImmediately {
			future <- Message("Payment in process")
		}
		// Initiate
		transactions, err := c.InitiatePayment(r, pr)

		if err != nil {
			if !returnAsyncImmediately { future <- MessageWithStatus(http.StatusInternalServerError,"Init failed") }

			return
		}

		// Verify
		ok, err := c.VerifyTransactions(r, pr, transactions)

		if !ok {
			if !returnAsyncImmediately { future <- MessageWithStatus(http.StatusInternalServerError,"Verification failed") }
			return
		}

		// Commit
		ok, err = c.FinalizePayment(r, transactions, pr)

		if !ok {
			if !returnAsyncImmediately { future <- MessageWithStatus(http.StatusInternalServerError,"Finalize failed") }
			return
		}

		if !returnAsyncImmediately { future <- MessageWithStatus(http.StatusOK,"Payment processing completed") }
	}(g.client, router, *paymentRequest)

	Respond(w, future)
}

