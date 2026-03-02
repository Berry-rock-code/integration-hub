package config

import (
	"os"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads .env for local development.
// In production (Cloud Run), environment variables should already be provided,
// so this is a no-op if .env doesn't exist.
func LoadDotEnv() {
	// Only try .env if it exists
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load(".env")
	}
}
