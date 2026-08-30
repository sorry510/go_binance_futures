package llm

import "context"

type Provider string

const (
	ProviderOpenAI           Provider = "openai"
	ProviderOpenAICompatible Provider = "openai_compatible"
	ProviderAnthropic        Provider = "anthropic"
	ProviderDeepSeek         Provider = "deepseek"
	ProviderGLM              Provider = "glm"
	ProviderMoonshot         Provider = "moonshot"
	ProviderOllama           Provider = "ollama"
	ProviderGemini           Provider = "gemini"
	ProviderClaudeSDK        Provider = "claude_sdk"
	ProviderCodexSDK         Provider = "codex_sdk"
)

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model       string    `json:"model,omitempty"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type Response struct {
	ID           string `json:"id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Model        string `json:"model,omitempty"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        Usage  `json:"usage,omitempty"`
}

type Client interface {
	Provider() Provider
	Generate(ctx context.Context, request Request) (*Response, error)
}
