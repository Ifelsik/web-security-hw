package server

import (
	"fmt"
	"net/http"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Got request. URL %s\n", r.URL.Path)
}

// TODO: implement panic and logging middleware
// func PanicMiddleWare()