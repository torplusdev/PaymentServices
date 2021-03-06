package torclient

import (
	"context"

	"paidpiper.com/payment-gateway/models"
)

type TorClient interface {
	GetRoute(ctx context.Context, sessionId string) (*models.RouteResponse, error)
}
