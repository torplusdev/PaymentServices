package controllers

import (
	"encoding/json"
	"net/http"
)

func Message(message string) map[string]interface{} {
	return map[string]interface{}{"message": message}
}

func Respond(status int, w http.ResponseWriter, data map[string]interface{}) {
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)

	if err != nil {
		// Log
	}
}

func RespondObject(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(200)
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)

	if err != nil {
		// Log
	}
}

