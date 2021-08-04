package dbtime

import (
	"database/sql"
	"time"
)

type SqlTime time.Time

var sqlForat = "2006-01-02 15:04:05.999999999Z07:00"

func (t SqlTime) String() string {
	return time.Time(t).Format(sqlForat)
}
func (t *SqlTime) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	switch val := v.(type) {
	case string:
		vt, err := time.Parse(sqlForat, val)
		if err != nil {
			return err
		}
		*t = SqlTime(vt)
	case []byte:
		vt, err := time.Parse(sqlForat, string(val))
		if err != nil {
			return err
		}
		*t = SqlTime(vt)
	}

	return nil
}
func Now() SqlTime {
	return SqlTime(time.Now())
}

type NullSqlTime sql.NullTime

func (t *NullSqlTime) Scan(v interface{}) error {
	// Should be more strictly to check this type.
	if v == nil {
		nullTime := sql.NullTime{
			Valid: false,
		}
		*t = NullSqlTime(nullTime)
	}
	switch val := v.(type) {
	case string:
		vt, err := time.Parse(sqlForat, val)
		if err != nil {
			return err
		}
		nullTime := sql.NullTime{
			Time:  vt,
			Valid: true,
		}
		*t = NullSqlTime(nullTime)
	case []byte:
		vt, err := time.Parse(sqlForat, string(val))
		if err != nil {
			return err
		}
		nullTime := sql.NullTime{
			Time:  vt,
			Valid: true,
		}
		*t = NullSqlTime(nullTime)
	}

	return nil
}
