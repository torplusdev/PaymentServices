package controllers

import (
	"github.com/stellar/go/keypair"
	"net/http"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/routing"
)

type GatewayController struct {
	NodeManager node.NodeManager
	Seed		*keypair.Full
}

func (g *GatewayController) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	request := &models.ProcessPaymentRequest{}

	rootApi := root.CreateRootApi(true)

	c := client.CreateClient(rootApi, g.Seed.Seed(), g.NodeManager)

	addr := make([]string, len(request.RouteAddresses) + 2)

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
		Respond(w, Message(false, "Init failed"))
		return
	}

	// Verify
	ok, err := c.VerifyTransactions(router, pr, transactions)

	if !ok {
		Respond(w, Message(false, "Verification failed"))
		return
	}

	// Commit
	ok, err = c.FinalizePayment(router, transactions, pr)

	if !ok {
		Respond(w, Message(false, "Finalize failed"))
		return
	}
}

