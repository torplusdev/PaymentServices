package sqlite

import (
	"database/sql"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

func New() (*liteDb, error) {
	regestryDb := &liteDb{}
	return regestryDb, regestryDb.init()
}

//TODO CLOSE METHOD
type liteDb struct {
	mutext sync.Mutex
	db     *sql.DB
}

func (prdb *liteDb) init() error {
	prdb.mutext.Lock()
	defer prdb.mutext.Unlock()

	err := prdb.Open()
	if err != nil {
		return err
	}
	err = prdb.createTables()
	if err != nil {
		return err
	}
	err = prdb.Close()
	if err != nil {
		return err
	}
	return nil
}

func (prdb *liteDb) createTables() error {
	err := prdb.createTablePaymentRequest()
	if err != nil {
		return err
	}
	err = prdb.createTableTransaction()
	if err != nil {
		return err
	}
	return nil
}

func (prdb *liteDb) exec(sql string) error {
	_, err := prdb.db.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func (prdb *liteDb) Open() error {
	db, err := sql.Open("sqlite3", "./db.db") // "file:locked.sqlite?cache=shared")
	if err != nil {
		return err
	}

	prdb.db = db
	return nil
}

func (prdb *liteDb) Close() error {
	if prdb.db != nil {
		prdb.db.Close()
	}
	return nil
}
