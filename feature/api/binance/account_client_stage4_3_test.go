package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStage43LeadReadOnlyRuleAndIncomeRequestsAreAccountBound(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		if r.Method != "GET" {
			t.Errorf("unexpected mutation: %s", r.Method)
		}
		// exchangeInfo is public and intentionally has no API Key header.
		if r.URL.Path != "/fapi/v1/exchangeInfo" && r.Header.Get("X-MBX-APIKEY") != "lead-key" {
			t.Errorf("signed account read used wrong key: %s", r.URL.Path)
		}
		if strings.HasPrefix(r.URL.Path, "/fapi/v1/income") || strings.Contains(r.URL.Path, "leverageBracket") {
			if r.URL.Query().Get("signature") == "" {
				t.Error("signed read is not signed")
			}
		}
		switch r.URL.Path {
		case "/fapi/v1/income":
			_, _ = w.Write([]byte(`[{"tranId":1,"incomeType":"REALIZED_PNL","income":"-1","asset":"USDT","time":1791619200000}]`))
		case "/fapi/v1/leverageBracket":
			_, _ = w.Write([]byte(`{"symbol":"BTCUSDT","brackets":[{"bracket":1,"initialLeverage":20,"notionalCap":100000}]} `))
		case "/fapi/v1/exchangeInfo":
			_, _ = w.Write([]byte(`{"symbols":[],"serverTime":1791619200000}`))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	lead := stage1TestAccount(t, LeadAccountID, server)
	if _, err := lead.GetIncomeHistoryContext(context.Background(), 1791619200000, 1791705600000, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := lead.GetLeadLeverageBracketContext(context.Background(), "BTCUSDT"); err != nil {
		t.Fatal(err)
	}
	if _, err := lead.GetLeadExchangeInfoContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 GETs, got %v", seen)
	}
	main := stage1TestAccount(t, MainAccountID, server)
	if _, err := main.GetIncomeHistoryContext(context.Background(), 1791619200000, 1791705600000, 1000); err == nil {
		t.Fatal("main income accepted as Lead")
	}
	if _, err := main.GetLeadLeverageBracketContext(context.Background(), "BTCUSDT"); err == nil {
		t.Fatal("main tier accepted")
	}
	if _, err := main.GetLeadExchangeInfoContext(context.Background()); err == nil {
		t.Fatal("main symbol rules accepted")
	}
	if len(seen) != 3 {
		t.Fatalf("main account unexpectedly touched lead private endpoint: %v", seen)
	}
}
