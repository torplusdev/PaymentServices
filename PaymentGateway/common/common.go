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

type PaymentTransactionPayload interface {
	GetPaymentTransaction() PaymentTransaction
	GetPaymentDestinationAddress() string
	UpdateTransactionXDR(xdr string) error
	UpdateStellarToken(token string) error
	Validate() error
}

type PaymentNode struct {
	Address string
	Fee TransactionAmount
}

type PaymentRouter interface {
	 CreatePaymentRoute(req PaymentRequest) []PaymentNode
	 GetNodeByAddress( address string) (PaymentNode,error)
}




