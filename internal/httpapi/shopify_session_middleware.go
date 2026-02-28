package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func extractShopDomain(shopURL string) (string, error) {
	shopURL = strings.TrimSpace(shopURL)
	shopURL = strings.TrimPrefix(shopURL, "https://")
	shopURL = strings.TrimPrefix(shopURL, "http://")
	shopURL = strings.TrimSuffix(shopURL, "/")
	if shopURL == "" {
		return "", errors.New("empty shop domain")
	}
	return strings.ToLower(shopURL), nil
}

func hostFromIssuer(issuer string) string {
	issuer = strings.TrimSpace(strings.ToLower(issuer))
	issuer = strings.TrimPrefix(issuer, "https://")
	issuer = strings.TrimPrefix(issuer, "http://")
	parts := strings.SplitN(issuer, "/", 2)
	return parts[0]
}

func ShopifySessionTokenMiddleware(apiKey, apiSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if tokenString == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			const tokenLeeway = 5 * time.Second
			claims := jwt.MapClaims{}
			parser := jwt.NewParser(
				jwt.WithLeeway(tokenLeeway),
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			)
			token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(apiSecret), nil
			})
			if err != nil || !token.Valid {
				log.Printf("invalid session token: err=%v token_valid=%t", err, token != nil && token.Valid)
				http.Error(w, "invalid session token", http.StatusUnauthorized)
				return
			}

			aud, err := claims.GetAudience()
			if err != nil || len(aud) == 0 || aud[0] != apiKey {
				http.Error(w, "invalid token audience", http.StatusUnauthorized)
				return
			}

			iss, err := claims.GetIssuer()
			if err != nil || iss == "" {
				http.Error(w, "missing issuer", http.StatusUnauthorized)
				return
			}

			destRaw, ok := claims["dest"]
			if !ok {
				http.Error(w, "missing destination", http.StatusUnauthorized)
				return
			}
			dest, ok := destRaw.(string)
			if !ok || strings.TrimSpace(dest) == "" {
				http.Error(w, "invalid destination", http.StatusUnauthorized)
				return
			}

			issHost := hostFromIssuer(iss)
			destHost, err := extractShopDomain(dest)
			if err != nil || issHost == "" || destHost == "" || issHost != destHost {
				http.Error(w, "issuer and destination mismatch", http.StatusUnauthorized)
				return
			}

			LogClaims(claims)
			ctx := context.WithValue(r.Context(), shopifyClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
