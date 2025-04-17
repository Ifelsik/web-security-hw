package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Ifelsik/web-security-hw/internal"
)

const (
	TLSConnectionEstablishStartLine = "HTTP/1.1 200 Connection established\r\n\r\n"
)

type ProxyServer struct {
	CertStorage *internal.CertStorage
}

func NewProxyServer(certDir string) *ProxyServer {
	server := new(ProxyServer)

	server.CertStorage = internal.NewCertStorage()
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

	fakeCertificates *internal.CertStorage
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

	var HTTPMessage bytes.Buffer
	buffer := bufio.NewReader(io.TeeReader(p.clientConn, &HTTPMessage))

	for {
		p.clientConn.SetDeadline(time.Now().Add(30 * time.Second))

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

		if request.Method == http.MethodConnect{ // got an HTTPS and serve CONNECT once
			err = p.establishTLS()

			if err != nil {
				log.Printf("Проблема с установкой TLS соединения: %v\n", err)
				return
			}

			HTTPMessage.Reset()
			buffer = bufio.NewReader(io.TeeReader(p.clientConn, &HTTPMessage))
			continue // need to read request to make reader of conn available (isn't blocked)
		}

		reqReader, err := ModifyRequest(&HTTPMessage)
		if err != nil {
			log.Printf("Неудалось прочитать отредактированный запрос: %v\n", err)
			return
		}

		err = p.ConnectToServer(request.Host)
		if err != nil {
			log.Printf("Проблема с обработой HTTP: %v\n", err)
			return
		}

		go io.Copy(p.serverConn, reqReader)
		io.Copy(p.clientConn, p.serverConn)

		HTTPMessage.Reset()
		log.Println("Запрос обработан")
	}
}

func (p *ProxyConn) establishTLS() error {
	log.Println("Установка TLS соединения с клиентом")
	p.clientConn.Write([]byte(TLSConnectionEstablishStartLine))

	conf := &tls.Config{
		NextProtos:   []string{"http/1.1"},
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
func ModifyRequest(request io.Reader) (io.Reader, error) {
	var buffer bytes.Buffer
	isStartLine := true

	scanner := bufio.NewScanner(request)
	for scanner.Scan() {
		line := scanner.Text()
		if isStartLine {
			isStartLine = false

			// <METHOD> <url> HTTP/<version>
			//    0       1        2
			tokens := strings.Split(line, " ")
			if len(tokens) != 3 {
				return nil, errors.New("invalid start line")
			}

			url, err := url.Parse(tokens[1])
			if err != nil {
				return nil, err
			}

			line = tokens[0] + " " + url.RequestURI() + " " + tokens[2]
		}
		if strings.Contains(line, "Proxy-Connection") {
			continue
		}
		_, err := buffer.WriteString(line + "\r\n")
		if err != nil {
			return nil, err
		}
	}

	buffer.WriteString("\r\n")

	return &buffer, nil
}
