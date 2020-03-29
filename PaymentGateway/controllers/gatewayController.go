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
	NodeManager *proxy.NodeManager
	Seed		*keypair.Full
}


func (g *GatewayController) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		Respond(500, w, Message("Invalid request"))
		return
	}

	pNode := g.NodeManager.GetProxyNode(response.NodeId)

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

	c := client.CreateClient(rootApi, g.Seed.Seed(), g.NodeManager)

	addr := make([]string, 0)

	addr = append(addr, g.Seed.Address())

	for _, a := range request.RouteAddresses {
		addr = append(addr, a)
	}

	addr = append(addr, request.Address)

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

