package controllers

import (
	"encoding/json"
	"log"
	"net/http"
)

type ResponseMessage struct {
	Status	int
	Data 	interface{}
}

func Message(message string) ResponseMessage {
	msg := ResponseMessage{
		Status: http.StatusOK,
		Data: map[string]interface{}{"message": message},
	}

	return msg
}

func MessageWithStatus(status int, message string) ResponseMessage {
	msg := ResponseMessage{
		Status: status,
		Data: map[string]interface{}{"message": message},
	}

	return msg
}

func MessageWithData(status int, data interface{}) ResponseMessage {
	msg := ResponseMessage{
		Status: status,
		Data: data,
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
			err = json.NewEncoder(w).Encode(msg.Data)

		case  chan ResponseMessage:
			responseChannel := data.(chan ResponseMessage)
			defer close(responseChannel)
			msg := <- responseChannel
			w.WriteHeader(msg.Status)
			err = json.NewEncoder(w).Encode(msg.Data)

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
