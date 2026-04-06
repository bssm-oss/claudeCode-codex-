package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

type Hook struct {
	Event   string
	Command string
	Source  string
}

type pluginManifest struct {
	Name  string              `json:"name"`
	Hooks []config.HookConfig `json:"hooks"`
}

var supportedEvents = map[string]struct{}{
	"session_start": {},
	"before_model":  {},
	"after_model":   {},
	"before_tool":   {},
	"after_tool":    {},
}

func LoadHooks(cfg config.Config) ([]Hook, error) {
	hooks := make([]Hook, 0, len(cfg.Hooks))
	for _, hook := range cfg.Hooks {
		normalized, err := normalizeHook(hook, "config")
		if err != nil {
			return nil, err
		}
		if normalized.Event != "" {
			hooks = append(hooks, normalized)
		}
	}

	pluginHooks, err := loadPluginHooks(cfg.PluginDirs)
	if err != nil {
		return nil, err
	}
	hooks = append(hooks, pluginHooks...)
	return hooks, nil
}

func loadPluginHooks(pluginDirs []string) ([]Hook, error) {
	hooks := []Hook{}
	for _, dir := range pluginDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read plugin dir %s: %w", dir, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(dir, entry.Name(), "plugin.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("read plugin manifest %s: %w", manifestPath, err)
			}
			var manifest pluginManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, fmt.Errorf("parse plugin manifest %s: %w", manifestPath, err)
			}
			source := manifest.Name
			if strings.TrimSpace(source) == "" {
				source = entry.Name()
			}
			for _, hook := range manifest.Hooks {
				normalized, err := normalizeHook(hook, "plugin:"+source)
				if err != nil {
					return nil, err
				}
				if normalized.Event != "" {
					hooks = append(hooks, normalized)
				}
			}
		}
	}
	return hooks, nil
}

func normalizeHook(hook config.HookConfig, source string) (Hook, error) {
	event := strings.TrimSpace(hook.Event)
	command := strings.TrimSpace(hook.Command)
	if event == "" || command == "" {
		return Hook{}, nil
	}
	if _, ok := supportedEvents[event]; !ok {
		return Hook{}, fmt.Errorf("unsupported hook event %q from %s", event, source)
	}
	return Hook{Event: event, Command: command, Source: source}, nil
}
