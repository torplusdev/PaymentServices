package common

type TransactionAmount = uint32

type PaymentRequest struct {
	ServiceSessionId string
	ServiceRef string
	Address string
	Amount TransactionAmount
	Asset string
}

type PaymentTransaction struct {
	TransactionSourceAddress  string
	ReferenceAmountIn         TransactionAmount
	AmountOut                 TransactionAmount
	XDR                       string
	PaymentSourceAddress	  string
	PaymentDestinationAddress string
	StellarNetworkToken       string
}

type PaymentNode struct {
	Address string
	Fee TransactionAmount
}

type PaymentRouter interface {
	 CreatePaymentRoute(req PaymentRequest) []PaymentNode
	 GetNodeByAddress( address string) (PaymentNode,error)
}




