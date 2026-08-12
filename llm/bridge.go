package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type bridgeClient struct {
	cfg Config
}

type bridgeRequest struct {
	Request
	WorkingDir string `json:"working_dir,omitempty"`
}

type bridgeResponse struct {
	ID           string `json:"id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Model        string `json:"model,omitempty"`
	Content      string `json:"content,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        Usage  `json:"usage,omitempty"`
	Error        string `json:"error,omitempty"`
}

func newBridgeClient(cfg Config) Client {
	return &bridgeClient{cfg: cfg}
}

func (client *bridgeClient) Provider() Provider {
	return client.cfg.Provider
}

func (client *bridgeClient) Generate(ctx context.Context, request Request) (*Response, error) {
	if len(request.Messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}
	for _, message := range request.Messages {
		if message.Role != RoleSystem && message.Role != RoleUser && message.Role != RoleAssistant {
			return nil, fmt.Errorf("unsupported SDK bridge message role %q", message.Role)
		}
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, client.cfg.Timeout)
		defer cancel()
	}
	request.Model = requestModel(request, client.cfg.Model)
	request.MaxTokens = requestMaxTokens(request, client.cfg.MaxTokens)
	request.Temperature = requestTemperature(request, client.cfg.Temperature)

	input, err := json.Marshal(bridgeRequest{Request: request, WorkingDir: client.cfg.WorkingDir})
	if err != nil {
		return nil, fmt.Errorf("encode SDK bridge request: %w", err)
	}

	command := exec.CommandContext(ctx, client.cfg.Command, client.cfg.Args...)
	command.Dir = client.cfg.WorkingDir
	command.Env = client.environment()
	command.Stdin = bytes.NewReader(input)
	output, err := command.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if len(stderr) > maxErrorBodyBytes {
				stderr = stderr[:maxErrorBodyBytes]
			}
			return nil, fmt.Errorf("%s bridge failed: %s", client.cfg.Provider, stderr)
		}
		return nil, fmt.Errorf("run %s bridge: %w", client.cfg.Provider, err)
	}

	var result bridgeResponse
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("decode %s bridge response: %w", client.cfg.Provider, err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("%s bridge: %s", client.cfg.Provider, result.Error)
	}
	return &Response{
		ID:           result.ID,
		SessionID:    result.SessionID,
		Model:        result.Model,
		Content:      result.Content,
		FinishReason: result.FinishReason,
		Usage:        result.Usage,
	}, nil
}

func (client *bridgeClient) environment() []string {
	environment := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, found := strings.Cut(item, "=")
		if found {
			environment[key] = value
		}
	}
	for key, value := range client.cfg.Environment {
		environment[key] = value
	}
	if client.cfg.APIKey != "" {
		switch client.cfg.Provider {
		case ProviderClaudeSDK:
			environment["ANTHROPIC_API_KEY"] = client.cfg.APIKey
		case ProviderCodexSDK:
			environment["OPENAI_API_KEY"] = client.cfg.APIKey
		}
	}

	result := make([]string, 0, len(environment))
	for key, value := range environment {
		result = append(result, key+"="+value)
	}
	return result
}
