package llm

import (
	"context"
	"fmt"
	"strings"
)

type anthropicClient struct {
	cfg       Config
	transport *httpTransport
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
}

type anthropicResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func newAnthropicClient(cfg Config) (Client, error) {
	transport, err := newHTTPTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &anthropicClient{cfg: cfg, transport: transport}, nil
}

func (client *anthropicClient) Provider() Provider {
	return ProviderAnthropic
}

func (client *anthropicClient) Generate(ctx context.Context, request Request) (*Response, error) {
	messages := make([]anthropicMessage, 0, len(request.Messages))
	systemParts := make([]string, 0, 2)
	if request.System != "" {
		systemParts = append(systemParts, request.System)
	}
	for _, message := range request.Messages {
		switch message.Role {
		case RoleSystem:
			systemParts = append(systemParts, message.Content)
		case RoleUser, RoleAssistant:
			messages = append(messages, anthropicMessage{Role: message.Role, Content: message.Content})
		default:
			return nil, fmt.Errorf("unsupported Anthropic message role %q", message.Role)
		}
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one user or assistant message is required")
	}

	payload := anthropicRequest{
		Model:       requestModel(request, client.cfg.Model),
		System:      strings.Join(systemParts, "\n\n"),
		Messages:    messages,
		MaxTokens:   requestMaxTokens(request, client.cfg.MaxTokens),
		Temperature: requestTemperature(request, client.cfg.Temperature),
	}
	headers := map[string]string{
		"x-api-key":         client.cfg.APIKey,
		"anthropic-version": client.cfg.APIVersion,
	}

	var result anthropicResponse
	if err := client.transport.postJSON(ctx, client.cfg.Endpoint, payload, &result, headers); err != nil {
		return nil, err
	}

	var content strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}
	return &Response{
		ID:           result.ID,
		Model:        result.Model,
		Content:      content.String(),
		FinishReason: result.StopReason,
		Usage: Usage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
			TotalTokens:  result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}
