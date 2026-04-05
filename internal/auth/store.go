package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

type Credentials struct {
	APIKey string `json:"api_key"`
}

type Store struct {
	paths config.Paths
}

func NewStore(paths config.Paths) Store {
	return Store{paths: paths}
}

func (s Store) Load(env map[string]string) (Credentials, error) {
	if apiKey := strings.TrimSpace(env["OPENAI_API_KEY"]); apiKey != "" {
		return Credentials{APIKey: apiKey}, nil
	}

	data, err := os.ReadFile(s.paths.AuthFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, nil
		}
		return Credentials{}, fmt.Errorf("read auth file: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return Credentials{}, fmt.Errorf("parse auth file: %w", err)
	}

	return creds, nil
}

func (s Store) Save(creds Credentials) error {
	if strings.TrimSpace(creds.APIKey) == "" {
		return errors.New("api key is required")
	}

	if err := os.MkdirAll(s.paths.ConfigDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(s.paths.AuthFile, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write auth file: %w", err)
	}

	return nil
}
