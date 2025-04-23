package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

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
	if request.URL.Scheme == "https" {
		p.log.Debug("Establishing TLS connection")
		err := p.establishTLS(request.Host + ":443")
		if err != nil {
			return nil, err
		}
		p.log.Debug("TLS connection established")
	}

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

func (p *ProxyClient) establishTLS(targetHost string) error {
	connectRequest := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: targetHost},
		Host:   targetHost,
		Header: http.Header{
			"Host": []string{targetHost},
		},
	}
	connectRequest.Write(p.Conn)

	resp, err := http.ReadResponse(bufio.NewReader(p.Conn), connectRequest)
	if err != nil {
		return fmt.Errorf("proxy CONNECT failed: %v", err)
	}
	if resp.StatusCode != 200 {
        return fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
    }

	tlsConn := tls.Client(p.Conn, &tls.Config{
		ServerName: targetHost,
		InsecureSkipVerify: true,
	})
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	p.Conn = tlsConn
	return nil
}
