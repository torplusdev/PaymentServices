package models

type ProcessPaymentRequest struct {
	RouteAddresses       []string	// stellar addresses

	CallbackUrl			 string 	// process command url

	PaymentRequest		 string 	// json body
}