package database

import (
	"database/sql"
	"testing"
	"time"

	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

func TestInsertPaymentRequest(t *testing.T) {
	db, err := NewLiteDB()

	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	serviceRef := "ServiceRef"
	base := &entity.DbPaymentRequest{
		Id:               0,
		SessionId:        "SessionId3",
		Amount:           5,
		Asset:            "Asset",
		ServiceRef:       serviceRef,
		ServiceSessionId: "ServiceSessionId",
		Address:          "Address",
		Date:             time.Date(2021, 6, 6, 6, 6, 6, 6, time.UTC),
		CompleteDate:     sql.NullTime{},
	}
	err = db.InsertPaymentRequest(base)
	if err != nil {
		t.Error(err)
	}
}
func TestUpdatePaymentRequestCompleteDate(t *testing.T) {
	TestInsertPaymentRequest(t)
	db, err := NewLiteDB()
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
		return
	}
	if len(items) == 0 {
		t.Errorf("PaymentRequest count is 0")
		return
	}
	item := items[0]
	now := time.Now()
	err = db.UpdatePaymentRequestCompleteDate(item.SessionId, now)
	if err != nil {
		t.Error(err)
		return
	}
	dbItem, err := db.SelectPaymentRequestById(item.Id)
	if err != nil {
		t.Error(err)
		return
	}
	if !dbItem.CompleteDate.Time.Equal(now) {
		t.Error("time not equal")
		return
	}
}
func TestSelectPaymentRequestGroup(t *testing.T) {

	db, err := NewLiteDB()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	serviceRef := "ServiceRef"
	from := time.Now().Add(-time.Hour)
	trs, err := db.SelectPaymentRequestGroup(serviceRef, time.Hour, from)
	if len(trs) == 0 {
		t.Error(err)
	}
}

func TestInsertTransaction(t *testing.T) {
	db, err := NewLiteDB()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	base := &entity.DbTransaction{
		Sequence:                  2,
		TransactionSourceAddress:  "TransactionSourceAddress",
		ReferenceAmountIn:         100,
		AmountOut:                 50,
		XDR:                       "XDR",
		PaymentSourceAddress:      "PaymentSourceAddress",
		PaymentDestinationAddress: "PaymentDestinationAddress",
		StellarNetworkToken:       "StellarNetworkToken",
		ServiceSessionId:          "ServiceSessionId",
	}
	err = db.InsertTransaction(base)
	if err != nil {
		t.Error(err)
	}

}
func TestSelectTransaction(t *testing.T) {
	TestInsertTransaction(t)
	db, err := NewLiteDB()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	trs, err := db.SelectTransaction(10)
	if err != nil {
		t.Error(err)
	}
	if len(trs) == 0 {
		t.Errorf("transactions count is zero")
		return
	}
}

func TestSelectTransactionGroup(t *testing.T) {
	TestInsertTransaction(t)
	db, err := NewLiteDB()
	if err != nil {
		t.Error(err)
	}
	err = db.Open()
	if err != nil {
		t.Error(err)
	}
	dateFrom := time.Now().Add(-time.Hour * 24)
	trs, err := db.SelectTransactionGroup(dateFrom, "GDZXFNXSJGHKFNQJZASNHYS2GPISB4MXIT4OGFBIRCSA2N6C2XKECALP")
	if err != nil {
		t.Error(err)
	}
	if len(trs) == 0 {
		t.Errorf("group transactions count is zero")
		return
	}
}

func TestFormat(t *testing.T) {
	tim, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", "2021-06-06 06:06:06.000000006+00:00")
	if err != nil {
		t.Error(err)
	}
	t.Log(tim)

}
