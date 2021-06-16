package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ProcessPaymentRequest struct {
	Route []RoutingNode

	CallbackUrl string // Payment command url

	StatusCallbackUrl string // Status callback command url

	PaymentRequest *PaymentRequest // json body

	NodeId PeerID // request reference identification
}

func (pr *ProcessPaymentRequest) MarshalJSON() (bs []byte, err error) {
	var typ struct {
		Route []RoutingNode

		CallbackUrl string // Payment command url

		StatusCallbackUrl string // Status callback command url

		PaymentRequest string // json body

		NodeId PeerID // request reference identification
	}
	typ.Route = pr.Route
	typ.CallbackUrl = pr.CallbackUrl
	typ.StatusCallbackUrl = pr.StatusCallbackUrl
	typ.NodeId = pr.NodeId
	bs, err = json.Marshal(&pr.PaymentRequest)
	if err != nil {
		return nil, err
	}
	typ.PaymentRequest = string(bs)
	return json.Marshal(&typ)
}

func (pr *ProcessPaymentRequest) UnmarshalJSON(b []byte) error {
	var typ struct {
		Route []RoutingNode

		CallbackUrl string // Payment command url

		StatusCallbackUrl string // Status callback command url

		PaymentRequest string // json body

		NodeId PeerID // request reference identification
	}
	err := json.Unmarshal(b, &typ)
	if err != nil {
		return err
	}
	pr.Route = typ.Route
	pr.CallbackUrl = typ.CallbackUrl
	pr.StatusCallbackUrl = typ.StatusCallbackUrl
	pr.NodeId = typ.NodeId
	paymentRequest := &PaymentRequest{}

	cleanPaymentRequest := strings.TrimRight(typ.PaymentRequest, "\r\n ")
	cleanPaymentRequest = strings.ReplaceAll(cleanPaymentRequest, "\\n", "")
	fmt.Println("JSON: ", cleanPaymentRequest)
	err = json.Unmarshal([]byte(cleanPaymentRequest), paymentRequest)
	if err != nil {
		return err
	}
	pr.PaymentRequest = paymentRequest
	return nil
}
