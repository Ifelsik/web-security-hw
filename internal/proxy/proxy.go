package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"

	"github.com/ifelsik/mitm-proxy/internal/utills/promise"
	"github.com/ifelsik/mitm-proxy/internal/utills/request"
	"go.uber.org/zap"
)

// Address that proxy listen for.
const listenAddress = "127.0.0.1"

type Proxy struct {
	log      *zap.SugaredLogger
	listener net.Listener

	isStopped atomic.Bool
}

func NewProxy(log *zap.SugaredLogger, port string) (*Proxy, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%s", listenAddress, port))
	if err != nil {
		return nil, fmt.Errorf("create proxy: %w", err)
	}

	return &Proxy{
		log:       log,
		listener:  l,
		isStopped: atomic.Bool{},
	}, nil
}

func (p *Proxy) Run(ctx context.Context) {
	go p.monitorCancel(ctx)
	p.log.Infoln("Proxy listen at", p.listener.Addr())
	for !p.isStopped.Load() {
		conn, err := p.listener.Accept()
		if err != nil {
			p.log.Errorln("accept new TCP connection:", err)
			continue
		}
		serve := p.panicWrapper(p.serveConn)
		go serve(ctx, conn)
	}
}

func (p *Proxy) monitorCancel(ctx context.Context) {
	<-ctx.Done()
	p.log.Info("Proxy is stopping")
	p.isStopped.Store(true)
	p.listener.Close()
}

const stackTraceBuffSize = 1024

func (p *Proxy) panicWrapper(next func(context.Context, net.Conn)) func(context.Context, net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			p.log.Warnln("panic recovered", err)

			buf := make([]byte, stackTraceBuffSize)

			n := runtime.Stack(buf, false)
			for n == len(buf) {
				buf = make([]byte, len(buf)*2)
				n = runtime.Stack(buf, false)
			}
			fmt.Printf("Stack trace: %s\n", buf[:n])
		}
	}()

	return func(ctx context.Context, c net.Conn) {
		p.log.Debug("panic recover wrapper")
		next(ctx, c)
	}
}

func (p *Proxy) readRequest(bufRd *bufio.Reader) (*http.Request, error) {
	req, err := http.ReadRequest(bufRd)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("client has closed the connection: %w", err)
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("connection broken: %s", err)
		}
		return nil, fmt.Errorf("read HTTP request from conn: %s", err)
	}
	return req, nil
}

func (p *Proxy) modifyRequest(req *http.Request) (*request.HTTPRequest, error) {
	r, err := request.ParseRawRequest(req)
	if err != nil {
		return nil, fmt.Errorf("parse request: %w", err)
	}
	// TODO: modify request, e.g. change User-Agent
	return r, nil
}

func (p *Proxy) readResponse(bufRd *bufio.Reader, req *http.Request) (*http.Response, error) {
	resp, err := http.ReadResponse(bufRd, req)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("remote server has closed the connection: %w", err)
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("connection broken: %s", err)
		}
		return nil, fmt.Errorf("read HTTP response from server conn: %s", err)
	}
	return resp, nil
}

func (p *Proxy) serveConn(ctx context.Context, inConn net.Conn) {
	defer func() {
		err := inConn.Close()
		p.log.Debug("inbound conn closed:", err)
	}()
	inBuffRW := bufio.NewReadWriter(
		bufio.NewReader(inConn),
		bufio.NewWriter(inConn),
	)

	req, err := p.readRequest(inBuffRW.Reader)
	if err != nil {
		// TODO: may be not to return. First request can be broken due transport.
		p.log.Error("read client request:", err)
		return
	}

	future := promise.Promise(func() (*request.HTTPRequest, error) {
		return p.modifyRequest(req)
	})

	outConn, err := net.Dial("tcp", req.Host+":80")
	if err != nil {
		// TODO: retry connect
		p.log.Error("establish outbound TCP conn:", err)
		return
	}
	defer func() {
		err := outConn.Close()
		p.log.Debug("outbound conn closed:", err)
	}()

	outBuffRW := bufio.NewReadWriter(
		bufio.NewReader(outConn),
		bufio.NewWriter(outConn),
	)

	result := <-future
	if result.Err != nil {
		p.log.Errorf("modify request: %s", result.Err)
	}

	_, err = io.Copy(outBuffRW, result.Value)
	if err != nil {
		p.log.Errorf("copy request to outbound conn: %s", err)
		return
	}

	resp, err := p.readResponse(outBuffRW.Reader, req)
	if err != nil {
		p.log.Errorf("read server response: %s", err)
		return
	}
	resp.Write(inBuffRW)
	inBuffRW.Flush()
}
