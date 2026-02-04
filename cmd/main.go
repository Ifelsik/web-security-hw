package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ifelsik/mitm-proxy/internal/proxy"
	"github.com/ifelsik/mitm-proxy/internal/utils/logger"
)

func main() {
	ctx := context.Background()
	stopCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log := logger.NewLogger()

	proxy, err := proxy.NewProxy(log, "8080")
	if err != nil {
		log.Fatalf("init proxy: %s", err)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		proxy.Run()
	})
	wg.Go(func() {
		<-stopCtx.Done()
		defer cancel()
		log.Info("Stopping server")
		err = proxy.Shutdown(context.TODO())
		if err != nil {
			log.Errorf("proxy shutdown: %s", err)
		}
	})

	wg.Wait()
	log.Info("Server stopped")
}
