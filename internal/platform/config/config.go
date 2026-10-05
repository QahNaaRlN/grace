package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
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
	switch cfg.AuthMode {
	case "locked":
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
