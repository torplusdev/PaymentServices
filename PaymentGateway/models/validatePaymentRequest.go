package models

import "encoding/json"

type ShapelessValidatePaymentRequest struct {
	ServiceType    string
	CommodityType  string
	PaymentRequest string
}
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

	prStr, err := json.Marshal(&ShapelessValidatePaymentRequest{
		ServiceType:    pr.ServiceType,
		CommodityType:  pr.CommodityType,
		PaymentRequest: string(bs),
	})
	if err != nil {
		return nil, err
	}
	bs = []byte(prStr)
	return
}

func (d *ValidatePaymentRequest) UnmarshalJSON(data []byte) error {
	typ := &ShapelessValidatePaymentRequest{}
	if err := json.Unmarshal(data, &typ); err != nil {
		return err
	}
	d.CommodityType = typ.CommodityType
	d.ServiceType = typ.ServiceType
	err := json.Unmarshal([]byte(typ.PaymentRequest), &d.PaymentRequest)
	if err != nil {
		return err
	}
	return nil
}

type ValidatePaymentResponse struct {
	Quantity uint32
}
