package auth

import (
	"context"
	"testing"
	"time"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

type codexBridgeDeleteStore struct {
	deleteCh chan string
}

func (s *codexBridgeDeleteStore) List(context.Context) ([]*Auth, error) { return nil, nil }

func (s *codexBridgeDeleteStore) Save(context.Context, *Auth) (string, error) { return "", nil }

func (s *codexBridgeDeleteStore) Delete(_ context.Context, id string) error {
	select {
	case s.deleteCh <- id:
	default:
	}
	return nil
}

func TestMarkResult_CodexBridgeUnauthorizedDeletesAuthFileWhenEnabled(t *testing.T) {
	t.Parallel()

	store := &codexBridgeDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		CodexBridge: internalconfig.CodexBridgeConfig{
			Enabled:            true,
			AutoDeleteOnExpiry: true,
		},
	})

	auth := &Auth{
		ID:       "codex-bridge-auth.json",
		Provider: "codex-bridge",
		Status:   StatusActive,
		Attributes: map[string]string{
			"base_url": "http://127.0.0.1:5005/v1",
			"api_key":  "at-123",
		},
		Metadata: map[string]any{
			"email": "user@example.com",
		},
	}
	mgr.auths[auth.ID] = auth.Clone()

	mgr.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: auth.Provider,
		Model:    "gpt-5.4",
		Success:  false,
		Error: &Error{
			Message:    "unauthorized",
			HTTPStatus: 401,
		},
	})

	select {
	case got := <-store.deleteCh:
		if got != auth.ID {
			t.Fatalf("Delete() id = %q, want %q", got, auth.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected codex bridge auth delete on 401")
	}
}

func TestMarkResult_CodexBridgeUnauthorizedDoesNotDeleteWhenDisabled(t *testing.T) {
	t.Parallel()

	store := &codexBridgeDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		CodexBridge: internalconfig.CodexBridgeConfig{
			Enabled:            true,
			AutoDeleteOnExpiry: false,
		},
	})

	auth := &Auth{
		ID:       "codex-bridge-auth.json",
		Provider: "codex-bridge",
		Status:   StatusActive,
		Metadata: map[string]any{
			"email": "user@example.com",
		},
	}
	mgr.auths[auth.ID] = auth.Clone()

	mgr.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: auth.Provider,
		Model:    "gpt-5.4",
		Success:  false,
		Error: &Error{
			Message:    "unauthorized",
			HTTPStatus: 401,
		},
	})

	select {
	case got := <-store.deleteCh:
		t.Fatalf("unexpected delete call for %q", got)
	case <-time.After(100 * time.Millisecond):
	}
}
