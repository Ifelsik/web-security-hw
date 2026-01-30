package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ifelsik/mitm-proxy/internal/proxy"
	"github.com/ifelsik/mitm-proxy/internal/utills/logger"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log := logger.NewLogger()

	proxy, err := proxy.NewProxy(log, "8080")
	if err != nil {
		log.Fatalf("init proxy: %s", err)
	}

	proxy.Run(ctx)

	<-ctx.Done()

	log.Info("Server stopped")
}
