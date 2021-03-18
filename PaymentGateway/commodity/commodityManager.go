package commodity

import (
	"fmt"

	"paidpiper.com/payment-gateway/models"
)

type Descriptor struct {
	UnitPrice float64
	Asset     string
}
type Manager interface {
	Calculate(commodiryRequest *models.CreatePaymentInfo) (*models.PaymentRequstBase, error)
	ReverseCalculate(service string, commodity string, price uint32, asset string) (*models.ValidatePaymentResponse, error)
}
type manager struct {
	priceTable map[string]map[string]Descriptor
}

func New() Manager {

	return &manager{priceTable: map[string]map[string]Descriptor{
		"ipfs": {
			"data": {
				UnitPrice: 0.00000002,
				Asset:     models.PPTokenAssetName,
			},
		},
		"tor": {
			"data": {
				UnitPrice: 0.1,
				Asset:     models.PPTokenAssetName,
			},
		},
		"http": {
			"attention": {
				UnitPrice: 0.1,
				Asset:     models.PPTokenAssetName,
			},
		},
	},
	}
}

func (cm *manager) Calculate(commodiryRequest *models.CreatePaymentInfo) (*models.PaymentRequstBase, error) {
	st, ok := cm.priceTable[commodiryRequest.ServiceType]

	if !ok {
		return nil, fmt.Errorf("unknown service %s", commodiryRequest.ServiceType)
	}

	d, ok := st[commodiryRequest.CommodityType]

	if !ok {
		return nil, fmt.Errorf("unknown commodity %s", commodiryRequest.CommodityType)
	}
	amount := uint32(d.UnitPrice * float64(commodiryRequest.Amount))
	return &models.PaymentRequstBase{
		ServiceRef: commodiryRequest.ServiceType,
		Asset:      d.Asset,
		Amount:     amount,
	}, nil
}

func (cm *manager) ReverseCalculate(service string, commodity string, price uint32, asset string) (*models.ValidatePaymentResponse, error) {
	st, ok := cm.priceTable[service]

	if !ok {
		return nil, fmt.Errorf("unknown service %s", service)
	}

	d, ok := st[commodity]

	if !ok {
		return nil, fmt.Errorf("unknown commodity %s", commodity)
	}

	if d.Asset != asset {
		return nil, fmt.Errorf("asset missmatch %s", asset)
	}

	quantity := uint32(float64(price) / d.UnitPrice)
	return &models.ValidatePaymentResponse{
		Quantity: quantity,
	}, nil
}
