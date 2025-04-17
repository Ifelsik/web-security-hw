package main

import (
	"log"
	"github.com/Ifelsik/web-security-hw/internal"
)

const address = "0.0.0.0:8080"
const certDir = "certificates"

func main() {
	log.Println("Запуск сервера", address)
	ProxyServer := internal.NewProxyServer(certDir)
	ProxyServer.ListenAndServe(address)
}
