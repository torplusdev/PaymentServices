package paymentmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stellar/go/support/log"
	"paidpiper.com/payment-gateway/models"
)

type ppClient struct {
	//controllers.GatewayController
	//controllers.UtilityController
	channelUrl        string
	sessionHandler    *SessionHandler
	commandListenPort int
}

func NewClient(channelUrl string, commandListenPort int, sessionHandler *SessionHandler) ClientHandler {
	return &ppClient{
		channelUrl:        channelUrl,
		commandListenPort: commandListenPort,
		sessionHandler:    sessionHandler,
	}
}

func (pm *ppClient) ProcessResponse(nodeId models.PeerID, msg *PaymentResponse) {
	req := &models.UtilityResponse{
		CommandResponseCore: models.CommandResponseCore{
			CommandId: msg.CommandId,
			NodeId:    nodeId.String(),
			SessionId: msg.SessionId,
		},
		CommandResponse: msg.CommandReply,
	}
	url := fmt.Sprintf("%s/api/gateway/processResponse", pm.channelUrl)
	err := post(url, req, nil)
	if err != nil {
		log.Errorf("process response failed: %s", err.Error())
		return
	}

}

func (pm *ppClient) ProcessCommand(nodeId models.PeerID, msg *PaymentCommand) error {
	req := &models.ShapelessUtilityCommand{
		CommandCore: models.CommandCore{
			CommandId:   msg.CommandId,
			CommandType: msg.CommandType,
			NodeId:      nodeId.String(),
			SessionId:   msg.SessionId,
		},
		CallbackUrl: fmt.Sprintf("http://localhost:%d/api/commandResponse", pm.commandListenPort),
		CommandBody: msg.CommandBody,
	}
	url := fmt.Sprintf("%s/api/utility/processCommand", pm.channelUrl)
	err := post(url, req, nil)
	if err != nil {
		log.Errorf("process command  failed: %s", err.Error())
		return err
	}
	return nil
}

func (pm *ppClient) ProcessPayment(nodeId models.PeerID, msg *InitiatePayment) {
	request := &models.ProcessPaymentRequest{
		CallbackUrl:    fmt.Sprintf("http://localhost:%d/api/command", pm.commandListenPort),
		PaymentRequest: msg.PaymentRequest, // TODO FIX WHEN
		NodeId:         nodeId,
		Route:          nil, // TODO: remove to start chain payment
	}
	url := fmt.Sprintf("%s/api/gateway/processPayment", pm.channelUrl)
	response := &models.ProcessPaymentAccepted{}
	err := post(url, request, response)
	if err != nil {
		log.Errorf("Initiate Payment failed: %s", err.Error())

	}

	pm.sessionHandler.Open(response.SessionId, nodeId)
}

func (pm *ppClient) ValidatePayment(request *models.ValidatePaymentRequest) (uint32, error) {
	url := fmt.Sprintf("%s/api/utility/validatePayment", pm.channelUrl)
	response := &models.ValidatePaymentResponse{}
	err := post(url, request, response)
	if err != nil {
		log.Errorf("Validate Payment Request failed: %s", err)
	}
	return response.Quantity, nil
}

//TODO CHECK WHY NOT DESEREALIZE
func (pm *ppClient) CreatePaymentInfo(amount uint32) (*models.PaymentRequest, error) {
	request := models.CreatePaymentInfo{
		ServiceType:   "ipfs",
		CommodityType: "data",
		Amount:        amount,
	}
	url := fmt.Sprintf("%s/api/utility/createPaymentInfo", pm.channelUrl)
	response := &models.PaymentRequest{}
	err := post(url, request, response)
	if err != nil {
		log.Errorf("Validate Payment Request failed: %s", err.Error())
		return nil, fmt.Errorf("Validate Payment Request failed: %s", err.Error())
	}
	return response, err
}

func (pm *ppClient) GetTransaction(sessionId string) (trx *models.PaymentTransaction, err error) {
	url := fmt.Sprintf("%s/api/utility/transaction/%s", pm.channelUrl, sessionId)
	trx = &models.PaymentTransaction{}
	err = get(url, trx)
	if err != nil {
		log.Errorf("failed to get transaction: %s", err)
		return nil, err
	}
	return
}

func post(url string, values interface{}, response interface{}) error {
	jsonValue, err := json.Marshal(values)

	if err != nil {
		return err
	}

	reply, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		return err
	}

	defer reply.Body.Close()
	if response != nil {
		err = json.NewDecoder(reply.Body).Decode(response)
		if err != nil {
			return err
		}
	}
	log.Infof("url: %v StatusCode: %d", url, reply.StatusCode)
	return nil
}

func get(url string, response interface{}) error {
	reply, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("get url error: %v", err)
	}
	defer reply.Body.Close()
	if response != nil {
		err = json.NewDecoder(reply.Body).Decode(response)
		if err != nil {
			return err
		}
	}
	log.Infof("url: %v StatusCode: %d", url, reply.StatusCode)
	return nil
}
