package httpapi

import (
	"net/http"

	mw "shopify-app-authentication/internal/middleware"
)

// HandlePing 处理 GET /ping，返回当前店铺上下文（公开接口，无需鉴权）。
func HandlePing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":   true,
		"shop": shop,
	})
}

// HandleAdminPing 处理 GET /admin/ping，返回店铺上下文和 JWT claims（需 Session Token 鉴权）。
func HandleAdminPing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	claims, _ := mw.ClaimsFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"shop":   shop,
		"claims": claims,
	})
}

// HandleAppPing 处理 GET /app/ping，返回店铺上下文（需 App Proxy HMAC 签名校验）。
func HandleAppPing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":   true,
		"shop": shop,
	})
}
