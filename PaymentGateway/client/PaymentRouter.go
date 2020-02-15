package client

import "paidpiper.com/payment-gateway/common"

type TorAwarePaymentRouter struct {

}

func (torRouter TorAwarePaymentRouter) CreatePaymentRoute(req common.PaymentRequest) []common.PaymentNode {
	//TODO: Implement
	return []common.PaymentNode{}
}
