package entity

import (
	"database/sql"
	"time"
)

type DbPaymentRequest struct {
	Id               int
	SessionId        string
	Amount           int
	Asset            string
	ServiceRef       string
	ServiceSessionId string
	Address          string
	Date             time.Time
	CompleteDate     sql.NullTime
}
