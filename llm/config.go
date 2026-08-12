package llm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	beegoConfig "github.com/beego/beego/v2/core/config"
)

const defaultTimeout = 60 * time.Second

type Config struct {
	Provider    Provider
	Model       string
	BaseURL     string
	Endpoint    string
	APIKey      string
	APIVersion  string
	ProxyURL    string
	Timeout     time.Duration
	MaxTokens   int
	Temperature *float64
	Headers     map[string]string

	Command     string
	Args        []string
	WorkingDir  string
	Environment map[string]string
}

func LoadConfig() (Config, error) {
	rawProvider := strings.TrimSpace(readString("llm::provider", ""))
	provider, section, err := normalizeProvider(rawProvider)
	if err != nil {
		return Config{}, err
	}

	prefix := section + "::"
	timeoutSeconds := readInt(prefix+"timeout_seconds", int(defaultTimeout/time.Second))
	maxTokens := readInt(prefix+"max_tokens", 1024)

	cfg := Config{
		Provider:    provider,
		Model:       strings.TrimSpace(readString(prefix+"model", "")),
		BaseURL:     strings.TrimSpace(readString(prefix+"base_url", defaultBaseURL(provider))),
		Endpoint:    strings.TrimSpace(readString(prefix+"endpoint", defaultEndpoint(provider))),
		APIKey:      strings.TrimSpace(readString(prefix+"api_key", "")),
		APIVersion:  strings.TrimSpace(readString(prefix+"api_version", defaultAPIVersion(provider))),
		ProxyURL:    strings.TrimSpace(readString(prefix+"proxy_url", "")),
		Timeout:     time.Duration(timeoutSeconds) * time.Second,
		MaxTokens:   maxTokens,
		Command:     strings.TrimSpace(readString(prefix+"command", defaultCommand(provider))),
		WorkingDir:  strings.TrimSpace(readString(prefix+"working_dir", ".")),
		Headers:     map[string]string{},
		Environment: map[string]string{},
	}

	if value := strings.TrimSpace(readString(prefix+"temperature", "")); value != "" {
		temperature, parseErr := strconv.ParseFloat(value, 64)
		if parseErr != nil {
			return Config{}, fmt.Errorf("invalid %stemperature: %w", prefix, parseErr)
		}
		cfg.Temperature = &temperature
	}

	if err := decodeJSONSetting(prefix+"headers", &cfg.Headers); err != nil {
		return Config{}, err
	}
	if err := decodeJSONSetting(prefix+"environment", &cfg.Environment); err != nil {
		return Config{}, err
	}
	if err := decodeJSONSetting(prefix+"args", &cfg.Args); err != nil {
		return Config{}, err
	}
	if len(cfg.Args) == 0 {
		cfg.Args = defaultArgs(provider)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.Timeout <= 0 {
		return fmt.Errorf("llm timeout must be greater than zero")
	}
	if cfg.MaxTokens <= 0 {
		return fmt.Errorf("llm max_tokens must be greater than zero")
	}

	switch cfg.Provider {
	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return fmt.Errorf("llm_openai::api_key is required")
		}
		fallthrough
	case ProviderOpenAICompatible:
		if cfg.BaseURL == "" || cfg.Endpoint == "" || cfg.Model == "" {
			return fmt.Errorf("base_url, endpoint and model are required for %s", cfg.Provider)
		}
	case ProviderAnthropic:
		if cfg.APIKey == "" {
			return fmt.Errorf("llm_anthropic::api_key is required")
		}
		if cfg.BaseURL == "" || cfg.Endpoint == "" || cfg.Model == "" || cfg.APIVersion == "" {
			return fmt.Errorf("base_url, endpoint, api_version and model are required for anthropic")
		}
	case ProviderClaudeSDK, ProviderCodexSDK:
		if cfg.Command == "" {
			return fmt.Errorf("command is required for %s", cfg.Provider)
		}
	default:
		return fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
	return nil
}

func normalizeProvider(value string) (Provider, string, error) {
	switch strings.ToLower(value) {
	case "openai", "chatgpt", "chatgpt_api":
		return ProviderOpenAI, "llm_openai", nil
	case "openai_compatible", "compatible":
		return ProviderOpenAICompatible, "llm_openai_compatible", nil
	case "anthropic", "claude", "claude_api":
		return ProviderAnthropic, "llm_anthropic", nil
	case "claude_sdk", "claude_agent_sdk":
		return ProviderClaudeSDK, "llm_claude_sdk", nil
	case "codex_sdk":
		return ProviderCodexSDK, "llm_codex_sdk", nil
	case "":
		return "", "", fmt.Errorf("llm::provider is required")
	default:
		return "", "", fmt.Errorf("unsupported llm::provider %q", value)
	}
}

func readString(key string, fallback string) string {
	value, err := beegoConfig.String(key)
	if err != nil || value == "" {
		return fallback
	}
	return value
}

func readInt(key string, fallback int) int {
	value, err := beegoConfig.Int(key)
	if err != nil {
		return fallback
	}
	return value
}

func decodeJSONSetting(key string, destination interface{}) error {
	value := strings.TrimSpace(readString(key, ""))
	if value == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(value), destination); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", key, err)
	}
	return nil
}

func defaultBaseURL(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "https://api.openai.com"
	case ProviderAnthropic:
		return "https://api.anthropic.com"
	default:
		return ""
	}
}

func defaultEndpoint(provider Provider) string {
	switch provider {
	case ProviderOpenAI, ProviderOpenAICompatible:
		return "/v1/chat/completions"
	case ProviderAnthropic:
		return "/v1/messages"
	default:
		return ""
	}
}

func defaultAPIVersion(provider Provider) string {
	if provider == ProviderAnthropic {
		return "2023-06-01"
	}
	return ""
}

func defaultCommand(provider Provider) string {
	switch provider {
	case ProviderClaudeSDK, ProviderCodexSDK:
		return "node"
	default:
		return ""
	}
}

func defaultArgs(provider Provider) []string {
	switch provider {
	case ProviderClaudeSDK:
		return []string{"llm/bridge/claude.mjs"}
	case ProviderCodexSDK:
		return []string{"llm/bridge/codex.mjs"}
	default:
		return nil
	}
}
