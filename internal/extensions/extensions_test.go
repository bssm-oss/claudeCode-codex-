package extensions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

func TestLoadHooksIncludesConfigAndPluginHooks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins")
	manifestDir := filepath.Join(pluginDir, "sample")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifest := `{
		"name": "sample-plugin",
		"hooks": [
			{"event": "before_model", "command": "printf plugin"},
			{"event": "", "command": "ignored"}
		]
	}`
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	hooks, err := LoadHooks(config.Config{
		Hooks:      []config.HookConfig{{Event: "session_start", Command: "printf config"}},
		PluginDirs: []string{pluginDir},
	})
	if err != nil {
		t.Fatalf("load hooks: %v", err)
	}
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}
	if hooks[0].Source != "plugin:sample-plugin" && hooks[1].Source != "plugin:sample-plugin" {
		t.Fatalf("expected plugin hook source, got %#v", hooks)
	}
}

func TestLoadHooksRejectsUnsupportedEvent(t *testing.T) {
	t.Parallel()

	_, err := LoadHooks(config.Config{
		Hooks: []config.HookConfig{{Event: "typo_event", Command: "printf nope"}},
	})
	if err == nil {
		t.Fatal("expected unsupported event to fail")
	}
}
