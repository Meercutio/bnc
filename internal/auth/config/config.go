package config

import (
	"os"
	"time"
)

type Config struct {
	PostgresDSN   string
	JWTSecret     string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	MigrationsDir string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func Load() Config {
	return Config{
		PostgresDSN:   getenv("POSTGRES_DSN", ""),
		JWTSecret:     getenv("JWT_SECRET", "dev-secret"),
		AccessTTL:     mustDuration(getenv("ACCESS_TOKEN_TTL", "15m"), 15*time.Minute),
		RefreshTTL:    mustDuration(getenv("REFRESH_TOKEN_TTL", "720h"), 30*24*time.Hour),
		MigrationsDir: getenv("MIGRATIONS_DIR", "migrations/auth"),
	}
}
