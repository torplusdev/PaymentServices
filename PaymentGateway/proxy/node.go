package proxy

import (
	"context"
	"encoding/json"
	"github.com/go-errors/errors"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/utilityService"
	"paidpiper.com/payment-gateway/ppsidechannel"
)

type NodeProxy struct {
	id string
	client ppsidechannel.PPPaymentUtilityServicesClient
}

func (n NodeProxy) AddPendingServicePayment(serviceSessionId string, amount common.TransactionAmount) {
	panic("implement me")
}

func (n NodeProxy) SetAccumulatingTransactionsMode(accumulateTransactions bool) {
	panic("implement me")
}

func (n NodeProxy) CreatePaymentRequest(serviceSessionId string) (common.PaymentRequest, error) {
	panic("implement me")
}

func (n NodeProxy) CreateTransaction(totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) common.PaymentTransactionPayload {
	var request = &utilityService.CreateTransactionRequest{
		TotalIn:       totalIn,
		TotalOut:      totalOut,
		SourceAddress: sourceAddress,
	}

	body, err := json.Marshal(request)

	if err != nil {

	}

	reply, err := n.client.ProcessCommand(context.Background(), &ppsidechannel.CommandRequest {
		CommandType:          0,
		CommandBody:          string(body),
	})

	var payload common.PaymentTransactionPayload

	err = json.Unmarshal([]byte(reply.ResponseBody), &payload)

	if err != nil {

	}

	return payload
}

func (n NodeProxy) SignTerminalTransactions(creditTransactionPayload common.PaymentTransactionPayload) *errors.Error {
//	var request = &utilityService.SignTerminalTransactionsRequest{
//		Transaction:
//	}
	panic("implement me")
}

func (n NodeProxy) SignChainTransactions(creditTransactionPayload common.PaymentTransactionPayload, debitTransactionPayload common.PaymentTransactionPayload) *errors.Error {
	var request = &utilityService.SignChainTransactionsRequest{
//		Debit:  debitTransactionPayload,
//		Credit: creditTransactionPayload,
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

func (n NodeProxy) CommitServiceTransaction(transaction common.PaymentTransactionPayload, pr common.PaymentRequest) (bool, error) {
	panic("implement me")
}

func (n NodeProxy) CommitPaymentTransaction(transactionPayload common.PaymentTransactionPayload) (ok bool, err error) {
	var request = &utilityService.CommitPaymentTransactionRequest {
//		Transaction: transactionPayload
	}

	body, err := json.Marshal(request)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	reply, err := n.client.ProcessCommand(context.Background(), &ppsidechannel.CommandRequest {
		CommandType:          3,
		CommandBody:          string(body),
	})

	var payload utilityService.CommitPaymentTransactionResponse

	err = json.Unmarshal([]byte(reply.ResponseBody), &payload)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return payload.Ok, nil
}



