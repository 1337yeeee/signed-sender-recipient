package config

import (
	"os"
	"strings"
)

type Config struct {
	APIPort string
	CORS    CORSConfig

	App             AppConfig
	DocumentStorage DocumentStorageConfig
	SMTP            SMTPConfig
}

type AppConfig struct {
	Role           string
	Email          string
	PrivateKeyPath string
	PublicKeyPath  string
	PrivateKeyPEM  string
	PublicKeyPEM   string
}

type DocumentStorageConfig struct {
	Path string
}

type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (Config, error) {
	return Config{
		APIPort: getEnv("API_PORT", "8080"),
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsCSV(
				"CORS_ALLOWED_ORIGINS",
				[]string{
					"http://localhost:3000",
					"http://127.0.0.1:3000",
					"http://localhost:8080",
					"http://127.0.0.1:8080",
				},
			),
		},

		App: AppConfig{
			Role:           getEnv("APP_ROLE", "sender"),
			Email:          getEnv("APP_EMAIL", "sender@example.com"),
			PrivateKeyPath: getEnv("APP_PRIVATE_KEY_PATH", "data/keys/private.pem"),
			PublicKeyPath:  getEnv("APP_PUBLIC_KEY_PATH", "data/keys/public.pem"),
			PrivateKeyPEM:  os.Getenv("APP_PRIVATE_KEY_PEM"),
			PublicKeyPEM:   os.Getenv("APP_PUBLIC_KEY_PEM"),
		},

		DocumentStorage: DocumentStorageConfig{
			Path: getEnv("DOCUMENT_STORAGE_PATH", "data/storage"),
		},

		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "localhost"),
			Port:     getEnv("SMTP_PORT", "1025"),
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     getEnv("SMTP_FROM", "server@example.com"),
		},
	}, nil
}

func (c Config) Get(key string) string {
	return os.Getenv(key)
}

func getEnv(key, def string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return def
}

func getEnvAsCSV(key string, def []string) []string {
	if val, ok := os.LookupEnv(key); ok {
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	return def
}
