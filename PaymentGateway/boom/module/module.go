package module

import (
	"encoding/base64"
	"fmt"
	"log"

	"paidpiper.com/payment-gateway/boom"
	"paidpiper.com/payment-gateway/boom/client"
	"paidpiper.com/payment-gateway/boom/data"
)

type IPFS interface {
	Get(b [][]byte) error
}

func Fill(selfProvider boom.BoomDataProvider, ipfs IPFS) error {
	contentIDs := map[string]*data.FrequencyContentMetadata{}
	conn, err := selfProvider.Connections()
	if err != nil {
		return fmt.Errorf("fail connections request: %v", err)
	}
	for _, host := range conn.Hosts {
		clientOfMain := client.New(host)
		els, err := clientOfMain.Elements()
		if err != nil {
			log.Fatalf("error request elements: %v", err)
			continue
		}
		for _, el := range els {
			key := base64.StdEncoding.EncodeToString(el.Cid)
			if item, ok := contentIDs[key]; ok {
				item.Frequency += el.Frequency //NEED?
			} else {
				contentIDs[key] = el
			}
		}
	}
	keys := [][]byte{}
	for _, item := range contentIDs {
		keys = append(keys, item.Cid)
	}
	return ipfs.Get(keys)

}
