package replay

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"go_binance_futures/agent/skill"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	marketregime "go_binance_futures/agent/skills/marketregime"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
)

func TestV1SkillReplayFixtures(t *testing.T) {
	cases := []struct {
		file          string
		definition    skill.Skill
		inputContract string
		contract      string
	}{
		{"market_regime_success.json", marketregime.New(), "market_regime_snapshot_v1", "market_regime_analysis_v1"},
		{"strategy_builder_success.json", strategybuilder.New(strategybuilder.Options{Validate: func([]byte) error { return nil }}), "strategy_builder_input_v1", "strategy_template_v1"},
		{"symbol_analysis_success.json", symbolanalysis.New(), "symbol_analysis_input_v1", "trading_plan_v1"},
		{"alert_analysis_success.json", alertanalysis.New(), "alert_analysis_input_v1", "alert_v1"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			fixture, err := Load("testdata/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			out := Run(context.Background(), fixture, tc.definition)
			if out.Err != nil {
				t.Fatalf("replay failed: %v task=%+v", out.Err, out.Task)
			}
			if out.Task == nil || out.Task.Status != task.StatusSucceeded || out.Result == nil {
				t.Fatalf("unexpected replay result: %+v", out)
			}
			if out.Task.RuntimeVersion == "" || out.Task.SkillVersion != "1.0.0" || out.Task.PromptVersion != "1.0.0" {
				t.Fatalf("missing frozen version metadata: %+v", out.Task.VersionMetadata())
			}
			if len(out.Task.PromptHash) != 64 || out.Task.ModelConfigID != fixture.ModelConfigID {
				t.Fatalf("unexpected prompt/model identity: %+v", out.Task.VersionMetadata())
			}
			if out.Task.InputContractVersion != tc.inputContract || out.Task.OutputContractVersion != tc.contract || out.Task.SkillSource != skill.DefaultSource {
				t.Fatalf("unexpected contract/source: %+v", out.Task.VersionMetadata())
			}
		})
	}
}

func TestReplayIsRepeatableWithSameFixture(t *testing.T) {
	fixture, err := Load("testdata/symbol_analysis_success.json")
	if err != nil {
		t.Fatal(err)
	}
	first := Run(context.Background(), fixture, symbolanalysis.New())
	second := Run(context.Background(), fixture, symbolanalysis.New())
	if first.Err != nil || second.Err != nil {
		t.Fatalf("replay errors: first=%v second=%v", first.Err, second.Err)
	}
	if string(first.Result.Raw) != string(second.Result.Raw) {
		t.Fatalf("same fixture produced different final results:\n%s\n%s", first.Result.Raw, second.Result.Raw)
	}
	if first.Task.PromptHash != second.Task.PromptHash || first.Task.OutputContractVersion != second.Task.OutputContractVersion {
		t.Fatal("same fixture produced different version identity")
	}
}

func TestRuntimeReplayBaselines(t *testing.T) {
	load := func(t *testing.T, name string) Fixture {
		t.Helper()
		fixture, err := Load("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return fixture
	}

	t.Run("malformed decision terminal error", func(t *testing.T) {
		fixture := load(t, "runtime_llm_json_error.json")
		out := Run(context.Background(), fixture, skill.Definition{SkillName: "test", Prompt: "baseline", Rounds: 1})
		if out.Err == nil || out.Task == nil || out.Task.Stage != "max_rounds" || !hasStage(out.Task, "repairing_decision") {
			t.Fatalf("JSON error baseline changed: err=%v task=%+v", out.Err, out.Task)
		}
	})

	t.Run("malformed decision repairs", func(t *testing.T) {
		fixture := load(t, "runtime_repair.json")
		out := Run(context.Background(), fixture, skill.Definition{SkillName: "test", Prompt: "baseline", Rounds: 3})
		if out.Err != nil || out.Task.Status != task.StatusSucceeded || !hasStage(out.Task, "repairing_decision") {
			t.Fatalf("repair baseline changed: err=%v task=%+v", out.Err, out.Task)
		}
	})

	t.Run("tool error returns to agent", func(t *testing.T) {
		fixture := load(t, "runtime_tool_error.json")
		definition := skill.Definition{SkillName: "test", Prompt: "baseline", AllowedTools: []string{"echo"}, Rounds: 3}
		out := Run(context.Background(), fixture, definition)
		if out.Err != nil || out.Task.Status != task.StatusSucceeded || !hasToolStatus(out.Task, "echo", "error") {
			t.Fatalf("tool error baseline changed: err=%v task=%+v", out.Err, out.Task)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		fixture := load(t, "runtime_timeout.json")
		out := Run(context.Background(), fixture, skill.Definition{SkillName: "test", Prompt: "baseline", Rounds: 1})
		if out.Err == nil || out.Task == nil || out.Task.Status != task.StatusFailed || out.Task.Stage != "timeout" {
			t.Fatalf("timeout baseline changed: err=%v task=%+v", out.Err, out.Task)
		}
	})

	t.Run("context too large", func(t *testing.T) {
		fixture := load(t, "runtime_context_too_large.json")
		out := Run(context.Background(), fixture, skill.Definition{SkillName: "test", Prompt: "baseline", Rounds: 1})
		if out.Err == nil || out.Task == nil || out.Task.Stage != "context_too_large" {
			t.Fatalf("context baseline changed: err=%v task=%+v", out.Err, out.Task)
		}
	})
}

func TestVersionDifferenceReportsPromptChange(t *testing.T) {
	from := task.VersionMetadata{RuntimeVersion: "1.0.0", SkillVersion: "1.0.0", PromptVersion: "1.0.0", PromptHash: "aaa"}
	to := from
	to.PromptVersion = "1.0.1"
	to.PromptHash = "bbb"
	diff := CompareVersions(from, to)
	raw, _ := json.Marshal(diff)
	if len(diff) != 2 || !strings.Contains(string(raw), "prompt_version") || !strings.Contains(string(raw), "prompt_hash") {
		t.Fatalf("unexpected version diff: %s", raw)
	}
}

func hasStage(item *task.Task, stage string) bool {
	for _, event := range item.Events {
		if event.Stage == stage {
			return true
		}
	}
	return false
}

func hasToolStatus(item *task.Task, toolName, status string) bool {
	for _, event := range item.Events {
		if event.Tool == toolName && event.Status == status {
			return true
		}
	}
	return false
}
