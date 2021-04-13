package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"paidpiper.com/payment-gateway/boom"
	"paidpiper.com/payment-gateway/boom/data"
)

func New(host string) boom.BoomDataProvider {
	return &boomClient{
		host: host,
	}
}

type boomClient struct {
	host string
}

func (bc *boomClient) Connections() (*data.Connections, error) {
	url := fmt.Sprintf("%s/api/boom/connections", bc.host)

	reply, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	response := &data.Connections{}
	defer reply.Body.Close()
	if response != nil {
		err = json.NewDecoder(reply.Body).Decode(response)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}
func (bc *boomClient) Elements() ([]*data.FrequencyContentMetadata, error) {
	url := fmt.Sprintf("%s/api/boom/elements", bc.host)

	reply, err := http.Get(url)

	if err != nil {
		return nil, err
	}
	response := []*data.FrequencyContentMetadata{}
	defer reply.Body.Close()
	if response != nil {
		err = json.NewDecoder(reply.Body).Decode(response)
		if err != nil {
			return nil, err
		}
	}
	return response, nil
}
