package config

import (
	"path/filepath"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths("/tmp/home", "/tmp/project")
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	if got, want := paths.ConfigFile, filepath.Join("/tmp/home", ".config", "claudecode-codex", "config.json"); got != want {
		t.Fatalf("unexpected config path: got %q want %q", got, want)
	}

	if got, want := paths.AuthFile, filepath.Join("/tmp/home", ".config", "claudecode-codex", "auth.json"); got != want {
		t.Fatalf("unexpected auth path: got %q want %q", got, want)
	}
}
