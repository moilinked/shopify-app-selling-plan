package httpapi

import (
	"net/http"

	"shopify-app-authentication/internal/config"
	mw "shopify-app-authentication/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

// NewRouter 构建并返回应用的 HTTP 路由。
// 路由分组:
//   - /ping          公开健康检查
//   - /admin/*       需要 Shopify Session Token (JWT) 鉴权
//   - /app/*         需要 Shopify App Proxy HMAC 签名校验
func NewRouter(cfg config.Config, db *gorm.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(mw.AllowAllCORS)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(mw.ShopContextMiddleware())

	r.Get("/ping", HandlePing)

	// /admin 路由组：所有请求必须通过 JWT Session Token 校验
	r.Route("/admin", func(admin chi.Router) {
		admin.Use(mw.ShopifySessionTokenMiddleware(cfg.ShopifyAPIKey, cfg.ShopifyAPISecret, cfg.DebugAuth))
		admin.Get("/ping", HandleAdminPing)

		if db != nil {
			shopHandler := &ShopHandler{DB: db}
			admin.Mount("/shops", shopHandler.Routes())
		}
	})

	// /app 路由组：所有请求必须通过 HMAC 签名校验（Shopify App Proxy）
	r.Route("/app", func(app chi.Router) {
		app.Use(mw.ShopifyAppProxySignatureMiddleware(cfg.ShopifyAPISecret, cfg.DebugAuth))
		app.Get("/ping", HandleAppPing)
	})

	return r
}
