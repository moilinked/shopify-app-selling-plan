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

// ShopFromContext retrieves the shop domain from the request context.
func ShopFromContext(ctx context.Context) (string, bool) {
	shop, ok := ctx.Value(shopContextKey).(string)
	return shop, ok && shop != ""
}

// ClaimsFromContext retrieves the Shopify JWT claims from the request context.
func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(shopifyClaimsContextKey).(jwt.MapClaims)
	return claims, ok
}

// LogClaims logs the Shopify session token claims as JSON.
func LogClaims(claims jwt.MapClaims) {
	claimsJSON, _ := json.Marshal(claims)
	logger.Log.Debug().RawJSON("claims", claimsJSON).Msg("shopify_session_token")
}

// ShopContextMiddleware extracts the shop domain from request headers
// (X-Shop-Domain or Shop) and stores it in the request context.
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

// reconcileShopContext sets the auth-resolved shop in context.
// If the header-based shop already in context doesn't match, shop is removed.
func reconcileShopContext(ctx context.Context, authShop string) context.Context {
	headerShop, hasHeader := ShopFromContext(ctx)
	if hasHeader && headerShop != authShop {
		return context.WithValue(ctx, shopContextKey, "")
	}
	return context.WithValue(ctx, shopContextKey, authShop)
}
