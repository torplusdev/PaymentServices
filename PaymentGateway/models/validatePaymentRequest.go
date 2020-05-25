package models

type ValidatePaymentRequest struct {
	ServiceType   		string
	CommodityType 		string
	PaymentRequest		string 		// json body
}

type ValidatePaymentResponse struct {
	Quantity	uint32
}
