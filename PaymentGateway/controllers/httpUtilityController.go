package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	pr, err := u.NewPaymentRequest(ctx, request)

	if err != nil {
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
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Bad request"))
		return
	}

	response, err := u.ValidatePayment(ctx, request)

	if err != nil {
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
		log.Fatal(err)

		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	status, err := u.ProcessCommand(ctx, command)
	if err != nil {
		Respond(w, MessageWithStatus(status, err.Error()))
	} else {
		Respond(w, MessageWithStatus(status, "success"))
	}
}

func (u *HttpUtilityController) HttpGetBalance(w http.ResponseWriter, r *http.Request) {

	response := &models.GetBalanceResponse{
		Balance:   100,
		Timestamp: time.Now(),
	}

	Respond(w, response)
}
