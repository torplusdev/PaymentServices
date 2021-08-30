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

func TestCreateTransactionDummyDb(t *testing.T) {
	now := time.Now()
	timeRange := time.Hour * 24 * 1
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
	currentAddress := "GDZXFNXSJGHKFNQJZASNHYS2GPISB4MXIT4OGFBIRCSA2N6C2XKECALP"
	for end.Sub(current) > 0 {
		amount := rand.Intn(max-min) + min
		credit := &entity.DbTransaction{
			Id:                        0,
			Sequence:                  0,
			TransactionSourceAddress:  xid.New().String(),
			XDR:                       "-",
			PaymentDestinationAddress: currentAddress,
			PaymentSourceAddress:      xid.New().String(),
			AmountOut:                 amount,
			Date:                      current,
			StellarNetworkToken:       "",
			ServiceSessionId:          xid.New().String(),
		}

		amount = rand.Intn(max-min) + min
		debit := &entity.DbTransaction{
			Id:                        0,
			Sequence:                  0,
			TransactionSourceAddress:  xid.New().String(),
			XDR:                       "-",
			PaymentDestinationAddress: xid.New().String(),
			PaymentSourceAddress:      currentAddress,
			AmountOut:                 amount,
			Date:                      current.Add(time.Second * 30),
			StellarNetworkToken:       "",
			ServiceSessionId:          xid.New().String(),
		}

		err := db.InsertTransaction(credit)
		if err != nil {
			t.Error(err)
		}
		err = db.InsertTransaction(debit)
		if err != nil {
			t.Error(err)
		}

		current = current.Add(time.Second * 90)
	}
}
