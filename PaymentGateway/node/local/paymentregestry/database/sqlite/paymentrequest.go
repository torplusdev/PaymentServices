package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/dbtime"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

func (prdb *liteDb) createTablePaymentRequest() error {
	return prdb.exec(`
	CREATE TABLE IF NOT EXISTS PaymentRequest (
		Id 					INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		SessionId 			TEXT NOT NULL,
		Amount    			INTEGER NOT NULL,
		Asset     			TEXT NOT NULL,
		ServiceRef       	TEXT NOT NULL,
		ServiceSessionId 	TEXT NOT NULL,
		Address          	TEXT NOT NULL,
		Date	    		LONG NOT NULL,
		CompleteDate 		LONG NULL
	)
	`)
}

func (prdb *liteDb) InsertPaymentRequest(item *entity.DbPaymentRequest) error {
	tx, err := prdb.db.Begin()
	if err != nil {
		log.Error(err)
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO PaymentRequest (
		SessionId,
		Amount,
		Asset,
		ServiceRef,
		ServiceSessionId,
		Address,
		Date,
		CompleteDate
	)
	VALUES (
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
		item.SessionId,
		item.Amount,
		item.Asset,
		item.ServiceRef,
		item.ServiceSessionId,
		item.Address,
		item.Date,
		item.CompleteDate,
	)

	if err != nil {
		return err
	}
	return tx.Commit()

}

func (prdb *liteDb) UpdatePaymentRequestCompleteDate(sessionId string, time time.Time) error {
	query := `UPDATE PaymentRequest set CompleteDate=?
	WHERE SessionId=?;
	`

	stmt, err := prdb.db.Prepare(query)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(time, sessionId)
	return err
}

func (prdb *liteDb) SelectPaymentRequestGroup(comodity string, group time.Duration, where time.Time) ([]*models.BookHistoryItem, error) {
	//select CAST((julianday('now') - 2440587.5)*86400.0  AS INT)/(60*60*24) as Timespan
	query := `
		SELECT Asset,
			MIN(Date) as Date,
			SUM(Amount)
		FROM PaymentRequest 
		WHERE Date> '%v' AND Date<= '%v' AND ServiceRef='%v' 
		GROUP BY Asset, CAST((julianday(Date) - 2440587.5)*86400.0  AS INT)/(%d);
	`
	/*
		datetime(((CAST((julianday(Date) - 2440587.5)*86400.0  AS INT)/(21600))*21600),'unixepoch') as Date,
	*/
	//TODO ADD ServiceRef='%v'
	//MIN(Date),
	now := time.Now()

	groupSeconds := int(group.Seconds())
	q := fmt.Sprintf(query, dbtime.SqlTime(where).String(), dbtime.SqlTime(now).String(), comodity, groupSeconds)
	res, err := prdb.db.Query(q)
	if err != nil {
		return nil, err
	}

	defer res.Close()
	var items []*models.BookHistoryItem
	for res.Next() {

		var asset string
		var date dbtime.SqlTime
		var sum int

		err := res.Scan(
			&asset,
			&date,
			&sum,
		)
		if err != nil {
			return nil, err
		}

		item := &models.BookHistoryItem{
			Date:   time.Time(date),
			Volume: int64(sum),
		}
		items = append(items, item)
	}
	return items, nil
}
func (prdb *liteDb) SelectPaymentRequestById(id int) (*entity.DbPaymentRequest, error) {
	query := `SELECT Id,
					SessionId,
					Amount,
					Asset,
					ServiceRef,
					ServiceSessionId,
					Address,
					Date,
					CompleteDate
				FROM PaymentRequest where Id=%d;
	`
	res, err := prdb.db.Query(fmt.Sprintf(query, id))
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if res.Next() {
		var id int
		var sessionId string
		var amount int
		var asset string
		var serviceRef string
		var serviceSessionId string
		var address string
		var date dbtime.SqlTime
		var completeDate dbtime.NullSqlTime

		err := res.Scan(
			&id,
			&sessionId,
			&amount,
			&asset,
			&serviceRef,
			&serviceSessionId,
			&address,
			&date,
			&completeDate,
		)
		if err != nil {
			return nil, err
		}
		return &entity.DbPaymentRequest{
			Id:               id,
			SessionId:        sessionId,
			Amount:           amount,
			Asset:            asset,
			ServiceRef:       serviceRef,
			ServiceSessionId: serviceSessionId,
			Address:          address,
			Date:             time.Time(date),
			CompleteDate:     sql.NullTime(completeDate),
		}, nil
	}
	return nil, fmt.Errorf("not found")

}
func (prdb *liteDb) SelectPaymentRequest() ([]*entity.DbPaymentRequest, error) {
	query := `SELECT Id,
					SessionId,
					Amount,
					Asset,
					ServiceRef,
					ServiceSessionId,
					Address,
					Date,
					CompleteDate
				FROM PaymentRequest;
	`
	res, err := prdb.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var items []*entity.DbPaymentRequest
	for res.Next() {
		var id int
		var sessionId string
		var amount int
		var asset string
		var serviceRef string
		var serviceSessionId string
		var address string
		var date dbtime.SqlTime
		var completeDate dbtime.NullSqlTime

		err := res.Scan(
			&id,
			&sessionId,
			&amount,
			&asset,
			&serviceRef,
			&serviceSessionId,
			&address,
			&date,
			&completeDate,
		)
		if err != nil {
			return nil, err
		}
		item := &entity.DbPaymentRequest{
			Id:               id,
			SessionId:        sessionId,
			Amount:           amount,
			Asset:            asset,
			ServiceRef:       serviceRef,
			ServiceSessionId: serviceSessionId,
			Address:          address,
			Date:             time.Time(date),
			CompleteDate:     sql.NullTime(completeDate),
		}
		items = append(items, item)
	}
	return items, nil
}
