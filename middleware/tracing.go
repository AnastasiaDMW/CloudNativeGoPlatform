package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
)

func Tracing(next http.Handler) http.Handler {
	tracer := otel.Tracer("go-postgres")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(
			r.Context(),
			r.Method+" "+r.URL.Path,
		)
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
