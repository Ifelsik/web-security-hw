package proxy

import (
	"bufio"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ProxyClient struct {
	log  *logrus.Entry
	Addr string
	Conn net.Conn
}

func NewProxyClient(log *logrus.Entry, proxyAddr string) *ProxyClient {
	return &ProxyClient{
		log:  log,
		Addr: proxyAddr,
	}
}

func (p *ProxyClient) Connect() error {
	conn, err := net.Dial("tcp", p.Addr)
	if err != nil {
		p.log.Errorf("Failed to connect to proxy: %v", err)
		return err
	}
	p.log.Debug("Connected to proxy")
	p.Conn = conn
	return nil
}

func (p *ProxyClient) SendHTTP(request *http.Request) (*http.Response, error) {
	// handle TLS
	err := request.Write(p.Conn)
	if err != nil {
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(p.Conn), request)
	if err != nil {
		return nil, err
	}
	
	return resp, nil
}

func (p *ProxyClient) Close() {
	p.Conn.Close()
}
