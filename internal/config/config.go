package config

import (
	"os"
)

type Config struct {
	Port     string
	APIToken string
}

func Load() Config {
	return Config{
		Port:     getEnv("PORT", "8080"),
		APIToken: getEnv("API_TOKEN", "dev-token"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
