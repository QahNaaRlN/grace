package config

import "testing"

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
