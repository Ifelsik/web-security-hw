package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Host struct {
	Addr string
	Port uint
}

func (h *Host) String() string {
	return fmt.Sprintf("%s:%d", h.Addr, h.Port)
}

const addrPortDelim = ":"

const portHTTP = 80

func parseHost(host string) (Host, error) {
	host = strings.TrimSpace(host)

	split := strings.Split(host, addrPortDelim)
	if !(len(split) == 1 || len(split) == 2) {
		return Host{}, fmt.Errorf("not a host: %s", host)
	}

	var parsedHost Host
	parsedHost.Addr = strings.TrimSpace(split[0])
	if len(split) == 1 || (len(split) == 2 && len(split[1]) == 0) {
		// if got HTTP request not HTTPS
		// port is omitted
		parsedHost.Port = portHTTP
	} else {
		port := strings.TrimSpace(split[1])
		portUint, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return Host{}, fmt.Errorf("invalid port: %s in %s", port, host)
		}
		parsedHost.Port = uint(portUint)
	}
	return parsedHost, nil
}

var ErrNilRequest = errors.New("request is nil")

func GetHost(r *http.Request) (Host, error) {
	if r == nil {
		return Host{}, ErrNilRequest
	}
	return parseHost(r.Host)
}
