package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"shopify-app-authentication/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

// ShopifySessionTokenMiddleware 校验前端通过 Authorization: Bearer 发送的 Shopify Session Token（JWT）。
// 校验项包括：HS256 签名、exp/nbf（带 5 秒容差）、aud、iss 与 dest 域名一致性。
// 校验通过后将 claims 和解析出的店铺域名写入请求上下文。
func ShopifySessionTokenMiddleware(apiKey, apiSecret string, debugAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path

			// 1. 提取 Authorization 头中的 Bearer Token
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

			// 2. 解析并验签 JWT（HS256，5 秒时钟容差）
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

			// 3. 校验 aud 是否为当前应用的 API Key
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

			// 4. 校验 iss 存在
			iss, err := claims.GetIssuer()
			if err != nil || iss == "" {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Err(err).Msg("session_jwt: missing issuer")
				}
				http.Error(w, "missing issuer", http.StatusUnauthorized)
				return
			}

			// 5. 校验 dest 存在且合法
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

			// 6. 校验 iss 和 dest 的顶级域名一致（防止跨店伪造）
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

			// 7. 将 claims 和店铺域名写入上下文
			ctx := context.WithValue(r.Context(), shopifyClaimsContextKey, claims)
			ctx = reconcileShopContext(ctx, destHost)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
