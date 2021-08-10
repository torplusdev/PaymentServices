package entity

import (
	"time"
)

type DbTransaction struct {
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
	Date                      time.Time
}
