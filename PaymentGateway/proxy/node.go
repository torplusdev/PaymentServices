package proxy

import (
	"context"
	"encoding/json"
	"github.com/go-errors/errors"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/ppsidechannel"
	"paidpiper.com/payment-gateway/utilityService"
)

type NodeProxy struct {
	id string
	client ppsidechannel.PPPaymentUtilityServicesClient
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

	reply, err := n.client.ProcessCommand(context.Background(), &ppsidechannel.CommandRequest {
		CommandType:          0,
		CommandBody:          string(body),
	})

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	var payload common.PaymentTransactionReplacing

	err = json.Unmarshal([]byte(reply.ResponseBody), &payload)

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

	_, err = n.client.ProcessCommand(context.Background(), &ppsidechannel.CommandRequest {
		CommandType:          2,
		CommandBody:          string(body),
	})

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

	reply, err := n.client.ProcessCommand(context.Background(), &ppsidechannel.CommandRequest {
		CommandType:          3,
		CommandBody:          string(body),
	})

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	var payload utilityService.CommitPaymentTransactionResponse

	err = json.Unmarshal([]byte(reply.ResponseBody), &payload)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return payload.Ok, nil
}



