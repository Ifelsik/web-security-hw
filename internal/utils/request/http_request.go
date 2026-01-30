package request

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var ErrRequestIsNil = errors.New("request is nil")
var ErrMetaIsBad = errors.New("unable to form HTTP message metadata")

type HTTPRequest struct {
	Host    Host // authority according RFC3986
	Method  string
	Path    string // path+query+fragment according RFC3986 (see section 3)
	Headers http.Header
	Body    io.ReadCloser
	IsTLS   bool

	buff         bytes.Buffer // used for HTTP message metadata
	metaPrepared bool
	dataSize     int // payload size
	dataRead     int // read offset
}

func ParseRawRequest(r *http.Request) (*HTTPRequest, error) {
	if r == nil {
		return nil, fmt.Errorf("parse request: %w", ErrRequestIsNil)
	}

	result := &HTTPRequest{
		Host:         NewHost(r.Host),
		Method:       r.Method,
		Path:         r.URL.RequestURI() + r.URL.Fragment,
		Headers:      r.Header,
		Body:         r.Body,
		metaPrepared: false,
	}

	return result, nil
}

func (hr *HTTPRequest) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if !hr.metaPrepared {
		err := hr.prepareMeta()
		if err != nil {
			return 0, fmt.Errorf("read request: %w", ErrMetaIsBad)
		}
		hr.metaPrepared = true
	}
	
	n := 0
	if hr.buff.Len() > 0 {
		n, err := hr.buff.Read(p)
		if err != nil && err != io.EOF {
			return n, fmt.Errorf("read request meta from buff: %w", err)
		}
		hr.dataRead += n
		if len(p) == 0 {
			return n, nil
		}
	}
	k, err := hr.Body.Read(p)
	if err != nil && err != io.EOF {
		_ = hr.Body.Close()
		return k, fmt.Errorf("read request body: %w", err)
	}
	if err == io.EOF {
		_ = hr.Body.Close()
	}
	return n + k, err
}

func (hr *HTTPRequest) prepareMeta() error {
	_, _ = fmt.Fprintf(&hr.buff, "%s %s HTTP/1.1\r\nHost: %s\r\n", hr.Method, hr.Path, hr.Host.String())
	err := hr.Headers.Write(&hr.buff)
	if err != nil {
		return err
	}
	_, _ = hr.buff.WriteString("\r\n")
	hr.dataSize = hr.buff.Len()
	return nil
}

func (hr *HTTPRequest) PrepareClientRequest(ctx context.Context) (*http.Request, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		hr.Method,
		hr.url(),
		hr.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare client request: %w", err)
	}

	req.Header = hr.Headers

	return req, nil
}

func (hr *HTTPRequest) url() string {
	var scheme string
	switch hr.Host.Port {
	case 80:
		scheme = "http"
	case 443:
		scheme = "https"
	default:
		scheme = "http"
	}

	return scheme + "://" + hr.Host.Domain + hr.Path
}

type Host struct {
	Domain string
	Port   uint16
}

// NewHost creates Host. It takes host in format <host:port>
func NewHost(host string) Host {
	var h Host
	fmt.Sscanf(host, "%s:%d\n", &h.Domain, &h.Port)
	return h
}

// String implements Stringer interface
func (h *Host) String() string {
	return fmt.Sprintf("%s:%d", h.Domain, h.Port)
}
