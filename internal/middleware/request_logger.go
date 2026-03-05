package middleware

import (
	"net/http"
	"time"

	"shopify-app-authentication/internal/logger"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger logs each HTTP request using the application logger.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(ww, r)

		logger.Log.Info().
			Str("method", r.Method).
			Str("path", r.URL.RequestURI()).
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Str("remote", r.RemoteAddr).
			Msg("request")
	})
}
