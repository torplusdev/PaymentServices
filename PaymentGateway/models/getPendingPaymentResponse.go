package models

import (
	"time"
)

type GetPendingPaymentResponse struct {
	Address        string
	PendingBalance TransactionAmount
	Timestamp      time.Time
}
