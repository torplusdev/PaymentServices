package paymentmanager

import (
	"testing"

	"paidpiper.com/payment-gateway/models"
)

type paymentHandlerMock struct {
	debtRegistry           map[models.PeerID]*Debt
	requestedPaymentAmount uint32
	paymentRequest         string
}

func (p *paymentHandlerMock) GetDebt(id models.PeerID) *Debt {
	return p.debtRegistry[id]
}

func (p *paymentHandlerMock) CreatePaymentInfo(amount uint32) (string, error) {
	p.requestedPaymentAmount = amount

	return p.paymentRequest, nil
}

func (p *paymentHandlerMock) ProcessCommand(nodeId models.PeerID, msg *PaymentCommand) error {
	panic("not implemented")
}

func (p *paymentHandlerMock) ProcessPayment(nodeId models.PeerID, msg *InitiatePayment) {
	panic("not implemented")
}

func (p *paymentHandlerMock) ProcessResponse(nodeId models.PeerID, msg *PaymentResponse) {
	panic("not implemented")
}

func (p *paymentHandlerMock) ValidatePayment(req *models.ShapelessValidatePaymentRequest) (uint32, error) {
	panic("not implemented")
}

func (p *paymentHandlerMock) GetTransaction(sessionId string) (*models.PaymentTransaction, error) {
	panic("not implemented")
}

type PeerHandlerMock struct {
	paymentRequests map[models.PeerID]PaymentData
}

func (p *PeerHandlerMock) SendPaymentDataMessage(target models.PeerID, data PaymentData) {
	p.paymentRequests[target] = data
}

func TestRequirePayment(t *testing.T) {

	msg := RequirePaymentHandler{
		target:  "TargetId",
		msgSize: 1024 * 1024,
	}

	debt := Debt{
		id:               "TargetId",
		requestedAmount:  0,
		transferredBytes: 49 * 1024 * 1024, // 52 Mega transferred
		receivedBytes:    0,
	}
	paymentRequestConst := "sampleRequestInJson"
	paymentMock := &paymentHandlerMock{
		debtRegistry: map[models.PeerID]*Debt{
			"TargetId": &debt,
		},
		requestedPaymentAmount: 0,
		paymentRequest:         paymentRequestConst,
	}

	peerMock := &PeerHandlerMock{
		paymentRequests: map[models.PeerID]PaymentData{},
	}

	msg.Handle(paymentMock, peerMock, paymentMock)

	var expectedAmount uint32 = 50 * 1024 * 1024

	if paymentMock.requestedPaymentAmount != expectedAmount {
		t.Errorf("Invalid amount")
		return
	}
	data, ok := peerMock.paymentRequests["TargetId"]
	if !ok {
		t.Errorf("Invalid request")
		return
	}

	obj, ok := data.(*InitiatePayment)
	if !ok {
		t.Errorf("Invalid type")
		return
	}
	if obj.PaymentRequest != paymentRequestConst {
		t.Errorf("Invalid Payment Request")
		return
	}
	if debt.transferredBytes != 0 {
		t.Errorf("Transferred bytes count not zero")
	}
}
