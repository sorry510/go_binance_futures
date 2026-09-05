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

func (d *Definition) Name() string    { return d.kind }
func (d *Definition) Tools() []string { return nil }
func (d *Definition) MaxRounds() int  { return 5 }
func (d *Definition) ModelRequirements() llm.ModelRequirements {
	req := llm.ModelRequirements{StructuredOutput: true, MinJSONReliability: 70}
	if d.kind == StrategyReviewName || strings.HasPrefix(d.kind, "strategy_experiment") || d.kind == AlertTriageName {
		req.Reasoning = true
		req.MinJSONReliability = 75
	}
	return req
}
func (d *Definition) VersionInfo() skill.VersionInfo {
	return skill.VersionInfo{SkillVersion: "1.0.0", PromptVersion: "1.0.0", InputContractVersion: inputVersion(d.kind), OutputContractVersion: outputVersion(d.kind), Source: skill.DefaultSource, SourceVersion: "v2-11"}
}
func (d *Definition) SystemPrompt() string {
	switch d.kind {
	case MarketScanName:
		return "You rank only the deterministic market candidates provided by the system. Never claim to scan the full market yourself. Return strict opportunity_set_v1 JSON. Do not invent missing data or trading execution."
	case StrategyReviewName:
		return "Review the supplied strategy snapshot, deterministic fee-adjusted test statistics, and market condition. Return strict strategy_review_v1 JSON. Proposals are advisory only; never modify a template."
	case StrategyExperimentProposeName:
		return "Propose one candidate strategy revision from the supplied template and goal. Return strict strategy_experiment_proposal_v1 JSON. The candidate will be validated and tested deterministically; do not claim it has passed tests."
	case StrategyExperimentSummaryName:
		return "Summarize the supplied strategy experiment proposal and deterministic test report. Return strict strategy_experiment_result_v1 JSON. Never overwrite or activate a production strategy."
	case AlertTriageName:
		return "Triage only the pre-grouped signal candidates provided by the deterministic incident builder. Decide which signals represent the same market event and whether to notify, suppress, or monitor. Return strict incident_set_v1 JSON."
	case DailyMarketBriefName:
		return "Create a concise fixed-schema daily market brief from the supplied market condition, deterministic scanner candidates, and recent signal summary. Return strict daily_market_brief_v1 JSON. Do not invent live facts."
	default:
		return "Return strict JSON."
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
				return nil, err
			}
			return validateOpportunitySet(out, input)
		case StrategyReviewName:
			var out StrategyReviewV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, err
			}
			return validateStrategyReview(out, input)
		case StrategyExperimentProposeName:
			var out StrategyExperimentProposalV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, err
			}
			return validateExperimentProposal(out, input)
		case StrategyExperimentSummaryName:
			var out StrategyExperimentResultV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, err
			}
			return validateExperimentResult(out, input)
		case AlertTriageName:
			var out IncidentSetV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, err
			}
			return validateIncidentSet(out, input)
		case DailyMarketBriefName:
			var out DailyMarketBriefV1
			if err := strictDecode(raw, &out); err != nil {
				return nil, err
			}
			return validateDailyBrief(out, input)
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
	if !sameOptionalInt(out.MarketCondition, in.MarketCondition) {
		return nil, fmt.Errorf("market_condition mismatch")
	}
	if out.Version != "daily_market_brief_v1" || !validAsOf(out.AsOf) || strings.TrimSpace(out.Headline) == "" || strings.TrimSpace(out.RegimeSummary) == "" {
		return nil, fmt.Errorf("invalid daily_market_brief_v1")
	}
	if out.Opportunities == nil || out.Incidents == nil || out.Watchlist == nil || out.Risks == nil || out.DataMissing == nil {
		return nil, fmt.Errorf("daily brief arrays required")
	}
	return out, nil
}
