package controllers

import (
	"encoding/json"
	"github.com/stellar/go/keypair"
	"net/http"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/proxy"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/routing"
)

type GatewayController struct {
	nodeManager 	*proxy.NodeManager
	seed			*keypair.Full
	torCommandUrl	string
	torRouteUrl		string
}

func New(nodeManager *proxy.NodeManager, seed *keypair.Full, torCommandUrl string, torRouteUrl string) *GatewayController {
	manager := &GatewayController {
		nodeManager,
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
		Respond(500, w, Message("Invalid request"))
		return
	}

	pNode := g.nodeManager.GetProxyNode(response.NodeId)

	pNode.ProcessResponse(response.CommandId, response.ResponseBody)
}

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	request := &models.ProcessPaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(500, w, Message("Bad request"))
		return
	}

	rootApi := root.CreateRootApi(true)

	c := client.CreateClient(rootApi, g.seed.Seed(), g.nodeManager)

	addr := make([]string, 0)

	addr = append(addr, g.seed.Address())

	if len(request.RouteAddresses) == 0 {
		resp, err := http.Get(g.torRouteUrl)

		if err != nil {
			Respond(500, w, Message("Cant get payment route"))
			return
		}

		routeResponse := &models.RouteResponse{}

		err = json.NewDecoder(resp.Body).Decode(routeResponse)

		if err != nil {
			Respond(500, w, Message("Cant get payment route"))
			return
		}

		request.RouteAddresses = routeResponse.RouteAddresses
	}

	for _, a := range request.RouteAddresses {
		addr = append(addr, a)

		n := proxy.NewProxy(a, g.torCommandUrl)

		g.nodeManager.AddNode(a, n)
	}

	addr = append(addr, request.Address)

	if request.CallbackUrl == "" {
		n := proxy.NewProxy(request.Address, request.CallbackUrl)

		g.nodeManager.AddNode(request.Address, n)
	} else {
		n := proxy.NewProxy(request.Address, g.torCommandUrl)

		g.nodeManager.AddNode(request.Address, n)
	}

	router := routing.CreatePaymentRouterStubFromAddresses(addr)

	pr := common.PaymentRequest{
		ServiceSessionId: request.ServiceSessionId,
		ServiceRef:       request.ServiceRef,
		Address:          request.Address,
		Amount:           request.TransactionAmount,
		Asset:            request.Asset,
	}

	// Initiate
	transactions, err := c.InitiatePayment(router, pr)

	if err != nil {
		Respond(500, w, Message("Init failed"))
		return
	}

	// Verify
	ok, err := c.VerifyTransactions(router, pr, transactions)

	if !ok {
		Respond(500, w, Message("Verification failed"))
		return
	}

	// Commit
	ok, err = c.FinalizePayment(router, transactions, pr)

	if !ok {
		Respond(500, w, Message("Finalize failed"))
		return
	}
}

