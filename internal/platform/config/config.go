package config

import (
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SMTPHost                string
	SMTPPort                string
	SMTPUsername            string
	SMTPPassword            string
	SMTPFrom                string
	SMTPSecurity            string
	EmailBudget             int
	AuthMode                string
	PublicURL               string
	TelegramClientID        string
	TelegramClientSecret    string
	Port                    string
	DatabaseURL             string
	CORSOrigins             []string
	AssistantRoutingEnabled bool
	TypeSafeAPIKey          string
}

func Load() (Config, error) {
	cfg := Config{
		SMTPHost:             strings.TrimSpace(os.Getenv("GRACE_SMTP_HOST")),
		SMTPPort:             valueOrDefault("GRACE_SMTP_PORT", "587"),
		SMTPUsername:         strings.TrimSpace(os.Getenv("GRACE_SMTP_USERNAME")),
		SMTPPassword:         os.Getenv("GRACE_SMTP_PASSWORD"),
		SMTPFrom:             strings.TrimSpace(os.Getenv("GRACE_SMTP_FROM")),
		SMTPSecurity:         valueOrDefault("GRACE_SMTP_SECURITY", "starttls"),
		AuthMode:             valueOrDefault("GRACE_AUTH_MODE", "locked"),
		PublicURL:            strings.TrimRight(valueOrDefault("GRACE_PUBLIC_URL", "http://localhost:5173"), "/"),
		TelegramClientID:     strings.TrimSpace(os.Getenv("TELEGRAM_CLIENT_ID")),
		TelegramClientSecret: strings.TrimSpace(os.Getenv("TELEGRAM_CLIENT_SECRET")),
		Port:                 valueOrDefault("APP_PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		CORSOrigins:          splitCSV(valueOrDefault("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		TypeSafeAPIKey:       strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY")),
	}
	enabledValue := valueOrDefault("GRACE_ASSISTANT_ROUTING_ENABLED", "false")
	if enabledValue != "true" && enabledValue != "false" {
		return Config{}, fmt.Errorf("GRACE_ASSISTANT_ROUTING_ENABLED must be true or false")
	}
	cfg.AssistantRoutingEnabled = enabledValue == "true"
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	u, err := url.Parse(cfg.PublicURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return Config{}, fmt.Errorf("GRACE_PUBLIC_URL must be an origin")
	}
	budget, err := strconv.Atoi(valueOrDefault("GRACE_EMAIL_MONTHLY_BUDGET", "1500"))
	if err != nil || budget < 1 || budget > 100000 {
		return Config{}, fmt.Errorf("invalid email budget")
	}
	cfg.EmailBudget = budget
	switch cfg.AuthMode {
	case "locked":
	case "email":
		if u.Scheme != "https" || cfg.SMTPHost == "" || cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
			return Config{}, fmt.Errorf("email auth requires HTTPS and configured SMTP")
		}
		from, parseErr := mail.ParseAddress(cfg.SMTPFrom)
		if parseErr != nil || strings.ContainsAny(cfg.SMTPFrom, "\r\n") || from.Address == "" {
			return Config{}, fmt.Errorf("invalid email sender")
		}
		port, portErr := strconv.Atoi(cfg.SMTPPort)
		if portErr != nil || port < 1 || port > 65535 || (cfg.SMTPSecurity != "starttls" && cfg.SMTPSecurity != "tls") {
			return Config{}, fmt.Errorf("SMTP TLS configuration required")
		}
	case "telegram":
		if u.Scheme != "https" || cfg.TelegramClientID == "" || cfg.TelegramClientSecret == "" {
			return Config{}, fmt.Errorf("telegram auth requires HTTPS public URL and client credentials")
		}
	case "development":
		if os.Getenv("APP_ENV") != "development" || u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") {
			return Config{}, fmt.Errorf("development auth requires APP_ENV=development and a loopback public URL")
		}
	default:
		return Config{}, fmt.Errorf("invalid GRACE_AUTH_MODE")
	}
	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
