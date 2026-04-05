package openaiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bssm-oss/claudeCode-codex-/internal/auth"
)

const (
	defaultModel          = "gpt-5.4-mini"
	defaultOpenAIBaseURL  = "https://api.openai.com/v1"
	defaultChatGPTBaseURL = "https://chatgpt.com/backend-api/codex"
)

type Tool struct {
	Name        string
	Description string
	Schema      map[string]any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type ToolOutput struct {
	CallID string
	Output string
}

type TurnInput struct {
	PreviousResponseID string
	Prompt             string
	ToolOutputs        []ToolOutput
}

type TurnResult struct {
	ResponseID string
	Text       string
	ToolCalls  []ToolCall
}

type Client struct {
	httpClient  *http.Client
	credentials auth.Credentials
	model       string
	baseURL     string
	chatMode    bool
}

type responsesRequest struct {
	Model              string              `json:"model"`
	Instructions       string              `json:"instructions"`
	PreviousResponseID string              `json:"previous_response_id,omitempty"`
	Input              []responseInputItem `json:"input"`
	Tools              []responseTool      `json:"tools,omitempty"`
	ToolChoice         string              `json:"tool_choice,omitempty"`
	ParallelToolCalls  bool                `json:"parallel_tool_calls"`
	Store              bool                `json:"store"`
	Stream             bool                `json:"stream"`
	Include            []string            `json:"include,omitempty"`
}

type responseInputItem struct {
	Type    string                `json:"type"`
	Role    string                `json:"role,omitempty"`
	Content []responseContentItem `json:"content,omitempty"`
	CallID  string                `json:"call_id,omitempty"`
	Output  string                `json:"output,omitempty"`
}

type responseContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responseTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type responsesResponse struct {
	ID     string                `json:"id"`
	Output []responsesOutputItem `json:"output"`
}

type responsesOutputItem struct {
	Type      string                `json:"type"`
	Role      string                `json:"role,omitempty"`
	Name      string                `json:"name,omitempty"`
	Arguments string                `json:"arguments,omitempty"`
	CallID    string                `json:"call_id,omitempty"`
	Content   []responsesOutputText `json:"content,omitempty"`
}

type responsesOutputText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func New(creds auth.Credentials, model, openAIBaseURL, chatGPTBaseURL string) (*Client, error) {
	if strings.TrimSpace(model) == "" {
		model = defaultModel
	}
	if strings.TrimSpace(openAIBaseURL) == "" {
		openAIBaseURL = defaultOpenAIBaseURL
	}
	if strings.TrimSpace(chatGPTBaseURL) == "" {
		chatGPTBaseURL = defaultChatGPTBaseURL
	}

	switch creds.Mode() {
	case auth.ModeAPIKey:
		if strings.TrimSpace(creds.OpenAIAPIKey) == "" {
			return nil, errors.New("missing OPENAI_API_KEY or saved API key")
		}
		return &Client{
			httpClient:  &http.Client{Timeout: 90 * time.Second},
			credentials: creds,
			model:       model,
			baseURL:     openAIBaseURL,
		}, nil
	case auth.ModeChatGPT:
		if !creds.HasChatGPTAccessToken() {
			return nil, errors.New("missing ChatGPT access token in auth.json")
		}
		if strings.TrimSpace(creds.AccountID()) == "" {
			return nil, errors.New("missing ChatGPT account id in auth.json")
		}
		return &Client{
			httpClient:  &http.Client{Timeout: 90 * time.Second},
			credentials: creds,
			model:       model,
			baseURL:     chatGPTBaseURL,
			chatMode:    true,
		}, nil
	default:
		return nil, errors.New("no supported auth mode is configured")
	}
}

func (c *Client) Complete(ctx context.Context, input TurnInput, instructions string, tools []Tool) (TurnResult, error) {
	requestBody := responsesRequest{
		Model:              c.model,
		Instructions:       instructions,
		PreviousResponseID: input.PreviousResponseID,
		Input:              buildInputItems(input),
		Tools:              buildTools(tools),
		ToolChoice:         "auto",
		ParallelToolCalls:  false,
		Store:              false,
		Stream:             false,
	}
	if c.chatMode {
		requestBody.Include = []string{"reasoning.encrypted_content"}
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return TurnResult{}, fmt.Errorf("marshal responses request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/responses", bytes.NewReader(payload))
	if err != nil {
		return TurnResult{}, fmt.Errorf("build responses request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.credentials.BearerToken())
	if c.chatMode {
		request.Header.Set("ChatGPT-Account-ID", c.credentials.AccountID())
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return TurnResult{}, fmt.Errorf("execute responses request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return TurnResult{}, fmt.Errorf("read responses body: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return TurnResult{}, fmt.Errorf("responses request failed: status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed responsesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return TurnResult{}, fmt.Errorf("parse responses body: %w", err)
	}

	result := TurnResult{ResponseID: parsed.ID}
	for _, item := range parsed.Output {
		switch item.Type {
		case "message":
			for _, content := range item.Content {
				if content.Type == "output_text" {
					if result.Text != "" {
						result.Text += "\n"
					}
					result.Text += content.Text
				}
			}
		case "function_call":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        item.CallID,
				Name:      item.Name,
				Arguments: json.RawMessage(item.Arguments),
			})
		}
	}

	return result, nil
}

func buildInputItems(input TurnInput) []responseInputItem {
	items := make([]responseInputItem, 0, len(input.ToolOutputs)+1)
	if strings.TrimSpace(input.Prompt) != "" {
		items = append(items, responseInputItem{
			Type: "message",
			Role: "user",
			Content: []responseContentItem{{
				Type: "input_text",
				Text: input.Prompt,
			}},
		})
	}
	for _, output := range input.ToolOutputs {
		items = append(items, responseInputItem{
			Type:   "function_call_output",
			CallID: output.CallID,
			Output: output.Output,
		})
	}
	return items
}

func buildTools(tools []Tool) []responseTool {
	result := make([]responseTool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, responseTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Schema,
		})
	}
	return result
}
