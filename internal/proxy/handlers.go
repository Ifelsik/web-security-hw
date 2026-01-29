package proxy

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/ifelsik/mitm-proxy/internal/utills/request"
	"go.uber.org/zap"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	req, err := request.ParseRawRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	clientReq, err := req.PrepareClientRequest(context.TODO())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	resp, err := makeClientRequest(clientReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	_ = resp.Write(w)
}

const stackTraceBuffSize = 1024

type PanicMiddleware struct {
	log *zap.SugaredLogger
}

func (pm *PanicMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				pm.log.Warnln("panic recovered", r)

				buf := make([]byte, stackTraceBuffSize)

				n := runtime.Stack(buf, false)
				for n == len(buf) {
					buf = make([]byte, len(buf)*2)
					n = runtime.Stack(buf, false)
				}
				fmt.Printf("Stack trace: %s\n", buf[:n])
			}
		}()

		pm.log.Debug("entering panic middleware")
		next.ServeHTTP(w, r)
	})
}

type LoggingMiddleware struct {
	log *zap.SugaredLogger
}

func (lm *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lm.log.Debug("entering logging middleware")

		log := lm.log.With(
			"method", r.Method,
			"URL", r.URL.Path,
			"request_id", uuid.New(),
		)
		log.Debug("new incoming request")

		start := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(start)

		log = log.With(
			"elapsed", elapsed,
		)
		log.Info("request handled")
	})
}
