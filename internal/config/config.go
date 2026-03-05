package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

func (d DatabaseConfig) IsEmpty() bool {
	return d.Host == ""
}

type Config struct {
	Port             string         `yaml:"port"`
	ShopifyAPIKey    string         `yaml:"shopify_api_key"`
	ShopifyAPISecret string         `yaml:"shopify_api_secret"`
	DebugAuth        bool           `yaml:"debug_auth"`
	LogLevel         string         `yaml:"log_level"`
	Database         DatabaseConfig `yaml:"database"`
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, err
	}

	cfg.Port = strings.TrimSpace(cfg.Port)
	if cfg.Port == "" {
		cfg.Port = "9998"
	}

	cfg.ShopifyAPIKey = strings.TrimSpace(cfg.ShopifyAPIKey)
	cfg.ShopifyAPISecret = strings.TrimSpace(cfg.ShopifyAPISecret)
	if cfg.ShopifyAPIKey == "" || cfg.ShopifyAPISecret == "" {
		return Config{}, errors.New("shopify_api_key and shopify_api_secret are required")
	}

	cfg.LogLevel = strings.TrimSpace(strings.ToLower(cfg.LogLevel))
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}
