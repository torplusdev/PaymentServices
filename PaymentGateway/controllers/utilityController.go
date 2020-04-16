package controllers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"strconv"
)

type UtilityController struct {
	Node *node.Node
}

func (u *UtilityController) CreateTransaction(commandBody string) (interface{}, error) {
	request := &models.CreateTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	transaction, err := u.Node.CreateTransaction(request.TotalIn, request.TotalIn-request.TotalOut, request.TotalOut, request.SourceAddress)

	if err != nil {
		return nil, err
	}

	response := &models.CreateTransactionResponse{
		Transaction: transaction,
	}

	return response, nil
}

func (u *UtilityController) SignTerminalTransaction(commandBody string) (interface{}, error) {
	request := &models.SignTerminalTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	err = u.Node.SignTerminalTransactions(&request.Transaction)

	if err != nil {
		return nil, err
	}

	response := models.SignTerminalTransactionResponse{
		Transaction: request.Transaction,
	}

	return response, nil
}

func (u *UtilityController) SignChainTransactions(commandBody string) (interface{}, error) {
	request :=  &models.SignChainTransactionsCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	err = u.Node.SignChainTransactions(&request.Credit, &request.Debit)

	if err != nil {
		return nil, err
	}

	response :=  &models.SignChainTransactionsResponse{
		Debit:  request.Debit,
		Credit: request.Credit,
	}

	return response, nil
}

func (u *UtilityController) CommitServiceTransaction(commandBody string) (interface{}, error) {
	request := &models.CommitServiceTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	ok, err := u.Node.CommitServiceTransaction(&request.Transaction, request.PaymentRequest)

	if err != nil {
		return nil, err
	}

	response := &models.CommitServiceTransactionResponse{
		Ok: ok,
	}

	return response, nil
}

func (u *UtilityController) CommitPaymentTransaction(commandBody string) (interface{}, error) {
	request := &models.CommitPaymentTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	ok, err := u.Node.CommitPaymentTransaction(&request.Transaction)

	if err != nil {
		return nil, err
	}

	response := &models.CommitPaymentTransactionResponse{
		Ok: ok,
	}

	return response, nil
}

func (u *UtilityController) CreatePaymentInfo(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	strAmount := params["amount"]

	serviceSessionId := xid.New().String()

	amount, err := strconv.Atoi(strAmount)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	err = u.Node.AddPendingServicePayment(serviceSessionId, uint32(amount))

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	pr, err := u.Node.CreatePaymentRequest(serviceSessionId)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	Respond(w, pr)
}


func (u *UtilityController) FlushTransactions(w http.ResponseWriter, r *http.Request) {

	results,err := u.Node.FlushTransactions()

	if err != nil {
		Respond(w,MessageWithStatus(http.StatusInternalServerError,"Error in FlushTransactions:..."))
	}

	for k,v := range results {
		switch v.(type) {
			case error:
				log.Printf("Error in transaction for node %s: %w",k,v)
			default:
		}
	}

	Respond(w, MessageWithStatus(http.StatusOK,"Transactions committed"))
}


func (u *UtilityController) GetStellarAddress(w http.ResponseWriter, r *http.Request) {
	response := &models.GetStellarAddressResponse {
		Address: u.Node.Address,
	}

	Respond(w, response)
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {
	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	var reply interface{}

	switch command.CommandType {
	case 0:
		reply, err = u.CreateTransaction(command.CommandBody)
	case 1:
		reply, err = u.SignTerminalTransaction(command.CommandBody)
	case 2:
		reply, err = u.SignChainTransactions(command.CommandBody)
	case 3:
		reply, err = u.CommitPaymentTransaction(command.CommandBody)
	case 4:
		reply, err = u.CommitServiceTransaction(command.CommandBody)
	}

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Request process failed"))
		return
	}

	Respond(w, reply)
}