package models

type CreatePaymentInfo struct {
	ServiceType   string
	CommodityType string
	Amount        uint32
}
