package binance

import "testing"

func TestLeadSwitchCannotUseConfiguredMainFuturesKey(t *testing.T) {
	beforeMain, beforeTestnet := api_key, testnet_api_key
	api_key, testnet_api_key = "main-production-key", "main-testnet-key"
	t.Cleanup(func() { api_key, testnet_api_key = beforeMain, beforeTestnet })
	for _, k := range []string{"main-production-key", "main-testnet-key"} {
		if !LeadKeyIsMainKey(k) {
			t.Fatalf("Main key %q was not recognized", k)
		}
		if _, err := NewLeadAccountClient(k, "secret", nil); err == nil {
			t.Fatal("Lead Client accepted Main private account credentials")
		}
	}
	if LeadKeyIsMainKey("distinct-lead-key") {
		t.Fatal("distinct Lead key incorrectly blocked")
	}
}
