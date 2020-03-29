package controllers

import (
	"encoding/json"
	"net/http"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
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

func (u *UtilityController) CreatePaymentRequest(commandBody string) (interface{}, error) {
	request := &models.CreatePaymentRequestCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	pr, err := u.Node.CreatePaymentRequest(request.ServiceSessionId)

	if err != nil {
		return nil, err
	}

	response := &models.CreatePaymentRequestResponse{
		PaymentRequest: pr,
	}

	return response, nil
}

func (u *UtilityController) GetStellarAddress(w http.ResponseWriter, r *http.Request) {
	response := &models.GetStellarAddressResponse {
		Address: u.Node.Address,
	}

	RespondObject(w, response)
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {
	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		Respond(500, w, Message("Invalid request"))
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
	case 5:
		reply, err = u.CreatePaymentRequest(command.CommandBody)
	}

	if err != nil {
		Respond(500, w, Message("Request process failed"))
		return
	}

	RespondObject(w, reply)
}