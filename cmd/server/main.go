package main

import (
	"flag"
	"net/http"
	"time"

	"shopify-app-authentication/internal/config"
	"shopify-app-authentication/internal/database"
	"shopify-app-authentication/internal/httpapi"
	"shopify-app-authentication/internal/logger"

	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("load config")
	}

	if err := logger.Init(cfg.LogLevel); err != nil {
		logger.Log.Fatal().Err(err).Msg("init logger")
	}

	// 仅在配置了数据库信息时才连接
	var db *gorm.DB
	if !cfg.Database.IsEmpty() {
		var err error
		db, err = database.Connect(cfg.Database.DSN())
		if err != nil {
			logger.Log.Fatal().Err(err).Msg("connect database")
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Log.Info().Str("addr", ":"+cfg.Port).Msgf("listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Log.Fatal().Err(err).Msg("server error")
	}
}
