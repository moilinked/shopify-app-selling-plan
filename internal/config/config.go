package config

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port             string `yaml:"port"`
	ShopifyAPIKey    string `yaml:"shopify_api_key"`
	ShopifyAPISecret string `yaml:"shopify_api_secret"`
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

	return cfg, nil
}
