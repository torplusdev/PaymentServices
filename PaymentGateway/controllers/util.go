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
	json.NewEncoder(w).Encode(data)
}

func RespondValue(w http.ResponseWriter, name string, value interface{}) {
	Respond(200, w, map[string]interface{}{name: value})
}



