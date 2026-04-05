package auth

import (
	"path/filepath"
	"testing"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

func TestStoreSaveAndLoad(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := config.Paths{ConfigDir: tempDir, AuthFile: filepath.Join(tempDir, "auth.json")}
	store := NewStore(paths)

	if err := store.Save(Credentials{APIKey: "test-key"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	creds, err := store.Load(map[string]string{})
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}

	if creds.APIKey != "test-key" {
		t.Fatalf("expected saved key, got %q", creds.APIKey)
	}
}
