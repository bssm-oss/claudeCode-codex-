package app

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bssm-oss/claudeCode-codex-/internal/auth"
	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

func TestChatOneShotWithConfiguredOpenAIBaseURL(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"resp_test","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello from mock"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := config.Save(paths, config.Config{Model: "gpt-5.4-mini", ApprovalMode: "ask", Workspace: projectDir, Transcripts: filepath.Join(homeDir, "transcripts"), OpenAIBaseURL: server.URL + "/v1"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := auth.NewStore(paths).Save(auth.Credentials{OpenAIAPIKey: "sk-test"}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	oldHome := os.Getenv("HOME")
	oldWD, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Chdir(oldWD)
	})
	_ = os.Setenv("HOME", homeDir)
	_ = os.Chdir(projectDir)

	var stdout bytes.Buffer
	application := New(strings.NewReader(""), &stdout, &stdout)
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run chat: %v", err)
	}
	if !strings.Contains(stdout.String(), "hello from mock") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestDoctorReportsChatGPTModeFromStoredAuth(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	jwt := "header.payload.sig"
	if err := auth.NewStore(paths).SaveChatGPTTokens(auth.TokenData{IDToken: jwt, AccessToken: "access", RefreshToken: "refresh", AccountID: "acct_test"}); err != nil {
		t.Fatalf("save auth: %v", err)
	}
	if err := config.Save(paths, config.Config{Model: "gpt-5.4-mini", ApprovalMode: "ask", Workspace: projectDir, Transcripts: filepath.Join(homeDir, "transcripts")}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	oldHome := os.Getenv("HOME")
	oldWD, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Chdir(oldWD)
	})
	_ = os.Setenv("HOME", homeDir)
	_ = os.Chdir(projectDir)

	var stdout bytes.Buffer
	application := New(strings.NewReader(""), &stdout, &stdout)
	if err := application.Run(context.Background(), []string{"doctor"}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	if !strings.Contains(stdout.String(), fmt.Sprintf(`"account_id": %q`, "acct_test")) {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}

func TestChatOneShotWithConfiguredChatGPTBaseURL(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/codex/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("ChatGPT-Account-ID"); got != "acct_chat" {
			t.Fatalf("unexpected account header: %s", got)
		}
		_, _ = w.Write([]byte(`{"id":"resp_chatgpt","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"chatgpt mock ok"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if err := os.MkdirAll(paths.ConfigDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := config.Save(paths, config.Config{Model: "gpt-5.4-mini", ApprovalMode: "ask", Workspace: projectDir, Transcripts: filepath.Join(homeDir, "transcripts"), ChatGPTBaseURL: server.URL + "/api/codex"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := auth.NewStore(paths).SaveChatGPTTokens(auth.TokenData{IDToken: "header.payload.sig", AccessToken: "access-chat", RefreshToken: "refresh-chat", AccountID: "acct_chat"}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	oldHome := os.Getenv("HOME")
	oldWD, _ := os.Getwd()
	t.Cleanup(func() {
		_ = os.Setenv("HOME", oldHome)
		_ = os.Chdir(oldWD)
	})
	_ = os.Setenv("HOME", homeDir)
	_ = os.Chdir(projectDir)

	var stdout bytes.Buffer
	application := New(strings.NewReader(""), &stdout, &stdout)
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run chat: %v", err)
	}
	if !strings.Contains(stdout.String(), "chatgpt mock ok") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}
