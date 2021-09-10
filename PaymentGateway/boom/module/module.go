package module

import (
	"encoding/base64"
	"fmt"
	"strings"

	"paidpiper.com/payment-gateway/boom"
	"paidpiper.com/payment-gateway/boom/client"
	"paidpiper.com/payment-gateway/boom/data"
	"paidpiper.com/payment-gateway/log"
)

type IPFS interface {
	Get(b [][]byte) error
}

func Fill(selfProvider boom.BoomDataProvider, ipfs IPFS) error {
	contentIDs := map[string]*data.FrequencyContentMetadata{}
	conn, err := selfProvider.Connections()
	if err != nil {
		log.Errorf("fail connections request: %v", err)
		return fmt.Errorf("fail connections request: %v", err)
	}
	for _, host := range conn.Hosts {
		if !strings.Contains(host, "onion") {
			continue
		}
		log.Infof("Call host for freq elements: %v", host)
		clientOfMain := client.New(host)
		els, err := clientOfMain.Elements()
		if err != nil {
			log.Errorf("error request elements: %v", err)
			continue
		}
		if len(els) == 0 {
			log.Infof("Freq elements empty")
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
