package torclient

import (
	"context"

	"paidpiper.com/payment-gateway/models"
)

type staticTorClient struct {
	route *models.RouteResponse
	err   error
}

func FromStatic(route *models.RouteResponse, err error) TorClient {
	return &staticTorClient{
		route: route,
	}
}

func (c *staticTorClient) GetRoute(ctx context.Context, sessionId string, excludeNodeId, excludeAddress string) (*models.RouteResponse, error) {

	return c.route, c.err
}
