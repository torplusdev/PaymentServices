package paymentmanager

import (
	"log"

	"paidpiper.com/payment-gateway/models"
)

type PaymentHandler interface {
	Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler)
}
type RegisterReceivedBytesHandler struct {
	from    models.PeerID
	msgSize int
}

func (r *RegisterReceivedBytesHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	debt := paymentHandler.GetDebt(r.from)

	debt.receivedBytes += uint32(r.msgSize)
}

type RequirePaymentHandler struct {
	target  models.PeerID
	msgSize int
}

func (r *RequirePaymentHandler) Handle(paymentHandler DebtRegestry, peerHandler PeerHandler, client ClientHandler) {
	debt := paymentHandler.GetDebt(r.target)

	debt.transferredBytes += uint32(r.msgSize)

	if debt.transferredBytes >= requestPaymentAfterBytes {
		amount := debt.transferredBytes

		paymentRequest, err := client.CreatePaymentInfo(amount)

		if err != nil {
			log.Fatalf("create payment info failed: %s", err.Error())
			return
		}
		initiatePayment := &InitiatePayment{ //TODO SERIALIZE PROPERTY LIKE BYTES
			PaymentRequest: paymentRequest,
		}
		peerHandler.SendPaymentDataMessage(r.target, initiatePayment)
		debt.requestedAmount += amount
		debt.transferredBytes = 0
	}
}
