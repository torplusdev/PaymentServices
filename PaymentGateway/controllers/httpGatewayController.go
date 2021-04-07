package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}
	//TODO context
	err = g.ProcessResponse(context.Background(), response)
	if err != nil {
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
		Respond(w, MessageWithStatus(http.StatusBadRequest, fmt.Sprintf("Bad request:%v", err)))
		return
	}
	res, err := g.ProcessPayment(ctx, request)
	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, err.Error()))
		return
	}
	if res != nil {
		Respond(w, MessageWithData(http.StatusCreated, res))
		return
	}
	Respond(w, MessageWithStatus(http.StatusOK, "Payment processing completed"))

}
