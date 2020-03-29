package routing

import (
	"errors"
	"github.com/stellar/go/keypair"
	"paidpiper.com/payment-gateway/common"
)

type PaymentRouter struct {
	nodes []common.PaymentNode
}

func CreatePaymentRouterStub(nodes []common.PaymentNode) PaymentRouter {
	stub := PaymentRouter{ }
	stub.nodes = nodes

	return stub
}

func CreatePaymentRouterStubFromAddresses(addresses []string) PaymentRouter {
	stub := PaymentRouter{ }

	for _,e := range addresses {
		kp,_ := keypair.ParseAddress(e)
		stub.nodes = append(stub.nodes,common.PaymentNode{ Address: kp.Address(), Fee:10})
	}

	return stub
}

func (router PaymentRouter) CreatePaymentRoute(req common.PaymentRequest) []common.PaymentNode {
	copyOfNodes := append([]common.PaymentNode(nil), router.nodes...)
	return copyOfNodes
}

func (router PaymentRouter) GetNodeByAddress( address string) (common.PaymentNode,error) {

	for _,n := range router.nodes {
		if n.Address == address {
			return n,nil
		}
	}

	return common.PaymentNode{},errors.New("non-existent address")
}