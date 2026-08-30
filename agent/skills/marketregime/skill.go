package marketregime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
	marketservice "go_binance_futures/service/market"
	markettypes "go_binance_futures/types"
)

const Name = "market_regime"

const systemPrompt = `You are a cryptocurrency futures market-regime classifier.
Analyze only the supplied deterministic market snapshot.
Do not give trading advice, positions, leverage, orders, or tool calls.
Return exactly one Agent Runtime final decision as JSON:
{"action":"final","summary":"short Chinese summary","result":{"market_condition":1,"confidence":0.0,"reason":"Chinese explanation within 200 characters"}}
The result must contain only market_condition, confidence and reason.`

type Analysis struct {
	MarketCondition int     `json:"market_condition"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason"`
}

func New() skill.Skill {
	return skill.Definition{
		SkillName:      Name,
		Prompt:         systemPrompt,
		Rounds:         2,
		BuildInputFunc: buildInput,
		FinalValidator: validator.Func(validateFinal),
	}
}
func buildInput(ctx context.Context, req skill.Request) ([]llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var snapshot marketservice.RegimeSnapshot
	if err := json.Unmarshal([]byte(req.Input), &snapshot); err != nil {
		return nil, fmt.Errorf("decode market regime snapshot: %w", err)
	}
	if snapshot.SymbolCount <= 0 {
		return nil, fmt.Errorf("market regime snapshot is empty")
	}
	compact, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode market regime snapshot: %w", err)
	}
	return []llm.Message{{Role: llm.RoleUser, Content: buildPrompt(string(compact))}}, nil
}

func buildPrompt(snapshotJSON string) string {
	return `Classify the market into exactly one condition:
1 Strong bull: majors and the broad market rise strongly together.
2 Bull: directional bias is positive but not broad or strong enough for 1 or 8.
3 Sideways: no clear direction and no special regime below dominates.
4 Bear: directional bias is negative but not broad or strong enough for 5 or 9.
5 Strong bear: majors and the broad market fall strongly together.
6 Bullish divergence: majors are weak or negative while broad-market breadth is resilient or positive.
7 Bearish divergence: majors are positive while broad-market breadth is weak or negative.
8 Broad rise: advancing breadth is exceptionally high, while major direction is not strong enough for 1.
9 Broad decline: declining breadth is exceptionally high, while major direction is not strong enough for 5.
10 High-volatility sideways: dispersion or intraday range is high without a reliable direction.
11 Low-volatility consolidation: direction, dispersion, and intraday range are all subdued.

Prefer 6-11 only when clearly supported; otherwise use 1-5.
Snapshot:
` + snapshotJSON
}
func validateFinal(ctx context.Context, raw json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	var analysis Analysis
	if err := decoder.Decode(&analysis); err != nil {
		return nil, fmt.Errorf("decode market regime result: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if !markettypes.IsValidMarketCondition(analysis.MarketCondition) {
		return nil, fmt.Errorf("unsupported market condition %d", analysis.MarketCondition)
	}
	if analysis.Confidence < 0 || analysis.Confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}
	analysis.Reason = marketservice.SanitizeReason(analysis.Reason)
	if analysis.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	return analysis, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing market regime result: %w", err)
	}
	return fmt.Errorf("market regime result contains multiple JSON values")
}
