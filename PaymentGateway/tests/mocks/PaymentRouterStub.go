package mocks

import (
	"errors"
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/common"
)

type PaymentRouterStub struct {
	nodes []common.PaymentNode
}

func CreatePaymentRouterStub(nodes []common.PaymentNode) PaymentRouterStub {
	stub := PaymentRouterStub{ }
	stub.nodes = nodes

	return stub
}

func CreatePaymentRouterStubFromAddresses(addresses []string) PaymentRouterStub {
	stub := PaymentRouterStub{ }

	for _,e := range addresses {
		kp,_ := keypair.ParseFull(e)
		stub.nodes = append(stub.nodes,common.PaymentNode{ Address: kp.Address(), Fee:10})
	}

	return stub
}

func (router PaymentRouterStub) CreatePaymentRoute(req common.PaymentRequest) []common.PaymentNode {
	copyOfNodes := append([]common.PaymentNode(nil), router.nodes...)
	return copyOfNodes
}

func (router PaymentRouterStub) GetNodeByAddress( address string) (common.PaymentNode,error) {

	for _,n := range router.nodes {
		if (n.Address == address) {
			return n,nil
		}
	}

	return common.PaymentNode{},errors.New("Non-existent address")
}