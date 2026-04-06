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

	if got, want := paths.PluginDir, filepath.Join("/tmp/home", ".config", "claudecode-codex", "plugins"); got != want {
		t.Fatalf("unexpected plugin dir: got %q want %q", got, want)
	}

	if got, want := paths.ProjectPluginDir, filepath.Join("/tmp/project", ".ccagent", "plugins"); got != want {
		t.Fatalf("unexpected project plugin dir: got %q want %q", got, want)
	}
}

func TestLoadDefaultsPluginDirsWithoutConfigFile(t *testing.T) {
	t.Parallel()

	paths, err := ResolvePaths("/tmp/home", "/tmp/project")
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	cfg, err := Load(paths, "/tmp/project")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.PluginDirs) != 2 {
		t.Fatalf("expected 2 default plugin dirs, got %#v", cfg.PluginDirs)
	}
	if got, want := cfg.PluginDirs[0], filepath.Join("/tmp/home", ".config", "claudecode-codex", "plugins"); got != want {
		t.Fatalf("unexpected first plugin dir: got %q want %q", got, want)
	}
	if got, want := cfg.PluginDirs[1], filepath.Join("/tmp/project", ".ccagent", "plugins"); got != want {
		t.Fatalf("unexpected second plugin dir: got %q want %q", got, want)
	}
}
