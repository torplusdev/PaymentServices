package paymentmanager

import (
	"paidpiper.com/payment-gateway/models"
)

type PPCallbackHandler interface {
	ProcessCommand(msg *models.ProcessCommand) (err error)
	ProcessCommandResponse(msg *models.UtilityResponse) (err error) //*models.ProcessCommandResponse
	ProcessPaymentResponse(msg *models.PaymentStatusResponseModel) (err error)
}

type PPCallback struct {
	peerHandler    PeerHandler
	sessionHandler *SessionHandler
	clientHandler  ClientHandler
}

func NewPPCallback(peerHandler PeerHandler,
	clientHandler ClientHandler,
	sessionHandler *SessionHandler) PPCallbackHandler {
	return &PPCallback{
		peerHandler,
		sessionHandler,
		clientHandler,
	}
}

func (ppc *PPCallback) ProcessCommand(msg *models.ProcessCommand) (err error) {

	ppc.peerHandler.SendPaymentDataMessage(models.PeerID(msg.CommandCore.NodeId),
		&PaymentCommand{ //TODO REPLACE with ProcessCommand
			CommandId:   msg.CommandId,
			CommandBody: msg.CommandBody,
			CommandType: msg.CommandType,
			SessionId:   msg.SessionId,
		})
	return
}

type IncomingCommandResponseModel struct { //TODO REMOVE
	CommandResponse []byte
	CommandId       string
	NodeId          models.PeerID
	SessionId       string
}

func (ppc *PPCallback) ProcessCommandResponse(msg *models.UtilityResponse) (err error) {
	peerID := models.PeerID(msg.CommandResponseCore.NodeId)

	// bs, err := json.Marshal(msg.Response)
	// if err != nil {
	// 	return err
	// }
	paymentResponse := &PaymentResponse{ //TODO replace UtilityResponse
		CommandId:    msg.CommandId,
		CommandReply: msg.CommandResponse,
		SessionId:    msg.SessionId,
		CommandType:  msg.CommandType,
	}
	ppc.peerHandler.SendPaymentDataMessage(peerID, paymentResponse)
	return
}

func (ppc *PPCallback) ProcessPaymentResponse(msg *models.PaymentStatusResponseModel) (err error) {
	session, err := ppc.sessionHandler.Close(msg.SessionId)
	if err != nil {
		return
	}

	targetId := session.OriginNodeId
	paymentStatusResponse := &PaymentStatusResponse{
		SessionId: msg.SessionId,
		Status:    msg.Status == 1,
	}
	ppc.peerHandler.SendPaymentDataMessage(targetId, paymentStatusResponse)
	return
}
