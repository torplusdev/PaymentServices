package database

import (
	"time"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/sqlite"
)

func NewLiteDB() (Db, error) { //TODO FILE NAME or test or prod
	db, err := sqlite.New()
	if err != nil {
		return nil, err
	}
	return db, nil
}

type Db interface {
	Open() error
	Close() error
	InsertPaymentRequest(item *entity.DbPaymentRequest) error
	UpdatePaymentRequestCompleteDate(sessionId string, time time.Time) error
	SelectPaymentRequest() ([]*entity.DbPaymentRequest, error)
	SelectPaymentRequestById(id int) (*entity.DbPaymentRequest, error)
	InsertTransaction(item *entity.DbTransaction) error
	SelectTransaction(limits int) ([]*entity.DbTransaction, error)
	SelectTransactionGroup(dateFrom time.Time) ([]*models.BookTransactionItem, error)

	SelectPaymentRequestGroup(comodity string, group time.Duration, where time.Time) ([]*models.BookHistoryItem, error)
}
