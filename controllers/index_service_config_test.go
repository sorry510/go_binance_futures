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
