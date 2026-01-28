package server

import (
	"context"
	"fmt"
	"net/http"
)

type Config struct {
	Host string
	Port uint32
}

type Server struct {
	conf *Config
	srv  *http.Server
}

func NewServer(c Config, router http.Handler) *Server {
	if c.Host == "" || c.Port == 0 {
		panic("host or port is not specified")
	}
	return &Server{
		conf: &c,
		srv: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", c.Host, c.Port),
			Handler: router,
		},
	}
}

func (s *Server) ListenAndServe() error {
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown() error {
	// TODO: implement context
	return s.srv.Shutdown(context.TODO())
}

// String implements Stringer interface
func (s *Server) String() string {
	return fmt.Sprintf("%s:%d", s.conf.Host, s.conf.Port)
}
