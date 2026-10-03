package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go_binance_futures/service/backtest"
	"go_binance_futures/service/historicalmarket"
)

type researchSourceAudit struct {
	Symbol         string                               `json:"symbol"`
	StartTime      int64                                `json:"start_time"`
	EndTime        int64                                `json:"end_time"`
	ARMAsOfMS      int64                                `json:"arm_as_of_ms"`
	ReadOnly       bool                                 `json:"read_only"`
	Series         map[string][]backtest.Bar            `json:"series"`
	Aggregates     map[string]backtest.Bar              `json:"aggregates"`
	ZeroMinutes    map[string]int                       `json:"zero_trade_minutes"`
	Archives       []historicalmarket.PublicDataArchive `json:"archives"`
	TradeIDGaps    int                                  `json:"trade_id_gaps,omitempty"`
	TradeRows      int                                  `json:"trade_rows,omitempty"`
	UniqueTradeIDs int                                  `json:"unique_trade_ids,omitempty"`
	TradeIDSpan    int64                                `json:"trade_id_span,omitempty"`
}

func main() {
	if err := auditResearchSource(); err != nil {
		fmt.Fprintln(os.Stderr, "source audit error:", err)
		os.Exit(1)
	}
}

func auditResearchSource() error {
	conf := flag.String("conf", "conf/app.conf", "configuration with the commented ARM block")
	symbol := flag.String("symbol", "", "one futures symbol")
	startText := flag.String("start", "", "aligned interval start in RFC3339 UTC")
	interval := flag.String("interval", "1h", "one complete interval: 1h or 4h")
	cacheRoot := flag.String("cache-root", "", "research cache root")
	proxy := flag.String("archive-proxy", "", "optional official archive proxy")
	output := flag.String("output", "", "source audit JSON output")
	includeTrades := flag.Bool("trades", false, "also reconstruct minutes from checksum-verified daily trades")
	flag.Parse()
	startDate, err := time.Parse(time.RFC3339, *startText)
	if err != nil || *symbol == "" || *cacheRoot == "" || *output == "" || (*interval != "1h" && *interval != "4h") {
		return fmt.Errorf("require -symbol, -start, -cache-root, -output and 1h/4h interval")
	}
	duration, _ := researchIntervalMs(*interval)
	start := startDate.UnixMilli()
	if start%duration != 0 {
		return fmt.Errorf("start is not aligned to %s", *interval)
	}
	end := start + duration - 1
	cfg, err := parseArmResearchConfig(*conf)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	db, err := openArmResearchDB(cfg, "go_binance")
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	audit := researchSourceAudit{Symbol: *symbol, StartTime: start, EndTime: end, ReadOnly: true,
		Series: map[string][]backtest.Bar{}, Aggregates: map[string]backtest.Bar{}, ZeroMinutes: map[string]int{}}
	if err := tx.QueryRowContext(ctx, "SELECT CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS UNSIGNED)").Scan(&audit.ARMAsOfMS); err != nil {
		return err
	}
	for _, resolution := range []string{"1m", *interval} {
		bars, err := readArmSeries(ctx, tx, *symbol, resolution, start, end)
		if err != nil {
			return err
		}
		if err := validateResearchBars(*symbol, resolution, bars, start, end); err != nil {
			return err
		}
		audit.Series["arm_"+resolution] = bars
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	archiveDir := filepath.Join(*cacheRoot, "archives")
	client, err := historicalmarket.NewPublicDataClient(historicalmarket.PublicDataClientConfig{CacheDir: archiveDir, ProxyURL: *proxy, Timeout: 45 * time.Second, MaxRetries: 1})
	if err != nil {
		return err
	}
	defer client.Close()
	for _, source := range []struct{ key, period, interval string }{
		{"monthly_1m", historicalmarket.ArchivePeriodMonthly, "1m"},
		{"daily_1m", historicalmarket.ArchivePeriodDaily, "1m"},
		{"monthly_" + *interval, historicalmarket.ArchivePeriodMonthly, *interval},
		{"daily_" + *interval, historicalmarket.ArchivePeriodDaily, *interval},
	} {
		archive, err := fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindKlines,
			Period: source.period, Symbol: *symbol, Interval: source.interval, Date: startDate.UTC()}, archiveDir)
		if err != nil {
			return err
		}
		audit.Archives = append(audit.Archives, archive)
		rows, err := client.ParseKlines(ctx, archive, start, end)
		if err != nil {
			return err
		}
		bars := make([]backtest.Bar, 0, len(rows))
		for _, row := range rows {
			bars = append(bars, backtest.Bar{Symbol: *symbol, Interval: source.interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime,
				Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume,
				TradeCount: row.TradeCount, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume})
		}
		if err := validateResearchBars(*symbol, source.interval, bars, start, end); err != nil {
			return err
		}
		audit.Series[source.key] = bars
	}
	if *includeTrades {
		archive, err := fetchResearchArchive(ctx, client, historicalmarket.PublicDataArchiveSpec{Kind: historicalmarket.ArchiveKindTrades,
			Period: historicalmarket.ArchivePeriodDaily, Symbol: *symbol, Date: startDate.UTC()}, archiveDir)
		if err != nil {
			return err
		}
		audit.Archives = append(audit.Archives, archive)
		trades, err := client.ParseTrades(ctx, archive, start, end)
		if err != nil {
			return err
		}
		audit.TradeRows = len(trades)
		seenIDs := make(map[int64]bool, len(trades))
		minimumID, maximumID := int64(0), int64(0)
		bars := make([]backtest.Bar, 0, int(duration/60_000))
		for i, trade := range trades {
			seenIDs[trade.TradeID] = true
			if i == 0 {
				minimumID, maximumID = trade.TradeID, trade.TradeID
			} else {
				minimumID, maximumID = min(minimumID, trade.TradeID), max(maximumID, trade.TradeID)
			}
			if i > 0 && trade.TradeID != trades[i-1].TradeID+1 {
				audit.TradeIDGaps++
			}
			bucket := trade.TradeTime / 60_000 * 60_000
			if len(bars) == 0 || bars[len(bars)-1].OpenTime != bucket {
				bars = append(bars, backtest.Bar{Symbol: *symbol, Interval: "1m", OpenTime: bucket, CloseTime: bucket + 59_999,
					Open: trade.Price, High: trade.Price, Low: trade.Price})
			}
			bar := &bars[len(bars)-1]
			bar.High, bar.Low = max(bar.High, trade.Price), min(bar.Low, trade.Price)
			bar.Close = trade.Price
			bar.Volume += trade.Quantity
			bar.QuoteVolume += trade.Price * trade.Quantity
			bar.TradeCount++
			if !trade.IsBuyerMaker {
				bar.TakerBuyQuoteVolume += trade.Price * trade.Quantity
			}
		}
		audit.UniqueTradeIDs = len(seenIDs)
		audit.TradeIDSpan = maximumID - minimumID + 1
		if err := validateResearchBars(*symbol, "1m", bars, start, end); err != nil {
			return err
		}
		audit.Series["daily_trades_1m"] = bars
	}
	for _, key := range []string{"arm_1m", "monthly_1m", "daily_1m", "daily_trades_1m"} {
		if bars, ok := audit.Series[key]; ok {
			audit.Aggregates[key], audit.ZeroMinutes[key] = aggregateResearchMinutes(bars, *interval)
		}
	}
	content, err := json.MarshalIndent(audit, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(*output, append(content, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Println("source audit saved:", *output)
	return nil
}
