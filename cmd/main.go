package main

import (
	"log"
	"net"
)

const (
	network = "tcp"
	address = ":8080"
)

func main() {
	log.Println("Запуск сервера", address)
	listner, err := net.ListenTCP(network, &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080})
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Printf("Не удается принять соединение")
			continue
		}

		go func() {
			proxy := NewProxyServer(conn)
			proxy.ServeTCP()
		}()
	}
}
