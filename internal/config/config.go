package config

import (
	"os"
)

type Config struct {
	Port     string
	APIToken string

	DBHost string
}

func Load() Config {
	return Config{
		Port:     getEnv("PORT", "8080"),
		APIToken: getEnv("API_TOKEN", "dev-token"),

		DBHost: getEnv("DB_HOST", "localhost"),
		DBport: getEnv("DB_PORT", "5432"),
		DBName: getEnv("DB_NAME", "assetdb"),
		DBUser: getEnv("DB_USER", "assetuser"),
		DBPass: getEnv("DB_PASS", "assetpass"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
