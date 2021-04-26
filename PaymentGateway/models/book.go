package models

import (
	"strconv"
	"strings"
	"time"
)

type JsonTime time.Time

func (t JsonTime) MarshalJSON() ([]byte, error) {
	if time.Time.IsZero(time.Time(t)) {
		return []byte(strconv.FormatInt(0, 10)), nil
	}
	return []byte(strconv.FormatInt(time.Time(t).Unix(), 10)), nil
}

func (t *JsonTime) UnmarshalJSON(s []byte) (err error) {
	r := strings.Replace(string(s), `"`, ``, -1)

	q, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	if q == 0 {
		*(*time.Time)(t) = time.Time{}
	} else {
		*(*time.Time)(t) = time.Unix(q/1000, 0)
	}
	return
}

func (t JsonTime) String() string  { return time.Time(t).String() }
func (t JsonTime) Time() time.Time { return time.Time(t) }

type BookBalanceResponse struct {
	Balance   float64
	Timestamp JsonTime
}
type BookHistoryResponse struct {
	Items []*BookHistoryItem
}
type BookHistoryItem struct {
	Date   time.Time
	Volume int64
}

type HistoryItem struct {
	Date   time.Time
	Amount float64
}
