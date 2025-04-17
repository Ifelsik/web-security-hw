package internal

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	TLSConnectionEstablishStartLine = "HTTP/1.1 200 Connection established\r\n\r\n"
)

type ProxyServer struct {
	CertStorage *CertStorage
}

func NewProxyServer(certDir string) *ProxyServer {
	server := new(ProxyServer)

	server.CertStorage = NewCertStorage()
	server.CertStorage.SetCertificatesDir(certDir)
	err := server.CertStorage.LoadCertificates(certDir)
	if err != nil {
		log.Fatal(err)
	}

	return server
}

func (ps *ProxyServer) ListenAndServe(addr string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Не удается принять соединение")
			continue
		}

		go func() {
			proxy := NewProxyConn(ps, conn)
			proxy.ServeTCP()
		}()
	}
}

type ProxyConn struct {
	ps *ProxyServer

	serverConn net.Conn
	clientConn net.Conn
	isTLS      bool

	fakeCertificates *CertStorage
}

func NewProxyConn(ps *ProxyServer, clientConn net.Conn) *ProxyConn {
	server := new(ProxyConn)

	server.ps = ps
	server.fakeCertificates = ps.CertStorage

	server.clientConn = clientConn
	return server
}

func (p *ProxyConn) ServeTCP() {
	defer p.clientConn.Close()

	buffer := bufio.NewReader(p.clientConn)

	for {
		p.clientConn.SetDeadline(time.Now().Add(60 * time.Second))

		request, err := http.ReadRequest(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Клиент закрыл соединение")
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Println("Соединение закрыто по таймауту")
			} else {
				log.Printf("Невозможно прочитать запрос: %v\n", err)
			}
			return
		}

		if request.Method == http.MethodConnect { // got an HTTPS and serve CONNECT once
			err = p.establishTLS()

			if err != nil {
				log.Printf("Проблема с установкой TLS соединения: %v\n", err)
				return
			}

			// need to update buffer p.clientConn to new p.clientConn
			// even update p.clientConn can't help to update p.clientConn in buffer
			// we should to it explicitly
			buffer = bufio.NewReader(p.clientConn)

			continue // need to read request to make reader of conn available (isn't blocked)
		}

		reqReader, err := ModifyRequest(request)
		if err != nil {
			log.Printf("Не удалось прочитать отредактированный запрос: %v\n", err)
			return
		}

		err = p.ConnectToServer(request.Host)
		if err != nil {
			log.Printf("Проблема с обработой HTTP: %v\n", err)
			return
		}

		done := make(chan struct{}, 2)
		go func() {
			io.Copy(p.serverConn, reqReader)
			done <- struct{}{}
		}()
		go func(){
			io.Copy(p.clientConn, p.serverConn)
			done <- struct{}{}
		}()
		<-done

		log.Println("Запрос обработан")
	}
}

func (p *ProxyConn) establishTLS() error {
	log.Println("Установка TLS соединения с клиентом")
	p.clientConn.Write([]byte(TLSConnectionEstablishStartLine))

	conf := &tls.Config{
		NextProtos: []string{"http/1.1"},
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := info.ServerName
			cert, err := p.fakeCertificates.GetOrCreateCertificate(serverName)
			if err != nil {
				return nil, err
			}
			return cert, nil
		},
	}

	TLSConn := tls.Server(p.clientConn, conf)
	if err := TLSConn.Handshake(); err != nil {
		log.Println("При установке TLS соединения с клиентом возникла ошибка: ", err)
		return err
	}

	log.Println("TLS соединение с клиентом установлено")

	p.isTLS = true
	p.clientConn = TLSConn
	return nil
}

func (p *ProxyConn) ConnectToServer(Host string) error {
	if p.serverConn != nil {
		return nil
	}

	if p.isTLS {
		log.Println("Обработка HTTPS запроса: ", Host)
		serverConn, err := tls.Dial("tcp", Host+":443", &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"http/1.1"},
		})
		if err != nil {
			return err
		}
		p.serverConn = serverConn
	} else {
		log.Println("Обработка HTTP запроса: ", Host)
		serverConn, err := net.Dial("tcp", Host+":80")
		if err != nil {
			return err
		}
		p.serverConn = serverConn
	}

	return nil
}

// Returns modified request as a io.Reader.
// Deletes Proxy-Connection header
func ModifyRequest(r *http.Request) (io.Reader, error) {
	r.Header.Del("Proxy-Connection")

	// remove absolute URL
	r.RequestURI = ""

	var buf bytes.Buffer
	err := r.Write(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}
