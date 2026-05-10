package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	APIPort string
	CORS    CORSConfig

	App             AppConfig
	DocumentStorage DocumentStorageConfig
	SMTP            SMTPConfig
	MailPolling     MailPollingConfig
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

type MailPollingConfig struct {
	Enabled  bool
	Interval time.Duration
	Source   MailSourceConfig
	Filter   MailFilterConfig
}

type MailSourceConfig struct {
	Type    string
	BaseURL string
	Limit   int
}

type MailFilterConfig struct {
	RecipientEmail   string
	SubjectPrefix    string
	AttachmentSuffix string
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (Config, error) {
	pollInterval, err := getEnvDuration("MAIL_POLL_INTERVAL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

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

		MailPolling: MailPollingConfig{
			Enabled:  getEnvAsBool("MAIL_POLL_ENABLED", true),
			Interval: pollInterval,
			Source: MailSourceConfig{
				Type:    getEnv("MAIL_SOURCE_TYPE", "mailpit"),
				BaseURL: strings.TrimRight(getEnv("MAIL_SOURCE_BASE_URL", "http://mailpit:8025"), "/"),
				Limit:   getEnvAsInt("MAIL_SOURCE_LIMIT", 25),
			},
			Filter: MailFilterConfig{
				RecipientEmail:   getEnv("MAIL_FILTER_RECIPIENT_EMAIL", getEnv("APP_EMAIL", "sender@example.com")),
				SubjectPrefix:    getEnv("MAIL_FILTER_SUBJECT_PREFIX", "Encrypted document package:"),
				AttachmentSuffix: getEnv("MAIL_FILTER_ATTACHMENT_SUFFIX", "_encrypted_package.json"),
			},
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

func getEnvAsBool(key string, def bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	switch strings.ToLower(strings.TrimSpace(val)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func getEnvAsInt(key string, def int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return def
	}

	return parsed
}

func getEnvDuration(key string, def time.Duration) (time.Duration, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(val))
	if err != nil {
		return 0, err
	}

	return parsed, nil
}
