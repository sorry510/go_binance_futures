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

// ConfigID returns the persisted LLM configuration id when the client was
// created from database-backed configuration. Test/custom clients may return 0.
func ConfigID(client Client) int64 {
	type configIdentified interface{ ConfigID() int64 }
	if identified, ok := client.(configIdentified); ok {
		return identified.ConfigID()
	}
	return 0
}
