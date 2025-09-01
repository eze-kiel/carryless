package config

import (
	"os"
)

type Config struct {
	DatabasePath        string
	Port               string
	AllowedOrigins     string
	MailgunDomain      string
	MailgunAPIKey      string
	MailgunSenderEmail string
	MailgunSenderName  string
}

func Load() *Config {
	cfg := &Config{
		DatabasePath:        getEnv("DATABASE_PATH", "carryless.db"),
		Port:               getEnv("PORT", "8080"),
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "http://localhost:8080,http://127.0.0.1:8080,https://carryless.plop.name,https://carryless.org"),
		MailgunDomain:      getEnv("MAILGUN_DOMAIN", ""),
		MailgunAPIKey:      getEnv("MAILGUN_API_KEY", ""),
		MailgunSenderEmail: getEnv("MAILGUN_SENDER_EMAIL", "noreply@carryless.org"),
		MailgunSenderName:  getEnv("MAILGUN_SENDER_NAME", "Carryless"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}