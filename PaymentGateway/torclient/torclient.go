package torclient

import (
	"context"
	"encoding/json"

	"github.com/go-errors/errors"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/log"
	"paidpiper.com/payment-gateway/models"
)

type torClient struct {
	torUrl string
}

func NewTorClient(url string) TorClient {
	return &torClient{
		torUrl: url,
	}
}

func (c *torClient) GetRoute(ctx context.Context, sessionId string) (*models.RouteResponse, error) {
	url := c.torUrl + sessionId
	resp, err := common.HttpGetWithContext(ctx, url)

	if err != nil {
		log.Errorf("HttpRequest error: url:", url)
		return nil, errors.Errorf("Cant get payment route: %v", err)
	}

	defer resp.Body.Close()

	routeResponse := &models.RouteResponse{}

	err = json.NewDecoder(resp.Body).Decode(routeResponse)

	if err != nil {
		return nil, errors.Errorf("Cant get payment route: %v", err)
	}
	return routeResponse, nil
}
