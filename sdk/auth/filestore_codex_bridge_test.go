package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileTokenStoreReadAuthFile_MarksCodexWithoutRefreshToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "codex-no-rt.json")
	if err := os.WriteFile(path, []byte(`{"type":"codex","access_token":"at-123","email":"user@example.com"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := &FileTokenStore{}
	auth, err := store.readAuthFile(path, dir)
	if err != nil {
		t.Fatalf("readAuthFile() error = %v", err)
	}
	if auth == nil {
		t.Fatal("readAuthFile() returned nil auth")
	}
	if got := auth.Attributes["codex_no_rt"]; got != "true" {
		t.Fatalf("codex_no_rt marker = %q, want %q", got, "true")
	}
}

func TestFileTokenStoreReadAuthFile_DoesNotMarkCodexWithRefreshToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "codex-with-rt.json")
	if err := os.WriteFile(path, []byte(`{"type":"codex","access_token":"at-123","refresh_token":"rt-456"}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := &FileTokenStore{}
	auth, err := store.readAuthFile(path, dir)
	if err != nil {
		t.Fatalf("readAuthFile() error = %v", err)
	}
	if auth == nil {
		t.Fatal("readAuthFile() returned nil auth")
	}
	if got := auth.Attributes["codex_no_rt"]; got != "" {
		t.Fatalf("codex_no_rt marker = %q, want empty", got)
	}
}
