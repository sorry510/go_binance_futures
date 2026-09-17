package historicalmarket

import (
	"encoding/binary"
	"hash/crc32"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/config"
)

const (
	replay1mCacheSubdir     = "backtest-1m"
	replay1mCacheMagic      = "BT1M0001"
	replay1mCacheHeaderSize = 16
	replay1mCacheRecordSize = 80
)

type ReplayKlineCacheStats struct {
	FilesHit  int
	FilesMiss int
	ReadMs    int64
	WriteMs   int64
}

func (stats *ReplayKlineCacheStats) add(other ReplayKlineCacheStats) {
	stats.FilesHit += other.FilesHit
	stats.FilesMiss += other.FilesMiss
	stats.ReadMs += other.ReadMs
	stats.WriteMs += other.WriteMs
}
func (repo *Repository) replay1mCacheRoot() string {
	if root := strings.TrimSpace(repo.Replay1mCacheRootDir); root != "" {
		return root
	}
	if !repo.Replay1mCacheEnabled {
		return ""
	}
	root, _ := config.String("binance::public_data_cache_dir")
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	return filepath.Join(root, replay1mCacheSubdir)
}

func cacheComponentSafe(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func replay1mCachePath(root, market, symbol string, month time.Time) (string, bool) {
	market = strings.TrimSpace(market)
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if !cacheComponentSafe(market) || !cacheComponentSafe(symbol) {
		return "", false
	}
	name := month.UTC().Format("2006-01") + ".bin"
	return filepath.Join(root, market, symbol, name), true
}
func replayMonthStart(value int64) time.Time {
	t := time.UnixMilli(value).UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func replayMonthRange(month time.Time) (int64, int64) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	next := start.AddDate(0, 1, 0)
	return start.UnixMilli(), next.UnixMilli() - 1
}

func (repo *Repository) replayMonthCacheable(month time.Time) bool {
	now := time.Now().UTC()
	if repo.Now != nil {
		now = repo.Now().UTC()
	}
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return month.Before(current)
}

func filterReplayKlines(rows []ReplayKline, start, end int64) []ReplayKline {
	first := sort.Search(len(rows), func(i int) bool { return rows[i].OpenTime >= start })
	last := sort.Search(len(rows), func(i int) bool { return rows[i].OpenTime > end })
	if first >= last {
		return nil
	}
	return rows[first:last]
}
func encodeReplay1mRows(rows []ReplayKline) []byte {
	body := make([]byte, len(rows)*replay1mCacheRecordSize)
	for i, row := range rows {
		offset := i * replay1mCacheRecordSize
		binary.LittleEndian.PutUint64(body[offset+0:], uint64(row.OpenTime))
		binary.LittleEndian.PutUint64(body[offset+8:], uint64(row.CloseTime))
		binary.LittleEndian.PutUint64(body[offset+16:], uint64(row.TradeCount))
		binary.LittleEndian.PutUint64(body[offset+24:], math.Float64bits(row.Open))
		binary.LittleEndian.PutUint64(body[offset+32:], math.Float64bits(row.High))
		binary.LittleEndian.PutUint64(body[offset+40:], math.Float64bits(row.Low))
		binary.LittleEndian.PutUint64(body[offset+48:], math.Float64bits(row.Close))
		binary.LittleEndian.PutUint64(body[offset+56:], math.Float64bits(row.Volume))
		binary.LittleEndian.PutUint64(body[offset+64:], math.Float64bits(row.QuoteVolume))
		binary.LittleEndian.PutUint64(body[offset+72:], math.Float64bits(row.TakerBuyQuoteVolume))
	}
	return body
}

func decodeReplay1mRows(body []byte, count int) []ReplayKline {
	rows := make([]ReplayKline, count)
	for i := 0; i < count; i++ {
		offset := i * replay1mCacheRecordSize
		rows[i] = ReplayKline{
			OpenTime:            int64(binary.LittleEndian.Uint64(body[offset+0:])),
			CloseTime:           int64(binary.LittleEndian.Uint64(body[offset+8:])),
			TradeCount:          int64(binary.LittleEndian.Uint64(body[offset+16:])),
			Open:                math.Float64frombits(binary.LittleEndian.Uint64(body[offset+24:])),
			High:                math.Float64frombits(binary.LittleEndian.Uint64(body[offset+32:])),
			Low:                 math.Float64frombits(binary.LittleEndian.Uint64(body[offset+40:])),
			Close:               math.Float64frombits(binary.LittleEndian.Uint64(body[offset+48:])),
			Volume:              math.Float64frombits(binary.LittleEndian.Uint64(body[offset+56:])),
			QuoteVolume:         math.Float64frombits(binary.LittleEndian.Uint64(body[offset+64:])),
			TakerBuyQuoteVolume: math.Float64frombits(binary.LittleEndian.Uint64(body[offset+72:])),
		}
	}
	return rows
}
func readReplay1mCache(path string) ([]ReplayKline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) < replay1mCacheHeaderSize || string(raw[:8]) != replay1mCacheMagic {
		return nil, os.ErrInvalid
	}
	count := int(binary.LittleEndian.Uint32(raw[8:12]))
	wantCRC := binary.LittleEndian.Uint32(raw[12:16])
	body := raw[replay1mCacheHeaderSize:]
	if count < 0 || len(body) != count*replay1mCacheRecordSize || crc32.ChecksumIEEE(body) != wantCRC {
		return nil, os.ErrInvalid
	}
	rows := decodeReplay1mRows(body, count)
	for i := 1; i < len(rows); i++ {
		if rows[i].OpenTime <= rows[i-1].OpenTime {
			return nil, os.ErrInvalid
		}
	}
	return rows, nil
}

func writeReplay1mCache(path string, rows []ReplayKline) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := encodeReplay1mRows(rows)
	header := make([]byte, replay1mCacheHeaderSize)
	copy(header[:8], replay1mCacheMagic)
	binary.LittleEndian.PutUint32(header[8:12], uint32(len(rows)))
	binary.LittleEndian.PutUint32(header[12:16], crc32.ChecksumIEEE(body))
	tmp, err := os.CreateTemp(filepath.Dir(path), ".1m-cache-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(header); err == nil {
		_, err = tmp.Write(body)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
func (repo *Repository) queryReplay1mCached(market, symbol string, start, end int64) ([]ReplayKline, ReplayKlineCacheStats, error) {
	var stats ReplayKlineCacheStats
	first, last, ok, err := expectedKlineBounds("1m", start, end)
	if err != nil || !ok {
		return nil, stats, err
	}
	root := repo.replay1mCacheRoot()
	if root == "" {
		rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", first, last)
		return rows, stats, err
	}
	out := make([]ReplayKline, 0)
	for month := replayMonthStart(first); !month.After(replayMonthStart(last)); month = month.AddDate(0, 1, 0) {
		monthStart, monthEnd := replayMonthRange(month)
		segmentStart, segmentEnd := first, last
		if segmentStart < monthStart {
			segmentStart = monthStart
		}
		if segmentEnd > monthEnd {
			segmentEnd = monthEnd
		}
		if !repo.replayMonthCacheable(month) {
			rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", segmentStart, segmentEnd)
			if err != nil {
				return nil, stats, err
			}
			out = append(out, rows...)
			continue
		}
		path, valid := replay1mCachePath(root, market, symbol, month)
		if !valid {
			rows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", segmentStart, segmentEnd)
			if err != nil {
				return nil, stats, err
			}
			out = append(out, rows...)
			continue
		}
		readStarted := time.Now()
		cached, readErr := readReplay1mCache(path)
		stats.ReadMs += time.Since(readStarted).Milliseconds()
		if readErr == nil {
			stats.FilesHit++
			out = append(out, filterReplayKlines(cached, segmentStart, segmentEnd)...)
			continue
		}
		stats.FilesMiss++
		if !os.IsNotExist(readErr) {
			_ = os.Remove(path)
		}
		monthRows, err := repo.queryReplayKlinesDBOpenRange(market, symbol, "1m", monthStart, monthEnd)
		if err != nil {
			return nil, stats, err
		}
		writeStarted := time.Now()
		_ = writeReplay1mCache(path, monthRows)
		stats.WriteMs += time.Since(writeStarted).Milliseconds()
		out = append(out, filterReplayKlines(monthRows, segmentStart, segmentEnd)...)
	}
	return out, stats, nil
}
func (repo *Repository) invalidateReplay1mCache(rows []Kline) {
	root := repo.replay1mCacheRoot()
	if root == "" {
		return
	}
	paths := make(map[string]struct{})
	for _, row := range rows {
		if strings.TrimSpace(row.Interval) != "1m" || row.OpenTime <= 0 {
			continue
		}
		path, ok := replay1mCachePath(root, row.Market, row.Symbol, replayMonthStart(row.OpenTime))
		if ok {
			paths[path] = struct{}{}
		}
	}
	for path := range paths {
		_ = os.Remove(path)
	}
}
