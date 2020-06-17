package models

type ProcessPaymentRequest struct {
	Route []RoutingNode

	CallbackUrl string // Payment command url

	StatusCallbackUrl string // Status callback command url

	PaymentRequest string // json body

	NodeId string // request reference identification
}

type ProcessPaymentAccepted struct {
	SessionId string
}
