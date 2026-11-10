package spotstrategy

import "testing"

// Spot template/client UI and the authoritative S1 server validator must
// agree that SHORT/CLOSE_SHORT is invalid even when enable=false or the
// symbol-level Spot strategy switch is off.
func TestSpotRulesNeverPersistFuturesDirections(t *testing.T) {
	for _, action := range []string{"short", "close_short"} {
		for _, ruleEnabled := range []bool{false, true} {
			for _, configEnabled := range []bool{false, true} {
				rule := "false"
				if ruleEnabled {
					rule = "true"
				}
				raw := `[{"name":"not-spot","type":"` + action + `","enable":`
				if ruleEnabled {
					raw += "true"
				} else {
					raw += "false"
				}
				raw += `,"code":"` + rule + `"}]`
				if err := ValidateRulesWithTechnology(raw, "{}", configEnabled); err == nil {
					t.Fatalf("invalid Spot direction %s accepted (rule=%t config=%t)",
						action, ruleEnabled, configEnabled)
				}
			}
		}
	}
	// Disabled valid directions remain allowed as drafts while the symbol
	// configuration is disabled; they cannot be interpreted as live trades.
	if err := ValidateRulesWithTechnology(`[{"name":"draft","type":"long","enable":false,"code":""}]`, "{}", false); err != nil {
		t.Fatalf("disabled long draft rejected: %v", err)
	}
}

// A configuration API update with a disabled SHORT rule must be rejected
// before any spot_symbols row is modified. The symbol-level enabled flag
// does not make an invalid Futures direction safe to persist.
func TestSpotSettingsRejectDisabledShortWithoutDatabaseMutation(t *testing.T) {
	setupSpotS1(t)
	disabled := false
	for _, kind := range []string{"short", "close_short"} {
		invalid := `[{"name":"forbidden","type":"` + kind + `","enable":false,"code":"false"}]`
		_, err := UpdateSetting(Patch{Symbol: "BTCUSDT", Enabled: &disabled, BudgetQuote: "0", Strategy: &invalid})
		if err == nil {
			t.Fatalf("stored forbidden Spot direction %s", kind)
		}
		current, loadErr := LoadSetting("BTCUSDT")
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		if current.Strategy != "[]" || current.Enabled {
			t.Fatalf("rejected Spot template modified DB: %+v", current)
		}
	}
}
