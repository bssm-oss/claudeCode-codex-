package openaiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bssm-oss/claudeCode-codex-/internal/auth"
)

func TestAPIKeyModeUsesResponsesEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/v1/responses" {
			t.Fatalf("unexpected request path: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_1",
			"output": []map[string]any{{"type": "message", "role": "assistant", "content": []map[string]any{{"type": "output_text", "text": "hello"}}}},
		})
	}))
	defer server.Close()

	client, err := New(auth.Credentials{AuthMode: auth.ModeAPIKey, OpenAIAPIKey: "sk-test-key"}, "gpt-5.4-mini")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.baseURL = server.URL + "/v1"

	result, err := client.Complete(context.Background(), TurnInput{Prompt: "hello"}, "instructions", nil)
	if err != nil {
		t.Fatalf("complete request: %v", err)
	}
	if result.Text != "hello" {
		t.Fatalf("unexpected text: %#v", result)
	}
}

func TestChatGPTModeSendsAccountHeaderAndParsesToolCall(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/codex/responses" {
			t.Fatalf("unexpected request path: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if got := r.Header.Get("ChatGPT-Account-ID"); got != "acct-1" {
			t.Fatalf("unexpected account header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_2",
			"output": []map[string]any{{"type": "function_call", "call_id": "call_1", "name": "list_files", "arguments": `{"max_entries":5}`}},
		})
	}))
	defer server.Close()

	client, err := New(auth.Credentials{AuthMode: auth.ModeChatGPT, Tokens: &auth.TokenData{AccessToken: "access-token", RefreshToken: "refresh-token", IDToken: "ignored", AccountID: "acct-1"}}, "gpt-5.4-mini")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	client.baseURL = server.URL + "/api/codex"

	result, err := client.Complete(context.Background(), TurnInput{Prompt: "inspect repo"}, "instructions", []Tool{{Name: "list_files", Description: "List files", Schema: map[string]any{"type": "object"}}})
	if err != nil {
		t.Fatalf("complete request: %v", err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "list_files" {
		t.Fatalf("unexpected tool calls: %#v", result)
	}
}
