package sqlite

import (
	"fmt"
	"strings"
	"time"

	log "paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/dbtime"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

func (prdb *liteDb) createTableTransaction() error {
	return prdb.exec(`
	CREATE TABLE IF NOT EXISTS Transactoin (
		Id 							INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		Sequence                  	INTEGER NOT NULL,
		TransactionSourceAddress  	TEXT NOT NULL,
		ReferenceAmountIn         	INTEGER NOT NULL,
		AmountOut                 	INTEGER NOT NULL,
		XDR                       	TEXT NOT NULL,
		PaymentSourceAddress      	TEXT NOT NULL,
		PaymentDestinationAddress 	TEXT NOT NULL,
		StellarNetworkToken       	TEXT NOT NULL,
		ServiceSessionId          	TEXT NOT NULL,
		Date                        LONG NOT NULL
	)
`)
}

func (prdb *liteDb) InsertTransaction(item *entity.DbTransaction) error {
	tx, err := prdb.db.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO Transactoin (
		Sequence,
		TransactionSourceAddress,
		ReferenceAmountIn,
		AmountOut,
		XDR,
		PaymentSourceAddress,
		PaymentDestinationAddress,
		StellarNetworkToken,
		ServiceSessionId,
		Date
	)
	VALUES (
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?,
		?
	);

`)
	if err != nil {
		log.Error(err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(
		item.Sequence,
		item.TransactionSourceAddress,
		item.ReferenceAmountIn,
		item.AmountOut,
		item.XDR,
		item.PaymentSourceAddress,
		item.PaymentDestinationAddress,
		item.StellarNetworkToken,
		item.ServiceSessionId,
		item.Date,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (prdb *liteDb) SelectTransaction(limits int) ([]*entity.DbTransaction, error) {
	var query strings.Builder

	query.WriteString(`
		SELECT Id,
			Sequence,
			TransactionSourceAddress,
			ReferenceAmountIn,
			AmountOut,
			XDR,
			PaymentSourceAddress,
			PaymentDestinationAddress,
			StellarNetworkToken,
			ServiceSessionId,
			Date
		FROM Transactoin
	`)

	if limits == 0 {
		limits = 1000
	}
	query.WriteString(fmt.Sprintf("LIMIT %v;", limits))

	queryStr := query.String()
	// fmt.Println(queryStr)
	res, err := prdb.db.Query(queryStr)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var items []*entity.DbTransaction
	for res.Next() {
		var id int
		var sequence int64
		var transactionSourceAddress string
		var referenceAmountIn int
		var amountOut int
		var xdr string
		var paymentSourceAddress string
		var paymentDestinationAddress string
		var stellarNetworkToken string
		var serviceSessionId string
		var date dbtime.SqlTime
		err := res.Scan(
			&id,
			&sequence,
			&transactionSourceAddress,
			&referenceAmountIn,
			&amountOut,
			&xdr,
			&paymentSourceAddress,
			&paymentDestinationAddress,
			&stellarNetworkToken,
			&serviceSessionId,
			&date,
		)
		if err != nil {
			return nil, err
		}
		item := &entity.DbTransaction{
			Id:                        id,
			Sequence:                  sequence,
			TransactionSourceAddress:  transactionSourceAddress,
			ReferenceAmountIn:         referenceAmountIn,
			AmountOut:                 amountOut,
			XDR:                       xdr,
			PaymentSourceAddress:      paymentSourceAddress,
			PaymentDestinationAddress: paymentDestinationAddress,
			StellarNetworkToken:       stellarNetworkToken,
			ServiceSessionId:          serviceSessionId,
			Date:                      time.Time(date),
		}
		items = append(items, item)
	}
	return items, nil
}

func (prdb *liteDb) SelectTransactionGroup(dateFrom time.Time, currentAddress string) ([]*models.BookTransactionItem, error) {
	var query strings.Builder
	query.WriteString(`
		SELECT 
			PaymentSourceAddress, 
			PaymentDestinationAddress, 
			MIN(Date) as Date,
			SUM(AmountOut) as Amount
		FROM Transactoin 
		WHERE Date>'%[1]v'
		AND Date<='%[2]v'
		AND PaymentSourceAddress = '%[3]v'
		GROUP BY %[4]v
		UNION
		SELECT PaymentSourceAddress,
			PaymentDestinationAddress,
			MIN(Date) as Date,
			SUM(AmountOut) as Amount
		FROM Transactoin
		WHERE Date>'%[1]v'
		AND Date<='%[2]v'
		AND PaymentDestinationAddress = '%[3]v'
		GROUP BY %[4]v
		ORDER BY Date
		`)

	now := time.Now()
	dateFromStr := dbtime.SqlTime(dateFrom).String()
	dateToStr := dbtime.SqlTime(now).String()

	q := fmt.Sprintf(query.String(),
		dateFromStr,
		dateToStr,
		currentAddress,
		"strftime('%H', Date)")

	fmt.Println(q)
	res, err := prdb.db.Query(q)
	if err != nil {
		return nil, err
	}

	defer res.Close()
	var items []*models.BookTransactionItem
	for res.Next() {
		var source string
		var target string
		var date dbtime.SqlTime
		var amount int

		err := res.Scan(
			&source,
			&target,
			&date,
			&amount,
		)
		if err != nil {
			return nil, err
		}

		item := &models.BookTransactionItem{
			SourceAddress: source,
			TargetAddress: target,
			Timestamp:     time.Time(date),
			Value:         int64(amount),
		}
		items = append(items, item)
	}
	fmt.Println(items)
	return items, nil
}

func (prdb *liteDb) SelectBookTransactionItems(limits int, direction string, currentAddress string) ([]*models.BookTransactionItem, error) {
	var query strings.Builder

	query.WriteString(`
		SELECT 
			PaymentSourceAddress, 
			PaymentDestinationAddress, 
			Date,
			AmountOut as Amount
		FROM Transactoin `)

	if strings.EqualFold(direction, "credit") {
		query.WriteString(fmt.Sprintf(`WHERE PaymentDestinationAddress = '%v'`, currentAddress))
	} else {
		query.WriteString(fmt.Sprintf(`WHERE PaymentSourceAddress = '%v'`, currentAddress))
	}

	if limits == 0 {
		limits = 1000
	}
	query.WriteString(fmt.Sprintf("LIMIT %v;", limits))

	q := query.String()

	fmt.Println(q)
	res, err := prdb.db.Query(q)
	if err != nil {
		return nil, err
	}

	defer res.Close()
	var items []*models.BookTransactionItem
	for res.Next() {
		var source string
		var target string
		var date dbtime.SqlTime
		var amount int

		err := res.Scan(
			&source,
			&target,
			&date,
			&amount,
		)
		if err != nil {
			return nil, err
		}

		item := &models.BookTransactionItem{
			SourceAddress: source,
			TargetAddress: target,
			Timestamp:     time.Time(date),
			Value:         int64(amount),
		}
		items = append(items, item)
	}
	fmt.Println(items)
	return items, nil
}
