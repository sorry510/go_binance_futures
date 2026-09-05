package llm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	beegoConfig "github.com/beego/beego/v2/core/config"
)

type Store struct {
	Alias string
}

type ConfigInput struct {
	Name              string  `json:"name"`
	Provider          string  `json:"provider"`
	APIURL            string  `json:"api_url"`
	APIKey            string  `json:"api_key"`
	Model             string  `json:"model"`
	APIVersion        string  `json:"api_version"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	Temperature       float64 `json:"temperature"`
	Enabled           int     `json:"enabled"`
	RouterCandidate   *int    `json:"router_candidate,omitempty"`
	StructuredOutput  *int    `json:"structured_output,omitempty"`
	NativeToolCalling *int    `json:"native_tool_calling,omitempty"`
	Reasoning         *int    `json:"reasoning,omitempty"`
	LongContext       *int    `json:"long_context,omitempty"`
	JSONReliability   *int    `json:"json_reliability,omitempty"`
	MaxContextTokens  *int    `json:"max_context_tokens,omitempty"`
	CostClass         string  `json:"cost_class,omitempty"`
	LatencyClass      string  `json:"latency_class,omitempty"`
}
type PublicConfig struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	Provider          string  `json:"provider"`
	APIURL            string  `json:"api_url"`
	Model             string  `json:"model"`
	APIVersion        string  `json:"api_version,omitempty"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	Temperature       float64 `json:"temperature"`
	Enabled           int     `json:"enabled"`
	RouterCandidate   int     `json:"router_candidate"`
	StructuredOutput  int     `json:"structured_output"`
	NativeToolCalling int     `json:"native_tool_calling"`
	Reasoning         int     `json:"reasoning"`
	LongContext       int     `json:"long_context"`
	JSONReliability   int     `json:"json_reliability"`
	MaxContextTokens  int     `json:"max_context_tokens"`
	CostClass         string  `json:"cost_class"`
	LatencyClass      string  `json:"latency_class"`
	HasAPIKey         bool    `json:"has_api_key"`
	APIKeyMasked      string  `json:"api_key_masked,omitempty"`
	CreatedAt         int64   `json:"created_at"`
	UpdatedAt         int64   `json:"updated_at"`
}

func (store Store) ormer() orm.Ormer {
	if strings.TrimSpace(store.Alias) != "" {
		return orm.NewOrmUsingDB(store.Alias)
	}
	return orm.NewOrm()
}

func (store Store) List(ctx context.Context) ([]PublicConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.LLMConfig
	if _, err := store.ormer().QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).OrderBy("-enabled", "id").All(&rows); err != nil {
		return nil, err
	}
	result := make([]PublicConfig, 0, len(rows))
	for _, row := range rows {
		result = append(result, toPublicConfig(row))
	}
	return result, nil
}
func (store Store) Get(ctx context.Context, id int64) (*models.LLMConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item := &models.LLMConfig{ID: id}
	if err := store.ormer().Read(item); err != nil {
		return nil, err
	}
	if item.Deleted == 1 {
		return nil, orm.ErrNoRows
	}
	return item, nil
}

func (store Store) ActiveConfig() (Config, error) {
	var row models.LLMConfig
	err := store.ormer().QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).Filter("Enabled", 1).OrderBy("-updated_at").One(&row)
	if err != nil {
		if err == orm.ErrNoRows {
			return Config{}, fmt.Errorf("no enabled LLM configuration; configure one in AI -> LLM 配置")
		}
		return Config{}, err
	}
	return configFromModel(row)
}

func (store Store) Create(ctx context.Context, input ConfigInput) (*PublicConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	normalized, err := normalizeConfigInput(input, "", nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	o := store.ormer()
	var existing models.LLMConfig
	findErr := o.QueryTable(new(models.LLMConfig)).Filter("Name", normalized.Name).One(&existing)
	if findErr == nil {
		if existing.Deleted != 1 {
			return nil, fmt.Errorf("LLM configuration name %q already exists", normalized.Name)
		}
		return store.restore(ctx, &existing, normalized, now)
	}
	if findErr != orm.ErrNoRows {
		return nil, fmt.Errorf("find LLM configuration: %w", findErr)
	}
	row := modelFromInput(normalized)
	row.CreatedAt, row.UpdatedAt = now, now
	if normalized.Enabled == 1 {
		if _, err := o.QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).Update(orm.Params{"enabled": 0}); err != nil {
			return nil, err
		}
	}
	id, err := o.Insert(&row)
	if err != nil {
		return nil, err
	}
	row.ID = id
	result := toPublicConfig(row)
	return &result, nil
}

func (store Store) restore(ctx context.Context, row *models.LLMConfig, input ConfigInput, now int64) (*PublicConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o := store.ormer()
	if input.Enabled == 1 {
		if _, err := o.QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).Update(orm.Params{"enabled": 0}); err != nil {
			return nil, err
		}
	}
	id, createdAt := row.ID, row.CreatedAt
	*row = modelFromInput(input)
	row.ID, row.CreatedAt, row.UpdatedAt, row.Deleted = id, createdAt, now, 0
	if _, err := o.Update(row); err != nil {
		return nil, err
	}
	result := toPublicConfig(*row)
	return &result, nil
}
func (store Store) Update(ctx context.Context, id int64, input ConfigInput) (*PublicConfig, error) {
	row, err := store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		apiKey = row.APIKey
	}
	normalized, err := normalizeConfigInput(input, apiKey, row)
	if err != nil {
		return nil, err
	}
	o := store.ormer()
	if normalized.Enabled == 1 {
		if _, err := o.QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).Exclude("ID", id).Update(orm.Params{"enabled": 0}); err != nil {
			return nil, err
		}
	}
	createdAt := row.CreatedAt
	*row = modelFromInput(normalized)
	row.ID, row.CreatedAt, row.UpdatedAt = id, createdAt, time.Now().UnixMilli()
	if _, err := o.Update(row); err != nil {
		return nil, err
	}
	result := toPublicConfig(*row)
	return &result, nil
}

func (store Store) Delete(ctx context.Context, id int64) error {
	row, err := store.Get(ctx, id)
	if err != nil {
		return err
	}
	row.Enabled, row.Deleted, row.UpdatedAt = 0, 1, time.Now().UnixMilli()
	_, err = store.ormer().Update(row, "Enabled", "Deleted", "UpdatedAt")
	return err
}
func normalizeConfigInput(input ConfigInput, preservedAPIKey string, existing *models.LLMConfig) (ConfigInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Provider = strings.TrimSpace(input.Provider)
	input.APIURL = strings.TrimSpace(input.APIURL)
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.Model = normalizeModelIdentifier(input.Model)
	input.APIVersion = strings.TrimSpace(input.APIVersion)
	if input.APIKey == "" {
		input.APIKey = strings.TrimSpace(preservedAPIKey)
	}
	if input.Name == "" {
		return input, fmt.Errorf("configuration name is required")
	}
	applyRoutingProfileDefaults(&input, existing)
	for name, value := range map[string]*int{"router_candidate": input.RouterCandidate, "structured_output": input.StructuredOutput, "native_tool_calling": input.NativeToolCalling, "reasoning": input.Reasoning, "long_context": input.LongContext} {
		if value == nil || (*value != 0 && *value != 1) {
			return input, fmt.Errorf("%s must be 0 or 1", name)
		}
	}
	if input.JSONReliability == nil || *input.JSONReliability < 0 || *input.JSONReliability > 100 {
		return input, fmt.Errorf("json_reliability must be between 0 and 100")
	}
	if input.MaxContextTokens == nil || *input.MaxContextTokens < 0 {
		return input, fmt.Errorf("max_context_tokens must be >= 0")
	}
	provider, err := normalizeProvider(input.Provider)
	if err != nil {
		return input, err
	}
	input.Provider = string(provider)
	if input.Enabled != 0 && input.Enabled != 1 {
		return input, fmt.Errorf("enabled must be 0 or 1")
	}
	if input.TimeoutSeconds <= 0 {
		input.TimeoutSeconds = int(defaultTimeout / time.Second)
	}
	if input.Temperature < 0 || input.Temperature > 2 {
		return input, fmt.Errorf("temperature must be between 0 and 2")
	}
	applyPresetDefaults(&input, provider)
	if _, err := configFromInput(input); err != nil {
		return input, err
	}
	return input, nil
}

func intValueOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func applyRoutingProfileDefaults(input *ConfigInput, existing *models.LLMConfig) {
	base := models.LLMConfig{StructuredOutput: 1, JSONReliability: 80, CostClass: "medium", LatencyClass: "medium"}
	if existing != nil {
		base = *existing
	}
	if input.RouterCandidate == nil {
		v := base.RouterCandidate
		input.RouterCandidate = &v
	}
	if input.StructuredOutput == nil {
		v := base.StructuredOutput
		input.StructuredOutput = &v
	}
	if input.NativeToolCalling == nil {
		v := base.NativeToolCalling
		input.NativeToolCalling = &v
	}
	if input.Reasoning == nil {
		v := base.Reasoning
		input.Reasoning = &v
	}
	if input.LongContext == nil {
		v := base.LongContext
		input.LongContext = &v
	}
	if input.JSONReliability == nil {
		v := base.JSONReliability
		input.JSONReliability = &v
	}
	if input.MaxContextTokens == nil {
		v := base.MaxContextTokens
		input.MaxContextTokens = &v
	}
	if strings.TrimSpace(input.CostClass) == "" {
		input.CostClass = base.CostClass
	}
	if strings.TrimSpace(input.LatencyClass) == "" {
		input.LatencyClass = base.LatencyClass
	}
	if *input.JSONReliability < 0 || *input.JSONReliability > 100 {
		*input.JSONReliability = 80
	}
	if *input.MaxContextTokens < 0 {
		*input.MaxContextTokens = 0
	}
	input.CostClass = normalizeClass(input.CostClass)
	input.LatencyClass = normalizeClass(input.LatencyClass)
}

func normalizeClass(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "medium"
	}
}

func applyPresetDefaults(input *ConfigInput, provider Provider) {
	for _, preset := range providerPresets {
		if preset.Provider != provider {
			continue
		}
		if input.APIURL == "" {
			input.APIURL = preset.APIURL
		}
		if input.APIVersion == "" {
			input.APIVersion = preset.APIVersion
		}
		return
	}
}

func configFromInput(input ConfigInput) (Config, error) {
	input.Model = normalizeModelIdentifier(input.Model)
	provider, err := normalizeProvider(input.Provider)
	if err != nil {
		return Config{}, err
	}
	temperature := input.Temperature
	cfg := Config{
		Provider: provider, Model: input.Model, APIURL: input.APIURL, APIKey: input.APIKey,
		APIVersion: input.APIVersion, Timeout: time.Duration(input.TimeoutSeconds) * time.Second,
		Temperature: &temperature, Command: defaultCommand(provider), Args: defaultArgs(provider), WorkingDir: ".",
		Environment: map[string]string{},
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func configFromModel(row models.LLMConfig) (Config, error) {
	cfg, err := configFromInput(ConfigInput{
		Name: row.Name, Provider: row.Provider, APIURL: row.APIURL, APIKey: row.APIKey, Model: row.Model,
		APIVersion: row.APIVersion, TimeoutSeconds: row.TimeoutSeconds, Temperature: row.Temperature, Enabled: row.Enabled,
	})
	if err != nil {
		return Config{}, err
	}
	cfg.ID = row.ID
	return cfg, nil
}
func modelFromInput(input ConfigInput) models.LLMConfig {
	return models.LLMConfig{
		Name: input.Name, Provider: input.Provider, APIURL: input.APIURL, APIKey: input.APIKey,
		Model: input.Model, APIVersion: input.APIVersion, TimeoutSeconds: input.TimeoutSeconds, Temperature: input.Temperature, Enabled: input.Enabled,
		RouterCandidate: intValueOr(input.RouterCandidate, 0), StructuredOutput: intValueOr(input.StructuredOutput, 1), NativeToolCalling: intValueOr(input.NativeToolCalling, 0),
		Reasoning: intValueOr(input.Reasoning, 0), LongContext: intValueOr(input.LongContext, 0), JSONReliability: intValueOr(input.JSONReliability, 80),
		MaxContextTokens: intValueOr(input.MaxContextTokens, 0), CostClass: normalizeClass(input.CostClass), LatencyClass: normalizeClass(input.LatencyClass),
	}
}

func toPublicConfig(row models.LLMConfig) PublicConfig {
	return PublicConfig{
		ID: row.ID, Name: row.Name, Provider: row.Provider, APIURL: row.APIURL, Model: row.Model,
		APIVersion: row.APIVersion, TimeoutSeconds: row.TimeoutSeconds, Temperature: row.Temperature,
		Enabled: row.Enabled, RouterCandidate: row.RouterCandidate, StructuredOutput: row.StructuredOutput, NativeToolCalling: row.NativeToolCalling,
		Reasoning: row.Reasoning, LongContext: row.LongContext, JSONReliability: row.JSONReliability, MaxContextTokens: row.MaxContextTokens,
		CostClass: normalizeClass(row.CostClass), LatencyClass: normalizeClass(row.LatencyClass), HasAPIKey: strings.TrimSpace(row.APIKey) != "", APIKeyMasked: maskAPIKey(row.APIKey),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func maskAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:3] + "********" + value[len(value)-4:]
}
func EnsureDatabaseConfigFromLegacy(ctx context.Context) error {
	store := Store{}
	count, err := store.ormer().QueryTable(new(models.LLMConfig)).Count()
	if err != nil || count > 0 {
		return err
	}
	rawProvider := strings.TrimSpace(readLegacyString("llm::provider"))
	if rawProvider == "" {
		return nil
	}
	provider, err := normalizeProvider(rawProvider)
	if err != nil {
		return nil
	}
	section := legacySection(provider)
	if section == "" {
		return nil
	}
	prefix := section + "::"
	baseURL := strings.TrimSpace(readLegacyString(prefix + "base_url"))
	endpoint := strings.TrimSpace(readLegacyString(prefix + "endpoint"))
	apiURL := joinLegacyAPIURL(baseURL, endpoint)
	timeoutSeconds := readLegacyInt(prefix+"timeout_seconds", int(defaultTimeout/time.Second))
	temperature := readLegacyFloat(prefix+"temperature", 0.2)
	input := ConfigInput{
		Name: "Migrated " + string(provider), Provider: string(provider), APIURL: apiURL,
		APIKey: strings.TrimSpace(readLegacyString(prefix + "api_key")), Model: strings.TrimSpace(readLegacyString(prefix + "model")),
		APIVersion: strings.TrimSpace(readLegacyString(prefix + "api_version")), TimeoutSeconds: timeoutSeconds,
		Temperature: temperature, Enabled: 1,
	}
	if _, err := normalizeConfigInput(input, "", nil); err != nil {
		return nil
	}
	_, err = store.Create(ctx, input)
	return err
}
func legacySection(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "llm_openai"
	case ProviderOpenAICompatible, ProviderDeepSeek, ProviderGLM, ProviderMoonshot, ProviderOllama, ProviderGemini:
		return "llm_openai_compatible"
	case ProviderAnthropic:
		return "llm_anthropic"
	case ProviderClaudeSDK:
		return "llm_claude_sdk"
	case ProviderCodexSDK:
		return "llm_codex_sdk"
	default:
		return ""
	}
}

func readLegacyString(key string) string {
	value, err := beegoConfig.String(key)
	if err != nil {
		return ""
	}
	return value
}

func readLegacyInt(key string, fallback int) int {
	value, err := beegoConfig.Int(key)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
func readLegacyFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(readLegacyString(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func joinLegacyAPIURL(baseURL, endpoint string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint = strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	if baseURL == "" {
		return ""
	}
	if endpoint == "" {
		return baseURL
	}
	return baseURL + "/" + endpoint
}
func BuildConfig(input ConfigInput, preservedAPIKey string) (Config, error) {
	normalized, err := normalizeConfigInput(input, preservedAPIKey, nil)
	if err != nil {
		return Config{}, err
	}
	return configFromInput(normalized)
}
