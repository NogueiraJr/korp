package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	InternalToken string
	AIBaseURL     string
	AIAPIKey      string
	AIModel       string
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8081"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://korp:korp_dev_2026@localhost:5432/korp_estoque?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "korp-jwt-secret-dev-change-me"),
		InternalToken: getEnv("INTERNAL_TOKEN", "korp-internal-token-dev"),
		AIBaseURL:     getEnv("AI_BASE_URL", ""),
		AIAPIKey:      getEnv("AI_API_KEY", ""),
		AIModel:       getEnv("AI_MODEL", "gpt-4o-mini"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
