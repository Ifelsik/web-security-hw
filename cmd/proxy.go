package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type LogWriter struct {
	prefix string
}

func (lw *LogWriter) Write(p []byte) (int, error) {
	fmt.Printf("[%s] %s", lw.prefix, p)
	return len(p), nil
}

const (
	TLSConnectionEstablishStartLine = "HTTP/1.1 200 Connection established\r\n\r\n"
)

type ProxyServer struct {
	serverConn net.Conn
	clientConn net.Conn
	isTLS      bool

	crtToClient tls.Certificate
}

func NewProxyServer(clientConn net.Conn) *ProxyServer {
	server := new(ProxyServer)

	crtToClient, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}
	server.crtToClient = crtToClient
	server.clientConn = clientConn
	return server
}

func (p *ProxyServer) ServeTCP() {
	defer p.clientConn.Close()

	var HTTPMessage bytes.Buffer
	buffer := bufio.NewReader(io.TeeReader(p.clientConn, &HTTPMessage))

	request, err := http.ReadRequest(buffer)
	log.Printf("%+v",request)
	if err != nil {
		log.Printf("Невозможно прочитать запрос: %v\n", err)
		return
	}

	if request.Method == http.MethodConnect { // got an HTTPS
		err = p.establishTLS()
	
		if err != nil {
			log.Printf("Проблема с установкой TLS соединения: %v\n", err)
			return
		}

		HTTPMessage.Reset()
		buffer = bufio.NewReader(io.TeeReader(p.clientConn, &HTTPMessage))
		request, err = http.ReadRequest(buffer)
		if err != nil {
			log.Printf("Невозможно прочитать HTTPS-запрос после TLS: %v\n", err)
			return
		}
	}
	log.Printf("%+v",request)

	reqReader, err := ModifyRequest(&HTTPMessage)
	if err != nil {
		log.Printf("Неудалось прочитать отредактированный запрос: %v\n", err)
		return
	}

	err = p.handleHTTP(request.Host)
	if err != nil {
		log.Printf("Проблема с обработой HTTP: %v\n", err)
		return
	}
	
	go io.Copy(p.serverConn, reqReader)
	io.Copy(p.clientConn, p.serverConn)


	log.Println("Запрос обработан")
	
	p.serverConn.Close()
}

func (p *ProxyServer) establishTLS() error {
	log.Println("Установка TLS соединения с клиентом")
	p.clientConn.Write([]byte(TLSConnectionEstablishStartLine))

	conf := &tls.Config{
		Certificates: []tls.Certificate{p.crtToClient},
		NextProtos:   []string{"http/1.1"},
	}

	TLSConn := tls.Server(p.clientConn, conf)
	if err := TLSConn.Handshake(); err != nil {
		log.Println("При установке TLS соединения возникла ошибка: ", err)
		return err
	}

	log.Println("TLS соединение с клиентом установлено")

	p.isTLS = true
	p.clientConn = TLSConn
	return nil
}

func (p *ProxyServer) handleHTTP(Host string) error {
	if p.isTLS {
		log.Println("Обработка HTTPS запроса")
		serverConn, err := tls.Dial("tcp", Host + ":443", &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"http/1.1"},
		})
		if err != nil {
			return err
		}
		p.serverConn = serverConn
	} else {
		log.Println("Обработка HTTP запроса")
		serverConn, err := net.Dial("tcp", Host + ":80")
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
	isStartLine := false

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

	// log.Println(&buffer)
	return &buffer, nil
}
