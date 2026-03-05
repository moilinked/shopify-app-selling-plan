package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"shopify-app-authentication/internal/logger"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	shopifyClaimsContextKey = contextKey("shopifyClaims")
	shopContextKey          = contextKey("shop")
)

// ShopFromContext 从请求上下文中取出已验证的店铺域名。
func ShopFromContext(ctx context.Context) (string, bool) {
	shop, ok := ctx.Value(shopContextKey).(string)
	return shop, ok && shop != ""
}

// ClaimsFromContext 从请求上下文中取出 Shopify JWT 的 claims。
func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(shopifyClaimsContextKey).(jwt.MapClaims)
	return claims, ok
}

// LogClaims 以 JSON 格式记录 Shopify Session Token 的完整 claims（debug 级别）。
func LogClaims(claims jwt.MapClaims) {
	claimsJSON, _ := json.Marshal(claims)
	logger.Log.Debug().RawJSON("claims", claimsJSON).Msg("shopify_session_token")
}

// ShopContextMiddleware 从请求头 X-Shop-Domain 或 Shop 中提取店铺域名，
// 并存入请求上下文，供后续 handler 使用。
func ShopContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shop := shopFromHeader(r); shop != "" {
				ctx := context.WithValue(r.Context(), shopContextKey, shop)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractShopDomain 去除协议前缀和尾部斜杠，返回小写的店铺域名。
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

// hostFromIssuer 从 JWT iss 字段中提取主机名部分（去除路径和协议）。
func hostFromIssuer(issuer string) string {
	issuer = strings.TrimSpace(strings.ToLower(issuer))
	issuer = strings.TrimPrefix(issuer, "https://")
	issuer = strings.TrimPrefix(issuer, "http://")
	parts := strings.SplitN(issuer, "/", 2)
	return parts[0]
}

// shopFromHeader 优先从 X-Shop-Domain 头读取店铺域名，其次从 Shop 头读取。
func shopFromHeader(r *http.Request) string {
	shop := strings.TrimSpace(r.Header.Get("X-Shop-Domain"))
	if shop == "" {
		shop = strings.TrimSpace(r.Header.Get("Shop"))
	}
	if shop == "" {
		return ""
	}
	domain, err := extractShopDomain(shop)
	if err != nil {
		return ""
	}
	return domain
}

// reconcileShopContext 对比鉴权后解析出的店铺域名与请求头中的店铺域名。
// 如果两者不一致则清空上下文中的店铺信息，防止伪造。
func reconcileShopContext(ctx context.Context, authShop string) context.Context {
	headerShop, hasHeader := ShopFromContext(ctx)
	if hasHeader && headerShop != authShop {
		return context.WithValue(ctx, shopContextKey, "")
	}
	return context.WithValue(ctx, shopContextKey, authShop)
}
