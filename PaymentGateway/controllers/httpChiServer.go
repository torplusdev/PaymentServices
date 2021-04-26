package controllers

import (
	"net/http"

	"paidpiper.com/payment-gateway/common"
	chi_server "paidpiper.com/payment-gateway/http/server"
)

//# chi implemetation

func (u *HttpUtilityController) GetBalance(w http.ResponseWriter, r *http.Request) {
	bb, err := u.GetBookBalance()
	if err != nil {
		Respond(w, common.Error(500, err.Error()))
	}
	var timestamp int64 = bb.Timestamp.Time().UnixNano() / 1000000
	Respond(w, &chi_server.Balance{
		Balance:   &bb.Balance,
		Timestamp: &timestamp,
	})
}

// Get history
// (GET /book/history/{commodity}/{hours}/{bins})
func (u *HttpUtilityController) GetHistory(w http.ResponseWriter, r *http.Request, commodity string, hours int32, bins int32) {
	res, err := u.GetBookHistory(commodity, int(bins), int(hours))

	if err != nil {
		Respond(w, common.Error(500, err.Error()))
	}
	items := []chi_server.HistoryStatisticItem{}
	for _, item := range res.Items {
		items = append(items, chi_server.HistoryStatisticItem{
			Date:   item.Date.UnixNano() / 1000000,
			Volume: &item.Volume,
		})
	}
	Respond(w, &chi_server.HistoryCollection{
		Items: items,
	})
}
