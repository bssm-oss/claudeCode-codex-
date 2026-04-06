package app

import (
	"bytes"
	"context"
	"encoding/json"
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

	entries, err := os.ReadDir(filepath.Join(homeDir, "transcripts"))
	if err != nil {
		t.Fatalf("read transcripts dir: %v", err)
	}
	transcripts := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".jsonl" {
			transcripts = append(transcripts, entry)
		}
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

func TestContinueUsesSavedSessionResponseID(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if requestCount == 1 {
			if _, ok := body["previous_response_id"]; ok {
				t.Fatalf("unexpected previous_response_id on first request: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":"resp_first","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"first"}]}]}`))
			return
		}
		if got, _ := body["previous_response_id"].(string); got != "resp_first" {
			t.Fatalf("unexpected previous_response_id: %#v", body)
		}
		_, _ = w.Write([]byte(`{"id":"resp_second","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"second"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
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

	var firstOut bytes.Buffer
	application := New(strings.NewReader(""), &firstOut, &firstOut)
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run first chat: %v", err)
	}
	if !strings.Contains(firstOut.String(), "first") {
		t.Fatalf("unexpected first output: %s", firstOut.String())
	}

	var continueOut bytes.Buffer
	application = New(strings.NewReader(""), &continueOut, &continueOut)
	if err := application.Run(context.Background(), []string{"continue", "resp_first", "follow up"}); err == nil {
		t.Fatal("expected invalid session selector to fail")
	}

	var sessionsOut bytes.Buffer
	application = New(strings.NewReader(""), &sessionsOut, &sessionsOut)
	if err := application.Run(context.Background(), []string{"sessions"}); err != nil {
		t.Fatalf("run sessions: %v", err)
	}
	fields := strings.Fields(sessionsOut.String())
	if len(fields) < 5 {
		t.Fatalf("unexpected sessions output: %s", sessionsOut.String())
	}
	sessionID := fields[4]

	continueOut.Reset()
	application = New(strings.NewReader(""), &continueOut, &continueOut)
	if err := application.Run(context.Background(), []string{"continue", sessionID, "follow up"}); err != nil {
		t.Fatalf("run continue: %v", err)
	}
	if !strings.Contains(continueOut.String(), "second") {
		t.Fatalf("unexpected continue output: %s", continueOut.String())
	}
	if !strings.Contains(continueOut.String(), "Session:") {
		t.Fatalf("missing session label: %s", continueOut.String())
	}
}

func TestContinueUnknownSelectorFails(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
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

	var out bytes.Buffer
	application := New(strings.NewReader(""), &out, &out)
	if err := application.Run(context.Background(), []string{"continue", "missing-session"}); err == nil {
		t.Fatal("expected unknown selector to fail")
	}
}

func TestSessionsRenameUpdatesSessionName(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"resp_name","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"named"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
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

	application := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run chat: %v", err)
	}

	var listOut bytes.Buffer
	application = New(strings.NewReader(""), &listOut, &listOut)
	if err := application.Run(context.Background(), []string{"sessions"}); err != nil {
		t.Fatalf("run sessions: %v", err)
	}
	fields := strings.Fields(listOut.String())
	if len(fields) < 5 {
		t.Fatalf("unexpected sessions output: %s", listOut.String())
	}
	id := fields[4]

	var renameOut bytes.Buffer
	application = New(strings.NewReader(""), &renameOut, &renameOut)
	if err := application.Run(context.Background(), []string{"sessions", "--rename", id, "main-chat"}); err != nil {
		t.Fatalf("rename session: %v", err)
	}

	listOut.Reset()
	application = New(strings.NewReader(""), &listOut, &listOut)
	if err := application.Run(context.Background(), []string{"sessions"}); err != nil {
		t.Fatalf("run sessions again: %v", err)
	}
	if !strings.Contains(listOut.String(), "(main-chat)") {
		t.Fatalf("expected renamed session in output: %s", listOut.String())
	}
}

func TestContinueRejectsWorkspaceMismatch(t *testing.T) {
	projectDir := t.TempDir()
	homeDir := t.TempDir()
	otherDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"resp_workspace","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"workspace"}]}]}`))
	}))
	defer server.Close()

	paths, err := config.ResolvePaths(homeDir, projectDir)
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
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

	application := New(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err := application.Run(context.Background(), []string{"chat", "hello"}); err != nil {
		t.Fatalf("run chat: %v", err)
	}

	if err := config.Save(paths, config.Config{Model: "gpt-5.4-mini", ApprovalMode: "ask", Workspace: otherDir, Transcripts: filepath.Join(homeDir, "transcripts"), OpenAIBaseURL: server.URL + "/v1"}); err != nil {
		t.Fatalf("save mismatched config: %v", err)
	}
	_ = os.Chdir(otherDir)
	var out bytes.Buffer
	application = New(strings.NewReader(""), &out, &out)
	if err := application.Run(context.Background(), []string{"continue"}); err == nil {
		t.Fatal("expected workspace mismatch to fail")
	}
}
