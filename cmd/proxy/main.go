package main

import (
	"net/http"
	"os"

	"github.com/Ifelsik/web-security-hw/internal"
	"github.com/Ifelsik/web-security-hw/internal/delivery"
	"github.com/Ifelsik/web-security-hw/internal/repository"
	"github.com/Ifelsik/web-security-hw/internal/usecase"
	"github.com/joho/godotenv"
)

const address = "0.0.0.0:8080"
const certDir = "certificates"

func main() {
	log := internal.NewLogger()

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Info(".env file read")

	dbConn, err := repository.ConnectPGSQL(
		"postgres",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		"5432",
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Connected to database")

	repo := repository.NewORMrepository(dbConn, log)
	usecase := usecase.NewHistoryUseCase(repo, log)
	handlers := delivery.NewHistoryHandlers(usecase, log)

	go func() {
		r := internal.HandleRoutes(handlers)

		log.Info("Web API server is starting on 0.0.0.0:8000")
		http.ListenAndServe("0.0.0.0:8000", r)
	}()

	ProxyServer := internal.NewProxyServer(certDir, usecase)

	log.Info("Proxy server is starting on ", address)
	ProxyServer.ListenAndServe(address)
}
