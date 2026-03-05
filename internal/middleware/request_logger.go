package middleware

import (
	"net/http"
	"time"

	"shopify-app-authentication/internal/logger"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// RequestLogger 记录每个 HTTP 请求的方法、路径、状态码、响应大小和耗时。
// 替代 chi 内置的 middleware.Logger，统一使用 zerolog 输出。
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
