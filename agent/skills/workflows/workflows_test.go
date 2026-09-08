package workflows

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputContractsAreJSONObjectExamples(t *testing.T) {
	for _, kind := range []string{
		MarketScanName,
		StrategyReviewName,
		StrategyExperimentProposeName,
		StrategyExperimentSummaryName,
		AlertTriageName,
		DailyMarketBriefName,
	} {
		t.Run(kind, func(t *testing.T) {
			var value map[string]any
			if err := json.Unmarshal([]byte(outputContract(kind)), &value); err != nil {
				t.Fatalf("output contract is not valid JSON: %v\n%s", err, outputContract(kind))
			}
			if value["version"] != outputVersion(kind) {
				t.Fatalf("contract version = %v, want %s", value["version"], outputVersion(kind))
			}
		})
	}
}

func TestDailyOutputContractUsesDeterministicInputValues(t *testing.T) {
	input := `{"version":"daily_market_brief_input_v1","as_of":"2026-09-07T15:25:20Z","market_condition":10,"candidates":[],"signals":{"total":0,"by_type":{},"by_severity":{},"symbols":[]},"data_missing":[]}`
	contract := outputContractForInput(DailyMarketBriefName, input)
	if !strings.Contains(contract, `"as_of":"2026-09-07T15:25:20Z"`) {
		t.Fatalf("contract does not copy input as_of: %s", contract)
	}
	if !strings.Contains(contract, `"market_condition":10`) {
		t.Fatalf("contract does not copy input market_condition: %s", contract)
	}
}
