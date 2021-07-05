package paymentmanager

import "context"

func NewHttpConnection(port int, addres string, server CallbackServer, peerHandler PeerHandler) PPConnection {
	sessionHandler := NewSessionHandler()
	client := NewClient(addres, port, sessionHandler)
	ppCallBack := &PPCallback{
		peerHandler,
		sessionHandler,
		client,
	}
	server.SetCallbackHandler(ppCallBack)
	return &httpPPConnectionConnection{
		ClientHandler:  client,
		CallbackServer: server,
	}
}

type httpPPConnectionConnection struct {
	ClientHandler
	CallbackServer
}

func (c *httpPPConnectionConnection) Start() {
	go c.CallbackServer.Start()
}

func (c *httpPPConnectionConnection) Shutdown(ctx context.Context) {
	c.CallbackServer.Shutdown(ctx)
}
