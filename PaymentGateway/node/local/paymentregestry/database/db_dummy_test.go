package database

import (
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/xid"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

func TestCreateaDummyDb(t *testing.T) {
	now := time.Now()
	timeRange := time.Hour * 24 * 4
	start := now.Add(-timeRange)
	end := now.Add(0)
	current := start
	db, err := NewLiteDB()
	if err != nil {
		t.Error(err)
		return
	}
	min := 10
	max := 30
	err = db.Open()
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()
	for end.Sub(current) > 0 {
		amount := rand.Intn(max-min) + min
		base := &entity.DbPaymentRequest{
			Id:               0,
			SessionId:        xid.New().String(),
			Amount:           amount,
			Asset:            "data",
			ServiceRef:       "ipfs",
			ServiceSessionId: xid.New().String(),
			Address:          xid.New().String(),
			Date:             current,
			CompleteDate:     sql.NullTime{},
		}
		err := db.InsertPaymentRequest(base)
		if err != nil {
			t.Error(err)
		}
		current = current.Add(time.Second * 30)
	}
}
