package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

func TestStoreSaveAndLoadAPIKey(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := config.Paths{ConfigDir: tempDir, AuthFile: filepath.Join(tempDir, "auth.json")}
	store := NewStore(paths)

	if err := store.Save(Credentials{OpenAIAPIKey: "test-key"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	creds, err := store.Load(map[string]string{})
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}

	if creds.Mode() != ModeAPIKey || creds.OpenAIAPIKey != "test-key" {
		t.Fatalf("unexpected loaded credentials: %#v", creds)
	}
}

func TestStoreSaveAndLoadChatGPTTokens(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := config.Paths{ConfigDir: tempDir, AuthFile: filepath.Join(tempDir, "auth.json")}
	store := NewStore(paths)
	jwt := makeJWT(map[string]any{"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct_123", "chatgpt_plan_type": "pro"}})

	if err := store.SaveChatGPTTokens(TokenData{IDToken: jwt, AccessToken: "access", RefreshToken: "refresh"}); err != nil {
		t.Fatalf("save chatgpt tokens: %v", err)
	}

	creds, err := store.Load(map[string]string{})
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if creds.Mode() != ModeChatGPT || creds.AccountID() != "acct_123" {
		t.Fatalf("unexpected loaded chatgpt credentials: %#v", creds)
	}
}

func TestDeviceCodeLoginFlow(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	paths := config.Paths{ConfigDir: tempDir, AuthFile: filepath.Join(tempDir, "auth.json")}
	store := NewStore(paths)
	jwt := makeJWT(map[string]any{"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "acct_321"}})
	polls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/deviceauth/usercode":
			_ = json.NewEncoder(w).Encode(map[string]string{"device_auth_id": "device-auth-123", "user_code": "CODE-12345", "interval": "0"})
		case "/api/accounts/deviceauth/token":
			polls++
			if polls == 1 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"authorization_code": "poll-code-321", "code_challenge": "challenge", "code_verifier": "verifier"})
		case "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]string{"id_token": jwt, "access_token": "access-token-123", "refresh_token": "refresh-token-123"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	deviceCode, err := RequestDeviceCode(ctx, server.URL, "client-id")
	if err != nil {
		t.Fatalf("request device code: %v", err)
	}
	if err := store.CompleteDeviceCodeLogin(ctx, server.URL, "client-id", deviceCode); err != nil {
		t.Fatalf("complete device code login: %v", err)
	}

	loaded, err := store.Load(map[string]string{})
	if err != nil {
		t.Fatalf("load stored auth: %v", err)
	}
	if loaded.Mode() != ModeChatGPT || loaded.BearerToken() != "access-token-123" || loaded.AccountID() != "acct_321" {
		t.Fatalf("unexpected loaded auth after device login: %#v", loaded)
	}
}

func makeJWT(payload map[string]any) string {
	encode := func(value any) string {
		body, _ := json.Marshal(value)
		return base64.RawURLEncoding.EncodeToString(body)
	}
	return encode(map[string]string{"alg": "none", "typ": "JWT"}) + "." + encode(payload) + ".sig"
}
