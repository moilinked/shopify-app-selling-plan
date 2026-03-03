package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"shopify-app-authentication/internal/config"
	"shopify-app-authentication/internal/database"
	"shopify-app-authentication/internal/httpapi"

	"gorm.io/gorm"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	var db *gorm.DB
	if !cfg.Database.IsEmpty() {
		var err error
		db, err = database.Connect(cfg.Database.DSN())
		if err != nil {
			log.Fatalf("connect database: %v", err)
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
