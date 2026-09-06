package symbolanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
	markettypes "go_binance_futures/types"
)

const Name = "symbol_analysis"

var chatSymbolPattern = regexp.MustCompile(`(?i)([A-Z0-9]{2,20}USDT)\b`)

type Input struct {
	Symbol string `json:"symbol,omitempty"`
	Prompt string `json:"prompt,omitempty"`
	Chat   bool   `json:"chat,omitempty"`
}

type PriceZone struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

type Evidence struct {
	Source  string `json:"source"`
	Finding string `json:"finding"`
}
type TradingPlanV1 struct {
	Version                string      `json:"version"`
	Symbol                 string      `json:"symbol"`
	AsOf                   string      `json:"as_of"`
	MarketCondition        *int        `json:"market_condition"`
	Direction              string      `json:"direction"`
	Confidence             float64     `json:"confidence"`
	Summary                string      `json:"summary"`
	EntryZones             []PriceZone `json:"entry_zones"`
	StopLoss               *float64    `json:"stop_loss"`
	TakeProfits            []float64   `json:"take_profits"`
	LongTrigger            string      `json:"long_trigger"`
	ShortTrigger           string      `json:"short_trigger"`
	InvalidationConditions []string    `json:"invalidation_conditions"`
	Risks                  []string    `json:"risks"`
	DataMissing            []string    `json:"data_missing"`
	Evidence               []Evidence  `json:"evidence"`
}

type Definition struct{}

func New() *Definition                   { return &Definition{} }
func (*Definition) Name() string         { return Name }
func (*Definition) SystemPrompt() string { return systemPrompt }
func (*Definition) VersionInfo() skill.VersionInfo {
	return skill.VersionInfo{
		SkillVersion: "1.0.0", PromptVersion: "1.0.0",
		InputContractVersion: "symbol_analysis_input_v1", OutputContractVersion: "trading_plan_v1",
		Source: skill.DefaultSource, SourceVersion: "v1",
	}
}
func (*Definition) Tools() []string {
	return []string{"get_symbol_analysis_context", "get_klines", "get_funding_rate", "get_liquidations", "get_symbol_snapshot", "get_market_condition"}
}
func (*Definition) ModelRequirements() llm.ModelRequirements {
	return llm.ModelRequirements{StructuredOutput: true, MinJSONReliability: 70}
}
func (*Definition) MaxRounds() int { return 15 }

func (*Definition) ChatEnabled() bool           { return true }
func (*Definition) PlainTextFinalAllowed() bool { return true }

func (*Definition) BuildChatInput(ctx context.Context, content string) (string, error) {
	return buildChatInput(ctx, content, nil, "")
}

func (*Definition) BuildChatInputWithContext(ctx context.Context, content string, previousInputs []string) (string, error) {
	return buildChatInput(ctx, content, previousInputs, "")
}

func (*Definition) BuildChatInputWithOptions(ctx context.Context, content string, previousInputs []string, options skill.ChatInputOptions) (string, error) {
	return buildChatInput(ctx, content, previousInputs, options.Symbol)
}

func buildChatInput(ctx context.Context, content string, previousInputs []string, explicitSymbol string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("chat content is required")
	}
	symbol := strings.ToUpper(strings.TrimSpace(explicitSymbol))
	if symbol != "" && !strings.HasSuffix(symbol, "USDT") {
		return "", fmt.Errorf("selected symbol must be a USDT futures contract")
	}
	if symbol == "" {
		if match := chatSymbolPattern.FindStringSubmatch(content); len(match) >= 2 {
			symbol = strings.ToUpper(match[1])
		}
	}
	if symbol == "" && !strings.Contains(strings.ToUpper(content), "USDT") && shouldReusePreviousSymbol(content) {
		for _, previous := range previousInputs {
			var input Input
			if json.Unmarshal([]byte(previous), &input) == nil {
				input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
				if strings.HasSuffix(input.Symbol, "USDT") {
					symbol = input.Symbol
					break
				}
			}
		}
	}
	raw, err := json.Marshal(Input{Symbol: symbol, Prompt: content, Chat: true})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func shouldReusePreviousSymbol(content string) bool {
	normalized := strings.ToLower(strings.TrimSpace(content))
	for _, marker := range []string{"刚才", "之前", "上面", "这个币", "该币", "它", "继续", "刚刚", "previous", "above", "this coin", "continue"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func (*Definition) RequiredTools(req skill.Request) []string {
	input, err := decodeInput(req.Input)
	if err == nil && input.Chat && input.Symbol == "" {
		return nil
	}
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
	focus := strings.TrimSpace(input.Prompt)
	if input.Chat && input.Symbol == "" {
		content := "[CHAT_MODE_NO_SYMBOL]\nUser message: " + focus + "\nNo exact Binance USDT contract symbol was resolved deterministically. Treat this as a normal conversation under the symbol-analysis skill. Do not invent a ticker or market data, and do not call symbol market-data tools without an exact valid symbol. If the user is not asking for a specific coin analysis, answer naturally."
		return []llm.Message{{Role: llm.RoleUser, Content: content}}, nil
	}
	if focus == "" {
		focus = "分析当前是否适合交易，并给出结构化交易计划"
	}
	content := fmt.Sprintf("Requested symbol: %s\nUser focus: %s\nFirst call get_symbol_analysis_context with this exact symbol. Then use optional tools only if more detail is needed.", input.Symbol, focus)
	return []llm.Message{{Role: llm.RoleUser, Content: content}}, nil
}

func (*Definition) Validator() validator.FinalValidator {
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return validatePlan(ctx, raw, "")
	})
}

func (*Definition) ValidatorFor(req skill.Request) validator.FinalValidator {
	input, err := decodeInput(req.Input)
	if err != nil {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) { return nil, err })
	}
	if input.Chat && input.Symbol == "" {
		return chatTextValidator()
	}
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		return validatePlan(ctx, raw, input.Symbol)
	})
}

func (*Definition) ValidatorForRun(req skill.Request, toolResults map[string]any) validator.FinalValidator {
	input, err := decodeInput(req.Input)
	if err != nil {
		return validator.Func(func(context.Context, json.RawMessage) (any, error) { return nil, err })
	}
	if input.Chat && input.Symbol == "" {
		return chatTextValidator()
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
		value, err := validatePlan(ctx, raw, input.Symbol)
		if err != nil {
			return nil, err
		}
		plan := value.(TradingPlanV1)
		if err := validatePlanAgainstContext(plan, analysisContext); err != nil {
			return nil, err
		}
		if err := validateEvidenceSourcesUsed(plan.Evidence, toolResults); err != nil {
			return nil, err
		}
		return plan, nil
	})
}

func (definition *Definition) ValidatorForRunWithEvidence(req skill.Request, toolResults map[string]any, evidence map[string]contextengine.Evidence) validator.FinalValidator {
	base := definition.ValidatorForRun(req, toolResults)
	input, err := decodeInput(req.Input)
	if err == nil && input.Chat && input.Symbol == "" {
		return base
	}
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		value, err := base.Validate(ctx, raw)
		if err != nil {
			return nil, err
		}
		result, ok := value.(TradingPlanV1)
		if !ok {
			return nil, fmt.Errorf("unexpected symbol analysis result type %T", value)
		}
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
		return Input{}, fmt.Errorf("decode symbol analysis input: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Input{}, err
	}
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.Prompt = strings.TrimSpace(input.Prompt)
	if input.Chat {
		if input.Prompt == "" {
			return Input{}, fmt.Errorf("chat prompt is required")
		}
		if input.Symbol != "" && !strings.HasSuffix(input.Symbol, "USDT") {
			return Input{}, fmt.Errorf("symbol must be a USDT futures contract when provided")
		}
		return input, nil
	}
	if input.Symbol == "" || !strings.HasSuffix(input.Symbol, "USDT") {
		return Input{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	return input, nil
}

func chatTextValidator() validator.FinalValidator {
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, fmt.Errorf("chat result must be a JSON string: %w", err)
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("chat result must not be empty")
		}
		return text, nil
	})
}
func validatePlan(ctx context.Context, raw json.RawMessage, expectedSymbol string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var plan TradingPlanV1
	if err := decoder.Decode(&plan); err != nil {
		return nil, fmt.Errorf("decode TradingPlanV1: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}
	if plan.Version != "trading_plan_v1" {
		return nil, fmt.Errorf("version must be trading_plan_v1")
	}
	plan.Symbol = strings.ToUpper(strings.TrimSpace(plan.Symbol))
	if expectedSymbol != "" && plan.Symbol != expectedSymbol {
		return nil, fmt.Errorf("symbol mismatch: got %s, expected %s", plan.Symbol, expectedSymbol)
	}
	if plan.MarketCondition != nil && !markettypes.IsValidMarketCondition(*plan.MarketCondition) {
		return nil, fmt.Errorf("market_condition must be between 1 and 11 or null")
	}
	if plan.Confidence < 0 || plan.Confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}
	if plan.Direction != "long" && plan.Direction != "short" && plan.Direction != "neutral" {
		return nil, fmt.Errorf("direction must be long, short or neutral")
	}
	asOf, err := time.Parse(time.RFC3339, strings.TrimSpace(plan.AsOf))
	if err != nil {
		return nil, fmt.Errorf("as_of must be RFC3339: %w", err)
	}
	if asOf.After(time.Now().UTC().Add(5 * time.Minute)) {
		return nil, fmt.Errorf("as_of must not be in the future")
	}
	if strings.TrimSpace(plan.Summary) == "" {
		return nil, fmt.Errorf("summary is required")
	}
	if plan.EntryZones == nil || plan.TakeProfits == nil || plan.InvalidationConditions == nil || plan.Risks == nil || plan.DataMissing == nil || plan.Evidence == nil {
		return nil, fmt.Errorf("entry_zones, take_profits, invalidation_conditions, risks, data_missing and evidence must be JSON arrays")
	}
	if len(plan.Risks) == 0 || len(plan.InvalidationConditions) == 0 {
		return nil, fmt.Errorf("risks and invalidation_conditions must each contain at least one item")
	}
	if err := validateStringList("data_missing", plan.DataMissing); err != nil {
		return nil, err
	}
	if err := validateEvidence(plan.Evidence); err != nil {
		return nil, err
	}
	if err := validatePrices(plan); err != nil {
		return nil, err
	}
	return plan, nil
}
func validatePlanAgainstContext(plan TradingPlanV1, analysisContext symbolanalysisservice.Context) error {
	if analysisContext.Symbol != plan.Symbol {
		return fmt.Errorf("tool context symbol %s does not match plan symbol %s", analysisContext.Symbol, plan.Symbol)
	}
	if analysisContext.MarketCondition == nil {
		if plan.MarketCondition != nil {
			return fmt.Errorf("market_condition must be null because tool context did not provide it")
		}
	} else {
		if plan.MarketCondition == nil || *plan.MarketCondition != analysisContext.MarketCondition.MarketCondition {
			return fmt.Errorf("market_condition must match tool context value %d", analysisContext.MarketCondition.MarketCondition)
		}
	}
	missingSet := make(map[string]bool, len(plan.DataMissing))
	for _, item := range plan.DataMissing {
		missingSet[strings.TrimSpace(item)] = true
	}
	for _, required := range analysisContext.DataMissing {
		if !missingSet[required] {
			return fmt.Errorf("data_missing must preserve tool context item %q", required)
		}
	}
	planTime, _ := time.Parse(time.RFC3339, plan.AsOf)
	contextTime, err := time.Parse(time.RFC3339, analysisContext.AsOf)
	if err == nil && planTime.Before(contextTime.Add(-time.Minute)) {
		return fmt.Errorf("as_of predates the symbol analysis context")
	}
	return validatePriceSanity(plan, analysisContext.Snapshot.Price)
}

func validatePriceSanity(plan TradingPlanV1, currentPrice float64) error {
	if currentPrice <= 0 {
		return nil
	}
	minimum, maximum := currentPrice*0.2, currentPrice*5
	check := func(label string, value float64) error {
		if value < minimum || value > maximum {
			return fmt.Errorf("%s %.8g is obviously inconsistent with current price %.8g", label, value, currentPrice)
		}
		return nil
	}
	for index, zone := range plan.EntryZones {
		if err := check(fmt.Sprintf("entry_zones[%d].low", index), zone.Low); err != nil {
			return err
		}
		if err := check(fmt.Sprintf("entry_zones[%d].high", index), zone.High); err != nil {
			return err
		}
	}
	if plan.StopLoss != nil {
		if err := check("stop_loss", *plan.StopLoss); err != nil {
			return err
		}
	}
	for index, target := range plan.TakeProfits {
		if err := check(fmt.Sprintf("take_profits[%d]", index), target); err != nil {
			return err
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

func validateEvidence(items []Evidence) error {
	if len(items) == 0 {
		return fmt.Errorf("evidence must contain at least one tool-backed item")
	}
	allowedNative := map[string]bool{
		"get_symbol_analysis_context": true,
		"get_klines":                  true,
		"get_funding_rate":            true,
		"get_liquidations":            true,
		"get_symbol_snapshot":         true,
		"get_market_condition":        true,
	}
	hasContext := false
	for index, item := range items {
		item.Source = strings.TrimSpace(item.Source)
		if !allowedNative[item.Source] && !strings.HasPrefix(item.Source, "mcp.") {
			return fmt.Errorf("evidence %d has unsupported source %q", index+1, item.Source)
		}
		if strings.TrimSpace(item.Finding) == "" {
			return fmt.Errorf("evidence %d finding is required", index+1)
		}
		if item.Source == "get_symbol_analysis_context" {
			hasContext = true
		}
	}
	if !hasContext {
		return fmt.Errorf("evidence must include get_symbol_analysis_context")
	}
	return nil
}

func validatePrices(plan TradingPlanV1) error {
	lowest, highest := 0.0, 0.0
	for index, zone := range plan.EntryZones {
		if zone.Low <= 0 || zone.High <= 0 || zone.Low > zone.High {
			return fmt.Errorf("entry_zones[%d] must contain positive low <= high", index)
		}
		if lowest == 0 || zone.Low < lowest {
			lowest = zone.Low
		}
		if zone.High > highest {
			highest = zone.High
		}
	}
	for index, target := range plan.TakeProfits {
		if target <= 0 {
			return fmt.Errorf("take_profits[%d] must be positive", index)
		}
	}
	if plan.StopLoss != nil && *plan.StopLoss <= 0 {
		return fmt.Errorf("stop_loss must be positive or null")
	}
	switch plan.Direction {
	case "long":
		if len(plan.EntryZones) == 0 || plan.StopLoss == nil || len(plan.TakeProfits) == 0 || strings.TrimSpace(plan.LongTrigger) == "" {
			return fmt.Errorf("long plan requires entry_zones, stop_loss, take_profits and long_trigger")
		}
		if *plan.StopLoss >= lowest {
			return fmt.Errorf("long stop_loss must be below every entry zone")
		}
		for _, target := range plan.TakeProfits {
			if target <= highest {
				return fmt.Errorf("long take_profits must be above every entry zone")
			}
		}
	case "short":
		if len(plan.EntryZones) == 0 || plan.StopLoss == nil || len(plan.TakeProfits) == 0 || strings.TrimSpace(plan.ShortTrigger) == "" {
			return fmt.Errorf("short plan requires entry_zones, stop_loss, take_profits and short_trigger")
		}
		if *plan.StopLoss <= highest {
			return fmt.Errorf("short stop_loss must be above every entry zone")
		}
		for _, target := range plan.TakeProfits {
			if target >= lowest {
				return fmt.Errorf("short take_profits must be below every entry zone")
			}
		}
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
