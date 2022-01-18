package commodity

import (
	"encoding/json"
	"fmt"
	"net/http"

	"paidpiper.com/payment-gateway/models"
)

type ratesResponse struct {
	Ipfs      float64 `json:"ipfs"`
	Tor       float64 `json:"tor"`
	Attention float64 `json:"attention"`
	Fee       int     `json:"fee"`
}

type Descriptor struct {
	UnitPrice float64
	Asset     string
}
type Manager interface {
	Calculate(commodiryRequest *models.CreatePaymentInfo) (*models.PaymentRequstBase, error)
	ReverseCalculate(service string, commodity string, price uint32, asset string) (*models.ValidatePaymentResponse, error)
	GetProxyNodeFee() uint32
}
type manager struct {
	priceTable   map[string]map[string]Descriptor
	proxyNodeFee uint32
}

func getConfigFromServer(address string) (*ratesResponse, error) {
	host := "https://rates.torplus.com"
	url := fmt.Sprintf("%v/api/%v/rates", host, address)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("curl %v error %v", url, err)
	}
	respModel := &ratesResponse{}
	err = json.NewDecoder(resp.Body).Decode(resp)
	if err != nil {
		return &ratesResponse{
			Fee:       10,
			Ipfs:      0.00000002,
			Tor:       0.1,
			Attention: 0.1,
		}, nil
	}
	return respModel, nil
}
func FromUrl(address string) (Manager, error) {
	respModel, err := getConfigFromServer(address)
	if err != nil {
		return nil, err
	}
	return &manager{
		priceTable: map[string]map[string]Descriptor{
			"ipfs": {
				"data": {
					UnitPrice: respModel.Ipfs,
					Asset:     models.PPTokenAssetName,
				},
			},
			"tor": {
				"data": {
					UnitPrice: respModel.Tor,
					Asset:     models.PPTokenAssetName,
				},
			},
			"http": {
				"attention": {
					UnitPrice: respModel.Attention,
					Asset:     models.PPTokenAssetName,
				},
			},
		},
		proxyNodeFee: uint32(respModel.Fee),
	}, nil
}

func New() Manager {
	return &manager{
		priceTable: map[string]map[string]Descriptor{
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
		proxyNodeFee: 10,
	}
}

func (cm *manager) GetProxyNodeFee() uint32 {
	return cm.proxyNodeFee
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
