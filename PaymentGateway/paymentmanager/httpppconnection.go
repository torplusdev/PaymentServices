package paymentmanager

func NewHttpConnection(port int, ppaddres string, peerHandler PeerHandler) PPConnection {
	sessionHandler := NewSessionHandler()
	client := NewClient(ppaddres, port, sessionHandler)
	ppCallBack := &PPCallback{
		peerHandler,
		sessionHandler,
		client,
	}
	server := NewServer(port, ppCallBack)
	return &httpPPConnectionConnection{
		ClientHandler:   client,
		CallbackHandler: server,
	}
}

type httpPPConnectionConnection struct {
	ClientHandler
	CallbackHandler
}

func (c *httpPPConnectionConnection) Start() {
	go c.Start()
}
