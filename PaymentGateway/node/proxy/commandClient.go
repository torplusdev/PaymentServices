package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"paidpiper.com/payment-gateway/log"

	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
)

type CommandClient interface {
	CreateTransaction(context.Context, *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error)
	SignServiceTransaction(context context.Context, request *models.SignServiceTransactionCommand) (*models.SignServiceTransactionResponse, error)
	SignChainTransaction(context context.Context, command *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error)
	CommitServiceTransaction(context context.Context, req *models.CommitServiceTransactionCommand) error
	CommitChainTransaction(context context.Context, request *models.CommitChainTransactionCommand) error
}
type CommandResponseHandler interface {
	ProcessResponse(context context.Context, commandId string, responseBody []byte) error
}

func NewCommandClient(
	url,
	sessionId,
	nodeId string) (CommandClient, CommandResponseHandler) {
	commandClient := &commandClient{
		torUrl:     url,
		chainStore: NewCommandChainStore(),
		sessionId:  sessionId,
		nodeId:     nodeId,
	}
	return commandClient, commandClient
}

type commandClient struct {
	torUrl     string
	chainStore *commandChannelStore
	sessionId  string //TODO REMOVE AFTER SESSION WRAPPER TO INTERFACE
	nodeId     string
}

func (cl *commandClient) CreateTransaction(context context.Context, request *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {
	response := &models.CreateTransactionResponse{}
	err := processCommandWrapper(cl, context, request, response)
	if err != nil {
		return nil, err
	}
	return response, err

}

func (cl *commandClient) SignServiceTransaction(context context.Context, request *models.SignServiceTransactionCommand) (*models.SignServiceTransactionResponse, error) {
	response := &models.SignServiceTransactionResponse{}
	err := processCommandWrapper(cl, context, request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (cl *commandClient) SignChainTransaction(context context.Context, request *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error) {
	var response = &models.SignChainTransactionResponse{}
	err := processCommandWrapper(cl, context, request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (cl *commandClient) CommitServiceTransaction(context context.Context, request *models.CommitServiceTransactionCommand) error {
	return processCommandWrapperNoRes(cl, context, request)

}

func (cl *commandClient) CommitChainTransaction(context context.Context, request *models.CommitChainTransactionCommand) error {
	return processCommandWrapperNoRes(cl, context, request)

}

func processCommandWrapperNoRes(cl *commandClient, context context.Context, request models.InCommandType) error {
	body, err := cl.WrapToCommand(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := cl.processCommand(context, body)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	var response = &struct{}{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	return nil
}

func processCommandWrapper(cl *commandClient, context context.Context, request models.InCommandType, out models.OutCommandType) error {
	body, err := cl.WrapToCommand(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := cl.processCommand(context, body)

	if err != nil {
		log.Errorf("Tor command error: %v", err)
		return errors.Errorf(err.Error())
	}

	err = json.Unmarshal(reply, out)

	if err != nil {
		log.Errorf("ResponseJSON: %v", string(reply))
		log.Errorf("Tor command unmarshal error: %v", err)
		return errors.Errorf(err.Error())
	}

	return nil
}

//TODO WRAPPER TO INTERFACE
func (cl *commandClient) WrapToCommand(cmd models.InCommandType) (*models.ProcessCommand, error) {
	body, err := json.Marshal(cmd)

	if err != nil {
		log.Errorf("WrapToCommand marshal error: %v", err)
		return nil, errors.Errorf(err.Error())
	}
	command := &models.ProcessCommand{
		CommandCore: models.CommandCore{
			SessionId:   cl.sessionId,
			NodeId:      cl.nodeId,
			CommandId:   uuid.New().String(),
			CommandType: cmd.Type(),
		},
		CommandBody: body,
	}

	return command, err

}

//TODO TO INTERFACE
func (cl *commandClient) processCommand(context context.Context, cmd *models.ProcessCommand) ([]byte, error) {
	commandId := cmd.CommandId
	ch := cl.chainStore.open(commandId, cmd.CommandType)

	defer cl.chainStore.close(commandId)

	log.Infof("Process command SessionId=%s, NodeId=%s, CommandId=%s CommandType:%d", cmd.SessionId, cl.nodeId, commandId, cmd.CommandType)
	//TODO ERROR
	jsonValue, _ := json.Marshal(cmd)
	log.Infof("Tor Request Body: %v", string(jsonValue))
	res, err := common.HttpPostWithoutContext(cl.torUrl, bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Errorf("request to tor error: %v", err)
		return nil, err
	}
	defer res.Body.Close()

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error("Read body error:", err)
	}
	if err == nil && len(bodyBytes) > 0 {
		return bodyBytes, nil
	}

	// Wait
	responseBody := <-ch

	return responseBody, nil
}

func (cl *commandClient) ProcessResponse(context context.Context, commandId string, responseBody []byte) error {
	ok := cl.chainStore.processResponse(commandId, responseBody)
	if !ok {
		err := fmt.Errorf("unknown command response: : %s on %s", commandId, cl.nodeId)
		log.Error(err)
		return err
	}
	return nil
}
