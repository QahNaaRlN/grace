package config

import "testing"

func TestEmailConfigurationRequiresDeliveryAndHTTPS(t *testing.T) {
	base := map[string]string{"DATABASE_URL": "postgres://test", "GRACE_ASSISTANT_ROUTING_ENABLED": "false", "GRACE_AUTH_MODE": "email", "GRACE_PUBLIC_URL": "https://grace.test", "GRACE_SMTP_HOST": "postbox.example.test", "GRACE_SMTP_PORT": "587", "GRACE_SMTP_SECURITY": "starttls", "GRACE_SMTP_USERNAME": "synthetic-user", "GRACE_SMTP_PASSWORD": "synthetic-password", "GRACE_SMTP_FROM": "Grace <no-reply@example.test>", "GRACE_EMAIL_MONTHLY_BUDGET": "1500"}
	for key, value := range base {
		t.Setenv(key, value)
	}
	if cfg, err := Load(); err != nil || cfg.AuthMode != "email" {
		t.Fatal("valid email configuration rejected", err)
	}
	for _, invalid := range []struct{ key, value string }{{"GRACE_PUBLIC_URL", "http://grace.test"}, {"GRACE_SMTP_PASSWORD", ""}, {"GRACE_SMTP_HOST", ""}, {"GRACE_SMTP_SECURITY", "plain"}, {"GRACE_SMTP_PORT", "0"}, {"GRACE_SMTP_FROM", "no-reply@example.test\r\nBcc: attacker@example.test"}, {"GRACE_EMAIL_MONTHLY_BUDGET", "0"}} {
		t.Run(invalid.key, func(t *testing.T) {
			t.Setenv(invalid.key, invalid.value)
			if _, err := Load(); err == nil {
				t.Fatal("unsafe email configuration accepted")
			}
		})
	}
}

func TestAuthConfigurationFailsClosed(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("GRACE_ASSISTANT_ROUTING_ENABLED", "false")
	t.Setenv("GRACE_AUTH_MODE", "")
	t.Setenv("GRACE_PUBLIC_URL", "https://grace.test")
	cfg, err := Load()
	if err != nil || cfg.AuthMode != "locked" {
		t.Fatalf("default must lock: %v", err)
	}
	t.Setenv("GRACE_AUTH_MODE", "telegram")
	t.Setenv("TELEGRAM_CLIENT_ID", "")
	if _, err = Load(); err == nil {
		t.Fatal("missing credentials accepted")
	}
	t.Setenv("TELEGRAM_CLIENT_ID", "client")
	t.Setenv("TELEGRAM_CLIENT_SECRET", "secret")
	if _, err = Load(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRACE_PUBLIC_URL", "http://grace.test")
	if _, err = Load(); err == nil {
		t.Fatal("HTTP Telegram accepted")
	}
	t.Setenv("GRACE_AUTH_MODE", "development")
	t.Setenv("GRACE_PUBLIC_URL", "http://localhost:5173")
	t.Setenv("APP_ENV", "production")
	if _, err = Load(); err == nil {
		t.Fatal("production development bypass accepted")
	}
	t.Setenv("APP_ENV", "development")
	if _, err = Load(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRACE_PUBLIC_URL", "http://0.0.0.0:5173")
	if _, err = Load(); err == nil {
		t.Fatal("non-loopback bypass accepted")
	}
}

func TestAssistantRoutingConfigurationIsOptIn(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("GRACE_ASSISTANT_ROUTING_ENABLED", "")
	t.Setenv("TYPESAFE_API_KEY", "")
	cfg, err := Load()
	if err != nil || cfg.AssistantRoutingEnabled || cfg.TypeSafeAPIKey != "" {
		t.Fatalf("unexpected default config: %#v %v", cfg, err)
	}
	t.Setenv("GRACE_ASSISTANT_ROUTING_ENABLED", "true")
	t.Setenv("TYPESAFE_API_KEY", "  secret  ")
	cfg, err = Load()
	if err != nil || !cfg.AssistantRoutingEnabled || cfg.TypeSafeAPIKey != "secret" {
		t.Fatalf("unexpected enabled config: %#v %v", cfg, err)
	}
}

func TestAssistantRoutingConfigurationRejectsInvalidBoolean(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test")
	t.Setenv("GRACE_ASSISTANT_ROUTING_ENABLED", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid feature flag to fail")
	}
}
