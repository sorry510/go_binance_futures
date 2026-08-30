package llm

import "fmt"

func NewFromConfig() (Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewClient(cfg)
}

func NewClient(cfg Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Provider {
	case ProviderOpenAI, ProviderOpenAICompatible, ProviderDeepSeek, ProviderGLM, ProviderMoonshot, ProviderOllama, ProviderGemini:
		return newOpenAIClient(cfg)
	case ProviderAnthropic:
		return newAnthropicClient(cfg)
	case ProviderClaudeSDK, ProviderCodexSDK:
		return newBridgeClient(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.Provider)
	}
}
