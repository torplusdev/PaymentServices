package client

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

const (
	PROXY_ADDR = "127.0.0.1:9050"
)

type DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

func newClient(dialContext DialContext) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialContext,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
		},
	}
}

func NewClientWithProxy() (*http.Client, error) {
	baseDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	var dialContext DialContext

	if PROXY_ADDR != "" {
		dialSocksProxy, err := proxy.SOCKS5("tcp", PROXY_ADDR, nil, baseDialer)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating SOCKS5 proxy")
		}
		if contextDialer, ok := dialSocksProxy.(proxy.ContextDialer); ok {
			dialContext = contextDialer.DialContext
		} else {
			return nil, errors.New("Failed type assertion to DialContext")
		}
		// logger.Debug("Using SOCKS5 proxy for http client",
		// 	zap.String("host", proxyHost),
		// )
	} else {
		dialContext = (baseDialer).DialContext
	}

	httpClient := newClient(dialContext)
	return httpClient, nil
}
