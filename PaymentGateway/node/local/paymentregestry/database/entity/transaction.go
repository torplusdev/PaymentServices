package entity

type DbTransactoin struct {
	Id                        int
	Sequence                  int64
	TransactionSourceAddress  string
	ReferenceAmountIn         int
	AmountOut                 int
	XDR                       string
	PaymentSourceAddress      string
	PaymentDestinationAddress string
	StellarNetworkToken       string
	ServiceSessionId          string
}
