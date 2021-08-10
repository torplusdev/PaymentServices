package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"paidpiper.com/payment-gateway/log"

	"github.com/gorilla/mux"
	"paidpiper.com/payment-gateway/common"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local"
)

type HttpUtilityController struct {
	local.LocalPPNode
}

func NewHttpUtilityController(n local.LocalPPNode) *HttpUtilityController {
	return &HttpUtilityController{
		n,
	}
}

func (u *HttpUtilityController) ListTransactions(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r, "requesthandler:ListTransactions")
	defer span.End()

	trx := u.GetTransactions()

	Respond(w, trx)
}

func (u *HttpUtilityController) HttpGetTransactionInfo(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r, "requesthandler:GetTransactionInfo")
	defer span.End()

	trx := u.GetTransactionInfo()

	Respond(w, trx)
}
func (u *HttpUtilityController) HttpGetTransaction(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r, "requesthandler:GetTransaction")
	defer span.End()

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	trx := u.GetTransaction(sessionId)

	Respond(w, trx)
}

func (u *HttpUtilityController) HttpFlushTransactions(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r, "requesthandler:FlushTransactions")
	defer span.End()
	err := u.FlushTransactions(ctx)
	if err != nil {
		log.Errorf("Error flushing transactions: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Error in FlushTransactions: "+err.Error()))
	}

	Respond(w, MessageWithStatus(http.StatusOK, "Transactions committed"))
}

func (u *HttpUtilityController) HttpGetStellarAddress(w http.ResponseWriter, r *http.Request) {
	response := u.GetStellarAddress()
	Respond(w, response)
}

func (u *HttpUtilityController) HttpNewPaymentRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "requesthandler:HttpNewPaymentRequest")

	defer span.End()

	request := &models.CreatePaymentInfo{}
	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		log.Errorf("Error decoding new payment request: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	pr, err := u.NewPaymentRequest(ctx, request)

	if err != nil {
		log.Errorf("Error creating new payment request (invalid commodity): %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid commodity"))
		return
	}
	Respond(w, pr)
}

func (u *HttpUtilityController) HttpValidatePayment(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r, "ValidatePayment")
	defer span.End()

	request := &models.ValidatePaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		log.Error("Error decoding payment validation request : %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Bad request"))
		return
	}

	response, err := u.ValidatePayment(ctx, request)

	if err != nil {
		log.Errorf("Error validating payment: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, err.Error()))
		return
	}

	Respond(w, response)
}

func (u *HttpUtilityController) HttpProcessCommand(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "requesthandler:ProcessCommand")
	defer span.End()

	command := &models.UtilityCommand{}

	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		log.Errorf("Error decoding process command request : %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	data, err := u.ProcessCommand(ctx, command)
	if err != nil {
		log.Errorf("Error processing command request : %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusConflict, err.Error()))
		return
	}
	if data != nil {
		log.Info("Process Command result success")
		Respond(w, MessageWithData(http.StatusOK, data))
		return
	}
	Respond(w, MessageWithStatus(http.StatusCreated, "success"))
}

func (u *HttpUtilityController) HttpGetBalance(w http.ResponseWriter, r *http.Request) {
	res, err := u.GetBookBalance()
	if err != nil {
		log.Errorf("Error processing get balance request : %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusConflict, err.Error()))
		return
	}
	response := &models.GetBalanceResponse{
		Balance:   res.Balance,
		Timestamp: res.Timestamp,
	}

	Respond(w, response)
}

func (u *HttpUtilityController) HttpBookHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commodity := vars["commodity"]
	hours := vars["hours"]
	bins := vars["bins"]
	binsValue, err := strconv.Atoi(bins)

	if err != nil {
		log.Errorf("Error - bad value for bins : %s", vars["bins"])
		Respond(w, common.Error(500, "HISTORY_BINS should be int"))
	}

	hoursValue, err := strconv.Atoi(hours)
	if err != nil {
		log.Errorf("Error - bad value for hours : %s", vars["hours"])
		Respond(w, common.Error(500, "hours should be int"))
	}
	res, err := u.GetBookHistory(commodity, binsValue, hoursValue)

	if err != nil {
		log.Errorf("Error retrieving book history: %s", vars["hours"])
		Respond(w, common.Error(500, err.Error()))
	}
	Respond(w, res)

}

func (u *HttpUtilityController) HttpBookTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	limits := vars["limits"]

	limitsValue, err := strconv.Atoi(limits)
	if err != nil {
		log.Errorf("Error - bad value for limits : %s", vars["limits"])
		Respond(w, common.Error(500, "limits should be int"))
	}

	direction := vars["direction"]
	if !common.IsDirectionValid(direction) {
		log.Errorf("Error - bad value for direction : %s", vars["direction"])
		Respond(w, common.Error(500, "direction should be 'credit' or 'debit'"))
	}
	direct, err := common.GetDirectionByString(direction)

	res, err := u.GetTransactionHistory(direct, limitsValue)
	if err != nil {
		log.Errorf("Error retrieving transaction history: %s", vars["limits"])
		Respond(w, common.Error(500, err.Error()))
	}

	Respond(w, res)
}

func (u *HttpUtilityController) HttpBookTransactionsGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hours := vars["hours"]

	hoursValue, err := strconv.Atoi(hours)
	if err != nil {
		log.Errorf("Error - bad value for hours : %s", vars["hours"])
		Respond(w, common.Error(400, "hours should be int"))
	}

	res, err := u.GetTransactionHistoryGroup(hoursValue)
	if err != nil {
		log.Errorf("Error retrieving transaction history: %s", vars["hours"])
		Respond(w, common.Error(500, err.Error()))
	}

	Respond(w, res)
}

func (u *HttpUtilityController) HttpBookBalance(w http.ResponseWriter, r *http.Request) {
	res, err := u.GetBookBalance()
	if err != nil {
		log.Errorf("Error retrieving book balance: %s", err.Error())
		Respond(w, common.Error(500, err.Error()))
	}
	Respond(w, res)
}
