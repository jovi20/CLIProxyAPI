package cliproxy

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func TestApplyCodexBridgeIfNeeded_RewritesCodexWithoutRefreshToken(t *testing.T) {
	service := &Service{
		cfg: &config.Config{},
	}
	original := &coreauth.Auth{
		ID:       "codex-bridge-auth",
		Provider: "codex",
		Attributes: map[string]string{
			"path":        "codex-bridge-auth.json",
			"codex_no_rt": "true",
		},
		Metadata: map[string]any{
			"access_token": "at-123",
			"email":        "user@example.com",
		},
	}

	rewritten := service.applyCodexBridgeIfNeeded(original)
	if rewritten == nil {
		t.Fatal("applyCodexBridgeIfNeeded() returned nil auth")
	}
	if original.Provider != "codex" {
		t.Fatalf("original provider mutated to %q", original.Provider)
	}
	if rewritten.Provider != "codex-bridge" {
		t.Fatalf("provider = %q, want %q", rewritten.Provider, "codex-bridge")
	}
	if got := rewritten.Attributes["base_url"]; got != "http://chat2api:5005/v1" {
		t.Fatalf("base_url = %q, want %q", got, "http://chat2api:5005/v1")
	}
	if got := rewritten.Attributes["api_key"]; got != "at-123" {
		t.Fatalf("api_key = %q, want %q", got, "at-123")
	}
	if got := rewritten.Attributes["bridge_source_provider"]; got != "codex" {
		t.Fatalf("bridge_source_provider = %q, want %q", got, "codex")
	}
	if got := rewritten.Attributes["auth_kind"]; got != "oauth" {
		t.Fatalf("auth_kind = %q, want %q", got, "oauth")
	}
}

func TestRebindExecutors_CodexBridgeDefaultsRewriteExistingAuth(t *testing.T) {
	service := &Service{
		cfg:         &config.Config{},
		coreManager: coreauth.NewManager(nil, nil, nil),
	}
	auth := &coreauth.Auth{
		ID:       "codex-rebind-auth",
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"codex_no_rt": "true",
		},
		Metadata: map[string]any{
			"access_token": "at-123",
			"email":        "user@example.com",
		},
	}
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(auth.ID)
	})
	if _, err := service.coreManager.Register(coreauth.WithSkipPersist(nil), auth); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	service.rebindExecutors()

	updated, ok := service.coreManager.GetByID(auth.ID)
	if !ok || updated == nil {
		t.Fatal("expected auth after rebind")
	}
	if updated.Provider != "codex-bridge" {
		t.Fatalf("provider = %q, want %q", updated.Provider, "codex-bridge")
	}
	if got := updated.Attributes["base_url"]; got != "http://chat2api:5005/v1" {
		t.Fatalf("base_url = %q, want %q", got, "http://chat2api:5005/v1")
	}
}

func TestRegisterModelsForAuth_CodexBridgeUsesCodexCatalogAndBuiltInAliases(t *testing.T) {
	service := &Service{
		cfg: &config.Config{},
	}
	auth := &coreauth.Auth{
		ID:       "codex-bridge-models",
		Provider: "codex-bridge",
		Status:   coreauth.StatusActive,
		Attributes: map[string]string{
			"plan_type": "pro",
			"base_url":  "http://127.0.0.1:5005/v1",
			"api_key":   "at-123",
		},
		Metadata: map[string]any{
			"email": "user@example.com",
		},
	}

	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(auth.ID)
	})

	service.registerModelsForAuth(auth)
	models := registry.GetGlobalRegistry().GetModelsForClient(auth.ID)
	if len(models) == 0 {
		t.Fatal("expected codex bridge models to be registered")
	}
	if !containsModelID(models, "gpt-5.4") {
		t.Fatal("expected codex bridge to expose codex base models")
	}
	if !containsModelID(models, "gpt-5.2") {
		t.Fatal("expected codex bridge built-in aliases to be applied")
	}
}

func containsModelID(models []*registry.ModelInfo, want string) bool {
	for _, model := range models {
		if model != nil && model.ID == want {
			return true
		}
	}
	return false
}
