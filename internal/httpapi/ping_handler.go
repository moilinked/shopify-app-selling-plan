package httpapi

import (
	"net/http"

	mw "shopify-app-authentication/internal/middleware"
)

// HandlePing responds to GET /ping with the current shop context.
func HandlePing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":   true,
		"shop": shop,
	})
}

// HandleAdminPing responds to GET /admin/ping with the shop context and JWT claims.
func HandleAdminPing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	claims, _ := mw.ClaimsFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"shop":   shop,
		"claims": claims,
	})
}

// HandleAppPing responds to GET /app/ping with the shop context (app proxy).
func HandleAppPing(w http.ResponseWriter, r *http.Request) {
	shop, _ := mw.ShopFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":   true,
		"shop": shop,
	})
}
