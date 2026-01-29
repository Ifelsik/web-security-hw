package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ifelsik/mitm-proxy/internal/proxy"
	"github.com/ifelsik/mitm-proxy/internal/utills/logger"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log := logger.NewLogger()

	router := proxy.NewRouter(log)
	srvConf := proxy.Config{
		Host: "0.0.0.0",
		Port: 8080,
	}
	srv := proxy.NewServer(srvConf, router)

	var wg sync.WaitGroup
	wg.Go(func() {
		log.Infof("Starting proxy server at %s", srv)
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalf("proxy server: %s\n", err)
		}
	})
	wg.Go(func() {
		_ = <-ctx.Done()
		err := srv.Shutdown()
		log.Info("Server is shutting down...")
		if err != nil {
			log.Infof("proxy server shutdown: %s\n", err)
		}
	})

	wg.Wait()
	log.Info("Server stopped")
}
