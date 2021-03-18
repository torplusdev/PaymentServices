package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"paidpiper.com/payment-gateway/common"
)

type ResponseMessage struct {
	Status int
	Data   interface{}
}

func Message(message string) ResponseMessage {
	msg := ResponseMessage{
		Status: http.StatusOK,
		Data:   map[string]interface{}{"message": message},
	}

	return msg
}

func MessageWithStatus(status int, message string) ResponseMessage {
	msg := ResponseMessage{
		Status: status,
		Data:   map[string]interface{}{"message": message},
	}

	return msg
}

func MessageWithData(status int, data interface{}) ResponseMessage {
	msg := ResponseMessage{
		Status: status,
		Data:   data,
	}

	return msg
}

func Respond(w http.ResponseWriter, data interface{}) {

	w.Header().Add("Content-Type", "application/json")

	var err error

	switch res := data.(type) {
	case ResponseMessage:

		w.WriteHeader(res.Status)
		err = json.NewEncoder(w).Encode(res.Data)

	case chan ResponseMessage:

		defer close(res)
		msg := <-res
		w.WriteHeader(msg.Status)
		err = json.NewEncoder(w).Encode(msg.Data)

	case common.HttpErrorMessage:

		err = res.WriteHttpError(w)
	default:
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(data)
	}

	if err != nil {
		log.Printf("Error encoding data for response: %v", err.Error())
	}
}
