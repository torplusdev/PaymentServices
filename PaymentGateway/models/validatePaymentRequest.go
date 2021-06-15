package models

import "encoding/json"

type ValidatePaymentRequest struct {
	ServiceType    string
	CommodityType  string
	PaymentRequest PaymentRequest // json body
}

func (pr *ValidatePaymentRequest) MarshalJSON() (bs []byte, err error) {

	bs, err = json.Marshal(&pr.PaymentRequest)
	if err != nil {
		return
	}
	var typ struct {
		ServiceType    string
		CommodityType  string
		PaymentRequest string // json body
	}
	typ.CommodityType = pr.CommodityType
	typ.ServiceType = pr.ServiceType
	typ.PaymentRequest = string(bs)
	prStr, err := json.Marshal(&typ)
	if err != nil {
		return nil, err
	}
	bs = []byte(prStr)
	return
}

func (d *ValidatePaymentRequest) UnmarshalJSON(data []byte) error {
	var typ struct {
		ServiceType    string
		CommodityType  string
		PaymentRequest string // json body
	}
	if err := json.Unmarshal(data, &typ); err != nil {
		return err
	}
	d.CommodityType = typ.CommodityType
	d.ServiceType = typ.ServiceType
	d.PaymentRequest = PaymentRequest{}
	err := json.Unmarshal([]byte(typ.PaymentRequest), &d.PaymentRequest)
	if err != nil {
		return err
	}
	return nil
}

type ValidatePaymentResponse struct {
	Quantity uint32
}
