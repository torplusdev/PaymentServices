package controllers

import (
	"encoding/json"
	"net/http"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/utilityService"
)

type UtilityController struct {
	Impl *utilityService.UtilityServiceImpl
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {
	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		Respond(w, Message(false, "Invalid request"))
		return
	}

	var reply string

	switch command.CommandType {
	case 0:
		reply, err = u.Impl.CreateTransaction(command.CommandBody)
	case 1:
		reply, err = u.Impl.SignTerminalTransaction(command.CommandBody)
	case 2:
		reply, err = u.Impl.SignChainTransactions(command.CommandBody)
	case 3:
		reply, err = u.Impl.CommitPaymentTransaction(command.CommandBody)
	}

	if err != nil {
		Respond(w, Message(false, "Invalid request"))
		return
	}

	RespondValue(w, "ResponseBody", reply)
}