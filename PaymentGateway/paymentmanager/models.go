package paymentmanager

import (
	"paidpiper.com/payment-gateway/log"

	"paidpiper.com/payment-gateway/models"
)

// PeerHandler sends changes out to the network as they get added to the payment list
type PaymentData interface {
	Handler(p models.PeerID) PaymentHandler
	//	Handle(paymentHandler PaymentHandler, peerHandler PeerHandler, client ClientHandler)
}

type InitiatePayment struct {
	PaymentRequest *models.PaymentRequest // json marshal
}

func (m *InitiatePayment) Handler(from models.PeerID) PaymentHandler {
	return &InitiatePaymentHandler{
		from: from,
		msg:  m,
	}
}

type InitiatePaymentHandler struct {
	from models.PeerID
	msg  *InitiatePayment
}

func (i *InitiatePaymentHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	req := &models.ValidatePaymentRequest{
		PaymentRequest: *i.msg.PaymentRequest,
		ServiceType:    "ipfs",
		CommodityType:  "data",
	}
	quantity, err := client.ValidatePayment(req)

	if err != nil {
		log.Errorf("payment validation failed: %s", err)
	}

	debt := paymentHandler.GetDebt(i.from)

	if quantity > debt.receivedBytes {
		log.Warnf("invalid quantity requested: quantity: %v  receivedBytes: %v", quantity, debt.receivedBytes)
	}

	client.ProcessPayment(i.from, i.msg)
}

type PaymentCommand struct { //TODO REMOVE
	CommandId   string
	CommandType models.CommandType
	SessionId   string
	CommandBody []byte
}

func (m *PaymentCommand) Handler(from models.PeerID) PaymentHandler {
	return &ProcessOutgoingPaymentHandler{
		from: from,
		msg:  m,
	}
}

type ProcessOutgoingPaymentHandler struct {
	from models.PeerID
	msg  *PaymentCommand
}

func (h *ProcessOutgoingPaymentHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	msg := h.msg
	err := client.ProcessCommand(h.from, msg)

	if err != nil {
		log.Errorf("process command failed: %v", err)
	}
}

type PaymentResponse struct {
	CommandId    string
	CommandReply []byte
	SessionId    string
	CommandType  models.CommandType
}

func (m *PaymentResponse) Handler(from models.PeerID) PaymentHandler {
	return &ProcessPaymentResponseHandler{
		from: from,
		msg:  m,
	}
}

type ProcessPaymentResponseHandler struct {
	from models.PeerID
	msg  *PaymentResponse
}

func (h *ProcessPaymentResponseHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	msg := h.msg
	client.ProcessResponse(h.from, msg)
}

type PaymentStatusResponse struct {
	SessionId string
	Status    bool
}

func (m *PaymentStatusResponse) Handler(from models.PeerID) PaymentHandler {
	return &ProcessPaymentStatusResponseHandler{
		from: from,
		msg:  m,
	}
}

type ProcessPaymentStatusResponseHandler struct {
	from models.PeerID
	msg  *PaymentStatusResponse
}

func (h *ProcessPaymentStatusResponseHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	m := h.msg
	if !m.Status {
		// TODO: retry ?
		return
	}

	trx, err := client.GetTransaction(m.SessionId)

	if err != nil {
		log.Errorf("Transaction not found: %v", err)
	}

	debt := paymentHandler.GetDebt(h.from)

	debt.requestedAmount -= trx.AmountOut
}
