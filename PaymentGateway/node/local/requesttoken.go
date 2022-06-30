package local

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RequestTokenModel struct {
	Address string `json:"address"`
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func requestToken(url, address string) error {
	request, err := json.Marshal(&RequestTokenModel{Address: address})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(request))
	if err != nil {
		fmt.Printf("create reqeust error: %v\n", err)
		return err
	}

	req.Header.Add("Authorization", "Basic "+basicAuth("torplus-accounting-77mSFQ", "cYGNKqKtwbhT3KP87fnhnPEaV63HeNkMbLSgu8jCeGmaSrpQZGeQkFpe334sPRRwxBJjDDTJnUmsmxA7ZESXsSd68JUAtvVSM3xH"))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Printf("Create reqeust error: %v\n", err)
		return err
	}
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response is not ok %v\n", res.Status)
		return fmt.Errorf("response is not ok %v", res.Status)
	}
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Read response body error %v\n", err)
		return fmt.Errorf("response is not ok %v body %v", res.Status, string(d))
	}
	return nil
}
