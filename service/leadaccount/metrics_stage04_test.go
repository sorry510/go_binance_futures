package leadaccount

import (
	"net/http"
	"testing"
	"time"

	"go_binance_futures/service/binanceapiusage"
)

func TestLead04UsageMetricsKeepSAPISeparateFromFutures(t *testing.T) {
	collector := binanceapiusage.Default()
	now := time.Now()
	// Distinct sources ensure another account's traffic cannot count as Lead.
	// Use a single run-specific timestamp and assert only attributable deltas.
	before := LeadUsageSnapshot(collector, nil, now)
	collector.Record(binanceapiusage.RequestEvent{
		At: now.UnixMilli(), Product: "futures", Environment: "mainnet",
		Source: "lead_trading", RequestType: "read", Method: http.MethodGet,
		Path: "/fapi/v2/account", StatusCode: 200, EstimatedWeight: 5,
	}, nil, "")
	collector.Record(binanceapiusage.RequestEvent{
		At: now.UnixMilli(), Product: "sapi", Environment: "mainnet",
		Source: "lead_trading_sapi", RequestType: "read", Method: http.MethodGet,
		Path: "/sapi/v1/copyTrading/futures/leadSymbol", StatusCode: 200, EstimatedWeight: 1,
	}, nil, "")
	collector.Record(binanceapiusage.RequestEvent{
		At: now.UnixMilli(), Product: "futures", Environment: "mainnet",
		Source: "start_trade", RequestType: "read", Method: http.MethodGet,
		Path: "/fapi/v2/account", StatusCode: 200, EstimatedWeight: 5,
	}, nil, "")
	after := LeadUsageSnapshot(collector, nil, now.Add(time.Millisecond))
	if after.FuturesRequests-before.FuturesRequests != 1 ||
		after.FuturesWeight-before.FuturesWeight != 5 ||
		after.SAPIRequests-before.SAPIRequests != 1 ||
		after.SAPIWeight-before.SAPIWeight != 1 ||
		after.Requests-before.Requests != 2 {
		t.Fatalf("Lead SAPI/Futures attribution incorrect before=%+v after=%+v", before, after)
	}
}
