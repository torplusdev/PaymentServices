package node

import (
	"context"

	"paidpiper.com/payment-gateway/models"
)

type PPNode interface {
	//TODO WRAPPERS TO sublayer
	CreateTransaction(ctx context.Context, command *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error)
	SignChainTransaction(ctx context.Context, command *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error)
	SignServiceTransaction(ctx context.Context, command *models.SignServiceTransactionCommand) (*models.SignServiceTransactionResponse, error)
	CommitChainTransaction(ctx context.Context, command *models.CommitChainTransactionCommand) error
	CommitServiceTransaction(ctx context.Context, command *models.CommitServiceTransactionCommand) error
	GetAddress() string
	GetFee() uint32
}
