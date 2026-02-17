package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port     string
	APIToken string

	DBHost string
	DBPort string
	DBName string
	DBUser string
	DBPass string

	// DBMaxOpenConns is the maximum number of open connections to the database (default 25).
	DBMaxOpenConns int
	// DBMaxIdleConns is the maximum number of idle connections (default 5).
	DBMaxIdleConns int

	JWTSecret string

	// Env is "dev" (default) or "prod". When "prod", JWT_SECRET must be set and not the default.
	Env string

	// JWTExpireHours is the token lifetime in hours (default 24). Set via JWT_EXPIRE_HOURS.
	JWTExpireHours int

	// NmapPath is the path to the nmap executable (e.g. "nmap" for Linux/Mac, or full Windows path).
	NmapPath string

	// TLSCertFile and TLSKeyFile enable HTTPS when both are set.
	// When empty, the API listens with plain HTTP.
	TLSCertFile string
	TLSKeyFile  string

	// LogFormat is "text" (default) or "json" for structured logging.
	LogFormat string

	// CORSAllowedOrigins is a list of origins allowed for CORS (e.g. https://app.example.com, http://localhost:3000).
	// Set via CORS_ALLOWED_ORIGINS (comma-separated). When empty, no CORS headers are sent (same-origin only).
	CORSAllowedOrigins []string
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

		DBMaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),

		JWTSecret: getEnv("JWT_SECRET", "supersecretkey"),
		Env:       getEnv("ENV", "dev"),
		JWTExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 24),

		// Default "nmap" works on Linux/Mac when nmap is in PATH; set NMAP_PATH for Windows or custom install.
		NmapPath: getEnv("NMAP_PATH", "nmap"),

		// Optional TLS configuration for HTTPS.
		TLSCertFile: getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  getEnv("TLS_KEY_FILE", ""),

		LogFormat: getEnv("LOG_FORMAT", "text"),

		CORSAllowedOrigins: parseCORSOrigins(getEnv("CORS_ALLOWED_ORIGINS", "")),
	}
}

// parseCORSOrigins splits a comma-separated list of origins and trims spaces. Empty strings are omitted.
func parseCORSOrigins(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
