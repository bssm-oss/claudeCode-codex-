package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bssm-oss/claudeCode-codex-/internal/config"
)

const (
	ModeAPIKey       = "apikey"
	ModeChatGPT      = "chatgpt"
	DefaultIssuerURL = "https://auth.openai.com"
	DefaultClientID  = "app_EMoamEEZ73f0CkXaXp7hrann"
)

type Credentials struct {
	AuthMode     string     `json:"auth_mode,omitempty"`
	OpenAIAPIKey string     `json:"OPENAI_API_KEY,omitempty"`
	Tokens       *TokenData `json:"tokens,omitempty"`
	LastRefresh  *time.Time `json:"last_refresh,omitempty"`
}

type TokenData struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id,omitempty"`
	Email        string `json:"-"`
	PlanType     string `json:"-"`
	UserID       string `json:"-"`
}

type DeviceCode struct {
	VerificationURL string
	UserCode        string
	DeviceAuthID    string
	IntervalSeconds int
}

type deviceCodeExchange struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeChallenge     string `json:"code_challenge"`
	CodeVerifier      string `json:"code_verifier"`
}

type userCodeResponse struct {
	DeviceAuthID string `json:"device_auth_id"`
	UserCode     string `json:"user_code"`
	Interval     string `json:"interval"`
}

type Store struct {
	paths config.Paths
}

func NewStore(paths config.Paths) Store {
	return Store{paths: paths}
}

func (c Credentials) Mode() string {
	if strings.TrimSpace(c.AuthMode) != "" {
		return strings.ToLower(strings.TrimSpace(c.AuthMode))
	}
	if strings.TrimSpace(c.OpenAIAPIKey) != "" {
		return ModeAPIKey
	}
	if c.Tokens != nil && strings.TrimSpace(c.Tokens.AccessToken) != "" {
		return ModeChatGPT
	}
	return ""
}

func (c Credentials) BearerToken() string {
	if c.Mode() == ModeChatGPT && c.Tokens != nil {
		return c.Tokens.AccessToken
	}
	return c.OpenAIAPIKey
}

func (c Credentials) AccountID() string {
	if c.Tokens == nil {
		return ""
	}
	if strings.TrimSpace(c.Tokens.AccountID) != "" {
		return c.Tokens.AccountID
	}
	return extractChatGPTClaims(c.Tokens.IDToken).ChatGPTAccountID
}

func (c Credentials) HasChatGPTAccessToken() bool {
	return c.Tokens != nil && strings.TrimSpace(c.Tokens.AccessToken) != ""
}

func (s Store) Load(env map[string]string) (Credentials, error) {
	if apiKey := strings.TrimSpace(env["OPENAI_API_KEY"]); apiKey != "" {
		return Credentials{AuthMode: ModeAPIKey, OpenAIAPIKey: apiKey}, nil
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
	if creds.Tokens != nil {
		claims := extractChatGPTClaims(creds.Tokens.IDToken)
		if creds.Tokens.AccountID == "" {
			creds.Tokens.AccountID = claims.ChatGPTAccountID
		}
		creds.Tokens.Email = claims.Email
		creds.Tokens.PlanType = claims.ChatGPTPlanType
		creds.Tokens.UserID = claims.ChatGPTUserID
	}

	return creds, nil
}

func (s Store) Save(creds Credentials) error {
	if strings.TrimSpace(creds.OpenAIAPIKey) == "" {
		return errors.New("api key is required")
	}
	creds.AuthMode = ModeAPIKey
	creds.Tokens = nil
	creds.LastRefresh = nil
	return s.write(creds)
}

func (s Store) SaveChatGPTTokens(tokens TokenData) error {
	if strings.TrimSpace(tokens.AccessToken) == "" || strings.TrimSpace(tokens.RefreshToken) == "" || strings.TrimSpace(tokens.IDToken) == "" {
		return errors.New("complete ChatGPT token bundle is required")
	}
	now := time.Now().UTC()
	creds := Credentials{
		AuthMode:    ModeChatGPT,
		Tokens:      &tokens,
		LastRefresh: &now,
	}
	return s.write(creds)
}

func (s Store) Logout() error {
	err := os.Remove(s.paths.AuthFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove auth file: %w", err)
	}
	return nil
}

func (s Store) write(creds Credentials) error {
	if creds.Tokens != nil && strings.TrimSpace(creds.Tokens.AccountID) == "" {
		creds.Tokens.AccountID = extractChatGPTClaims(creds.Tokens.IDToken).ChatGPTAccountID
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

func RequestDeviceCode(ctx context.Context, issuerURL, clientID string) (DeviceCode, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(issuerURL), "/")
	if baseURL == "" {
		baseURL = DefaultIssuerURL
	}
	if strings.TrimSpace(clientID) == "" {
		clientID = DefaultClientID
	}

	body, err := json.Marshal(map[string]string{"client_id": clientID})
	if err != nil {
		return DeviceCode{}, fmt.Errorf("marshal device code request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/accounts/deviceauth/usercode", bytes.NewReader(body))
	if err != nil {
		return DeviceCode{}, fmt.Errorf("build device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("request device code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("read device code response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return DeviceCode{}, fmt.Errorf("device code request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var parsed userCodeResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return DeviceCode{}, fmt.Errorf("parse device code response: %w", err)
	}
	interval := 5
	if parsed.Interval != "" {
		if parsedValue, err := strconv.Atoi(strings.TrimSpace(parsed.Interval)); err == nil && parsedValue >= 0 {
			interval = parsedValue
		}
	}

	return DeviceCode{
		VerificationURL: baseURL + "/codex/device",
		UserCode:        parsed.UserCode,
		DeviceAuthID:    parsed.DeviceAuthID,
		IntervalSeconds: interval,
	}, nil
}

func (s Store) CompleteDeviceCodeLogin(ctx context.Context, issuerURL, clientID string, code DeviceCode) error {
	baseURL := strings.TrimRight(strings.TrimSpace(issuerURL), "/")
	if baseURL == "" {
		baseURL = DefaultIssuerURL
	}
	if strings.TrimSpace(clientID) == "" {
		clientID = DefaultClientID
	}

	exchange, err := pollForAuthorizationCode(ctx, baseURL, code)
	if err != nil {
		return err
	}
	tokens, err := exchangeCodeForTokens(ctx, baseURL, clientID, exchange)
	if err != nil {
		return err
	}
	return s.SaveChatGPTTokens(tokens)
}

func pollForAuthorizationCode(ctx context.Context, issuerURL string, code DeviceCode) (deviceCodeExchange, error) {
	deadline := time.Now().Add(15 * time.Minute)
	for {
		body, err := json.Marshal(map[string]string{"device_auth_id": code.DeviceAuthID, "user_code": code.UserCode})
		if err != nil {
			return deviceCodeExchange{}, fmt.Errorf("marshal token poll request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuerURL+"/api/accounts/deviceauth/token", bytes.NewReader(body))
		if err != nil {
			return deviceCodeExchange{}, fmt.Errorf("build token poll request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return deviceCodeExchange{}, fmt.Errorf("poll device token: %w", err)
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return deviceCodeExchange{}, fmt.Errorf("read token poll response: %w", readErr)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			var parsed deviceCodeExchange
			if err := json.Unmarshal(data, &parsed); err != nil {
				return deviceCodeExchange{}, fmt.Errorf("parse token poll response: %w", err)
			}
			return parsed, nil
		case http.StatusForbidden, http.StatusNotFound:
			if time.Now().After(deadline) {
				return deviceCodeExchange{}, errors.New("device auth timed out after 15 minutes")
			}
			wait := time.Duration(code.IntervalSeconds) * time.Second
			select {
			case <-ctx.Done():
				return deviceCodeExchange{}, ctx.Err()
			case <-time.After(wait):
			}
		default:
			return deviceCodeExchange{}, fmt.Errorf("device auth failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
	}
}

func exchangeCodeForTokens(ctx context.Context, issuerURL, clientID string, exchange deviceCodeExchange) (TokenData, error) {
	form := strings.NewReader(fmt.Sprintf(
		"grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&code_verifier=%s",
		urlEncode(exchange.AuthorizationCode),
		urlEncode(strings.TrimRight(issuerURL, "/")+"/deviceauth/callback"),
		urlEncode(clientID),
		urlEncode(exchange.CodeVerifier),
	))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(issuerURL, "/")+"/oauth/token", form)
	if err != nil {
		return TokenData{}, fmt.Errorf("build oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TokenData{}, fmt.Errorf("exchange oauth code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenData{}, fmt.Errorf("read oauth token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return TokenData{}, fmt.Errorf("oauth token exchange failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var parsed struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return TokenData{}, fmt.Errorf("parse oauth token response: %w", err)
	}
	claims := extractChatGPTClaims(parsed.IDToken)
	return TokenData{
		IDToken:      parsed.IDToken,
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		AccountID:    claims.ChatGPTAccountID,
		Email:        claims.Email,
		PlanType:     claims.ChatGPTPlanType,
		UserID:       claims.ChatGPTUserID,
	}, nil
}

type tokenClaims struct {
	Email            string
	ChatGPTPlanType  string
	ChatGPTUserID    string
	ChatGPTAccountID string
}

func extractChatGPTClaims(jwt string) tokenClaims {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return tokenClaims{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}
	}
	var parsed map[string]any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return tokenClaims{}
	}
	claims := tokenClaims{Email: stringValue(parsed["email"])}
	if profile, ok := parsed["https://api.openai.com/profile"].(map[string]any); ok && claims.Email == "" {
		claims.Email = stringValue(profile["email"])
	}
	if authPayload, ok := parsed["https://api.openai.com/auth"].(map[string]any); ok {
		claims.ChatGPTPlanType = stringValue(authPayload["chatgpt_plan_type"])
		claims.ChatGPTUserID = stringValue(authPayload["chatgpt_user_id"])
		if claims.ChatGPTUserID == "" {
			claims.ChatGPTUserID = stringValue(authPayload["user_id"])
		}
		claims.ChatGPTAccountID = stringValue(authPayload["chatgpt_account_id"])
	}
	return claims
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func urlEncode(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"+", "%2B",
		"&", "%26",
		"=", "%3D",
		"?", "%3F",
		"/", "%2F",
		":", "%3A",
	)
	return replacer.Replace(value)
}
