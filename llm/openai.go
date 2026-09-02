package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type openAIClient struct {
	cfg       Config
	transport *httpTransport
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func newOpenAIClient(cfg Config) (Client, error) {
	transport, err := newHTTPTransport(cfg)
	if err != nil {
		return nil, err
	}
	return &openAIClient{cfg: cfg, transport: transport}, nil
}

func (client *openAIClient) Provider() Provider {
	return client.cfg.Provider
}

func (client *openAIClient) ConfigID() int64 { return client.cfg.ID }

func (client *openAIClient) Generate(ctx context.Context, request Request) (*Response, error) {
	messages := make([]openAIMessage, 0, len(request.Messages)+1)
	if request.System != "" {
		messages = append(messages, openAIMessage{Role: RoleSystem, Content: request.System})
	}
	for _, message := range request.Messages {
		if message.Role != RoleSystem && message.Role != RoleUser && message.Role != RoleAssistant {
			return nil, fmt.Errorf("unsupported OpenAI message role %q", message.Role)
		}
		messages = append(messages, openAIMessage{Role: message.Role, Content: message.Content})
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	payload := openAIRequest{
		Model:       requestModel(request, client.cfg.Model),
		Messages:    messages,
		MaxTokens:   request.MaxTokens,
		Temperature: requestTemperature(request, client.cfg.Temperature),
	}
	headers := map[string]string{}
	if client.cfg.APIKey != "" {
		headers["Authorization"] = "Bearer " + client.cfg.APIKey
	}

	var result openAIResponse
	if err := client.transport.postJSON(ctx, client.cfg.APIURL, payload, &result, headers); err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI response contains no choices")
	}

	content, err := decodeOpenAIContent(result.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	return &Response{
		ID:           result.ID,
		Model:        result.Model,
		Content:      content,
		FinishReason: result.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	}, nil
}

func decodeOpenAIContent(raw json.RawMessage) (string, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", fmt.Errorf("decode OpenAI message content: %w", err)
	}

	var builder strings.Builder
	for _, part := range parts {
		if part.Type == "text" || part.Type == "output_text" {
			builder.WriteString(part.Text)
		}
	}
	return builder.String(), nil
}
