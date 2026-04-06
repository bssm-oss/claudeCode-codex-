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

func TestChatOneShotRunsConfigAndPluginHooks(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"resp_hooks","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hook test ok"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".ccagent", "plugins", "sample"), 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifest := `{
		"name": "sample-plugin",
		"hooks": [
			{"event": "after_model", "command": "printf '%s:%s\n' \"$CCAGENT_HOOK_EVENT\" \"$CCAGENT_HOOK_SOURCE\" >> hook.log"}
		]
	}`
	if err := os.WriteFile(filepath.Join(projectDir, ".ccagent", "plugins", "sample", "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
	if err := config.Save(paths, config.Config{
		Model:         "gpt-5.4-mini",
		ApprovalMode:  "ask",
		Workspace:     projectDir,
		Transcripts:   filepath.Join(homeDir, "transcripts"),
		OpenAIBaseURL: server.URL + "/v1",
		Hooks: []config.HookConfig{{
			Event:   "session_start",
			Command: "printf '%s:%s\n' \"$CCAGENT_HOOK_EVENT\" \"$CCAGENT_HOOK_SOURCE\" >> hook.log",
		}},
	}); err != nil {
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
	application := New(strings.NewReader("y\ny\n"), &stdout, &stdout)
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run chat with hooks: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectDir, "hook.log"))
	if err != nil {
		t.Fatalf("read hook log: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "session_start:config") {
		t.Fatalf("missing config hook output: %s", text)
	}
	if !strings.Contains(text, "after_model:plugin:sample-plugin") {
		t.Fatalf("missing plugin hook output: %s", text)
	}

	transcripts, err := os.ReadDir(filepath.Join(homeDir, "transcripts"))
	if err != nil {
		t.Fatalf("read transcripts dir: %v", err)
	}
	if len(transcripts) != 1 {
		t.Fatalf("expected 1 transcript file, got %d", len(transcripts))
	}
	transcriptData, err := os.ReadFile(filepath.Join(homeDir, "transcripts", transcripts[0].Name()))
	if err != nil {
		t.Fatalf("read transcript file: %v", err)
	}
	transcriptText := string(transcriptData)
	if !strings.Contains(transcriptText, `"command":"printf '%s:%s\n' \"$CCAGENT_HOOK_EVENT\" \"$CCAGENT_HOOK_SOURCE\"`) {
		t.Fatalf("missing hook command in transcript: %s", transcriptText)
	}
	if !strings.Contains(transcriptText, `"status":"completed"`) {
		t.Fatalf("missing hook status in transcript: %s", transcriptText)
	}
}

func TestSessionsCommandListsAndSearchesTranscripts(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}
	if err := os.MkdirAll(paths.TranscriptDir, 0o755); err != nil {
		t.Fatalf("mkdir transcripts dir: %v", err)
	}
	transcript := filepath.Join(paths.TranscriptDir, "sample.jsonl")
	fixture := strings.Join([]string{
		`{"time":"2026-04-06T00:00:00Z","type":"user","payload":{"prompt":"hello codex"}}`,
		`{"time":"2026-04-06T00:00:01Z","type":"assistant","payload":{"content":"world"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(transcript, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write transcript fixture: %v", err)
	}
	if err := config.Save(paths, config.Config{Model: "gpt-5.4-mini", ApprovalMode: "ask", Workspace: projectDir, Transcripts: paths.TranscriptDir}); err != nil {
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

	var listOut bytes.Buffer
	application := New(strings.NewReader(""), &listOut, &listOut)
	if err := application.Run(context.Background(), []string{"sessions"}); err != nil {
		t.Fatalf("run sessions: %v", err)
	}
	if !strings.Contains(listOut.String(), "events=2") {
		t.Fatalf("unexpected sessions output: %s", listOut.String())
	}

	var searchOut bytes.Buffer
	application = New(strings.NewReader(""), &searchOut, &searchOut)
	if err := application.Run(context.Background(), []string{"sessions", "--query", "codex"}); err != nil {
		t.Fatalf("run session search: %v", err)
	}
	if !strings.Contains(searchOut.String(), "hello codex") {
		t.Fatalf("unexpected session search output: %s", searchOut.String())
	}
}
