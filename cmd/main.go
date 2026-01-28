package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ifelsik/mitm-proxy/internal/proxy/server"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srvConf := server.Config{
		Host: "0.0.0.0",
		Port: 8080,
	}
	srv := server.NewServer(srvConf)

	var wg sync.WaitGroup
	wg.Go(func() {
		log.Printf("Starting server at %s", srv)
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalf("server: %s\n", err)
		}
	})
	wg.Go(func () {
		_ = <- ctx.Done()
		err := srv.Shutdown()
		log.Println("Server is shutting down...")
		if err != nil {
			log.Printf("server shutdown: %s\n", err)
		}
	})

	wg.Wait()
	log.Println("Server stopped")
}
