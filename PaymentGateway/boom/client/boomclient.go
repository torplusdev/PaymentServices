package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"paidpiper.com/payment-gateway/boom"
	"paidpiper.com/payment-gateway/boom/data"
	"paidpiper.com/payment-gateway/log"
)

func New(host string, proxy string) boom.BoomDataProvider {
	host = strings.ReplaceAll(host, ":", ".onion:")
	host = strings.ReplaceAll(host, "/onion3/", "http://")
	host = strings.ReplaceAll(host, "4001", "30500")
	return &boomClient{
		host:  host,
		proxy: proxy,
	}
}

type boomClient struct {
	host  string
	proxy string
}

func (bc *boomClient) Connections() (*data.Connections, error) {

	httpClient, err := NewClientWithProxy(bc.proxy)
	if err != nil {
		log.Errorf("create client error: %v", err)
		return nil, err
	}
	url := fmt.Sprintf("%s/api/boom/connections", bc.host)
	// create a request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("create request error: %v", err)
		return nil, err
	}
	reply, err := httpClient.Do(req)

	if err != nil {
		log.Errorf("process request error: %v", err)
		return nil, err
	}
	response := &data.Connections{}
	defer reply.Body.Close()

	err = json.NewDecoder(reply.Body).Decode(response)
	if err != nil {
		log.Errorf("decode response  error: %v", err)
		return nil, err
	}

	return response, nil
}
func (bc *boomClient) Elements() ([]*data.FrequencyContentMetadata, error) {
	httpClient, err := NewClientWithProxy(bc.proxy)
	if err != nil {
		log.Errorf("create client error: %v", err)
		return nil, err
	}
	url := fmt.Sprintf("%s/api/boom/elements", bc.host)
	log.Debugf("create client error: %v", err)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("create http request error")
		return nil, err
	}
	reply, err := httpClient.Do(req)

	if err != nil {
		log.Errorf("process http request error")
		return nil, err
	}
	response := []*data.FrequencyContentMetadata{}
	defer reply.Body.Close()
	responseString, err := ioutil.ReadAll(reply.Body)
	if err != nil {
		log.Errorf("Read body error")
		return nil, err
	}
	log.Debug(string(responseString))

	err = json.Unmarshal(responseString, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}
