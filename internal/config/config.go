package config

import (
	"os"
)

type Config struct {
	Port     string
	APIToken string

	DBHost string
	DBPort string
	DBName string
	DBUser string
	DBPass string

	JWTSecret string

	// NmapPath is the path to the nmap executable (e.g. "nmap" for Linux/Mac, or full Windows path).
	NmapPath string
}

func Load() Config {
	return Config{
		Port:     getEnv("PORT", "8080"),
		APIToken: getEnv("API_TOKEN", "dev-token"),

		DBHost: getEnv("DB_HOST", "localhost"),
		DBPort: getEnv("DB_PORT", "5432"),
		DBName: getEnv("DB_NAME", "assetdb"),
		DBUser: getEnv("DB_USER", "assetuser"),
		DBPass: getEnv("DB_PASS", "assetpass"),

		JWTSecret: getEnv("JWT_SECRET", "supersecretkey"),

		// Default "nmap" works on Linux/Mac when nmap is in PATH; set NMAP_PATH for Windows or custom install.
		NmapPath: getEnv("NMAP_PATH", "nmap"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
