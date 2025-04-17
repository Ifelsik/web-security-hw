package main

import (
	"log"
)

const address = "0.0.0.0:8080"
const certDir = "certificates"

func main() {
	log.Println("Запуск сервера", address)
	ProxyServer := NewProxyServer(certDir)
	ProxyServer.ListenAndServe(address)
}
