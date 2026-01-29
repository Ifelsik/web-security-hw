package proxy

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	maxIdleConns          = 100
	idleConnTimeout       = 90 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	expectContinueTimeout = 1 * time.Second
)

const (
	dialTimeout            = 10 * time.Second
	keepAliveProbeInterval = 30 * time.Second
)

// proxyTransport has a pool of TCP connections to improve performance.
var proxyTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: keepAliveProbeInterval,
	}).DialContext,
	ForceAttemptHTTP2:     false,
	MaxIdleConns:          maxIdleConns,
	IdleConnTimeout:       idleConnTimeout,
	TLSHandshakeTimeout:   tlsHandshakeTimeout,
	ExpectContinueTimeout: expectContinueTimeout,
}

var client = &http.Client{
	Transport: proxyTransport,
	Timeout:   0, // TODO: consider about timeout
}

// makeClientRequest makes HTTP request to server.
// Returned Response should be closed manually.
func makeClientRequest(req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make client request: %w", err)
	}
	return resp, nil
}
