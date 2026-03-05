package httpapi

import (
	"net/http"

	"shopify-app-authentication/internal/config"
	mw "shopify-app-authentication/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewRouter(cfg config.Config, db *gorm.DB) http.Handler {
	r := chi.NewRouter()
	r.Use(mw.AllowAllCORS)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mw.RequestLogger)
	r.Use(middleware.Recoverer)
	r.Use(mw.ShopContextMiddleware())

	r.Get("/ping", HandlePing)

	r.Route("/admin", func(admin chi.Router) {
		admin.Use(mw.ShopifySessionTokenMiddleware(cfg.ShopifyAPIKey, cfg.ShopifyAPISecret, cfg.DebugAuth))
		admin.Get("/ping", HandleAdminPing)

		if db != nil {
			shopHandler := &ShopHandler{DB: db}
			admin.Mount("/shops", shopHandler.Routes())
		}
	})

	r.Route("/app", func(app chi.Router) {
		app.Use(mw.ShopifyAppProxySignatureMiddleware(cfg.ShopifyAPISecret, cfg.DebugAuth))
		app.Get("/ping", HandleAppPing)
	})

	return r
}
