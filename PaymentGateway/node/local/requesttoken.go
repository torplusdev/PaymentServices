package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RequestTokenModel struct {
	Address string `json:"address"`
}

func requestToken(url, address string) error {
	request, err := json.Marshal(&RequestTokenModel{Address: address})
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Post(url, "application/json", bytes.NewBuffer(request))
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		return nil
	}
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("error: %v", d)
}
