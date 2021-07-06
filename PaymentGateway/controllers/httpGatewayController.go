package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local"
)

type HttpGatewayController struct {
	local.LocalPPNode
}

func NewHttpGatewayController(n local.LocalPPNode) *HttpGatewayController {
	return &HttpGatewayController{
		n,
	}
}

func (g *HttpGatewayController) HttpProcessResponse(w http.ResponseWriter, r *http.Request) {

	response := &models.UtilityResponseFixModel{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error read request body: %s", err.Error())
	}
	log.Infof("Http process response %v", string(data))
	err = json.NewDecoder(bytes.NewReader(data)).Decode(response)

	if err != nil {
		log.Errorf("Error decoding request: %s", err.Error())
		log.Tracef("Body: %v", string(data))

		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}
	res := models.NewShapelessProcessCommandResponse(response)
	err = g.ProcessResponse(context.Background(), res)
	if err != nil {
		log.Errorf("Error processing response: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusConflict, err.Error()))
		return
	}

}

func (g *HttpGatewayController) HttpProcessPayment(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "ProcessPayment")
	defer span.End()

	request := &models.ProcessPaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		log.Errorf("Error decoding payment request: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, fmt.Sprintf("Bad request:%v", err)))
		return
	}
	res, err := g.ProcessPayment(ctx, request)
	if err != nil {
		log.Errorf("Error processing payment request: %s", err.Error())
		Respond(w, MessageWithStatus(http.StatusBadRequest, err.Error()))
		return
	}
	if res != nil {
		log.Infof("Payment processing complete (session %s)", res.SessionId)
		Respond(w, MessageWithData(http.StatusCreated, res))
		return
	}
	Respond(w, MessageWithStatus(http.StatusOK, "Payment processing completed"))

}
