package config

import "os"

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	InternalToken string
	EstoqueURL    string
	AdminUsername string
	AdminPassword string
	TokenTTLHours int
}

func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8082"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://korp:korp_dev_2026@localhost:5432/korp_faturamento?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "korp-jwt-secret-dev-change-me"),
		InternalToken: getEnv("INTERNAL_TOKEN", "korp-internal-token-dev"),
		EstoqueURL:    getEnv("ESTOQUE_URL", "http://localhost:8081"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		TokenTTLHours: 24,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}