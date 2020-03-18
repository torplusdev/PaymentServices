package proxy

import (
	"bytes"
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"net/http"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/utilityService"
	"strconv"
)

type NodeProxy struct {
	id string
	torUrl string
	commandChannel map[string]chan string
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

func (n NodeProxy) AddPendingServicePayment(serviceSessionId string, amount common.TransactionAmount) {
	panic("implement me")
}

func (n NodeProxy) CreatePaymentRequest(serviceSessionId string) (common.PaymentRequest, error) {
	panic("implement me")
}

func (n NodeProxy) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {
	var request = &utilityService.CreateTransactionRequest{
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

	var payload common.PaymentTransactionReplacing

	err = json.Unmarshal([]byte(reply), &payload)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	return payload, nil
}

func (n NodeProxy) SignTerminalTransactions(creditTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
//	var request = &utilityService.SignTerminalTransactionsRequest{
//		Transaction:
//	}
	panic("implement me")
}

func (n NodeProxy) SignChainTransactions(creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) *errors.Error {
	var request = &utilityService.SignChainTransactionsRequest{
		Debit:  *debitTransactionPayload,
		Credit: *creditTransactionPayload,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	_, err = n.ProcessCommand(2,  string(body))

	if err != nil {
		return errors.Errorf(err.Error())
	}

	return nil
}

func (n NodeProxy) CommitServiceTransaction(transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {
	panic("implement me")
}

func (n NodeProxy) CommitPaymentTransaction(transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {
	var request = &utilityService.CommitPaymentTransactionRequest {
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

	var payload utilityService.CommitPaymentTransactionResponse

	err = json.Unmarshal([]byte(reply), &payload)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return payload.Ok, nil
}



