package internal

import (
	"github.com/Ifelsik/web-security-hw/internal/delivery"

	"github.com/gorilla/mux"
)

func HandleRoutes(handlers *delivery.ProxyControlHandlers) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/requests", handlers.GetRequestsHistory).Methods("GET")
	r.HandleFunc("/requests/{id:[0-9]+}", handlers.GetRequestByID).Methods("GET")

	// TODO: repeat/id and scan/id
	r.HandleFunc("/repeat/{id:[0-9]+}", handlers.RepeatRequest).Methods("GET")
	r.HandleFunc("/scan/{id:[0-9]+}", handlers.Scan).Methods("GET")
	
	return r
}
