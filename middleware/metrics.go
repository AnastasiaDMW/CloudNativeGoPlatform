package middleware

import (
	"net/http"
	"time"

	"go-postgres/telemetry"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Не собираем HTTP-метрики для самого Prometheus endpoint.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		telemetry.HTTPRequestsInFlight.Inc()
		defer telemetry.HTTPRequestsInFlight.Dec()

		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(sw, r)

		telemetry.ObserveRequest(
			r.Method,
			r.URL.Path,
			sw.status,
			time.Since(start),
		)
	})
}