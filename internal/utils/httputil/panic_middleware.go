package httputil

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
