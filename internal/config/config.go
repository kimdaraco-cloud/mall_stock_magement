// @ai-modified 2026-07-02 add env-based config loading
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from environment variables.
type Config struct {
	AppEnv        string
	Port          string
	SessionSecret string
	DatabaseURL   string
	LogLevel      string
}

// Load reads .env (if present) and builds a Config from the environment.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional; real env vars win

	cfg := &Config{
		AppEnv:        getenv("APP_ENV", "development"),
		Port:          getenv("PORT", "8080"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("load config: DATABASE_URL is required")
	}
	if cfg.SessionSecret == "" && cfg.AppEnv != "development" {
		return nil, fmt.Errorf("load config: SESSION_SECRET is required outside development")
	}
	return cfg, nil
}

// IsDev reports whether the app runs in development mode.
func (c *Config) IsDev() bool { return c.AppEnv == "development" }

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
