package local

import (
	"bytes"
	"encoding/json"

	"github.com/stellar/go/support/log"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
)

type CallbackerFactory func(cmd *models.UtilityCommand) CallBacker
type CallBacker interface {
	call(reply models.OutCommandType, err error) error
}
type callBackerImpl struct {
	url string
	cmd *models.UtilityCommand
}

func newCallbacker(cmd *models.UtilityCommand) CallBacker {
	return &callBackerImpl{
		cmd.CallbackUrl,
		cmd,
	}
}

func (cb *callBackerImpl) call(reply models.OutCommandType, err error) error {
	if cb.url == "" || err != nil {
		log.Errorf("Call backer not call reason error: %v", err)
		return nil
	}
	data, err := json.Marshal(reply)

	if err != nil {
		log.Errorf("Command response marshal failed: %v", err)
		return err
	}
	cmd := cb.cmd
	values := &models.UtilityResponse{
		CommandResponse: data,
		CommandResponseCore: models.CommandResponseCore{
			CommandId:   cmd.CommandId,
			NodeId:      cmd.NodeId,
			SessionId:   cmd.CommandCore.SessionId,
			CommandType: cb.cmd.CommandType,
		},
	}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		log.Errorf("Callbacker marshal error: %v", err)
	}
	log.Info("Callbacker Body: %v", string(jsonValue))
	err = common.HttpPostWithoutResponseContext(cmd.CallbackUrl, bytes.NewBuffer(jsonValue))

	if err != nil {
		log.Errorf("Callback url execution failed: : %v", err)
		log.Fatal(err)
		return err
	}

	return nil
}
