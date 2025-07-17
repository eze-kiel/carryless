package config

import (
	"os"
)

type Config struct {
	DatabasePath string
	Port         string
	SecretKey    string
}

func Load() *Config {
	cfg := &Config{
		DatabasePath: getEnv("DATABASE_PATH", "carryless.db"),
		Port:         getEnv("PORT", "8080"),
		SecretKey:    getEnv("SECRET_KEY", "your-secret-key-change-this-in-production"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}