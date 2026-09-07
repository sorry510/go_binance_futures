package agenttrade

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestV3BaselineExecutionStatusContract(t *testing.T) {
	got := []string{
		StatusRiskRejected,
		StatusAwaitingApproval,
		StatusApproved,
		StatusRejected,
		StatusExecuting,
		StatusExecuted,
		StatusExecutionFailed,
		StatusExecutionUncertain,
		StatusExpired,
	}
	want := []string{
		"risk_rejected",
		"awaiting_approval",
		"approved",
		"rejected",
		"executing",
		"executed",
		"execution_failed",
		"execution_uncertain",
		"expired",
	}
	if len(got) != len(want) {
		t.Fatalf("status contract length changed: got=%d want=%d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("execution status contract changed at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
	if RiskPass != "pass" || RiskFail != "fail" {
		t.Fatalf("risk status contract changed: pass=%q fail=%q", RiskPass, RiskFail)
	}
}

func BenchmarkV3BaselineProposalToExecution(b *testing.B) {
	prepareTradeTestDB(b)
	now := time.UnixMilli(1_800_000_000_000).UTC()
	data := passingRiskData(now)
	broker := &fakeBroker{submit: BrokerOrderResult{ExchangeOrderID: "bench", AveragePrice: 100.1}}
	service := Service{
		Store: Store{}, Risk: RiskEngine{Store: Store{}, Data: data, Now: func() time.Time { return now }},
		Broker: broker, Now: func() time.Time { return now },
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proposal := baseProposal(now)
		proposal.ProposalID = fmt.Sprintf("bench_%d", i)
		proposal.SourceTaskID = fmt.Sprintf("bench_task_%d", i)
		if err := service.Store.SaveProposal(ctx, &proposal); err != nil {
			b.Fatal(err)
		}
		if _, err := service.Approve(ctx, proposal.ProposalID, "benchmark"); err != nil {
			b.Fatal(err)
		}
		if _, _, err := service.Execute(ctx, proposal.ProposalID, "benchmark"); err != nil {
			b.Fatal(err)
		}
	}
}
