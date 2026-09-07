package llm

import (
	"fmt"
	"net/url"
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
	ProxyURL    string
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
	if err := validateProxyURL(cfg.ProxyURL); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.ProxyURL) != "" && (cfg.Provider == ProviderClaudeSDK || cfg.Provider == ProviderCodexSDK) {
		return fmt.Errorf("llm proxy_url is only supported for HTTP API providers")
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

func validateProxyURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("llm proxy_url is invalid")
	}
	if strings.ToLower(parsed.Scheme) != "socks5h" {
		return fmt.Errorf("llm proxy_url must use socks5h://")
	}
	if strings.TrimSpace(parsed.Hostname()) == "" || strings.TrimSpace(parsed.Port()) == "" {
		return fmt.Errorf("llm proxy_url must include host and port")
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("llm proxy_url must not contain path, query or fragment")
	}
	return nil
}

func maskProxyURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "configured"
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(username, "********")
		}
	}
	return parsed.String()
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
