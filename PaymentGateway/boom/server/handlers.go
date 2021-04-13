package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func AddHandlers(router *mux.Router) {
	router.HandleFunc("/api/boom/elements", httpProcessResponse).Methods("GET")
	router.HandleFunc("/api/boom/connections", httpConnections).Methods("GET")

}

func httpConnections(w http.ResponseWriter, r *http.Request) {
	res, err := globConnectionSource.Connections()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	writeJson(w, res)
}

func httpProcessResponse(w http.ResponseWriter, r *http.Request) {
	res, err := FrequentElements()
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
