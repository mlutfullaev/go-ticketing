package log

import (
	"log/slog"
	"net/http"
	"time"
)

func LoggerMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		logger.Info("request", r.Method, r.URL.Path, time.Since(start))
	})
}
