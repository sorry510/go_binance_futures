package controllers

import (
	"math"
	"strings"
	"testing"

	"go_binance_futures/models"
)

func TestApplyServiceConfigPatchOpportunityConfidence(t *testing.T) {
	base := models.Config{
		ID:                            1,
		AgentOpportunityMinConfidence: 0.70,
		AgentOpportunityWatchEnable:   1,
		FutureMaxCount:                12,
	}
	for _, tt := range []struct {
		name    string
		body    string
		want    float64
		wantErr string
	}{
		{name: "80 percent", body: `{"agent_opportunity_min_confidence":0.8}`, want: 0.8},
		{name: "73.5 percent", body: `{"agent_opportunity_min_confidence":0.735}`, want: 0.735},
		{name: "minimum", body: `{"agent_opportunity_min_confidence":0.01}`, want: 0.01},
		{name: "maximum", body: `{"agent_opportunity_min_confidence":1}`, want: 1},
		{name: "empty", body: `{}`, wantErr: "cannot be empty"},
		{name: "zero defaults unexpectedly", body: `{"agent_opportunity_min_confidence":0}`, wantErr: "between 0.01 and 1"},
		{name: "negative", body: `{"agent_opportunity_min_confidence":-0.1}`, wantErr: "between 0.01 and 1"},
		{name: "over 100 percent", body: `{"agent_opportunity_min_confidence":1.01}`, wantErr: "between 0.01 and 1"},
		{name: "wrong type", body: `{"agent_opportunity_min_confidence":"80"}`, wantErr: "invalid opportunity confidence"},
		{name: "null", body: `{"agent_opportunity_min_confidence":null}`, wantErr: "between 0.01 and 1"},
		{name: "malformed", body: `{"agent_opportunity_min_confidence":`, wantErr: "invalid config JSON"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyServiceConfigPatch(base, []byte(tt.body))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v; want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got.AgentOpportunityMinConfidence-tt.want) > 1e-9 {
				t.Fatalf("confidence = %v; want %v", got.AgentOpportunityMinConfidence, tt.want)
			}
			if got.ID != base.ID || got.AgentOpportunityWatchEnable != base.AgentOpportunityWatchEnable || got.FutureMaxCount != base.FutureMaxCount {
				t.Fatalf("patch corrupted unrelated config fields: %#v", got)
			}
		})
	}
}

func TestApplyServiceConfigPatchUnrelatedField(t *testing.T) {
	base := models.Config{ID: 1, AgentOpportunityMinConfidence: 0.7, FutureMaxCount: 10}
	got, err := applyServiceConfigPatch(base, []byte(`{"future_max_count":15}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.FutureMaxCount != 15 || got.AgentOpportunityMinConfidence != 0.7 || got.ID != 1 {
		t.Fatalf("unexpected patch result: %#v", got)
	}
}

func TestApplyServiceConfigPatchSpotQuant(t *testing.T) {
	base := models.Config{ID: 1, SpotQuantEnable: 0, SpotEnable: 1, SpotNewEnable: 1, FutureEnable: 1}
	for _, tc := range []struct {
		body    string
		want    int
		wantErr bool
	}{
		{body: `{"spot_quant_enable":1}`, want: 1},
		{body: `{"spot_quant_enable":0}`, want: 0},
		{body: `{"spot_quant_enable":2}`, wantErr: true},
		{body: `{"spot_quant_enable":-1}`, wantErr: true},
		{body: `{"spot_quant_enable":"1"}`, wantErr: true},
		{body: `{"spot_quant_enable":true}`, wantErr: true},
		{body: `{"spot_quant_enable":null}`, wantErr: true},
		{body: `{"spotQuantEnable":1}`, wantErr: true},
		{body: `{"spot_quant_enable":1,"future_enable":1}`, wantErr: true},
		{body: `{"spot_quant_enable":0,"spot_enable":0}`, wantErr: true},
	} {
		got, err := applyServiceConfigPatch(base, []byte(tc.body))
		if tc.wantErr {
			if err == nil || (!strings.Contains(err.Error(), "spot_quant_enable") && !strings.Contains(err.Error(), "spotQuantEnable")) {
				t.Errorf("%s: expected spot quant validation error, got %v", tc.body, err)
			}
			continue
		}
		if err != nil || int(got.SpotQuantEnable) != tc.want {
			t.Errorf("%s: got enabled=%d, err=%v", tc.body, got.SpotQuantEnable, err)
		}
		if got.ID != base.ID || got.SpotEnable != 1 || got.SpotNewEnable != 1 || got.FutureEnable != 1 {
			t.Errorf("%s: patch changed unrelated market settings: %+v", tc.body, got)
		}
	}
	got, err := applyServiceConfigPatch(base, []byte(`{"future_enable":0}`))
	if err != nil || got.SpotQuantEnable != 0 {
		t.Errorf("unrelated patch unexpectedly enabled spot quant: %+v, %v", got, err)
	}
}
