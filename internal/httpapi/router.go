package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"shopify-app-authentication/internal/config"
)

type contextKey string

const shopifyClaimsContextKey = contextKey("shopifyClaims")

func NewRouter(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(AllowAllCORS)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("pong"))
	})

	r.With(ShopifySessionTokenMiddleware(cfg.ShopifyAPIKey, cfg.ShopifyAPISecret)).Get("/protected/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		claims, _ := r.Context().Value(shopifyClaimsContextKey).(jwt.MapClaims)
		body, _ := json.Marshal(map[string]interface{}{
			"ok":     true,
			"claims": claims,
		})
		_, _ = w.Write(body)
	})

	return r
}

func AllowAllCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(shopifyClaimsContextKey).(jwt.MapClaims)
	return claims, ok
}

func LogClaims(claims jwt.MapClaims) {
	claimsJSON, _ := json.Marshal(claims)
	log.Printf("shopify_session_token claims=%s", claimsJSON)
}
