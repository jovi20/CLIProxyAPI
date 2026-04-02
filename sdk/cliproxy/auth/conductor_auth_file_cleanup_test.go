package auth

import (
	"context"
	"testing"
	"time"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

type authFileDeleteStore struct {
	deleteCh chan string
}

func (s *authFileDeleteStore) List(context.Context) ([]*Auth, error) { return nil, nil }

func (s *authFileDeleteStore) Save(context.Context, *Auth) (string, error) { return "", nil }

func (s *authFileDeleteStore) Delete(_ context.Context, id string) error {
	select {
	case s.deleteCh <- id:
	default:
	}
	return nil
}

func newFileBackedAuth(id, provider string) *Auth {
	return &Auth{
		ID:       id,
		FileName: id,
		Provider: provider,
		Status:   StatusActive,
		Attributes: map[string]string{
			"path": id,
		},
		Metadata: map[string]any{
			"email": "user@example.com",
		},
	}
}

func markFailure(mgr *Manager, auth *Auth, status int, message string) {
	mgr.auths[auth.ID] = auth.Clone()
	mgr.MarkResult(context.Background(), Result{
		AuthID:   auth.ID,
		Provider: auth.Provider,
		Model:    "test-model",
		Success:  false,
		Error: &Error{
			Message:    message,
			HTTPStatus: status,
		},
	})
}

func waitForDeletedAuth(t *testing.T, ch <-chan string, want string) {
	t.Helper()

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("Delete() id = %q, want %q", got, want)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected delete call for %q", want)
	}
}

func waitForNoDeletedAuth(t *testing.T, ch <-chan string) {
	t.Helper()

	select {
	case got := <-ch:
		t.Fatalf("unexpected delete call for %q", got)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestMarkResult_FileAuthPaymentRequiredDeletesWhenEnabled(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("claude-bad.json", "claude")
	markFailure(mgr, auth, 402, "payment required")
	waitForDeletedAuth(t, store.deleteCh, auth.ID)
}

func TestMarkResult_FileAuthUnauthorizedInvalidSignalDeletesWhenEnabled(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("qwen-bad.json", "qwen")
	markFailure(mgr, auth, 401, "invalid access token")
	waitForDeletedAuth(t, store.deleteCh, auth.ID)
}

func TestMarkResult_FileAuthForbiddenExpiredSignalDeletesWhenEnabled(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("iflow-bad.json", "iflow")
	markFailure(mgr, auth, 403, "refresh token expired")
	waitForDeletedAuth(t, store.deleteCh, auth.ID)
}

func TestMarkResult_FileAuthUnauthorizedWithoutSignalDoesNotDelete(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("compat-baseurl.json", "openai-compatibility")
	markFailure(mgr, auth, 401, "missing provider baseURL")
	waitForNoDeletedAuth(t, store.deleteCh)
}

func TestMarkResult_FileAuthForbiddenWithoutSignalDoesNotDelete(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("region-locked.json", "gemini")
	markFailure(mgr, auth, 403, "permission denied")
	waitForNoDeletedAuth(t, store.deleteCh)
}

func TestMarkResult_FileAuthQuotaDoesNotDelete(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("quota.json", "qwen")
	markFailure(mgr, auth, 429, "quota exhausted")
	waitForNoDeletedAuth(t, store.deleteCh)
}

func TestMarkResult_FileAuthMissingRefreshTokenDoesNotDelete(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := newFileBackedAuth("bridge-no-rt.json", "codex-bridge")
	markFailure(mgr, auth, 401, "missing refresh token")
	waitForNoDeletedAuth(t, store.deleteCh)
}

func TestMarkResult_ConfigBackedAuthDoesNotDelete(t *testing.T) {
	t.Parallel()

	store := &authFileDeleteStore{deleteCh: make(chan string, 1)}
	mgr := NewManager(store, nil, nil)
	mgr.SetConfig(&internalconfig.Config{
		AuthFileCleanup: internalconfig.AuthFileCleanupConfig{
			Enabled: true,
		},
	})

	auth := &Auth{
		ID:       "openrouter-config",
		Provider: "openai-compatibility",
		Status:   StatusActive,
		Attributes: map[string]string{
			"provider_key": "openrouter",
			"compat_name":  "openrouter",
		},
	}
	markFailure(mgr, auth, 402, "payment required")
	waitForNoDeletedAuth(t, store.deleteCh)
}
