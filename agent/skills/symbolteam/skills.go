package symbolteam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
)

type kind string

const (
	kindTechnical  kind = "technical"
	kindFlow       kind = "flow"
	kindNews       kind = "news"
	kindSupervisor kind = "supervisor"
)

type Definition struct{ kind kind }

func Technical() *Definition  { return &Definition{kind: kindTechnical} }
func Flow() *Definition       { return &Definition{kind: kindFlow} }
func News() *Definition       { return &Definition{kind: kindNews} }
func Supervisor() *Definition { return &Definition{kind: kindSupervisor} }

func (d *Definition) Name() string {
	switch d.kind {
	case kindTechnical:
		return TechnicalSkillName
	case kindFlow:
		return FlowSkillName
	case kindNews:
		return NewsSkillName
	default:
		return SupervisorSkillName
	}
}

func (d *Definition) SystemPrompt() string {
	switch d.kind {
	case kindTechnical:
		return technicalPrompt
	case kindFlow:
		return flowPrompt
	case kindNews:
		return newsPrompt
	default:
		return supervisorPrompt
	}
}

func (*Definition) Tools() []string { return nil }
func (*Definition) MaxRounds() int  { return 3 }
func (*Definition) ModelRequirements() llm.ModelRequirements {
	return llm.ModelRequirements{StructuredOutput: true, MinJSONReliability: 75}
}

func (d *Definition) VersionInfo() skill.VersionInfo {
	input, output := "symbol_team_shared_input_v1", "technical_analysis_v1"
	if d.kind == kindFlow {
		output = "flow_analysis_v1"
	}
	if d.kind == kindNews {
		output = "news_analysis_v1"
	}
	if d.kind == kindSupervisor {
		input, output = "symbol_team_supervisor_input_v1", "symbol_team_supervisor_v1"
	}
	return skill.VersionInfo{
		SkillVersion: "1.1.0", PromptVersion: "1.1.0", InputContractVersion: input,
		OutputContractVersion: output, Source: skill.DefaultSource, SourceVersion: "v3-2",
	}
}

func (d *Definition) ValidateInput(req skill.Request) error {
	if d.kind == kindSupervisor {
		var input SupervisorInput
		if err := strictDecode(req.Input, &input); err != nil {
			return fmt.Errorf("decode supervisor input: %w", err)
		}
		if !validSymbol(input.Symbol) {
			return fmt.Errorf("symbol must be a USDT futures contract")
		}
		return nil
	}
	var input SharedInput
	if err := strictDecode(req.Input, &input); err != nil {
		return fmt.Errorf("decode shared input: %w", err)
	}
	if !validSymbol(input.Symbol) {
		return fmt.Errorf("symbol must be a USDT futures contract")
	}
	if len(input.SharedContext) == 0 || string(input.SharedContext) == "null" {
		return fmt.Errorf("shared_context is required")
	}
	return nil
}

func (d *Definition) BuildInput(ctx context.Context, req skill.Request) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := d.ValidateInput(req); err != nil {
		return nil, err
	}
	if d.kind == kindSupervisor {
		return []llm.Message{{Role: llm.RoleUser, Content: "Synthesize the typed child-agent results below. Missing or failed members must remain explicit; never invent their evidence.\n" + req.Input}}, nil
	}
	focus := "technical structure, multi-timeframe trend, volatility and key price levels"
	if d.kind == kindFlow {
		focus = "funding, open interest, taker flow, order-book depth and liquidations"
	}
	if d.kind == kindNews {
		focus = "only shared_context.market_intelligence events: official announcements, Alpha listings, external news and deterministic local signals; distinguish event_time from observed_at and treat stale items as stale"
	}
	return []llm.Message{{Role: llm.RoleUser, Content: "Analyze only this focus: " + focus + ". The shared_context was collected once by the Team Coordinator through get_symbol_analysis_context. Do not request new data and do not infer unavailable fields.\n" + req.Input}}, nil
}

func (d *Definition) Validator() validator.FinalValidator {
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		switch d.kind {
		case kindTechnical:
			var out TechnicalAnalysisV1
			if err := strictDecodeRaw(raw, &out); err != nil {
				return nil, err
			}
			if err := validateTechnical(out); err != nil {
				return nil, err
			}
			return out, nil
		case kindFlow:
			var out FlowAnalysisV1
			if err := strictDecodeRaw(raw, &out); err != nil {
				return nil, err
			}
			if err := validateFlow(out); err != nil {
				return nil, err
			}
			return out, nil
		case kindNews:
			var out NewsAnalysisV1
			if err := strictDecodeRaw(raw, &out); err != nil {
				return nil, err
			}
			if err := validateNews(out); err != nil {
				return nil, err
			}
			return out, nil
		default:
			var out SupervisorResultV1
			if err := strictDecodeRaw(raw, &out); err != nil {
				return nil, err
			}
			if err := validateSupervisor(out); err != nil {
				return nil, err
			}
			return out, nil
		}
	})
}

func (d *Definition) ValidatorFor(req skill.Request) validator.FinalValidator {
	base := d.Validator()
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		value, err := base.Validate(ctx, raw)
		if err != nil {
			return nil, err
		}
		var symbol string
		if d.kind == kindSupervisor {
			var input SupervisorInput
			_ = json.Unmarshal([]byte(req.Input), &input)
			symbol = input.Symbol
		} else {
			var input SharedInput
			_ = json.Unmarshal([]byte(req.Input), &input)
			symbol = input.Symbol
		}
		switch out := value.(type) {
		case TechnicalAnalysisV1:
			if out.Symbol != symbol {
				return nil, fmt.Errorf("symbol mismatch")
			}
		case FlowAnalysisV1:
			if out.Symbol != symbol {
				return nil, fmt.Errorf("symbol mismatch")
			}
		case NewsAnalysisV1:
			if out.Symbol != symbol {
				return nil, fmt.Errorf("symbol mismatch")
			}
		case SupervisorResultV1:
			if out.Symbol != symbol {
				return nil, fmt.Errorf("symbol mismatch")
			}
		}
		return value, nil
	})
}

func validSymbol(value string) bool {
	value = strings.ToUpper(strings.TrimSpace(value))
	return len(value) > 4 && strings.HasSuffix(value, "USDT")
}

func strictDecode(raw string, target any) error { return strictDecodeRaw(json.RawMessage(raw), target) }
func strictDecodeRaw(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON")
		}
		return err
	}
	return nil
}

func validateTechnical(out TechnicalAnalysisV1) error {
	if out.Version != "technical_analysis_v1" || !validSymbol(out.Symbol) || strings.TrimSpace(out.AsOf) == "" {
		return fmt.Errorf("invalid technical identity")
	}
	if out.Trend != "bullish" && out.Trend != "bearish" && out.Trend != "mixed" {
		return fmt.Errorf("invalid technical trend")
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	for _, level := range out.KeyLevels {
		if level.Price <= 0 || strings.TrimSpace(level.Kind) == "" {
			return fmt.Errorf("invalid key level")
		}
	}
	return validateEvidence(out.Evidence)
}

func validateFlow(out FlowAnalysisV1) error {
	if out.Version != "flow_analysis_v1" || !validSymbol(out.Symbol) || strings.TrimSpace(out.AsOf) == "" {
		return fmt.Errorf("invalid flow identity")
	}
	if out.Bias != "bullish" && out.Bias != "bearish" && out.Bias != "mixed" {
		return fmt.Errorf("invalid flow bias")
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	return validateEvidence(out.Evidence)
}

func validateNews(out NewsAnalysisV1) error {
	if out.Version != "news_analysis_v1" || !validSymbol(out.Symbol) || strings.TrimSpace(out.AsOf) == "" {
		return fmt.Errorf("invalid news identity")
	}
	if out.Bias != "bullish" && out.Bias != "bearish" && out.Bias != "mixed" && out.Bias != "neutral" {
		return fmt.Errorf("invalid news bias")
	}
	if out.Impact != "high" && out.Impact != "medium" && out.Impact != "low" && out.Impact != "none" {
		return fmt.Errorf("invalid news impact")
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	if strings.TrimSpace(out.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	for _, item := range out.Evidence {
		if item.Source != "market_intelligence" || strings.TrimSpace(item.Finding) == "" {
			return fmt.Errorf("invalid market-intelligence evidence")
		}
	}
	return nil
}

func validateEvidence(items []Evidence) error {
	if len(items) == 0 {
		return fmt.Errorf("evidence is required")
	}
	for _, item := range items {
		if item.Source != "get_symbol_analysis_context" || strings.TrimSpace(item.Finding) == "" {
			return fmt.Errorf("invalid shared-context evidence")
		}
	}
	return nil
}

func validateSupervisor(out SupervisorResultV1) error {
	if out.Version != "symbol_team_supervisor_v1" || !validSymbol(out.Symbol) || strings.TrimSpace(out.AsOf) == "" {
		return fmt.Errorf("invalid supervisor identity")
	}
	if out.Direction != "long" && out.Direction != "short" && out.Direction != "neutral" {
		return fmt.Errorf("invalid supervisor direction")
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		return fmt.Errorf("confidence must be 0..1")
	}
	if strings.TrimSpace(out.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	for _, item := range out.Evidence {
		if (item.Role != RoleTechnical && item.Role != RoleFlow && item.Role != RoleNews) || strings.TrimSpace(item.Source) == "" || strings.TrimSpace(item.Finding) == "" {
			return fmt.Errorf("invalid supervisor evidence")
		}
	}
	return nil
}

const technicalPrompt = `You are the Technical Analyst in a bounded trading-analysis team. Use only shared_context. Never call tools, never place orders, and never invent missing data. Return exactly one Runtime decision JSON object with top-level action="final" and result=<TechnicalAnalysisV1>. Do not place TechnicalAnalysisV1 fields at the top level. The result object must contain: version="technical_analysis_v1", symbol, as_of, trend(bullish|bearish|mixed), structure, volatility, key_levels:[{kind,price}], confidence(0..1), data_missing:[], evidence:[{source:"get_symbol_analysis_context",finding}]. Example shape: {"action":"final","result":{"version":"technical_analysis_v1",...}}.`

const flowPrompt = `You are the Flow Analyst in a bounded trading-analysis team. Use only shared_context. Never call tools, never place orders, and never invent missing data. Return exactly one Runtime decision JSON object with top-level action="final" and result=<FlowAnalysisV1>. Do not place FlowAnalysisV1 fields at the top level. The result object must contain: version="flow_analysis_v1", symbol, as_of, bias(bullish|bearish|mixed), funding, open_interest, taker, depth, liquidation, confidence(0..1), data_missing:[], evidence:[{source:"get_symbol_analysis_context",finding}]. Example shape: {"action":"final","result":{"version":"flow_analysis_v1",...}}.`

const newsPrompt = `You are the News Analyst in a bounded trading-analysis team. Use only shared_context.market_intelligence. Never call tools, never place orders, and never invent missing news or sources. Respect event_time, observed_at and freshness; stale events must not be presented as new catalysts. Return exactly one Runtime decision JSON object with top-level action="final" and result=<NewsAnalysisV1>. The result object must contain: version="news_analysis_v1", symbol, as_of, bias(bullish|bearish|mixed|neutral), impact(high|medium|low|none), summary, confidence(0..1), data_missing:[], evidence:[{source:"market_intelligence",finding}]. If there are no relevant fresh events, return bias="neutral", impact="none", explain that no fresh catalyst was found, and keep evidence empty rather than inventing it. Example shape: {"action":"final","result":{"version":"news_analysis_v1",...}}.`

const supervisorPrompt = `You are the Supervisor of a bounded trading-analysis team. You receive typed Technical, Flow and News child results. Do not call tools, do not place orders, and do not invent missing child evidence. A failed child is data_missing, not permission to guess. Return exactly one Runtime decision JSON object with top-level action="final" and result=<SymbolTeamSupervisorV1>. Do not place SymbolTeamSupervisorV1 fields at the top level. The result object must contain: version="symbol_team_supervisor_v1", symbol, as_of, direction(long|short|neutral), confidence(0..1), summary, consensus:[], disagreements:[], data_missing:[], evidence:[{role:"technical_analyst"|"flow_analyst"|"news_analyst",source,finding}]. Prefer neutral when child evidence conflicts or is insufficient. Example shape: {"action":"final","result":{"version":"symbol_team_supervisor_v1",...}}.`
