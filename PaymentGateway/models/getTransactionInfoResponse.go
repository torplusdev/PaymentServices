package models

import "time"

type GetTransactionInfoResponse struct {
	TotalPending TransactionAmount
	LastSyncTimestamp time.Time
}
