package models

type ProcessPaymentRequest struct {
	Route []RoutingNode

	CallbackUrl string // Payment command url

	StatusCallbackUrl string // Status callback command url

	PaymentRequest *PaymentRequest // json body

	NodeId PeerID // request reference identification
}
