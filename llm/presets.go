package llm

type ProviderPreset struct {
	Provider       Provider `json:"provider"`
	DisplayName    string   `json:"display_name"`
	APIURL         string   `json:"api_url"`
	APIVersion     string   `json:"api_version,omitempty"`
	APIKeyRequired bool     `json:"api_key_required"`
	Description    string   `json:"description"`
}

var providerPresets = []ProviderPreset{
	{Provider: ProviderOpenAI, DisplayName: "OpenAI", APIURL: "https://api.openai.com/v1/chat/completions", APIKeyRequired: true, Description: "OpenAI Chat Completions API"},
	{Provider: ProviderAnthropic, DisplayName: "Anthropic", APIURL: "https://api.anthropic.com/v1/messages", APIVersion: "2023-06-01", APIKeyRequired: true, Description: "Anthropic Messages API"},
	{Provider: ProviderDeepSeek, DisplayName: "DeepSeek", APIURL: "https://api.deepseek.com/chat/completions", APIKeyRequired: true, Description: "DeepSeek OpenAI-compatible API"},
	{Provider: ProviderGLM, DisplayName: "GLM / 智谱", APIURL: "https://open.bigmodel.cn/api/paas/v4/chat/completions", APIKeyRequired: true, Description: "智谱 GLM OpenAI-compatible API"},
	{Provider: ProviderMoonshot, DisplayName: "Moonshot / Kimi", APIURL: "https://api.moonshot.cn/v1/chat/completions", APIKeyRequired: true, Description: "Moonshot OpenAI-compatible API"},
	{Provider: ProviderOllama, DisplayName: "Ollama", APIURL: "http://127.0.0.1:11434/v1/chat/completions", APIKeyRequired: false, Description: "Local Ollama OpenAI-compatible API"},
	{Provider: ProviderGemini, DisplayName: "Gemini", APIURL: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", APIKeyRequired: true, Description: "Google Gemini OpenAI-compatible API"},
	{Provider: ProviderOpenAICompatible, DisplayName: "自定义 OpenAI-Compatible", APIURL: "", APIKeyRequired: false, Description: "Custom OpenAI-compatible Chat Completions endpoint"},
}

func ProviderPresets() []ProviderPreset {
	return append([]ProviderPreset(nil), providerPresets...)
}
