package sqlite

import (
	"log"

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
		ServiceSessionId          	TEXT  NOT NULL
	)
`)
}

func (prdb *liteDb) InsertTransaction(item *entity.DbTransactoin) error {
	tx, err := prdb.db.Begin()
	if err != nil {
		log.Fatal(err)
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
		ServiceSessionId
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
	);

`)
	if err != nil {
		log.Fatal(err)
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
	)
	if err != nil {
		return err
	}
	return tx.Commit()

}

func (prdb *liteDb) SelectTransaction() ([]*entity.DbTransactoin, error) {
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
			ServiceSessionId
		FROM Transactoin;
	`

	res, err := prdb.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var items []*entity.DbTransactoin
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
		)
		if err != nil {
			return nil, err
		}
		item := &entity.DbTransactoin{
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
		}
		items = append(items, item)
	}
	return items, nil
}
