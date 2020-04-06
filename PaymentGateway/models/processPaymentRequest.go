package models

type ProcessPaymentRequest struct {
	RouteAddresses       []string
	ServiceSessionId     string
	ServiceRef           string
	Address              string
	TransactionAmount    uint32
	Asset                string
	CallbackUrl			 string
}