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
	"time"

	"github.com/google/uuid"
	"github.com/ifelsik/mitm-proxy/internal/utils/httputil"
	"github.com/ifelsik/mitm-proxy/internal/utils/promise"
	"github.com/ifelsik/mitm-proxy/internal/utils/request"
	"go.uber.org/zap"
)

// Address that proxy listen for.
const listenAddress = "127.0.0.1"

type Proxy struct {
	log      *zap.SugaredLogger
	listener net.Listener

	isStopped atomic.Bool

	pool *BytePool
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
		pool:      pool,
	}, nil
}

func (p *Proxy) Run() {
	p.log.Infoln("Proxy listen at", p.listener.Addr())
	for !p.isStopped.Load() {
		conn, err := p.listener.Accept()
		if err != nil {
			p.log.Errorf("accept new TCP connection: %s", err)
			continue
		}
		serve := p.panicWrapper(p.loggingWrapper(p.serveConn))
		go serve(context.TODO(), conn)
	}
}

// Shutdown by default stops to accept new incoming connections.
// All existent connections are handled till they close.
// If ctx context timeout exceeds method throws error context.DeadlineExceeded
func (p *Proxy) Shutdown(ctx context.Context) error {
	p.isStopped.Store(true)
	_ = p.listener.Close()

	return nil
}

const stackTraceBuffSize = 4 * 1024

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

func (p *Proxy) loggingWrapper(next func(context.Context, net.Conn)) func(context.Context, net.Conn) {
	uuid, _ := uuid.NewV7()
	log := p.log.With(
		"connection_id", uuid.String(),
	)

	return func(ctx context.Context, c net.Conn) {
		log.Debug("logging wrapper")
		start := time.Now()
		next(ctx, c)
		alive := time.Since(start)

		log.With("alive", alive).Info("connection closed")
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

func (p *Proxy) handleTunnel(ctx context.Context, inBuffRW, outBuffRW *bufio.ReadWriter) error {
	for {
		req, err := p.readRequest(inBuffRW.Reader)
		if err != nil {
			return err
		}

		modifiedReq, err := p.modifyRequest(req)
		if err != nil {
			return err
		}

		copyBuff := p.pool.Get()
		_, err = io.CopyBuffer(outBuffRW.Writer, modifiedReq, copyBuff)
		if err != nil {
			return err
		}
		p.pool.Put(copyBuff)

		_ = outBuffRW.Flush()

		resp, err := http.ReadResponse(outBuffRW.Reader, req)
		if err != nil {
			return err
		}

		_ = resp.Write(inBuffRW)
		err = inBuffRW.Flush()
		if err != nil {
			return err
		}
	}
	return nil
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

	host, err := httputil.GetHost(req)
	if err != nil {
		p.log.Errorf("get request host: %s", err)
		return
	}
	outConn, err := net.Dial("tcp", host.String())
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

	copyBuf := p.pool.Get()
	_, err = io.CopyBuffer(outBuffRW, result.Value, copyBuf)
	if err != nil {
		p.log.Errorf("copy request to outbound conn: %s", err)
		return
	}
	p.pool.Put(copyBuf)

	_ = outBuffRW.Flush()

	resp, err := p.readResponse(outBuffRW.Reader, req)
	if err != nil {
		p.log.Errorf("read server response: %s", err)
		return
	}
	_ = resp.Write(inBuffRW)
	_ = inBuffRW.Flush()

	err = p.handleTunnel(ctx, inBuffRW, outBuffRW)
	if err != nil {
		p.log.Errorf("handle client-server tunnel: %s", err)
		return
	}
}
