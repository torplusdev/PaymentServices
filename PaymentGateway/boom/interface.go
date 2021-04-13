package boom

import "paidpiper.com/payment-gateway/boom/data"

type BoomDataProvider interface {
	Elements() ([]*data.FrequencyContentMetadata, error)
	Connections() (*data.Connections, error)
}
