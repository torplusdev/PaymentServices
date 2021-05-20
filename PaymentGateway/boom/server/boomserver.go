package server

import (
	"fmt"

	"paidpiper.com/payment-gateway/boom/data"
)

type BoomDataSource interface {
	Elements() []*data.FrequencyContentMetadata
}

type ConnectionSource interface {
	Connections() ([]*data.Connections, error)
}

var globSource BoomDataSource
var globConnectionSource ConnectionSource

func AddSource(source BoomDataSource) {
	globSource = source
}

func FrequentElements() ([]*data.FrequencyContentMetadata, error) {
	if globSource == nil {
		return nil, fmt.Errorf("boot source not inited")
	}
	return globSource.Elements(), nil
}
