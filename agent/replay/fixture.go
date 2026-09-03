package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/llm"
)

const FixtureVersion = "agent_replay_v1"

type LLMStep struct {
	Content      string    `json:"content,omitempty"`
	Model        string    `json:"model,omitempty"`
	FinishReason string    `json:"finish_reason,omitempty"`
	Error        string    `json:"error,omitempty"`
	DelayMs      int       `json:"delay_ms,omitempty"`
	Usage        llm.Usage `json:"usage,omitempty"`
}

type ToolStep struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
	DelayMs int             `json:"delay_ms,omitempty"`
}

type ToolMetadata struct {
	Risk           permission.RiskLevel `json:"risk,omitempty"`
	Idempotent     bool                 `json:"idempotent,omitempty"`
	SourceType     string               `json:"source_type,omitempty"`
	ProviderRef    string               `json:"provider_ref,omitempty"`
	TimeoutMs      int                  `json:"timeout_ms,omitempty"`
	CacheTTLms     int                  `json:"cache_ttl_ms,omitempty"`
	MaxResultBytes int                  `json:"max_result_bytes,omitempty"`
	InputSchema    json.RawMessage      `json:"input_schema,omitempty"`
	OutputSchema   json.RawMessage      `json:"output_schema,omitempty"`
}

type Fixture struct {
	Version         string                  `json:"version"`
	Name            string                  `json:"name"`
	Skill           string                  `json:"skill"`
	Input           string                  `json:"input"`
	ModelConfigID   int64                   `json:"model_config_id,omitempty"`
	LLM             []LLMStep               `json:"llm"`
	Tools           map[string][]ToolStep   `json:"tools,omitempty"`
	ToolMetadata    map[string]ToolMetadata `json:"tool_metadata,omitempty"`
	TimeoutMs       int                     `json:"timeout_ms,omitempty"`
	MaxContextBytes int                     `json:"max_context_bytes,omitempty"`
}

func Load(path string) (Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return Fixture{}, fmt.Errorf("decode replay fixture: %w", err)
	}
	if fixture.Version != FixtureVersion {
		return Fixture{}, fmt.Errorf("unsupported replay fixture version %q", fixture.Version)
	}
	if strings.TrimSpace(fixture.Name) == "" || strings.TrimSpace(fixture.Skill) == "" {
		return Fixture{}, fmt.Errorf("replay fixture requires name and skill")
	}
	expandFixtureTemplates(&fixture, time.Now().UTC())
	return fixture, nil
}

func expandFixtureTemplates(fixture *Fixture, now time.Time) {
	values := map[string]string{
		"{{NOW_RFC3339}}":          now.Format(time.RFC3339),
		"{{NOW_PLUS_10M_RFC3339}}": now.Add(10 * time.Minute).Format(time.RFC3339),
	}
	for index := range fixture.LLM {
		for key, value := range values {
			fixture.LLM[index].Content = strings.ReplaceAll(fixture.LLM[index].Content, key, value)
		}
	}
	for name, steps := range fixture.Tools {
		for index := range steps {
			raw := string(steps[index].Result)
			for key, value := range values {
				raw = strings.ReplaceAll(raw, key, value)
			}
			steps[index].Result = json.RawMessage(raw)
		}
		fixture.Tools[name] = steps
	}
}
