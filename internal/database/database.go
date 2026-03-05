package database

import (
	"fmt"

	"shopify-app-authentication/internal/logger"
	"shopify-app-authentication/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect 使用 DSN 连接 PostgreSQL 并自动迁移表结构。
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(&models.Shop{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	logger.Log.Info().Msg("database connected and migrated successfully")
	return db, nil
}
