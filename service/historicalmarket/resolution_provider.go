package historicalmarket

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/config"
)

type ResolutionEvidence struct {
	Resolution    string `json:"resolution"`
	SourceRef     string `json:"source_ref,omitempty"`
	ArchiveURL    string `json:"archive_url,omitempty"`
	ArchiveSHA256 string `json:"archive_sha256,omitempty"`
	EvidenceHash  string `json:"evidence_hash"`
	Rows          int    `json:"rows"`
	CacheHit      bool   `json:"cache_hit"`
}

type ResolutionStats struct {
	SecondCacheHits  int   `json:"second_cache_hits"`
	TradeCacheHits   int   `json:"trade_cache_hits"`
	ArchiveCacheHits int   `json:"archive_cache_hits"`
	ArchiveDownloads int   `json:"archive_downloads"`
	DownloadBytes    int64 `json:"download_bytes"`
}

type ResolutionProvider interface {
	MinuteBars(context.Context, string, string, string, int64, int64) ([]Kline, ResolutionEvidence, error)
	SecondBars(context.Context, string, string, int64, int64) ([]Kline, ResolutionEvidence, error)
	Trades(context.Context, string, string, int64, int64) ([]PublicDataTrade, ResolutionEvidence, error)
	Stats() ResolutionStats
	Close() error
}

type AdaptiveResolutionProvider struct {
	repo   *Repository
	public *PublicDataClient
	mu     sync.Mutex
	stats  ResolutionStats
}

func DefaultAdaptiveResolutionProvider() (ResolutionProvider, error) {
	proxyURL, _ := config.String("binance::proxy_url")
	tempRootDir, _ := config.String("binance::public_data_cache_dir")
	if strings.TrimSpace(tempRootDir) == "" {
		tempRootDir = "./cache/tmp"
	}
	client, err := NewPublicDataClient(PublicDataClientConfig{ProxyURL: proxyURL, TempRootDir: tempRootDir})
	if err != nil {
		return nil, err
	}
	provider, err := NewAdaptiveResolutionProvider(DefaultRepository(), client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	return provider, nil
}

func NewAdaptiveResolutionProvider(repo *Repository, public *PublicDataClient) (*AdaptiveResolutionProvider, error) {
	if repo == nil {
		return nil, fmt.Errorf("historical repository is required")
	}
	if public == nil {
		return nil, fmt.Errorf("public data client is required")
	}
	return &AdaptiveResolutionProvider{repo: repo, public: public}, nil
}

func (provider *AdaptiveResolutionProvider) SetActivity(activity func(string)) {
	if provider == nil || provider.public == nil {
		return
	}
	provider.public.SetActivity(activity)
}

func (provider *AdaptiveResolutionProvider) Close() error {
	if provider == nil || provider.public == nil {
		return nil
	}
	return provider.public.Close()
}

func (provider *AdaptiveResolutionProvider) Stats() ResolutionStats {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	return provider.stats
}

func (provider *AdaptiveResolutionProvider) MinuteBars(ctx context.Context, market, symbol, interval string, start, end int64) ([]Kline, ResolutionEvidence, error) {
	rows, err := provider.repo.LoadKlines(ctx, market, symbol, interval, start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	evidence := ResolutionEvidence{Resolution: interval, Rows: len(rows), EvidenceHash: klineEvidenceHash("minute", start, end, rows)}
	if len(rows) > 0 {
		evidence.SourceRef = rows[0].SourceRef
	}
	return rows, evidence, nil
}

func (provider *AdaptiveResolutionProvider) SecondBars(ctx context.Context, market, symbol string, start, end int64) ([]Kline, ResolutionEvidence, error) {
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, ResolutionEvidence{}, err
	}
	local, err := provider.repo.LoadSparseSecondBars(ctx, market, symbol, start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	if sparseSecondRangeCached(local, start, end) {
		provider.mu.Lock()
		provider.stats.SecondCacheHits++
		provider.mu.Unlock()
		evidence := ResolutionEvidence{Resolution: SparseSecondInterval, Rows: len(local), CacheHit: true, EvidenceHash: klineEvidenceHash("second-local", start, end, local)}
		if len(local) > 0 {
			evidence.SourceRef = local[0].SourceRef
			evidence.ArchiveSHA256 = archiveHashFromSourceRef(local[0].SourceRef)
		}
		return local, evidence, nil
	}

	date, err := publicDataArchiveDate(start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	spec := PublicDataArchiveSpec{Kind: ArchiveKindKlines, Period: ArchivePeriodDaily, Symbol: symbol, Interval: SparseSecondInterval, Date: date}
	archive, fetchErr := provider.public.FetchArchive(ctx, spec)
	if fetchErr == nil {
		provider.recordArchive(archive)
		rows, parseErr := provider.public.ParseKlines(ctx, archive, start, end)
		if parseErr != nil {
			return nil, ResolutionEvidence{}, parseErr
		}
		if secondRangeComplete(rows, start, end) {
			if _, err := provider.repo.StoreSparseSecondBars(ctx, rows); err != nil {
				return nil, ResolutionEvidence{}, err
			}
			return rows, archiveKlineEvidence("1s", archive, start, end, rows), nil
		}
	} else if !errors.Is(fetchErr, ErrPublicDataArchiveNotFound) {
		return nil, ResolutionEvidence{}, fetchErr
	}

	trades, tradeEvidence, err := provider.Trades(ctx, market, symbol, start, end)
	if err != nil {
		if fetchErr != nil {
			return nil, ResolutionEvidence{}, fmt.Errorf("1s archive unavailable (%v) and trades fallback failed: %w", fetchErr, err)
		}
		return nil, ResolutionEvidence{}, fmt.Errorf("1s archive incomplete and trades fallback failed: %w", err)
	}
	if len(trades) == 0 {
		return nil, ResolutionEvidence{}, fmt.Errorf("verified trades archive contains no trades for %s %d-%d", strings.ToUpper(symbol), start, end)
	}
	bars := aggregateTradesToSeconds(trades, start, end)
	rangeTag := resolutionRangeTag(start, end)
	for i := range bars {
		bars[i].SourceRef = tradeEvidence.SourceRef + "&derived=trades&" + rangeTag
	}
	if _, err := provider.repo.StoreSparseSecondBars(ctx, bars); err != nil {
		return nil, ResolutionEvidence{}, err
	}
	evidence := ResolutionEvidence{
		Resolution: "1s_from_trades", SourceRef: tradeEvidence.SourceRef, ArchiveURL: tradeEvidence.ArchiveURL,
		ArchiveSHA256: tradeEvidence.ArchiveSHA256, Rows: len(bars), EvidenceHash: klineEvidenceHash("second-from-trades", start, end, bars),
		CacheHit: tradeEvidence.CacheHit,
	}
	return bars, evidence, nil
}

func (provider *AdaptiveResolutionProvider) Trades(ctx context.Context, market, symbol string, start, end int64) ([]PublicDataTrade, ResolutionEvidence, error) {
	if err := validateRange(market, symbol, start, end); err != nil {
		return nil, ResolutionEvidence{}, err
	}
	local, err := provider.repo.LoadSparseTrades(ctx, market, symbol, start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	rangeTag := resolutionRangeTag(start, end)
	if sparseTradeRangeCached(local, start, end) {
		provider.mu.Lock()
		provider.stats.TradeCacheHits++
		provider.mu.Unlock()
		evidence := ResolutionEvidence{Resolution: "trades", Rows: len(local), CacheHit: true, EvidenceHash: tradeEvidenceHash(start, end, local)}
		if len(local) > 0 {
			evidence.SourceRef = local[0].SourceRef
			evidence.ArchiveSHA256 = local[0].ArchiveSHA256
		}
		return local, evidence, nil
	}

	date, err := publicDataArchiveDate(start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	archive, err := provider.public.FetchArchive(ctx, PublicDataArchiveSpec{Kind: ArchiveKindTrades, Period: ArchivePeriodDaily, Symbol: symbol, Date: date})
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	provider.recordArchive(archive)
	rows, err := provider.public.ParseTrades(ctx, archive, start, end)
	if err != nil {
		return nil, ResolutionEvidence{}, err
	}
	sourceRef := archive.URL + "#sha256=" + archive.SHA256 + "&" + rangeTag
	for i := range rows {
		rows[i].SourceRef = sourceRef
		rows[i].ArchiveSHA256 = archive.SHA256
	}
	if len(rows) > 0 {
		if _, err := provider.repo.StoreSparseTrades(ctx, rows); err != nil {
			return nil, ResolutionEvidence{}, err
		}
	}
	evidence := ResolutionEvidence{
		Resolution: "trades", SourceRef: sourceRef, ArchiveURL: archive.URL, ArchiveSHA256: archive.SHA256,
		Rows: len(rows), CacheHit: archive.FromCache, EvidenceHash: tradeEvidenceHash(start, end, rows),
	}
	return rows, evidence, nil
}

func (provider *AdaptiveResolutionProvider) recordArchive(archive PublicDataArchive) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if archive.FromCache {
		provider.stats.ArchiveCacheHits++
		return
	}
	provider.stats.ArchiveDownloads++
	provider.stats.DownloadBytes += archive.Size
}

func publicDataArchiveDate(start, end int64) (time.Time, error) {
	first := time.UnixMilli(start).UTC()
	last := time.UnixMilli(end).UTC()
	if first.Year() != last.Year() || first.YearDay() != last.YearDay() {
		return time.Time{}, fmt.Errorf("high-resolution request must stay within one UTC archive date: %d-%d", start, end)
	}
	return time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, time.UTC), nil
}

func secondRangeComplete(rows []Kline, start, end int64) bool {
	first := ((start + 999) / 1000) * 1000
	last := (end / 1000) * 1000
	if last+999 > end {
		last -= 1000
	}
	if last < first {
		return len(rows) == 0
	}
	expected := int((last-first)/1000) + 1
	if len(rows) != expected {
		return false
	}
	for i, row := range rows {
		if row.OpenTime != first+int64(i)*1000 {
			return false
		}
	}
	return true
}

func sparseSecondRangeCached(rows []Kline, start, end int64) bool {
	if secondRangeComplete(rows, start, end) {
		return true
	}
	if len(rows) == 0 {
		return false
	}
	tag := resolutionRangeTag(start, end)
	for _, row := range rows {
		if !strings.Contains(row.SourceRef, "derived=trades") || !strings.Contains(row.SourceRef, tag) {
			return false
		}
	}
	return true
}

func sparseTradeRangeCached(rows []PublicDataTrade, start, end int64) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		coverageStart, coverageEnd, ok := sourceRefRange(row.SourceRef)
		if !ok || coverageStart > start || coverageEnd < end || !validSHA256Hex(row.ArchiveSHA256) {
			return false
		}
	}
	return true
}

func sourceRefRange(sourceRef string) (int64, int64, bool) {
	marker := "range="
	index := strings.LastIndex(sourceRef, marker)
	if index < 0 {
		return 0, 0, false
	}
	value := sourceRef[index+len(marker):]
	if amp := strings.IndexByte(value, '&'); amp >= 0 {
		value = value[:amp]
	}
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < start {
		return 0, 0, false
	}
	return start, end, true
}

func resolutionRangeTag(start, end int64) string {
	return fmt.Sprintf("range=%d-%d", start, end)
}

func archiveHashFromSourceRef(sourceRef string) string {
	marker := "sha256="
	index := strings.Index(sourceRef, marker)
	if index < 0 {
		return ""
	}
	value := sourceRef[index+len(marker):]
	if amp := strings.IndexByte(value, '&'); amp >= 0 {
		value = value[:amp]
	}
	if len(value) == 64 {
		return value
	}
	return ""
}

func archiveKlineEvidence(resolution string, archive PublicDataArchive, start, end int64, rows []Kline) ResolutionEvidence {
	sourceRef := archive.URL + "#sha256=" + archive.SHA256
	return ResolutionEvidence{
		Resolution: resolution, SourceRef: sourceRef, ArchiveURL: archive.URL, ArchiveSHA256: archive.SHA256,
		EvidenceHash: klineEvidenceHash(resolution, start, end, rows), Rows: len(rows), CacheHit: archive.FromCache,
	}
}

func klineEvidenceHash(kind string, start, end int64, rows []Kline) string {
	payload := struct {
		Kind  string  `json:"kind"`
		Start int64   `json:"start"`
		End   int64   `json:"end"`
		Rows  []Kline `json:"rows"`
	}{kind, start, end, rows}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func tradeEvidenceHash(start, end int64, rows []PublicDataTrade) string {
	payload := struct {
		Start int64             `json:"start"`
		End   int64             `json:"end"`
		Rows  []PublicDataTrade `json:"rows"`
	}{start, end, rows}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func aggregateTradesToSeconds(trades []PublicDataTrade, start, end int64) []Kline {
	if len(trades) == 0 {
		return nil
	}
	ordered := append([]PublicDataTrade(nil), trades...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].TradeTime == ordered[j].TradeTime {
			return ordered[i].TradeID < ordered[j].TradeID
		}
		return ordered[i].TradeTime < ordered[j].TradeTime
	})
	result := make([]Kline, 0)
	var current *Kline
	for _, trade := range ordered {
		if trade.TradeTime < start || trade.TradeTime > end || trade.Price <= 0 || math.IsNaN(trade.Price) {
			continue
		}
		openTime := (trade.TradeTime / 1000) * 1000
		if current == nil || current.OpenTime != openTime {
			if current != nil {
				result = append(result, *current)
			}
			current = &Kline{Market: trade.Market, Symbol: trade.Symbol, Interval: SparseSecondInterval, Source: trade.Source, SourceRef: trade.SourceRef, OpenTime: openTime, CloseTime: openTime + 999, Open: trade.Price, High: trade.Price, Low: trade.Price, Close: trade.Price}
		}
		if trade.Price > current.High {
			current.High = trade.Price
		}
		if trade.Price < current.Low {
			current.Low = trade.Price
		}
		current.Close = trade.Price
		current.Volume += trade.Quantity
		current.QuoteVolume += trade.QuoteQuantity
		current.TradeCount++
		if !trade.IsBuyerMaker {
			current.TakerBuyBaseVolume += trade.Quantity
			current.TakerBuyQuoteVolume += trade.QuoteQuantity
		}
	}
	if current != nil {
		result = append(result, *current)
	}
	return result
}
