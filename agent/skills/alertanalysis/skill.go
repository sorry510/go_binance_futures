package alertanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/lang"
	"go_binance_futures/llm"
	signalservice "go_binance_futures/service/signal"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
)

const Name = "alert_analysis"

type Input struct {
	AlertID string               `json:"alert_id"`
	Signal  signalservice.Signal `json:"signal"`
}

type Evidence struct {
	Source  string `json:"source"`
	Finding string `json:"finding"`
}

type AlertV1 struct {
	Version       string                 `json:"version"`
	AlertID       string                 `json:"alert_id"`
	SignalID      string                 `json:"signal_id"`
	Symbol        string                 `json:"symbol"`
	Type          signalservice.Type     `json:"type"`
	Severity      signalservice.Severity `json:"severity"`
	Summary       string                 `json:"summary"`
	MarketContext string                 `json:"market_context"`
	ConfirmedBy   []string               `json:"confirmed_by"`
	Risks         []string               `json:"risks"`
	Action        string                 `json:"action"`
	CooldownUntil string                 `json:"cooldown_until"`
	DataMissing   []string               `json:"data_missing"`
	Evidence      []Evidence             `json:"evidence"`
}

type Definition struct{}

func New() *Definition           { return &Definition{} }
func (*Definition) Name() string { return Name }
func (*Definition) VersionInfo() skill.VersionInfo {
	return skill.VersionInfo{
		SkillVersion: "1.0.0", PromptVersion: "1.0.0",
		InputContractVersion: "alert_analysis_input_v1", OutputContractVersion: "alert_v1",
		Source: skill.DefaultSource, SourceVersion: "v1",
	}
}
func (*Definition) SystemPrompt() string {
	language := "English"
	if strings.HasPrefix(strings.ToLower(lang.GetLanguage()), "zh") {
		language = "Simplified Chinese"
	}
	return systemPrompt + "\nAll human-readable values in summary, market_context, confirmed_by, risks, data_missing and evidence.finding must use " + language + ". Preserve symbols, IDs, enum values, tool names and market units exactly."
}
func (*Definition) Tools() []string {
	return []string{"get_symbol_analysis_context", "get_klines", "get_funding_rate", "get_liquidations", "get_symbol_snapshot", "get_market_condition"}
}
func (*Definition) MaxRounds() int { return 6 }

func (*Definition) RequiredTools(skill.Request) []string {
	return []string{"get_symbol_analysis_context"}
}

func (*Definition) ValidateInput(req skill.Request) error {
	_, err := decodeInput(req.Input)
	return err
}

func (*Definition) BuildInput(ctx context.Context, req skill.Request) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	input, err := decodeInput(req.Input)
	if err != nil {
		return nil, err
	}
	rawSignal, err := json.Marshal(input.Signal)
	if err != nil {
		return nil, err
	}
	content := fmt.Sprintf(
		"Alert ID: %s\nPre-filtered signal: %s\nFirst call get_symbol_analysis_context with symbol %s. Then decide whether current evidence confirms, weakens or invalidates the signal.",
		input.AlertID, string(rawSignal), input.Signal.Symbol,
	)
	return []llm.Message{{Role: llm.RoleUser, Content: content}}, nil
}

func (*Definition) Validator() validator.FinalValidator {
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return validateAlert(ctx, raw, Input{})
	})
}

func (*Definition) ValidatorFor(req skill.Request) validator.FinalValidator {
	input, err := decodeInput(req.Input)
	if err != nil {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) { return nil, err })
	}
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return validateAlert(ctx, raw, input)
	})
}

func (*Definition) ValidatorForRun(req skill.Request, toolResults map[string]any) validator.FinalValidator {
	input, err := decodeInput(req.Input)
	if err != nil {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) { return nil, err })
	}
	contextValue, ok := toolResults["get_symbol_analysis_context"]
	if !ok {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) {
			return nil, fmt.Errorf("get_symbol_analysis_context result is required")
		})
	}
	analysisContext, ok := contextValue.(symbolanalysisservice.Context)
	if !ok {
		if pointer, pointerOK := contextValue.(*symbolanalysisservice.Context); pointerOK && pointer != nil {
			analysisContext, ok = *pointer, true
		}
	}
	if !ok {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) {
			return nil, fmt.Errorf("unexpected symbol analysis context type %T", contextValue)
		})
	}
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		value, err := validateAlert(ctx, raw, input)
		if err != nil {
			return nil, err
		}
		alert := value.(AlertV1)
		if err := validateAgainstContext(alert, analysisContext); err != nil {
			return nil, err
		}
		if err := validateEvidenceSourcesUsed(alert.Evidence, toolResults); err != nil {
			return nil, err
		}
		return alert, nil
	})
}

func (definition *Definition) ValidatorForRunWithEvidence(req skill.Request, toolResults map[string]any, evidence map[string]contextengine.Evidence) validator.FinalValidator {
	base := definition.ValidatorForRun(req, toolResults)
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		value, err := base.Validate(ctx, raw)
		if err != nil {
			return nil, err
		}
		result := value.(AlertV1)
		if err := validateStructuredEvidenceSources(result.Evidence, evidence); err != nil {
			return nil, err
		}
		return result, nil
	})
}

func decodeInput(raw string) (Input, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var input Input
	if err := decoder.Decode(&input); err != nil {
		return Input{}, fmt.Errorf("decode alert analysis input: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Input{}, err
	}
	input.AlertID = strings.TrimSpace(input.AlertID)
	input.Signal.Symbol = strings.ToUpper(strings.TrimSpace(input.Signal.Symbol))
	if input.AlertID == "" {
		return Input{}, fmt.Errorf("alert_id is required")
	}
	if input.Signal.SignalID == "" || input.Signal.EventID == "" {
		return Input{}, fmt.Errorf("signal_id and event_id are required")
	}
	if input.Signal.Symbol == "" || !strings.HasSuffix(input.Signal.Symbol, "USDT") {
		return Input{}, fmt.Errorf("signal symbol must be a USDT futures contract")
	}
	if input.Signal.Type == "" {
		return Input{}, fmt.Errorf("signal type is required")
	}
	if !signalservice.SeverityAtLeast(input.Signal.Severity, signalservice.SeverityLow) {
		return Input{}, fmt.Errorf("signal severity is invalid")
	}
	if input.Signal.Metrics == nil || input.Signal.Labels == nil || input.Signal.Evidence == nil {
		return Input{}, fmt.Errorf("signal metrics, labels and evidence must be JSON objects/arrays")
	}
	return input, nil
}

func validateAlert(ctx context.Context, raw json.RawMessage, expected Input) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var alert AlertV1
	if err := decoder.Decode(&alert); err != nil {
		return nil, fmt.Errorf("decode AlertV1: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	if alert.Version != "alert_v1" {
		return nil, fmt.Errorf("version must be alert_v1")
	}
	alert.AlertID = strings.TrimSpace(alert.AlertID)
	alert.SignalID = strings.TrimSpace(alert.SignalID)
	alert.Symbol = strings.ToUpper(strings.TrimSpace(alert.Symbol))
	if expected.AlertID != "" && alert.AlertID != expected.AlertID {
		return nil, fmt.Errorf("alert_id mismatch")
	}
	if expected.Signal.SignalID != "" && alert.SignalID != expected.Signal.SignalID {
		return nil, fmt.Errorf("signal_id mismatch")
	}
	if expected.Signal.Symbol != "" && alert.Symbol != expected.Signal.Symbol {
		return nil, fmt.Errorf("symbol mismatch: got %s, expected %s", alert.Symbol, expected.Signal.Symbol)
	}
	if expected.Signal.Type != "" && alert.Type != expected.Signal.Type {
		return nil, fmt.Errorf("type mismatch: got %s, expected %s", alert.Type, expected.Signal.Type)
	}
	if !signalservice.SeverityAtLeast(alert.Severity, signalservice.SeverityLow) {
		return nil, fmt.Errorf("severity must be low, medium, high or critical")
	}
	if alert.Action != "notify" && alert.Action != "record" && alert.Action != "ignore" {
		return nil, fmt.Errorf("action must be notify, record or ignore")
	}
	if strings.TrimSpace(alert.Summary) == "" || strings.TrimSpace(alert.MarketContext) == "" {
		return nil, fmt.Errorf("summary and market_context are required")
	}
	if alert.ConfirmedBy == nil || alert.Risks == nil || alert.DataMissing == nil || alert.Evidence == nil {
		return nil, fmt.Errorf("confirmed_by, risks, data_missing and evidence must be JSON arrays")
	}
	if len(alert.Risks) == 0 {
		return nil, fmt.Errorf("risks must contain at least one item")
	}
	if err := validateStringList("confirmed_by", alert.ConfirmedBy); err != nil {
		return nil, err
	}
	if err := validateStringList("risks", alert.Risks); err != nil {
		return nil, err
	}
	if err := validateStringList("data_missing", alert.DataMissing); err != nil {
		return nil, err
	}
	cooldownUntil, err := time.Parse(time.RFC3339, strings.TrimSpace(alert.CooldownUntil))
	if err != nil {
		return nil, fmt.Errorf("cooldown_until must be RFC3339: %w", err)
	}
	now := time.Now().UTC()
	if cooldownUntil.Before(now.Add(-time.Minute)) || cooldownUntil.After(now.Add(24*time.Hour)) {
		return nil, fmt.Errorf("cooldown_until must be between now and 24 hours from now")
	}
	if err := validateEvidence(alert.Evidence); err != nil {
		return nil, err
	}
	return alert, nil
}

func validateAgainstContext(alert AlertV1, analysisContext symbolanalysisservice.Context) error {
	if analysisContext.Symbol != alert.Symbol {
		return fmt.Errorf("tool context symbol %s does not match alert symbol %s", analysisContext.Symbol, alert.Symbol)
	}
	missingSet := make(map[string]bool, len(alert.DataMissing))
	for _, item := range alert.DataMissing {
		missingSet[strings.TrimSpace(item)] = true
	}
	for _, required := range analysisContext.DataMissing {
		if !missingSet[required] {
			return fmt.Errorf("data_missing must preserve tool context item %q", required)
		}
	}
	return nil
}

func validateEvidence(items []Evidence) error {
	if len(items) == 0 {
		return fmt.Errorf("evidence must contain at least one tool-backed item")
	}
	allowed := map[string]bool{
		"get_symbol_analysis_context": true,
		"get_klines":                  true, "get_funding_rate": true, "get_liquidations": true,
		"get_symbol_snapshot": true, "get_market_condition": true,
	}
	hasContext := false
	for index, item := range items {
		source := strings.TrimSpace(item.Source)
		if !allowed[source] {
			return fmt.Errorf("evidence %d has unsupported source %q", index+1, source)
		}
		if strings.TrimSpace(item.Finding) == "" {
			return fmt.Errorf("evidence %d finding is required", index+1)
		}
		if source == "get_symbol_analysis_context" {
			hasContext = true
		}
	}
	if !hasContext {
		return fmt.Errorf("evidence must include get_symbol_analysis_context")
	}
	return nil
}

func validateStructuredEvidenceSources(items []Evidence, registry map[string]contextengine.Evidence) error {
	for index, item := range items {
		source := strings.TrimSpace(item.Source)
		found := false
		usable := false
		for _, evidence := range registry {
			if strings.TrimSpace(evidence.Source) != source {
				continue
			}
			found = true
			if evidence.ContentHash != "" && evidence.Freshness != contextengine.FreshnessMissing {
				usable = true
			}
		}
		if !found {
			return fmt.Errorf("evidence %d source %q has no structured runtime evidence", index+1, source)
		}
		if !usable {
			return fmt.Errorf("evidence %d source %q has no usable structured runtime evidence", index+1, source)
		}
	}
	return nil
}

func validateEvidenceSourcesUsed(items []Evidence, toolResults map[string]any) error {
	for index, item := range items {
		if _, ok := toolResults[strings.TrimSpace(item.Source)]; !ok {
			return fmt.Errorf("evidence %d source %q was not successfully called in this run", index+1, item.Source)
		}
	}
	return nil
}

func validateStringList(name string, values []string) error {
	seen := make(map[string]bool, len(values))
	for index, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("%s[%d] must not be empty", name, index)
		}
		if seen[value] {
			return fmt.Errorf("%s contains duplicate %q", name, value)
		}
		seen[value] = true
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return fmt.Errorf("JSON contains multiple root values")
}
