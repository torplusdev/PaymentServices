package utilityService

import (
	"context"
	"encoding/json"
	"errors"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node"
	pp "paidpiper.com/payment-gateway/ppsidechannel"
)

type UtilityServiceImpl struct {
	node *node.Node
}

type CreateTransactionRequest struct {
	TotalIn       uint32
	TotalOut      uint32
	SourceAddress string
}

type SignTerminalTransactionsRequest struct {
	Transaction common.PaymentTransactionReplacing
}

type SignChainTransactionsRequest struct {
	Debit  common.PaymentTransactionReplacing
	Credit common.PaymentTransactionReplacing
}

type CommitPaymentTransactionRequest struct {
	Transaction common.PaymentTransactionReplacing
}

type CommitPaymentTransactionResponse struct {
	Ok bool
}

func (s *UtilityServiceImpl) CreateTransaction(commandBody string) (string, error) {
	var request CreateTransactionRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return "", err
	}

	transaction, err := s.node.CreateTransaction(request.TotalIn, request.TotalIn-request.TotalOut, request.TotalOut, request.SourceAddress)

	if err != nil {
		return "", err
	}

	value, err := json.Marshal(&transaction)

	if err != nil {
		return "", err
	}

	return string(value), nil
}

func (s *UtilityServiceImpl) SignTerminalTransaction(commandBody string) (string, error) {
	var request SignTerminalTransactionsRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return "", err
	}

	err = s.node.SignTerminalTransactions(&request.Transaction)

	if err != nil {
		return "", err
	}

	value, err := json.Marshal(&request)

	return string(value), nil
}

func (s *UtilityServiceImpl) SignChainTransactions(commandBody string) (string, error) {
	var request SignChainTransactionsRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return "", err
	}

	err = s.node.SignChainTransactions(&request.Credit, &request.Debit)

	if err != nil {
		return "", err
	}

	value, err := json.Marshal(&request)

	if err != nil {
		return "", err
	}

	return string(value), nil
}

func (s *UtilityServiceImpl) CommitPaymentTransaction(commandBody string) (string, error) {
	var request CommitPaymentTransactionRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return "", err
	}

	ok, err := s.node.CommitPaymentTransaction(&request.Transaction)

	if err != nil {
		return "", err
	}

	value, err := json.Marshal(&CommitPaymentTransactionResponse{Ok: ok})

	if err != nil {
		return "", err
	}

	return string(value), nil
}

func (s *UtilityServiceImpl) ProcessCommand(ctx context.Context, command *pp.CommandRequest) (*pp.CommandReply, error) {
	var reply string
	var err error

	switch command.CommandType {
	case 0:
		reply, err = s.CreateTransaction(command.CommandBody)
	case 1:
		reply, err = s.SignTerminalTransaction(command.CommandBody)
	case 2:
		reply, err = s.SignChainTransactions(command.CommandBody)
	case 3:
		reply, err = s.CommitPaymentTransaction(command.CommandBody)
	default:
		return nil, errors.New("unknown command")
	}

	return &pp.CommandReply{
		ResponseBody:         reply,
	}, err
}
