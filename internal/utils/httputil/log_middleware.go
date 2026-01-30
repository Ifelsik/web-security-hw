package httputil

import (
	"fmt"
	"net/http"
	"runtime"

	"go.uber.org/zap"
)

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
