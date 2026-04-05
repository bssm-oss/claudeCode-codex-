package openaiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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

type Message struct {
	Role       string
	Content    string
	ToolCallID string
	ToolName   string
	ToolCalls  []ToolCall
}

type TurnResult struct {
	Text      string
	ToolCalls []ToolCall
	Message   Message
}

type Client struct {
	inner *openai.Client
	model string
}

func New(apiKey, model string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("missing OPENAI_API_KEY or saved API key")
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-5.4-mini"
	}
	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithMaxRetries(2))
	return &Client{inner: &client, model: model}, nil
}

func (c *Client) Complete(ctx context.Context, messages []Message, tools []Tool) (TurnResult, error) {
	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.model),
		Messages: buildMessages(messages),
		Tools:    buildTools(tools),
	}

	completion, err := c.inner.Chat.Completions.New(ctx, params)
	if err != nil {
		return TurnResult{}, fmt.Errorf("create completion: %w", err)
	}

	message := completion.Choices[0].Message
	result := TurnResult{
		Text: strings.TrimSpace(message.Content),
		Message: Message{
			Role:    "assistant",
			Content: strings.TrimSpace(message.Content),
		},
	}

	for _, toolCall := range message.ToolCalls {
		if toolCall.Function.Name == "" {
			continue
		}
		call := ToolCall{
			ID:        toolCall.ID,
			Name:      toolCall.Function.Name,
			Arguments: json.RawMessage(toolCall.Function.Arguments),
		}
		result.ToolCalls = append(result.ToolCalls, call)
		result.Message.ToolCalls = append(result.Message.ToolCalls, call)
	}

	return result, nil
}

func buildMessages(history []Message) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history))
	for _, msg := range history {
		switch msg.Role {
		case "developer":
			messages = append(messages, openai.DeveloperMessage(msg.Content))
		case "assistant":
			if len(msg.ToolCalls) == 0 {
				messages = append(messages, openai.AssistantMessage(msg.Content))
				continue
			}

			toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(msg.ToolCalls))
			for _, call := range msg.ToolCalls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: call.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      call.Name,
							Arguments: string(call.Arguments),
						},
						Type: "function",
					},
				})
			}
			assistant := &openai.ChatCompletionAssistantMessageParam{
				Role:      "assistant",
				ToolCalls: toolCalls,
			}
			if msg.Content != "" {
				assistant.Content = openai.ChatCompletionAssistantMessageParamContentUnion{OfString: openai.String(msg.Content)}
			}
			messages = append(messages, openai.ChatCompletionMessageParamUnion{OfAssistant: assistant})
		case "tool":
			messages = append(messages, openai.ToolMessage(msg.Content, msg.ToolCallID))
		default:
			messages = append(messages, openai.UserMessage(msg.Content))
		}
	}
	return messages
}

func buildTools(tools []Tool) []openai.ChatCompletionToolUnionParam {
	result := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		result = append(result, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  tool.Schema,
		}))
	}
	return result
}
