package controllers

import (
	"encoding/json"
	"log"
	"net/http"
)

type ResponseMessage struct {
	Status int
	Fields map[string]interface{}
}

func Message(message string) ResponseMessage {
	msg := ResponseMessage{
		Status: http.StatusOK,
		Fields: map[string]interface{}{"message": message},
	}

	return msg
}

func MessageWithStatus(status int, message string) ResponseMessage {
	msg := ResponseMessage{
		Status: status,
		Fields: map[string]interface{}{"message": message},
	}

	return msg
}


func Respond(w http.ResponseWriter, data interface{}) {

	w.Header().Add("Content-Type", "application/json")

	var err error

	switch data.(type) {
		case ResponseMessage:
			msg := data.(ResponseMessage)
			w.WriteHeader(msg.Status)
			err = json.NewEncoder(w).Encode(msg)

		case  chan ResponseMessage:
			waitChannel := data.(chan ResponseMessage)
			msg := <- waitChannel
			w.WriteHeader(msg.Status)
			err = json.NewEncoder(w).Encode(msg)

		default:
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(data)
	}

	if err != nil {
		log.Printf("Error encoding data for response: %v",err.Error())
	}
}



//func Respond(status int, w http.ResponseWriter, data map[string]interface{}) {
//	w.WriteHeader(status)
//	w.Header().Add("Content-Type", "application/json")
//	err := json.NewEncoder(w).Encode(data)
//
//	if err != nil {
//		// Log
//	}
//}

//func RespondObject(w http.ResponseWriter, data interface{}) {
//	w.WriteHeader(200)
//	w.Header().Add("Content-Type", "application/json")
//	err := json.NewEncoder(w).Encode(data)
//
//	if err != nil {
//		// Log
//	}
//}
//
