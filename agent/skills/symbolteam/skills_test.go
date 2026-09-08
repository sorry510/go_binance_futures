package symbolteam

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go_binance_futures/agent/skill"
)

func TestAnalystSkillsUseSharedContextWithoutTools(t *testing.T) {
	input := `{"symbol":"BTCUSDT","shared_context":{"symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z"},"shared_context_hash":"abc"}`
	for _, definition := range []*Definition{Technical(), Flow(), News()} {
		if tools := definition.Tools(); len(tools) != 0 {
			t.Fatalf("%s unexpectedly has tools: %v", definition.Name(), tools)
		}
		if err := definition.ValidateInput(skill.Request{Input: input}); err != nil {
			t.Fatalf("%s rejected shared input: %v", definition.Name(), err)
		}
		messages, err := definition.BuildInput(context.Background(), skill.Request{Input: input})
		if err != nil || len(messages) != 1 {
			t.Fatalf("%s build input: messages=%v err=%v", definition.Name(), messages, err)
		}
	}
}

func TestTypedValidatorsRejectInventedEvidence(t *testing.T) {
	raw := json.RawMessage(`{"version":"technical_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z","trend":"bullish","structure":"higher highs","volatility":"normal","key_levels":[{"kind":"support","price":100}],"confidence":0.8,"data_missing":[],"evidence":[{"source":"made_up_source","finding":"x"}]}`)
	if _, err := Technical().Validator().Validate(context.Background(), raw); err == nil {
		t.Fatal("technical validator accepted invented evidence source")
	}
}

func TestSupervisorHasNoToolsAndRequiresTypedDirection(t *testing.T) {
	if len(Supervisor().Tools()) != 0 {
		t.Fatalf("supervisor must not receive tools: %v", Supervisor().Tools())
	}
	raw := json.RawMessage(`{"version":"symbol_team_supervisor_v1","symbol":"BTCUSDT","as_of":"2026-09-06T10:00:00Z","direction":"buy-now","confidence":0.8,"summary":"x","consensus":[],"disagreements":[],"data_missing":[],"evidence":[]}`)
	if _, err := Supervisor().Validator().Validate(context.Background(), raw); err == nil {
		t.Fatal("supervisor validator accepted invalid direction")
	}
}

func TestTeamRolePromptsRequireRuntimeFinalEnvelope(t *testing.T) {
	for _, definition := range []*Definition{Technical(), Flow(), News(), Supervisor()} {
		prompt := definition.SystemPrompt()
		if !strings.Contains(prompt, `action="final"`) || !strings.Contains(prompt, `"action":"final"`) {
			t.Fatalf("skill %s prompt must require Runtime final envelope: %s", definition.Name(), prompt)
		}
		version := definition.VersionInfo()
		if version.PromptVersion != "1.1.0" {
			t.Fatalf("skill %s prompt version=%s", definition.Name(), version.PromptVersion)
		}
	}
}

func TestNewsValidatorUsesMarketIntelligenceEvidenceAndAllowsNoFreshCatalyst(t *testing.T) {
	valid := json.RawMessage(`{"version":"news_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-08T05:00:00Z","bias":"bullish","impact":"medium","summary":"fresh official listing catalyst","confidence":0.7,"data_missing":[],"evidence":[{"source":"market_intelligence","finding":"fresh announcement event_time precedes observed_at by 2s"}]}`)
	if _, err := News().Validator().Validate(context.Background(), valid); err != nil {
		t.Fatalf("news validator rejected market intelligence evidence: %v", err)
	}
	neutral := json.RawMessage(`{"version":"news_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-08T05:00:00Z","bias":"neutral","impact":"none","summary":"no fresh catalyst","confidence":0.8,"data_missing":[],"evidence":[]}`)
	if _, err := News().Validator().Validate(context.Background(), neutral); err != nil {
		t.Fatalf("news validator rejected no-catalyst result: %v", err)
	}
	invented := json.RawMessage(`{"version":"news_analysis_v1","symbol":"BTCUSDT","as_of":"2026-09-08T05:00:00Z","bias":"bullish","impact":"high","summary":"invented","confidence":0.9,"data_missing":[],"evidence":[{"source":"twitter_guess","finding":"rumor"}]}`)
	if _, err := News().Validator().Validate(context.Background(), invented); err == nil {
		t.Fatal("news validator accepted non-Market-Intelligence evidence")
	}
}
