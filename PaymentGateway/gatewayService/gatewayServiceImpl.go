package gatewayService

import (
	"context"
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/client"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	"paidpiper.com/payment-gateway/ppsidechannel"
	"paidpiper.com/payment-gateway/root"
	"paidpiper.com/payment-gateway/routing"
)

type GatewayServiceImpl struct {
 	NodeManager node.NodeManager
 	Seed		*keypair.Full
}

func (g *GatewayServiceImpl) ProcessPayment(ctx context.Context, request *ppsidechannel.PaymentRequest) (*ppsidechannel.PaymentReply, error) {
	rootApi := root.CreateRootApi(true)

	client := client.CreateClient(rootApi, g.Seed.Seed(), g.NodeManager)

	addrs := make([]string, len(request.RouteAddresses) + 2)

	addrs = append(addrs, g.Seed.Address())

	for _, a := range request.RouteAddresses {
		addrs = append(addrs, a)
	}

	addrs = append(addrs, request.Address)

	router := routing.CreatePaymentRouterStubFromAddresses(addrs)

	pr := common.PaymentRequest{
		ServiceSessionId: request.ServiceSessionId,
		ServiceRef:       request.ServiceRef,
		Address:          request.Address,
		Amount:           request.TransactionAmount,
		Asset:            request.Asset,
	}

	// Initiate
	transactions, err := client.InitiatePayment(router, pr)

	if err != nil {
		return nil, err
	}

	// Verify
	ok, err := client.VerifyTransactions(router, pr, transactions)

	if !ok {
		return nil, err
	}

	// Commit
	ok, err = client.FinalizePayment(router, transactions, pr)

	if !ok {
		return nil, err
	}

	return &ppsidechannel.PaymentReply{}, nil
}