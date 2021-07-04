package paymentmanager

import (
	"net"
	"time"
)

// The following code is copied from Go's source code (server.go from net/http).
// This is an unexported type in that package that needs to be used in order to
// enable keep-alive on HTTP connections.

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	if err := tc.SetKeepAlive(true); err != nil {
		return tc, err
	}

	const keepAlivePeriod = 3 * time.Minute

	if err := tc.SetKeepAlivePeriod(keepAlivePeriod); err != nil {
		return tc, err
	}

	return tc, nil
}
