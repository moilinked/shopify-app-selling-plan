package middleware

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

// ShopifySessionTokenMiddleware validates the Shopify session token (JWT)
// from the Authorization header. On success it stores the claims and
// the resolved shop domain in the request context.
func ShopifySessionTokenMiddleware(apiKey, apiSecret string, debugAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				if debugAuth {
					log.Printf("session_jwt: missing Authorization header path=%s", requestPath)
				}
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				if debugAuth {
					log.Printf("session_jwt: invalid Authorization format path=%s", requestPath)
				}
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if tokenString == "" {
				if debugAuth {
					log.Printf("session_jwt: missing bearer token path=%s", requestPath)
				}
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
				if debugAuth {
					log.Printf("session_jwt: parse/verify failed path=%s err=%v token_valid=%t", requestPath, err, token != nil && token.Valid)
				}
				http.Error(w, "invalid session token", http.StatusUnauthorized)
				return
			}

			aud, err := claims.GetAudience()
			if err != nil || len(aud) == 0 || aud[0] != apiKey {
				if debugAuth {
					log.Printf("session_jwt: invalid audience path=%s aud=%v expected=%s err=%v", requestPath, aud, apiKey, err)
				}
				http.Error(w, "invalid token audience", http.StatusUnauthorized)
				return
			}

			iss, err := claims.GetIssuer()
			if err != nil || iss == "" {
				if debugAuth {
					log.Printf("session_jwt: missing issuer path=%s err=%v", requestPath, err)
				}
				http.Error(w, "missing issuer", http.StatusUnauthorized)
				return
			}

			destRaw, ok := claims["dest"]
			if !ok {
				if debugAuth {
					log.Printf("session_jwt: missing destination claim path=%s", requestPath)
				}
				http.Error(w, "missing destination", http.StatusUnauthorized)
				return
			}
			dest, ok := destRaw.(string)
			if !ok || strings.TrimSpace(dest) == "" {
				if debugAuth {
					log.Printf("session_jwt: invalid destination claim path=%s dest=%v", requestPath, destRaw)
				}
				http.Error(w, "invalid destination", http.StatusUnauthorized)
				return
			}

			issHost := hostFromIssuer(iss)
			destHost, err := extractShopDomain(dest)
			if err != nil || issHost == "" || destHost == "" || issHost != destHost {
				if debugAuth {
					log.Printf("session_jwt: issuer/dest mismatch path=%s iss_host=%s dest_host=%s err=%v", requestPath, issHost, destHost, err)
				}
				http.Error(w, "issuer and destination mismatch", http.StatusUnauthorized)
				return
			}

			if debugAuth {
				log.Printf("session_jwt: verified path=%s iss=%s dest=%s sub=%v", requestPath, issHost, destHost, claims["sub"])
				LogClaims(claims)
			}
			ctx := context.WithValue(r.Context(), shopifyClaimsContextKey, claims)
			ctx = reconcileShopContext(ctx, destHost)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
