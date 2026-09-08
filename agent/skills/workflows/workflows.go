package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
	"go_binance_futures/scanner"
)

const (
	MarketScanName                = "market_scan"
	StrategyReviewName            = "strategy_review"
	StrategyExperimentProposeName = "strategy_experiment_propose"
	StrategyExperimentSummaryName = "strategy_experiment_summary"
	AlertTriageName               = "alert_triage"
	DailyMarketBriefName          = "daily_market_brief"
)

type TemplateSnapshot struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Technology string `json:"technology"`
	Strategy   string `json:"strategy"`
	UpdatedAt  int64  `json:"updated_at"`
}

type StrategyStats struct {
	Total       int     `json:"total"`
	Closed      int     `json:"closed"`
	Wins        int     `json:"wins"`
	Losses      int     `json:"losses"`
	WinRate     float64 `json:"win_rate"`
	GrossProfit float64 `json:"gross_profit"`
	NetProfit   float64 `json:"net_profit"`
	Fees        float64 `json:"fees"`
	AverageNet  float64 `json:"average_net"`
	LongTrades  int     `json:"long_trades"`
	ShortTrades int     `json:"short_trades"`
	WindowStart int64   `json:"window_start"`
	WindowEnd   int64   `json:"window_end"`
}

type MarketScanInput struct {
	Version         string                       `json:"version"`
	Prompt          string                       `json:"prompt,omitempty"`
	GeneratedAt     int64                        `json:"generated_at"`
	AsOf            string                       `json:"as_of,omitempty"`
	MarketCondition *int                         `json:"market_condition"`
	Candidates      []scanner.PrefilterCandidate `json:"candidates"`
	DataMissing     []string                     `json:"data_missing"`
}

type Opportunity struct {
	Rank       int      `json:"rank"`
	Symbol     string   `json:"symbol"`
	Score      float64  `json:"score"`
	Direction  string   `json:"direction"`
	Confidence float64  `json:"confidence"`
	Thesis     string   `json:"thesis"`
	Risks      []string `json:"risks"`
	Evidence   []string `json:"evidence"`
}

type OpportunitySetV1 struct {
	Version         string        `json:"version"`
	AsOf            string        `json:"as_of"`
	MarketCondition *int          `json:"market_condition"`
	Opportunities   []Opportunity `json:"opportunities"`
	DataMissing     []string      `json:"data_missing"`
}

type StrategyReviewInput struct {
	Version         string           `json:"version"`
	Prompt          string           `json:"prompt,omitempty"`
	Template        TemplateSnapshot `json:"template"`
	Stats           StrategyStats    `json:"stats"`
	MarketCondition *int             `json:"market_condition"`
	DataMissing     []string         `json:"data_missing"`
}

type StrategyReviewV1 struct {
	Version              string   `json:"version"`
	TemplateID           int64    `json:"template_id"`
	MarketCondition      *int     `json:"market_condition"`
	Verdict              string   `json:"verdict"`
	Confidence           float64  `json:"confidence"`
	Summary              string   `json:"summary"`
	SuitableEnvironments []int    `json:"suitable_environments"`
	FailureModes         []string `json:"failure_modes"`
	Proposals            []string `json:"proposals"`
	Evidence             []string `json:"evidence"`
}

type StrategyExperimentProposalInput struct {
	Version         string           `json:"version"`
	Template        TemplateSnapshot `json:"template"`
	Goal            string           `json:"goal"`
	MarketCondition *int             `json:"market_condition"`
}

type StrategyExperimentProposalV1 struct {
	Version        string   `json:"version"`
	BaseTemplateID int64    `json:"base_template_id"`
	CandidateName  string   `json:"candidate_name"`
	TechnologyJSON string   `json:"technology_json"`
	StrategyJSON   string   `json:"strategy_json"`
	Rationale      []string `json:"rationale"`
	Risks          []string `json:"risks"`
}

type ExperimentTestReport struct {
	Version          string   `json:"version"`
	Valid            bool     `json:"valid"`
	RuleCount        int      `json:"rule_count"`
	EnabledRuleCount int      `json:"enabled_rule_count"`
	CompiledRules    int      `json:"compiled_rules"`
	ScenarioRuns     int      `json:"scenario_runs"`
	ScenarioPasses   int      `json:"scenario_passes"`
	Errors           []string `json:"errors"`
}

type StrategyExperimentSummaryInput struct {
	Version  string                       `json:"version"`
	Proposal StrategyExperimentProposalV1 `json:"proposal"`
	Test     ExperimentTestReport         `json:"test"`
}

type StrategyExperimentResultV1 struct {
	Version         string               `json:"version"`
	BaseTemplateID  int64                `json:"base_template_id"`
	CandidateName   string               `json:"candidate_name"`
	TechnologyJSON  string               `json:"technology_json"`
	StrategyJSON    string               `json:"strategy_json"`
	Verdict         string               `json:"verdict"`
	Summary         string               `json:"summary"`
	Test            ExperimentTestReport `json:"test"`
	ProposedChanges []string             `json:"proposed_changes"`
	Risks           []string             `json:"risks"`
}

type IncidentSignal struct {
	SignalID  string `json:"signal_id"`
	Symbol    string `json:"symbol"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	CreatedAt int64  `json:"created_at"`
}

type IncidentCandidate struct {
	CandidateID string           `json:"candidate_id"`
	WindowStart int64            `json:"window_start"`
	WindowEnd   int64            `json:"window_end"`
	Symbols     []string         `json:"symbols"`
	Signals     []IncidentSignal `json:"signals"`
}

type AlertTriageInput struct {
	Version     string              `json:"version"`
	WindowStart int64               `json:"window_start"`
	WindowEnd   int64               `json:"window_end"`
	Candidates  []IncidentCandidate `json:"candidates"`
}

type Incident struct {
	IncidentID string   `json:"incident_id"`
	SignalIDs  []string `json:"signal_ids"`
	Symbols    []string `json:"symbols"`
	Severity   string   `json:"severity"`
	Action     string   `json:"action"`
	Summary    string   `json:"summary"`
	Rationale  string   `json:"rationale"`
}

type IncidentSetV1 struct {
	Version   string     `json:"version"`
	AsOf      string     `json:"as_of"`
	Incidents []Incident `json:"incidents"`
}

type SignalSummary struct {
	Total      int            `json:"total"`
	ByType     map[string]int `json:"by_type"`
	BySeverity map[string]int `json:"by_severity"`
	Symbols    []string       `json:"symbols"`
}

type DailyMarketBriefInput struct {
	Version         string                       `json:"version"`
	Prompt          string                       `json:"prompt,omitempty"`
	AsOf            string                       `json:"as_of"`
	MarketCondition *int                         `json:"market_condition"`
	Candidates      []scanner.PrefilterCandidate `json:"candidates"`
	Signals         SignalSummary                `json:"signals"`
	DataMissing     []string                     `json:"data_missing"`
}

type BriefOpportunity struct {
	Symbol string `json:"symbol"`
	Why    string `json:"why"`
}

type DailyMarketBriefV1 struct {
	Version         string             `json:"version"`
	AsOf            string             `json:"as_of"`
	MarketCondition *int               `json:"market_condition"`
	Headline        string             `json:"headline"`
	RegimeSummary   string             `json:"regime_summary"`
	Opportunities   []BriefOpportunity `json:"opportunities"`
	Incidents       []string           `json:"incidents"`
	Watchlist       []string           `json:"watchlist"`
	Risks           []string           `json:"risks"`
	DataMissing     []string           `json:"data_missing"`
}

type Definition struct{ kind string }

func MarketScan() *Definition                { return &Definition{kind: MarketScanName} }
func StrategyReview() *Definition            { return &Definition{kind: StrategyReviewName} }
func StrategyExperimentPropose() *Definition { return &Definition{kind: StrategyExperimentProposeName} }
func StrategyExperimentSummary() *Definition { return &Definition{kind: StrategyExperimentSummaryName} }
func AlertTriage() *Definition               { return &Definition{kind: AlertTriageName} }
func DailyMarketBrief() *Definition          { return &Definition{kind: DailyMarketBriefName} }

func (d *Definition) Name() string                 { return d.kind }
func (d *Definition) Tools() []string              { return nil }
func (d *Definition) MaxRounds() int               { return 0 }
func (d *Definition) DirectJSONFinalAllowed() bool { return true }
func (d *Definition) ModelRequirements() llm.ModelRequirements {
	req := llm.ModelRequirements{StructuredOutput: true, MinJSONReliability: 70}
	if d.kind == StrategyReviewName || strings.HasPrefix(d.kind, "strategy_experiment") || d.kind == AlertTriageName {
		req.Reasoning = true
		req.MinJSONReliability = 75
	}
	return req
}
func (d *Definition) VersionInfo() skill.VersionInfo {
	return skill.VersionInfo{SkillVersion: "1.0.0", PromptVersion: "1.0.2", InputContractVersion: inputVersion(d.kind), OutputContractVersion: outputVersion(d.kind), Source: skill.DefaultSource, SourceVersion: "v2-11"}
}
func (d *Definition) SystemPrompt() string {
	finalEnvelope := ` Return exactly one Agent Runtime final decision JSON object with this top-level shape: {"action":"final","summary":"concise summary","result":{...}}. Put the complete workflow output inside result. Never return the workflow output object at the top level. Do not return tool or parallel_tools actions.`
	contract := " The result field must match this exact schema and must not contain any other fields: " + outputContract(d.kind)
	switch d.kind {
	case MarketScanName:
		return "You rank only the deterministic market candidates provided by the system. Never claim to scan the full market yourself. Do not invent missing data or trading execution. Copy input.as_of and input.market_condition exactly, including null. Preserve every input.data_missing item. Use only candidate symbols from the input, at most once each. direction must be long, short, watch, or avoid; confidence must be 0..1. Do not copy scanner-only fields such as grade, price, percent_change_24h, quote_volume_24h, trade_count_24h, high_24h, low_24h, open_24h, reasons, missing, or last_update_time into an opportunity object." + contract + finalEnvelope
	case StrategyReviewName:
		return "Review the supplied strategy snapshot, deterministic fee-adjusted test statistics, and market condition. Copy input.template.id and input.market_condition exactly. Proposals are advisory only; never modify a template." + contract + finalEnvelope
	case StrategyExperimentProposeName:
		return "Propose one candidate strategy revision from the supplied template and goal. Copy input.template.id into base_template_id. technology_json and strategy_json must each be JSON encoded as a string. The candidate will be validated and tested deterministically; do not claim it has passed tests." + contract + finalEnvelope
	case StrategyExperimentSummaryName:
		return "Summarize the supplied strategy experiment proposal and deterministic test report. Copy proposal identity, technology_json, strategy_json, and the complete input.test object exactly. Never overwrite or activate a production strategy." + contract + finalEnvelope
	case AlertTriageName:
		return "Triage only the pre-grouped signal candidates provided by the deterministic incident builder. Every input signal_id must appear in exactly one incident. Decide whether to notify, suppress, or monitor; severity must be low, medium, high, or critical." + contract + finalEnvelope
	case DailyMarketBriefName:
		return "Create a concise daily market brief only from the supplied market condition, deterministic scanner candidates, and recent signal summary. Copy input.as_of and input.market_condition exactly, including null. Preserve every input.data_missing item. opportunities and watchlist may only reference symbols present in input.candidates or input.signals.symbols. Do not copy scanner fields into the result and do not invent live facts, prices, indicators, news, or fields that are not in the schema." + contract + finalEnvelope
	default:
		return "Return strict JSON inside an Agent Runtime final decision." + contract + finalEnvelope
	}
}
func (d *Definition) ValidateInput(req skill.Request) error {
	switch d.kind {
	case MarketScanName:
		var v MarketScanInput
		return strictDecodeString(req.Input, &v)
	case StrategyReviewName:
		var v StrategyReviewInput
		return strictDecodeString(req.Input, &v)
	case StrategyExperimentProposeName:
		var v StrategyExperimentProposalInput
		return strictDecodeString(req.Input, &v)
	case StrategyExperimentSummaryName:
		var v StrategyExperimentSummaryInput
		return strictDecodeString(req.Input, &v)
	case AlertTriageName:
		var v AlertTriageInput
		return strictDecodeString(req.Input, &v)
	case DailyMarketBriefName:
		var v DailyMarketBriefInput
		return strictDecodeString(req.Input, &v)
	}
	return fmt.Errorf("unsupported workflow skill %q", d.kind)
}
func (d *Definition) BuildInput(ctx context.Context, req skill.Request) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := d.ValidateInput(req); err != nil {
		return nil, err
	}
	return []llm.Message{{Role: llm.RoleUser, Content: req.Input}}, nil
}
func (d *Definition) Validator() validator.FinalValidator { return d.validatorFor("") }
func (d *Definition) ValidatorFor(req skill.Request) validator.FinalValidator {
	return d.validatorFor(req.Input)
}
func (d *Definition) validatorFor(input string) validator.FinalValidator {
	return validator.Func(func(ctx context.Context, raw json.RawMessage) (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		switch d.kind {
		case MarketScanName:
			var out OpportunitySetV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateOpportunitySet(out, input)
			return outputContractResult(d.kind, input, value, err)
		case StrategyReviewName:
			var out StrategyReviewV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateStrategyReview(out, input)
			return outputContractResult(d.kind, input, value, err)
		case StrategyExperimentProposeName:
			var out StrategyExperimentProposalV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateExperimentProposal(out, input)
			return outputContractResult(d.kind, input, value, err)
		case StrategyExperimentSummaryName:
			var out StrategyExperimentResultV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateExperimentResult(out, input)
			return outputContractResult(d.kind, input, value, err)
		case AlertTriageName:
			var out IncidentSetV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateIncidentSet(out, input)
			return outputContractResult(d.kind, input, value, err)
		case DailyMarketBriefName:
			var out DailyMarketBriefV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, outputContractError(d.kind, input, err)
			}
			value, err := validateDailyBrief(out, input)
			return outputContractResult(d.kind, input, value, err)
		}
		return nil, fmt.Errorf("unsupported workflow skill %q", d.kind)
	})
}

func inputVersion(kind string) string {
	switch kind {
	case MarketScanName:
		return "market_scan_input_v1"
	case StrategyReviewName:
		return "strategy_review_input_v1"
	case StrategyExperimentProposeName:
		return "strategy_experiment_proposal_input_v1"
	case StrategyExperimentSummaryName:
		return "strategy_experiment_summary_input_v1"
	case AlertTriageName:
		return "alert_triage_input_v1"
	case DailyMarketBriefName:
		return "daily_market_brief_input_v1"
	default:
		return "workflow_input_v1"
	}
}
func outputVersion(kind string) string {
	switch kind {
	case MarketScanName:
		return "opportunity_set_v1"
	case StrategyReviewName:
		return "strategy_review_v1"
	case StrategyExperimentProposeName:
		return "strategy_experiment_proposal_v1"
	case StrategyExperimentSummaryName:
		return "strategy_experiment_result_v1"
	case AlertTriageName:
		return "incident_set_v1"
	case DailyMarketBriefName:
		return "daily_market_brief_v1"
	default:
		return "workflow_output_v1"
	}
}

func outputContract(kind string) string {
	switch kind {
	case MarketScanName:
		return `{"version":"opportunity_set_v1","as_of":"<copy input.as_of exactly>","market_condition":3,"opportunities":[{"rank":1,"symbol":"SOLUSDT","score":100,"direction":"long","confidence":0.8,"thesis":"concise thesis","risks":["concrete risk"],"evidence":["fact from supplied candidate"]}],"data_missing":["preserve input items"]}`
	case StrategyReviewName:
		return `{"version":"strategy_review_v1","template_id":1,"market_condition":3,"verdict":"revise","confidence":0.8,"summary":"concise review","suitable_environments":[1,2],"failure_modes":["..."],"proposals":["..."],"evidence":["..."]}`
	case StrategyExperimentProposeName:
		return `{"version":"strategy_experiment_proposal_v1","base_template_id":1,"candidate_name":"...","technology_json":"{\"ma\":[]}","strategy_json":"[]","rationale":["..."],"risks":["..."]}`
	case StrategyExperimentSummaryName:
		return `{"version":"strategy_experiment_result_v1","base_template_id":1,"candidate_name":"...","technology_json":"<copy proposal exactly>","strategy_json":"<copy proposal exactly>","verdict":"promising","summary":"...","test":{"version":"strategy_experiment_test_v1","valid":true,"rule_count":1,"enabled_rule_count":1,"compiled_rules":1,"scenario_runs":3,"scenario_passes":3,"errors":[]},"proposed_changes":["..."],"risks":["..."]}`
	case AlertTriageName:
		return `{"version":"incident_set_v1","as_of":"<RFC3339>","incidents":[{"incident_id":"...","signal_ids":["..."],"symbols":["BTCUSDT"],"severity":"high","action":"notify","summary":"...","rationale":"..."}]}`
	case DailyMarketBriefName:
		return `{"version":"daily_market_brief_v1","as_of":"<copy input.as_of exactly>","market_condition":3,"headline":"concise headline","regime_summary":"concise market-regime summary","opportunities":[{"symbol":"SOLUSDT","why":"why this supplied candidate matters"}],"incidents":["concise incident summary derived from input.signals"],"watchlist":["SOLUSDT"],"risks":["market or data-quality risk"],"data_missing":["preserve input items"]}`
	default:
		return `{}`
	}
}
func outputContractForInput(kind, input string) string {
	contract := outputContract(kind)
	if strings.TrimSpace(input) == "" {
		return contract
	}
	replaceValue := func(key, example string, value any) {
		raw, err := json.Marshal(value)
		if err == nil {
			contract = strings.Replace(contract, fmt.Sprintf(`%q:%s`, key, example), fmt.Sprintf(`%q:%s`, key, raw), 1)
		}
	}
	switch kind {
	case MarketScanName:
		var in MarketScanInput
		if strictDecodeString(input, &in) == nil {
			contract = strings.Replace(contract, `"as_of":"<copy input.as_of exactly>"`, `"as_of":`+string(mustJSON(in.AsOf)), 1)
			replaceValue("market_condition", "3", in.MarketCondition)
		}
	case StrategyReviewName:
		var in StrategyReviewInput
		if strictDecodeString(input, &in) == nil {
			replaceValue("template_id", "1", in.Template.ID)
			replaceValue("market_condition", "3", in.MarketCondition)
		}
	case StrategyExperimentProposeName:
		var in StrategyExperimentProposalInput
		if strictDecodeString(input, &in) == nil {
			replaceValue("base_template_id", "1", in.Template.ID)
		}
	case StrategyExperimentSummaryName:
		var in StrategyExperimentSummaryInput
		if strictDecodeString(input, &in) == nil {
			replaceValue("base_template_id", "1", in.Proposal.BaseTemplateID)
			replaceValue("candidate_name", `"..."`, in.Proposal.CandidateName)
		}
	case DailyMarketBriefName:
		var in DailyMarketBriefInput
		if strictDecodeString(input, &in) == nil {
			contract = strings.Replace(contract, `"as_of":"<copy input.as_of exactly>"`, `"as_of":`+string(mustJSON(in.AsOf)), 1)
			replaceValue("market_condition", "3", in.MarketCondition)
		}
	}
	return contract
}

func mustJSON(value any) json.RawMessage {
	raw, _ := json.Marshal(value)
	return raw
}

func outputContractError(kind, input string, err error) error {
	return fmt.Errorf("output does not match %s: %v. Expected exact result schema: %s", outputVersion(kind), err, outputContractForInput(kind, input))
}

func outputContractResult(kind, input string, value any, err error) (any, error) {
	if err != nil {
		return nil, outputContractError(kind, input, err)
	}
	return value, nil
}
func strictDecodeString(raw string, target any) error {
	return strictDecode(json.RawMessage(raw), target)
}
func strictDecode(raw json.RawMessage, target any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON")
		}
		return err
	}
	return nil
}
func validAsOf(value string) bool {
	_, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	return err == nil
}
func validConfidence(v float64) bool { return v >= 0 && v <= 1 }
func nonEmptyList(v []string) bool {
	for _, x := range v {
		if strings.TrimSpace(x) != "" {
			return true
		}
	}
	return false
}

func sameOptionalInt(left, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validateOpportunitySet(out OpportunitySetV1, input string) (any, error) {
	if out.Version != "opportunity_set_v1" || !validAsOf(out.AsOf) {
		return nil, fmt.Errorf("invalid opportunity_set_v1 version/as_of")
	}
	var in MarketScanInput
	_ = strictDecodeString(input, &in)
	if in.AsOf != "" && out.AsOf != in.AsOf {
		return nil, fmt.Errorf("as_of mismatch")
	}
	if !sameOptionalInt(out.MarketCondition, in.MarketCondition) {
		return nil, fmt.Errorf("market_condition mismatch")
	}
	allowed := map[string]bool{}
	for _, c := range in.Candidates {
		allowed[strings.ToUpper(c.Symbol)] = true
	}
	seen := map[string]bool{}
	for i := range out.Opportunities {
		o := &out.Opportunities[i]
		o.Symbol = strings.ToUpper(strings.TrimSpace(o.Symbol))
		if !allowed[o.Symbol] || seen[o.Symbol] {
			return nil, fmt.Errorf("opportunity symbol %q is not a unique input candidate", o.Symbol)
		}
		seen[o.Symbol] = true
		if !validConfidence(o.Confidence) || strings.TrimSpace(o.Thesis) == "" || !nonEmptyList(o.Risks) {
			return nil, fmt.Errorf("invalid opportunity %s", o.Symbol)
		}
		if o.Direction != "long" && o.Direction != "short" && o.Direction != "watch" && o.Direction != "avoid" {
			return nil, fmt.Errorf("invalid direction")
		}
	}
	if out.Opportunities == nil || out.DataMissing == nil {
		return nil, fmt.Errorf("opportunities and data_missing must be arrays")
	}
	missing := make(map[string]bool, len(out.DataMissing))
	for _, item := range out.DataMissing {
		missing[strings.TrimSpace(item)] = true
	}
	for _, required := range in.DataMissing {
		if !missing[strings.TrimSpace(required)] {
			return nil, fmt.Errorf("data_missing must preserve input item %q", required)
		}
	}
	return out, nil
}
func validateStrategyReview(out StrategyReviewV1, input string) (any, error) {
	if out.Version != "strategy_review_v1" || !validConfidence(out.Confidence) || strings.TrimSpace(out.Summary) == "" {
		return nil, fmt.Errorf("invalid strategy_review_v1")
	}
	var in StrategyReviewInput
	_ = strictDecodeString(input, &in)
	if !sameOptionalInt(out.MarketCondition, in.MarketCondition) {
		return nil, fmt.Errorf("market_condition mismatch")
	}
	if in.Template.ID > 0 && out.TemplateID != in.Template.ID {
		return nil, fmt.Errorf("template_id mismatch")
	}
	if out.Verdict != "keep" && out.Verdict != "revise" && out.Verdict != "retire" && out.Verdict != "insufficient_data" {
		return nil, fmt.Errorf("invalid verdict")
	}
	if out.FailureModes == nil || out.Proposals == nil || out.Evidence == nil || out.SuitableEnvironments == nil {
		return nil, fmt.Errorf("review arrays are required")
	}
	return out, nil
}
func validateExperimentProposal(out StrategyExperimentProposalV1, input string) (any, error) {
	if out.Version != "strategy_experiment_proposal_v1" || strings.TrimSpace(out.CandidateName) == "" || !nonEmptyList(out.Rationale) || !nonEmptyList(out.Risks) {
		return nil, fmt.Errorf("invalid experiment proposal")
	}
	var in StrategyExperimentProposalInput
	_ = strictDecodeString(input, &in)
	if in.Template.ID > 0 && out.BaseTemplateID != in.Template.ID {
		return nil, fmt.Errorf("base_template_id mismatch")
	}
	var a, b any
	if json.Unmarshal([]byte(out.TechnologyJSON), &a) != nil || json.Unmarshal([]byte(out.StrategyJSON), &b) != nil {
		return nil, fmt.Errorf("technology_json and strategy_json must contain JSON")
	}
	return out, nil
}
func validateExperimentResult(out StrategyExperimentResultV1, input string) (any, error) {
	if out.Version != "strategy_experiment_result_v1" || strings.TrimSpace(out.Summary) == "" || out.Test.Version != "strategy_experiment_test_v1" {
		return nil, fmt.Errorf("invalid experiment result")
	}
	if out.Verdict != "promising" && out.Verdict != "reject" && out.Verdict != "needs_more_data" {
		return nil, fmt.Errorf("invalid experiment verdict")
	}
	var in StrategyExperimentSummaryInput
	_ = strictDecodeString(input, &in)
	if out.BaseTemplateID != in.Proposal.BaseTemplateID || out.CandidateName != in.Proposal.CandidateName {
		return nil, fmt.Errorf("experiment identity mismatch")
	}
	if out.TechnologyJSON != in.Proposal.TechnologyJSON || out.StrategyJSON != in.Proposal.StrategyJSON {
		return nil, fmt.Errorf("experiment candidate payload mismatch")
	}
	if !reflect.DeepEqual(out.Test, in.Test) {
		return nil, fmt.Errorf("deterministic test report mismatch")
	}
	if out.ProposedChanges == nil || out.Risks == nil {
		return nil, fmt.Errorf("experiment arrays required")
	}
	return out, nil
}
func validateIncidentSet(out IncidentSetV1, input string) (any, error) {
	if out.Version != "incident_set_v1" || !validAsOf(out.AsOf) || out.Incidents == nil {
		return nil, fmt.Errorf("invalid incident_set_v1")
	}
	var in AlertTriageInput
	_ = strictDecodeString(input, &in)
	allowed := map[string]bool{}
	for _, c := range in.Candidates {
		for _, signal := range c.Signals {
			allowed[signal.SignalID] = true
		}
	}
	seen := map[string]bool{}
	for _, incident := range out.Incidents {
		if strings.TrimSpace(incident.IncidentID) == "" || len(incident.IncidentID) > 80 || !nonEmptyList(incident.SignalIDs) || strings.TrimSpace(incident.Summary) == "" {
			return nil, fmt.Errorf("invalid incident")
		}
		if incident.Action != "notify" && incident.Action != "suppress" && incident.Action != "monitor" {
			return nil, fmt.Errorf("invalid incident action")
		}
		if incident.Severity != "low" && incident.Severity != "medium" && incident.Severity != "high" && incident.Severity != "critical" {
			return nil, fmt.Errorf("invalid incident severity")
		}
		for _, id := range incident.SignalIDs {
			if !allowed[id] {
				return nil, fmt.Errorf("unknown signal_id %q", id)
			}
			if seen[id] {
				return nil, fmt.Errorf("signal_id %q appears in multiple incidents", id)
			}
			seen[id] = true
		}
	}
	for id := range allowed {
		if !seen[id] {
			return nil, fmt.Errorf("signal_id %q was not triaged", id)
		}
	}
	return out, nil
}
func validateDailyBrief(out DailyMarketBriefV1, input string) (any, error) {
	var in DailyMarketBriefInput
	if err := strictDecodeString(input, &in); err != nil {
		return nil, err
	}
	if out.Version != "daily_market_brief_v1" || !validAsOf(out.AsOf) || strings.TrimSpace(out.Headline) == "" || strings.TrimSpace(out.RegimeSummary) == "" {
		return nil, fmt.Errorf("invalid daily_market_brief_v1 version/as_of/headline/regime_summary")
	}
	if out.AsOf != in.AsOf {
		return nil, fmt.Errorf("as_of mismatch")
	}
	if !sameOptionalInt(out.MarketCondition, in.MarketCondition) {
		return nil, fmt.Errorf("market_condition mismatch")
	}
	if out.Opportunities == nil || out.Incidents == nil || out.Watchlist == nil || out.Risks == nil || out.DataMissing == nil {
		return nil, fmt.Errorf("daily brief arrays are required")
	}
	if in.Signals.Total == 0 && len(out.Incidents) > 0 {
		return nil, fmt.Errorf("incidents must be empty when input signal total is zero")
	}
	allowed := make(map[string]bool, len(in.Candidates)+len(in.Signals.Symbols))
	for _, candidate := range in.Candidates {
		allowed[strings.ToUpper(strings.TrimSpace(candidate.Symbol))] = true
	}
	for _, symbol := range in.Signals.Symbols {
		allowed[strings.ToUpper(strings.TrimSpace(symbol))] = true
	}
	seenOpportunities := map[string]bool{}
	for i := range out.Opportunities {
		symbol := strings.ToUpper(strings.TrimSpace(out.Opportunities[i].Symbol))
		out.Opportunities[i].Symbol = symbol
		if symbol == "" || !allowed[symbol] || seenOpportunities[symbol] || strings.TrimSpace(out.Opportunities[i].Why) == "" {
			return nil, fmt.Errorf("invalid or duplicate opportunity symbol %q", symbol)
		}
		seenOpportunities[symbol] = true
	}
	for i := range out.Watchlist {
		symbol := strings.ToUpper(strings.TrimSpace(out.Watchlist[i]))
		out.Watchlist[i] = symbol
		if symbol == "" || !allowed[symbol] {
			return nil, fmt.Errorf("watchlist symbol %q is not present in supplied input", symbol)
		}
	}
	missing := make(map[string]bool, len(out.DataMissing))
	for _, item := range out.DataMissing {
		missing[strings.TrimSpace(item)] = true
	}
	for _, required := range in.DataMissing {
		if !missing[strings.TrimSpace(required)] {
			return nil, fmt.Errorf("data_missing must preserve input item %q", required)
		}
	}
	return out, nil
}
