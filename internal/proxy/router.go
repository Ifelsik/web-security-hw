package proxy

import (
	"net/http"

	"go.uber.org/zap"
)

func NewRouter(log *zap.SugaredLogger) http.Handler {
	loggingMw := &LoggingMiddleware{
		log: log,
	}
	panicMw := &PanicMiddleware{
		log: log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", ProxyHandler)

	return panicMw.Middleware(loggingMw.Middleware(mux))
}
