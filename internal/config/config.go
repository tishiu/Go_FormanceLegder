package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL    string
	ServerPort     string
	JWTSecret      []byte
	APIKeySecret   []byte
	SessionTimeout time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ledger_kiro?sslmode=disable"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		JWTSecret:      []byte(getEnv("JWT_SECRET", "change-me-in-production")),
		APIKeySecret:   []byte(getEnv("API_KEY_SECRET", "change-me-in-production")),
		SessionTimeout: time.Hour * 24,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
