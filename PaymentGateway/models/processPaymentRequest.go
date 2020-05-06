package models

type ProcessPaymentRequest struct {
	Route				[]RoutingNode

	CallbackUrl			string 		// process command url

	PaymentRequest		string 		// json body

	NodeId				string		// request reference identification
}
