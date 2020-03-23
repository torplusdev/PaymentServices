package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"net/http"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"strconv"
)

type NodeProxy struct {
	id string
	torUrl string
	commandChannel map[string]chan string
}

func (n NodeProxy) ProcessCommandNoReply(commandType int, commandBody string) (error) {
	id := uuid.New().String()

	values := map[string]string{"CommandId": id, "CommandType": strconv.Itoa(commandType), "CommandBody": commandBody, "NodeId": n.id}

	jsonValue, _ := json.Marshal(values)

	_, err := http.Post(n.torUrl, "application/json", bytes.NewBuffer(jsonValue))

	return err
}

func (n NodeProxy) ProcessCommand(commandType int, commandBody string) (string, error) {
	id := uuid.New().String()

	values := map[string]string{"CommandId": id, "CommandType": strconv.Itoa(commandType), "CommandBody": commandBody, "NodeId": n.id}

	jsonValue, _ := json.Marshal(values)

	ch := make(chan string, 2)

	n.commandChannel[id] = ch

	defer delete (n.commandChannel, id)
	defer close (ch)

	_, err := http.Post(n.torUrl, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		return "", err
	}

	// Wait
	responseBody := <- ch

	return responseBody, nil
}

func (n NodeProxy) ProcessResponse(commandId string, responseBody string) {
	n.commandChannel[commandId] <- responseBody
}

func (n NodeProxy) AddPendingServicePayment(serviceSessionId string, amount common.TransactionAmount) error {
	var request = &models.AddPendingServicePaymentCommand{
		ServiceSessionId: serviceSessionId,
		Amount: amount,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return err
	}

	err = n.ProcessCommandNoReply(0, string(body))

	return err
}

func (n NodeProxy) CreatePaymentRequest(serviceSessionId string) (common.PaymentRequest, error) {
	var request = &models.CreatePaymentRequestCommand{
		ServiceSessionId: serviceSessionId,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return common.PaymentRequest{}, err
	}

	reply, err := n.ProcessCommand(0, string(body))

	if err != nil {
		return common.PaymentRequest{}, err
	}

	response := &models.CreatePaymentRequestResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return common.PaymentRequest{}, err
	}

	return response.PaymentRequest, nil
}

func (n NodeProxy) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {
	var request = &models.CreateTransactionCommand{
		TotalIn:       totalIn,
		TotalOut:      totalOut,
		SourceAddress: sourceAddress,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	reply, err := n.ProcessCommand(0, string(body))

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	response := &models.CreateTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	return response.Transaction, nil
}

func (n NodeProxy) SignTerminalTransactions(creditTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	var request = &models.SignTerminalTransactionCommand{
		Transaction: *creditTransactionPayload,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(1,  string(body))

	if err != nil {
		return errors.Errorf(err.Error())
	}

	var response = &models.SignTerminalTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	creditTransactionPayload = &response.Transaction

	return nil
}

func (n NodeProxy) SignChainTransactions(creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	var request = &models.SignChainTransactionsCommand{
		Debit:  *debitTransactionPayload,
		Credit: *creditTransactionPayload,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(2,  string(body))

	if err != nil {
		return errors.Errorf(err.Error())
	}

	var response = &models.SignChainTransactionsResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	creditTransactionPayload = &response.Credit
	debitTransactionPayload = &response.Debit

	return nil
}

func (n NodeProxy) CommitServiceTransaction(transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {
	var request = &models.CommitServiceTransactionCommand {
		Transaction: *transaction,
		PaymentRequest: pr,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(4, string(body))

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	var response = &models.CommitServiceTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return response.Ok, nil
}

func (n NodeProxy) CommitPaymentTransaction(transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {
	var request = &models.CommitPaymentTransactionCommand {
		Transaction: *transactionPayload,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(3, string(body))

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	var response = &models.CommitPaymentTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return response.Ok, nil
}



