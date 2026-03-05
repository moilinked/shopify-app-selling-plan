package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"shopify-app-authentication/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

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
					logger.Log.Debug().Str("path", requestPath).Msg("session_jwt: missing Authorization header")
				}
				http.Error(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				if debugAuth {
					logger.Log.Debug().Str("path", requestPath).Msg("session_jwt: invalid Authorization format")
				}
				http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
			if tokenString == "" {
				if debugAuth {
					logger.Log.Debug().Str("path", requestPath).Msg("session_jwt: missing bearer token")
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
					logger.Log.Warn().Str("path", requestPath).Err(err).
						Bool("token_valid", token != nil && token.Valid).
						Msg("session_jwt: parse/verify failed")
				}
				http.Error(w, "invalid session token", http.StatusUnauthorized)
				return
			}

			aud, err := claims.GetAudience()
			if err != nil || len(aud) == 0 || aud[0] != apiKey {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).
						Strs("aud", aud).Str("expected", apiKey).Err(err).
						Msg("session_jwt: invalid audience")
				}
				http.Error(w, "invalid token audience", http.StatusUnauthorized)
				return
			}

			iss, err := claims.GetIssuer()
			if err != nil || iss == "" {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Err(err).Msg("session_jwt: missing issuer")
				}
				http.Error(w, "missing issuer", http.StatusUnauthorized)
				return
			}

			destRaw, ok := claims["dest"]
			if !ok {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Msg("session_jwt: missing destination claim")
				}
				http.Error(w, "missing destination", http.StatusUnauthorized)
				return
			}
			dest, ok := destRaw.(string)
			if !ok || strings.TrimSpace(dest) == "" {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Interface("dest", destRaw).
						Msg("session_jwt: invalid destination claim")
				}
				http.Error(w, "invalid destination", http.StatusUnauthorized)
				return
			}

			issHost := hostFromIssuer(iss)
			destHost, err := extractShopDomain(dest)
			if err != nil || issHost == "" || destHost == "" || issHost != destHost {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).
						Str("iss_host", issHost).Str("dest_host", destHost).Err(err).
						Msg("session_jwt: issuer/dest mismatch")
				}
				http.Error(w, "issuer and destination mismatch", http.StatusUnauthorized)
				return
			}

			if debugAuth {
				logger.Log.Info().Str("path", requestPath).
					Str("iss", issHost).Str("dest", destHost).Interface("sub", claims["sub"]).
					Msg("session_jwt: verified")
				LogClaims(claims)
			}
			ctx := context.WithValue(r.Context(), shopifyClaimsContextKey, claims)
			ctx = reconcileShopContext(ctx, destHost)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
