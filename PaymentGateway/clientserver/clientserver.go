package clientserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	boom "github.com/tylertreat/BoomFilters"
)

type BoomDataSource interface {
	Elements() []*boom.Element
}

var globSource BoomDataSource

type FrequencyContentMetadata struct {
	Cid         []byte
	Frequency   uint64
	LastUpdated time.Time
}
type PpBoomElement struct {
	Data []byte
	Freq uint64
}

func AddHandler(source BoomDataSource) {
	globSource = source
}

func BoomElements() ([]*boom.Element, error) {
	if globSource == nil {
		return nil, fmt.Errorf("boot source not inited")
	}
	return globSource.Elements(), nil
}

func RouteElements() ([]*FrequencyContentMetadata, error) {

	items, err := BoomElements()
	if err != nil {
		return nil, err
	}
	listings := []*FrequencyContentMetadata{}
	ts := time.Now()
	for _, e := range items {
		l := &FrequencyContentMetadata{
			Cid:         e.Data,
			Frequency:   e.Freq,
			LastUpdated: ts,
		}
		listings = append(listings, l)
	}
	return listings, nil

}

func HttpProcessResponse(w http.ResponseWriter, r *http.Request) {
	res, err := BoomElements()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	writeJson(w, res)
}

func writeJson(w http.ResponseWriter, items interface{}) {
	err := json.NewEncoder(w).Encode(items)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)

}
