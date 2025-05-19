package api

import (
	"net/http"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/go-chi/chi/v5/middleware"
)

// LoggerMiddleware returns a middleware that logs requests using the provided logger.
func LoggerMiddleware(logger userprefs.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t0 := time.Now()
			defer func() {
				logger.Info("Served request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"latency_ms", float64(time.Since(t0).Microseconds())/1000.0,
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
