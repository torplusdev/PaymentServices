package sqlite

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

func TestInsert(t *testing.T) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	db, err := New()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	err = db.InsertPaymentRequest(&entity.DbPaymentRequest{
		SessionId:        "SessionId",
		Amount:           5,
		Asset:            "Asset",
		ServiceRef:       "ServiceRef",
		ServiceSessionId: "ServiceSessionId",
		Address:          "Address",
		Date:             time.Now(),
		CompleteDate:     sql.NullTime{Valid: false},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSelect(t *testing.T) {
	db, err := New()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	items, err := db.SelectPaymentRequest()
	if err != nil {
		t.Error(err)
	}
	if len(items) == 0 {
		t.Errorf("Select count is null")
	}
}
