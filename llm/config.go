package llm

import (
	"fmt"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second
const defaultAnthropicMaxTokens = 4096

var modelIdentifierDashReplacer = strings.NewReplacer(
	"\u2010", "-", // hyphen
	"\u2011", "-", // non-breaking hyphen
	"\u2012", "-", // figure dash
	"\u2013", "-", // en dash
	"\u2014", "-", // em dash
	"\u2212", "-", // minus sign
	"\uff0d", "-", // full-width hyphen-minus
)

func normalizeModelIdentifier(value string) string {
	return modelIdentifierDashReplacer.Replace(strings.TrimSpace(value))
}

type Config struct {
	ID          int64
	Provider    Provider
	Model       string
	APIURL      string
	APIKey      string
	APIVersion  string
	Timeout     time.Duration
	Temperature *float64

	Command     string
	Args        []string
	WorkingDir  string
	Environment map[string]string
}

func LoadConfig() (Config, error) {
	return (Store{}).ActiveConfig()
}
func (cfg Config) Validate() error {
	if cfg.Timeout <= 0 {
		return fmt.Errorf("llm timeout must be greater than zero")
	}

	switch cfg.Provider {
	case ProviderOpenAI, ProviderDeepSeek, ProviderGLM, ProviderMoonshot, ProviderGemini:
		if strings.TrimSpace(cfg.APIKey) == "" {
			return fmt.Errorf("api_key is required for %s", cfg.Provider)
		}
		fallthrough
	case ProviderOpenAICompatible, ProviderOllama:
		if strings.TrimSpace(cfg.APIURL) == "" || strings.TrimSpace(cfg.Model) == "" {
			return fmt.Errorf("api_url and model are required for %s", cfg.Provider)
		}
	case ProviderAnthropic:
		if strings.TrimSpace(cfg.APIKey) == "" {
			return fmt.Errorf("api_key is required for anthropic")
		}
		if strings.TrimSpace(cfg.APIURL) == "" || strings.TrimSpace(cfg.Model) == "" || strings.TrimSpace(cfg.APIVersion) == "" {
			return fmt.Errorf("api_url, api_version and model are required for anthropic")
		}
	case ProviderClaudeSDK, ProviderCodexSDK:
		if strings.TrimSpace(cfg.Command) == "" {
			return fmt.Errorf("command is required for %s", cfg.Provider)
		}
	default:
		return fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
	return nil
}
func normalizeProvider(value string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "openai", "chatgpt", "chatgpt_api":
		return ProviderOpenAI, nil
	case "openai_compatible", "compatible", "custom":
		return ProviderOpenAICompatible, nil
	case "anthropic", "claude", "claude_api":
		return ProviderAnthropic, nil
	case "deepseek":
		return ProviderDeepSeek, nil
	case "glm", "zhipu", "bigmodel":
		return ProviderGLM, nil
	case "moonshot", "kimi":
		return ProviderMoonshot, nil
	case "ollama":
		return ProviderOllama, nil
	case "gemini", "google":
		return ProviderGemini, nil
	case "claude_sdk", "claude_agent_sdk":
		return ProviderClaudeSDK, nil
	case "codex_sdk":
		return ProviderCodexSDK, nil
	case "":
		return "", fmt.Errorf("llm provider is required")
	default:
		return "", fmt.Errorf("unsupported llm provider %q", value)
	}
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
