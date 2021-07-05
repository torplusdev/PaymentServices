package paymentmanager

import (
	"context"

	"paidpiper.com/payment-gateway/models"
)

type PeerHandler interface {
	SendPaymentDataMessage(id models.PeerID, data PaymentData)
}
type PaymentManager interface {
	RequirePayment(ctx context.Context, id models.PeerID, msgSize int)
	RegisterReceivedBytes(ctx context.Context, id models.PeerID, msgSize int)
	ReceivePaymentDataMessage(ctx context.Context, id models.PeerID, data PaymentData)
	SetHttpConnection(commandListenPort int, channelUrl string, server CallbackServer)
	Startup()
}

// Payment manager manages payment requests and process actual payments over the Stellar network
type paymentManager struct {
	paymentMessages chan PaymentHandler
	ctx             context.Context
	cancel          func()
	peerHandler     PeerHandler
	debtRegistry    DebtRegestry
	ppConnection    PPConnection
}

const (
	requestPaymentAfterBytes = 50 * 1024 * 1024 // Pey per each 50 MB including transaction fee => 50 * 0.00002 + 0.00001 = 0.00101 XLM , 1 XLM pays for 49,5GB of data
)

// New initializes a new WantManager for a given context.
func New(ctx context.Context, peerHandler PeerHandler) PaymentManager {
	ctx, cancel := context.WithCancel(ctx)

	return &paymentManager{

		paymentMessages: make(chan PaymentHandler, 10),
		ctx:             ctx,
		cancel:          cancel,
		peerHandler:     peerHandler,

		debtRegistry: &debtRegestryImpl{store: make(map[models.PeerID]*Debt)},
	}
}

func (pm *paymentManager) SetHttpConnection(commandListenPort int, channelUrl string, server CallbackServer) {

	conn := NewHttpConnection(commandListenPort, channelUrl, server, pm.peerHandler)
	pm.SetConnection(conn)
}

func (pm *paymentManager) SetConnection(ppConnection PPConnection) {
	pm.ppConnection = ppConnection
}

// Startup starts processing for the PayManager.
func (pm *paymentManager) Startup() {

	go pm.run()

	if pm.ppConnection != nil {
		pm.ppConnection.Start()
	}
}

// Shutdown ends processing for the pay manager.
func (pm *paymentManager) Shutdown() {
	pm.cancel()

	pm.ppConnection.Shutdown(pm.ctx)
}

func (pm *paymentManager) run() {
	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	for {
		select {
		case message := <-pm.paymentMessages:
			message.Handle(pm.debtRegistry, pm.peerHandler, pm.ppConnection)
		case <-pm.ctx.Done():
			return
		}
	}
}

func (pm *paymentManager) ReceivePaymentDataMessage(ctx context.Context, id models.PeerID, data PaymentData) {
	select {
	case pm.paymentMessages <- data.Handler(id):
	case <-pm.ctx.Done():
	case <-ctx.Done():
	}
}

func (pm *paymentManager) SendPaymentDataMessage(id models.PeerID, data PaymentData) {
	pm.peerHandler.SendPaymentDataMessage(id, data)
}

func (pm *paymentManager) RegisterReceivedBytes(ctx context.Context, id models.PeerID, msgSize int) {
	select {
	case pm.paymentMessages <- &RegisterReceivedBytesHandler{from: id, msgSize: msgSize}:
	case <-pm.ctx.Done():
	case <-ctx.Done():
	}
}

// Register {msgSize} bytes sent to {id} peer and initiate payment request
func (pm *paymentManager) RequirePayment(ctx context.Context, id models.PeerID, msgSize int) {
	select {
	case pm.paymentMessages <- &RequirePaymentHandler{target: id, msgSize: msgSize}:
	case <-pm.ctx.Done():
	case <-ctx.Done():
	}
}
