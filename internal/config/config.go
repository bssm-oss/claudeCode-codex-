package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultModel        = "gpt-5.4-mini"
	defaultApprovalMode = "ask"
	configDirName       = "claudecode-codex"
)

type Config struct {
	Model          string `json:"model"`
	ApprovalMode   string `json:"approval_mode"`
	Workspace      string `json:"workspace,omitempty"`
	Transcripts    string `json:"transcripts,omitempty"`
	OpenAIBaseURL  string `json:"openai_base_url,omitempty"`
	ChatGPTBaseURL string `json:"chatgpt_base_url,omitempty"`
}

type Paths struct {
	HomeDir        string
	ConfigDir      string
	ConfigFile     string
	AuthFile       string
	TranscriptDir  string
	ChangeLogDir   string
	DocsDir        string
	ProjectRootDir string
}

func ResolvePaths(homeDir, projectRoot string) (Paths, error) {
	if strings.TrimSpace(homeDir) == "" {
		return Paths{}, errors.New("home directory is required")
	}

	configDir := filepath.Join(homeDir, ".config", configDirName)
	transcripts := filepath.Join(configDir, "transcripts")

	return Paths{
		HomeDir:        homeDir,
		ConfigDir:      configDir,
		ConfigFile:     filepath.Join(configDir, "config.json"),
		AuthFile:       filepath.Join(configDir, "auth.json"),
		TranscriptDir:  transcripts,
		ChangeLogDir:   filepath.Join(projectRoot, "docs", "changes"),
		DocsDir:        filepath.Join(projectRoot, "docs"),
		ProjectRootDir: projectRoot,
	}, nil
}

func Default(projectRoot string) Config {
	return Config{
		Model:        defaultModel,
		ApprovalMode: defaultApprovalMode,
		Workspace:    projectRoot,
	}
}

func Load(paths Paths, projectRoot string) (Config, error) {
	cfg := Default(projectRoot)
	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.Transcripts = paths.TranscriptDir
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultModel
	}
	if strings.TrimSpace(cfg.ApprovalMode) == "" {
		cfg.ApprovalMode = defaultApprovalMode
	}
	if strings.TrimSpace(cfg.Workspace) == "" {
		cfg.Workspace = projectRoot
	}
	if strings.TrimSpace(cfg.Transcripts) == "" {
		cfg.Transcripts = paths.TranscriptDir
	}

	return cfg, nil
}

func Save(paths Paths, cfg Config) error {
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(paths.ConfigFile, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
