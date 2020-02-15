package common

type TransactionAmount = uint32

type PaymentRequest struct {
	ServiceRef string
	Address string
	Amount TransactionAmount
	Asset string
}

type PaymentTransaction struct {
	TransactionSource string
	ReferenceAmountIn TransactionAmount
	AmountOut         TransactionAmount
	XDR               string
	Address           string
	Seed 			  string
	Network 		  string
}

type PaymentNode struct {
	Address string
	Fee TransactionAmount
	Seed string
}

type PaymentRouter interface {
	 CreatePaymentRoute(req PaymentRequest) []PaymentNode
	 GetNodeByAddress( address string) (PaymentNode,error)
}