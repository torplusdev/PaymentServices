package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"net/http"
)

type cidWithMetadata struct {
	Cid       string
	Frequency uint64
}

func AddHandlers(router *mux.Router) {
	router.HandleFunc("/api/boom/elements", httpElements).Methods("GET")
	router.HandleFunc("/api/boom/cids", httpCids).Methods("GET")
	router.HandleFunc("/api/boom/connections", httpConnections).Methods("GET")

}

func httpConnections(w http.ResponseWriter, r *http.Request) {
	if globConnectionSource != nil {
		res, err := globConnectionSource.Connections()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
		writeJson(w, res)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprint(w, "globConnectionSource is null")
	return

}

func httpElements(w http.ResponseWriter, r *http.Request) {
	res, err := FrequentElements()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	writeJson(w, res)
}

func httpCids(w http.ResponseWriter, r *http.Request) {
	encodedElements, err := FrequentElements()

	results := []cidWithMetadata{}
	for _, e := range encodedElements {
		_, actualCid, err := cid.CidFromBytes(e.Cid)

		if err == nil {
			results = append(results, cidWithMetadata{
				Frequency: e.Frequency,
				Cid:       actualCid.String(),
			})
		}
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	writeJson(w, results)
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
