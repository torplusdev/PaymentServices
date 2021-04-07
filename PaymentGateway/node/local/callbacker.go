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
		return nil
	}
	data, err := json.Marshal(reply)

	if err != nil {
		log.Fatalf("Command response marshal failed: %v", err)
		return err
	}
	cmd := cb.cmd
	values := &models.UtilityResponse{
		CommandResponse: data,
		CommandResponseCore: models.CommandResponseCore{
			CommandId: cmd.CommandCore.CommandId,
			NodeId:    cmd.CommandCore.NodeId,
			SessionId: cmd.CommandCore.SessionId,
		},
	}
	jsonValue, _ := json.Marshal(values)

	err = common.HttpPostWithoutResponseContext(cmd.CallbackUrl, bytes.NewBuffer(jsonValue))

	if err != nil {
		log.Errorf("Callback url execution failed: : %v", err)
		log.Fatal(err)
		return err
	}

	return nil
}
