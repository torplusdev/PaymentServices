package sqlite

import (
	log "paidpiper.com/payment-gateway/log"

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
		ServiceSessionId          	TEXT  NOT NULL,
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

func (prdb *liteDb) SelectTransaction() ([]*entity.DbTransaction, error) {
	query := `
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
		FROM Transactoin;
	`

	res, err := prdb.db.Query(query)
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
			Date:                      date,
		}
		items = append(items, item)
	}
	return items, nil
}
