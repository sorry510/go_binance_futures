package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStage41HedgeModeReadOnlySignedToLeadOnly(t *testing.T) {
	seen := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		if r.Method != "GET" || r.URL.Path != "/fapi/v1/positionSide/dual" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-MBX-APIKEY") != "lead-key" || r.URL.Query().Get("signature") == "" {
			t.Errorf("hedge mode read not signed by Lead key")
		}
		_, _ = w.Write([]byte(`{"dualSidePosition":true}`))
	}))
	defer server.Close()
	lead := stage1TestAccount(t, LeadAccountID, server)
	result, err := lead.GetPositionModeContext(context.Background())
	if err != nil || result == nil || !result.DualSidePosition {
		t.Fatalf("hedge read failed: result=%v error=%v", result, err)
	}
	if seen != 1 {
		t.Fatalf("unexpected REST request count %d", seen)
	}
}
