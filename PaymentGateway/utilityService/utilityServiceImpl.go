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

func (s *UtilityServiceImpl) CreateTransaction(ctx context.Context, commandBody string) (*pp.CommandReply, error) {
	var request CreateTransactionRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return nil, err
	}

	transaction, err := s.node.CreateTransaction(request.TotalIn, request.TotalIn-request.TotalOut, request.TotalOut, request.SourceAddress)

	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(&transaction)

	if err != nil {
		return nil, err
	}

	return &pp.CommandReply{
		ResponseBody: string(value),
	}, nil
}

func (s *UtilityServiceImpl) SignTerminalTransaction(ctx context.Context, commandBody string) (*pp.CommandReply, error) {
	var request SignTerminalTransactionsRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return nil, err
	}

	err = s.node.SignTerminalTransactions(&request.Transaction)

	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(&request)

	return &pp.CommandReply{
		ResponseBody: string(value),
	}, nil
}

func (s *UtilityServiceImpl) SignChainTransactions(ctx context.Context, commandBody string) (*pp.CommandReply, error) {
	var request SignChainTransactionsRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return nil, err
	}

	err = s.node.SignChainTransactions(&request.Credit, &request.Debit)

	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(&request)

	if err != nil {
		return nil, err
	}

	return &pp.CommandReply{
		ResponseBody: string(value),
	}, nil
}

func (s *UtilityServiceImpl) CommitPaymentTransaction(ctx context.Context, commandBody string) (*pp.CommandReply, error) {
	var request CommitPaymentTransactionRequest

	err := json.Unmarshal([]byte(commandBody), &request)

	if err != nil {
		return nil, err
	}

	ok, err := s.node.CommitPaymentTransaction(&request.Transaction)

	if err != nil {
		return nil, err
	}

	value, err := json.Marshal(&CommitPaymentTransactionResponse{Ok: ok})

	if err != nil {
		return nil, err
	}

	return &pp.CommandReply{
		ResponseBody: string(value),
	}, nil
}

func (s *UtilityServiceImpl) ProcessCommand(ctx context.Context, command *pp.CommandRequest) (*pp.CommandReply, error) {
	switch command.CommandType {
	case 0:
		return s.CreateTransaction(ctx, command.CommandBody)
	case 1:
		return s.SignTerminalTransaction(ctx, command.CommandBody)
	case 2:
		return s.SignChainTransactions(ctx, command.CommandBody)
	case 3:
		return s.CommitPaymentTransaction(ctx, command.CommandBody)
	}

	return nil, errors.New("unknown command")
}
