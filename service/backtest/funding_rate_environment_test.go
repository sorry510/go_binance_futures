package backtest

import "testing"

func TestFundingRateDataUsesOnlySettledRowsVisibleAtAsOf(t *testing.T) {
	builder := historicalEnvironment{dataset: Dataset{Funding: []Funding{
		{FundingTime: 100, FundingRate: 0.001},
		{FundingTime: 200, FundingRate: -0.002},
		{FundingTime: 300, FundingRate: 0.003},
	}}}
	got := builder.fundingRateData(250, 2)
	if len(got.Data) != 2 || len(got.Time) != 2 {
		t.Fatalf("unexpected funding data length: %+v", got)
	}
	if got.Time[0] != 200 || got.Data[0] != -0.002 || got.Time[1] != 100 || got.Data[1] != 0.001 {
		t.Fatalf("funding data must be newest-first and exclude future rows: %+v", got)
	}
}
