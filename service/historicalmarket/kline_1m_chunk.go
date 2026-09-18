package historicalmarket

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"
	"sort"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/klauspost/compress/zstd"
)

const (
	kline1mChunkEncoding    = "binary_v1"
	kline1mChunkCompression = "zstd"
	kline1mChunkMagic       = "K1MC0001"
	kline1mChunkHeaderSize  = 16
	kline1mChunkRecordSize  = 90
)

type kline1mSourceManifestEntry struct {
	Source    string `json:"source"`
	SourceRef string `json:"source_ref,omitempty"`
}

type kline1mChunkRow struct {
	Market         string `orm:"column(market)"`
	Symbol         string `orm:"column(symbol)"`
	MonthStart     int64  `orm:"column(month_start)"`
	StartTime      int64  `orm:"column(start_time)"`
	EndTime        int64  `orm:"column(end_time)"`
	PointCount     int    `orm:"column(point_count)"`
	Encoding       string `orm:"column(encoding)"`
	Compression    string `orm:"column(compression)"`
	Checksum       int64  `orm:"column(checksum)"`
	SourceManifest string `orm:"column(source_manifest)"`
	Payload        []byte `orm:"column(payload)"`
	CreatedAt      int64  `orm:"column(created_at)"`
	UpdatedAt      int64  `orm:"column(updated_at)"`
}

func kline1mMonthStart(value int64) time.Time {
	at := time.UnixMilli(value).UTC()
	return time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func kline1mMonthBounds(month time.Time) (int64, int64) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	next := start.AddDate(0, 1, 0)
	return start.UnixMilli(), next.Add(-time.Millisecond).UnixMilli()
}

func (repo *Repository) kline1mMonthClosed(month time.Time) bool {
	now := time.Now().UTC()
	if repo.Now != nil {
		now = repo.Now().UTC()
	}
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return month.Before(current)
}

func validateContiguous1mRows(rows []Kline) error {
	if len(rows) == 0 {
		return fmt.Errorf("1m chunk requires at least one row")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTime < rows[j].OpenTime })
	for i := range rows {
		if strings.TrimSpace(rows[i].Interval) != "1m" {
			return fmt.Errorf("1m chunk contains interval %q", rows[i].Interval)
		}
		if i > 0 && rows[i].OpenTime != rows[i-1].OpenTime+int64(time.Minute/time.Millisecond) {
			return fmt.Errorf("1m chunk gap between %d and %d", rows[i-1].OpenTime, rows[i].OpenTime)
		}
	}
	return nil
}

func encodeKline1mChunk(rows []Kline) ([]byte, string, uint32, error) {
	if err := validateContiguous1mRows(rows); err != nil {
		return nil, "", 0, err
	}
	manifest := make([]kline1mSourceManifestEntry, 0, 4)
	sourceIndexes := make(map[string]uint16)
	indexes := make([]uint16, len(rows))
	for i, row := range rows {
		key := row.Source + "\x00" + row.SourceRef
		index, ok := sourceIndexes[key]
		if !ok {
			if len(manifest) >= int(^uint16(0)) {
				return nil, "", 0, fmt.Errorf("too many 1m chunk source variants")
			}
			index = uint16(len(manifest))
			sourceIndexes[key] = index
			manifest = append(manifest, kline1mSourceManifestEntry{
				Source: row.Source, SourceRef: row.SourceRef,
			})
		}
		indexes[i] = index
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, "", 0, err
	}
	raw := make([]byte, kline1mChunkHeaderSize+len(rows)*kline1mChunkRecordSize)
	copy(raw[:8], kline1mChunkMagic)
	binary.LittleEndian.PutUint32(raw[8:12], uint32(len(rows)))
	binary.LittleEndian.PutUint32(raw[12:16], uint32(len(manifest)))
	for i, row := range rows {
		offset := kline1mChunkHeaderSize + i*kline1mChunkRecordSize
		binary.LittleEndian.PutUint64(raw[offset+0:], uint64(row.OpenTime))
		binary.LittleEndian.PutUint64(raw[offset+8:], uint64(row.CloseTime))
		binary.LittleEndian.PutUint64(raw[offset+16:], uint64(row.TradeCount))
		binary.LittleEndian.PutUint64(raw[offset+24:], math.Float64bits(row.Open))
		binary.LittleEndian.PutUint64(raw[offset+32:], math.Float64bits(row.High))
		binary.LittleEndian.PutUint64(raw[offset+40:], math.Float64bits(row.Low))
		binary.LittleEndian.PutUint64(raw[offset+48:], math.Float64bits(row.Close))
		binary.LittleEndian.PutUint64(raw[offset+56:], math.Float64bits(row.Volume))
		binary.LittleEndian.PutUint64(raw[offset+64:], math.Float64bits(row.QuoteVolume))
		binary.LittleEndian.PutUint64(raw[offset+72:], math.Float64bits(row.TakerBuyBaseVolume))
		binary.LittleEndian.PutUint64(raw[offset+80:], math.Float64bits(row.TakerBuyQuoteVolume))
		binary.LittleEndian.PutUint16(raw[offset+88:], indexes[i])
	}
	checksum := crc32.ChecksumIEEE(raw)
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedFastest),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		return nil, "", 0, err
	}
	defer encoder.Close()
	return encoder.EncodeAll(raw, nil), string(manifestJSON), checksum, nil
}
func decodeKline1mChunk(row kline1mChunkRow) ([]Kline, error) {
	if row.Encoding != kline1mChunkEncoding || row.Compression != kline1mChunkCompression {
		return nil, fmt.Errorf("unsupported 1m chunk codec %s/%s", row.Encoding, row.Compression)
	}
	decoder, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil, err
	}
	raw, err := decoder.DecodeAll(row.Payload, nil)
	decoder.Close()
	if err != nil {
		return nil, err
	}
	if uint32(row.Checksum) != crc32.ChecksumIEEE(raw) {
		return nil, fmt.Errorf("1m chunk checksum mismatch for %s %s %d", row.Market, row.Symbol, row.MonthStart)
	}
	if len(raw) < kline1mChunkHeaderSize || string(raw[:8]) != kline1mChunkMagic {
		return nil, fmt.Errorf("invalid 1m chunk header")
	}
	count := int(binary.LittleEndian.Uint32(raw[8:12]))
	manifestCount := int(binary.LittleEndian.Uint32(raw[12:16]))
	if count != row.PointCount || len(raw) != kline1mChunkHeaderSize+count*kline1mChunkRecordSize {
		return nil, fmt.Errorf("invalid 1m chunk length: count=%d metadata=%d bytes=%d", count, row.PointCount, len(raw))
	}
	var manifest []kline1mSourceManifestEntry
	if err := json.Unmarshal([]byte(row.SourceManifest), &manifest); err != nil {
		return nil, fmt.Errorf("decode 1m source manifest: %w", err)
	}
	if len(manifest) != manifestCount {
		return nil, fmt.Errorf("1m source manifest count=%d want=%d", len(manifest), manifestCount)
	}
	rows := make([]Kline, count)
	for i := 0; i < count; i++ {
		offset := kline1mChunkHeaderSize + i*kline1mChunkRecordSize
		sourceIndex := int(binary.LittleEndian.Uint16(raw[offset+88:]))
		if sourceIndex >= len(manifest) {
			return nil, fmt.Errorf("invalid 1m source index %d", sourceIndex)
		}
		rows[i] = Kline{
			Market: row.Market, Symbol: row.Symbol, Interval: "1m",
			Source: manifest[sourceIndex].Source, SourceRef: manifest[sourceIndex].SourceRef,
			OpenTime:            int64(binary.LittleEndian.Uint64(raw[offset+0:])),
			CloseTime:           int64(binary.LittleEndian.Uint64(raw[offset+8:])),
			TradeCount:          int64(binary.LittleEndian.Uint64(raw[offset+16:])),
			Open:                math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+24:])),
			High:                math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+32:])),
			Low:                 math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+40:])),
			Close:               math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+48:])),
			Volume:              math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+56:])),
			QuoteVolume:         math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+64:])),
			TakerBuyBaseVolume:  math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+72:])),
			TakerBuyQuoteVolume: math.Float64frombits(binary.LittleEndian.Uint64(raw[offset+80:])),
		}
		if i > 0 && rows[i].OpenTime != rows[i-1].OpenTime+int64(time.Minute/time.Millisecond) {
			return nil, fmt.Errorf("decoded 1m chunk is not contiguous")
		}
	}
	if len(rows) > 0 && (rows[0].OpenTime != row.StartTime || rows[len(rows)-1].OpenTime != row.EndTime) {
		return nil, fmt.Errorf("decoded 1m chunk range mismatch")
	}
	return rows, nil
}

func loadKline1mChunksRange(market, symbol string, minMonthStart, maxMonthStart int64) (map[int64]kline1mChunkRow, error) {
	result := make(map[int64]kline1mChunkRow)
	if minMonthStart <= 0 || maxMonthStart < minMonthStart {
		return result, nil
	}
	db, err := orm.GetDB("default")
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		"SELECT market,symbol,month_start,start_time,end_time,point_count,encoding,compression,checksum,source_manifest,payload,created_at,updated_at "+
			"FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start>=? AND month_start<=? ORDER BY month_start",
		market, strings.ToUpper(symbol), minMonthStart, maxMonthStart,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row kline1mChunkRow
		if err := rows.Scan(
			&row.Market, &row.Symbol, &row.MonthStart, &row.StartTime, &row.EndTime, &row.PointCount,
			&row.Encoding, &row.Compression, &row.Checksum, &row.SourceManifest, &row.Payload, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result[row.MonthStart] = row
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func loadKline1mChunk(market, symbol string, monthStart int64) (kline1mChunkRow, bool, error) {
	db, err := orm.GetDB("default")
	if err != nil {
		return kline1mChunkRow{}, false, err
	}
	var row kline1mChunkRow
	err = db.QueryRow(
		"SELECT market,symbol,month_start,start_time,end_time,point_count,encoding,compression,checksum,source_manifest,payload,created_at,updated_at "+
			"FROM market_klines_1m_chunks WHERE market=? AND symbol=? AND month_start=?",
		market, strings.ToUpper(symbol), monthStart,
	).Scan(
		&row.Market, &row.Symbol, &row.MonthStart, &row.StartTime, &row.EndTime, &row.PointCount,
		&row.Encoding, &row.Compression, &row.Checksum, &row.SourceManifest, &row.Payload, &row.CreatedAt, &row.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return kline1mChunkRow{}, false, nil
	}
	if err != nil {
		return kline1mChunkRow{}, false, err
	}
	return row, true, nil
}

func buildKline1mChunkRow(repo *Repository, rows []Kline) (kline1mChunkRow, error) {
	if len(rows) == 0 {
		return kline1mChunkRow{}, fmt.Errorf("cannot build empty 1m chunk")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTime < rows[j].OpenTime })
	payload, manifest, checksum, err := encodeKline1mChunk(rows)
	if err != nil {
		return kline1mChunkRow{}, err
	}
	now := repo.nowMillis()
	monthStart := kline1mMonthStart(rows[0].OpenTime).UnixMilli()
	return kline1mChunkRow{
		Market: rows[0].Market, Symbol: strings.ToUpper(rows[0].Symbol), MonthStart: monthStart,
		StartTime: rows[0].OpenTime, EndTime: rows[len(rows)-1].OpenTime,
		PointCount: len(rows), Encoding: kline1mChunkEncoding,
		Compression: kline1mChunkCompression, Checksum: int64(checksum),
		SourceManifest: manifest, Payload: payload, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (repo *Repository) upsertKline1mChunkSQL(ctx context.Context, tx *sql.Tx, row kline1mChunkRow) error {
	base := "INSERT INTO market_klines_1m_chunks " +
		"(market,symbol,month_start,start_time,end_time,point_count,encoding,compression,checksum,source_manifest,payload,created_at,updated_at) " +
		"VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)"
	if repo.mysql() {
		base += " ON DUPLICATE KEY UPDATE start_time=VALUES(start_time),end_time=VALUES(end_time),point_count=VALUES(point_count)," +
			"encoding=VALUES(encoding),compression=VALUES(compression),checksum=VALUES(checksum),source_manifest=VALUES(source_manifest)," +
			"payload=VALUES(payload),updated_at=VALUES(updated_at)"
	} else {
		base += " ON CONFLICT(market,symbol,month_start) DO UPDATE SET start_time=excluded.start_time,end_time=excluded.end_time," +
			"point_count=excluded.point_count,encoding=excluded.encoding,compression=excluded.compression,checksum=excluded.checksum," +
			"source_manifest=excluded.source_manifest,payload=excluded.payload,updated_at=excluded.updated_at"
	}
	_, err := tx.ExecContext(ctx, base,
		row.Market, row.Symbol, row.MonthStart, row.StartTime, row.EndTime, row.PointCount,
		row.Encoding, row.Compression, row.Checksum, row.SourceManifest, row.Payload, row.CreatedAt, row.UpdatedAt,
	)
	return err
}

func (repo *Repository) replaceKline1mChunk(ctx context.Context, rows []Kline) error {
	if len(rows) == 0 {
		return nil
	}
	row, err := buildKline1mChunkRow(repo, rows)
	if err != nil {
		return err
	}
	db, err := orm.GetDB("default")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	if err := repo.upsertKline1mChunkSQL(ctx, tx, row); err != nil {
		return rollback(err)
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM market_klines_1m WHERE market=? AND symbol=? AND open_time>=? AND open_time<=?",
		row.Market, row.Symbol, row.StartTime, row.EndTime,
	); err != nil {
		return rollback(err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	invalidateReplay1mMemoryChunk(row.Market, row.Symbol, row.MonthStart)
	return nil
}
func mergeKline1mRows(baseRows, updates []Kline) []Kline {
	byTime := make(map[int64]Kline, len(baseRows)+len(updates))
	for _, row := range baseRows {
		byTime[row.OpenTime] = row
	}
	for _, row := range updates {
		byTime[row.OpenTime] = row
	}
	out := make([]Kline, 0, len(byTime))
	for _, row := range byTime {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OpenTime < out[j].OpenTime })
	return out
}

func rowsAreOfficialPublicData(rows []Kline) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if row.Source != SourceBinancePublicData || strings.TrimSpace(row.Interval) != "1m" {
			return false
		}
	}
	return true
}

func rowsCoverNaturalMonth(rows []Kline) bool {
	if len(rows) == 0 {
		return false
	}
	month := kline1mMonthStart(rows[0].OpenTime)
	monthStart, monthEnd := kline1mMonthBounds(month)
	lastOpen := monthEnd - int64(time.Minute/time.Millisecond) + 1
	return rows[0].OpenTime == monthStart && rows[len(rows)-1].OpenTime == lastOpen
}
func (repo *Repository) upsertKline1mHybrid(ctx context.Context, rows []Kline) (int, error) {
	groups := make(map[int64][]Kline)
	for _, row := range rows {
		groups[kline1mMonthStart(row.OpenTime).UnixMilli()] = append(groups[kline1mMonthStart(row.OpenTime).UnixMilli()], row)
	}
	written := 0
	for monthStart, group := range groups {
		sort.Slice(group, func(i, j int) bool { return group[i].OpenTime < group[j].OpenTime })
		month := time.UnixMilli(monthStart).UTC()
		if !repo.kline1mMonthClosed(month) {
			count, err := repo.upsertKlinesRows("1m", group)
			written += count
			if err != nil {
				return written, err
			}
			continue
		}
		existing, ok, err := loadKline1mChunk(group[0].Market, group[0].Symbol, monthStart)
		if err != nil {
			return written, err
		}
		if ok {
			existingRows, err := decodeKline1mChunk(existing)
			if err != nil {
				return written, err
			}
			merged := mergeKline1mRows(existingRows, group)
			if err := repo.replaceKline1mChunk(ctx, merged); err != nil {
				return written, err
			}
			written += len(group)
			continue
		}
		if err := validateContiguous1mRows(group); err == nil && (rowsAreOfficialPublicData(group) || rowsCoverNaturalMonth(group)) {
			if err := repo.replaceKline1mChunk(ctx, group); err != nil {
				return written, err
			}
			written += len(group)
			continue
		}
		count, err := repo.upsertKlinesRows("1m", group)
		written += count
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func filterKlinesByOpenTime(rows []Kline, start, end int64) []Kline {
	first := sort.Search(len(rows), func(i int) bool { return rows[i].OpenTime >= start })
	last := sort.Search(len(rows), func(i int) bool { return rows[i].OpenTime > end })
	if first >= last {
		return nil
	}
	return rows[first:last]
}

func (repo *Repository) queryKlines1mHybrid(market, symbol string, start, end int64) ([]Kline, error) {
	first, last, ok, err := expectedKlineBounds("1m", start, end)
	if err != nil || !ok {
		return nil, err
	}
	out := make([]Kline, 0)
	for month := kline1mMonthStart(first); !month.After(kline1mMonthStart(last)); month = month.AddDate(0, 1, 0) {
		monthStart, monthEnd := kline1mMonthBounds(month)
		monthLastOpen := monthEnd - int64(time.Minute/time.Millisecond) + 1
		segmentStart, segmentEnd := first, last
		if segmentStart < monthStart {
			segmentStart = monthStart
		}
		if segmentEnd > monthLastOpen {
			segmentEnd = monthLastOpen
		}
		chunk, found, err := loadKline1mChunk(market, symbol, monthStart)
		if err != nil {
			return nil, err
		}
		if found {
			rows, err := decodeKline1mChunk(chunk)
			if err != nil {
				return nil, err
			}
			out = append(out, filterKlinesByOpenTime(rows, segmentStart, segmentEnd)...)
			minuteMs := int64(time.Minute / time.Millisecond)
			if segmentStart < chunk.StartTime {
				rowEnd := chunk.StartTime - minuteMs
				if rowEnd > segmentEnd {
					rowEnd = segmentEnd
				}
				if rowEnd >= segmentStart {
					extra, err := repo.queryKlinesRowsOpenRange(market, symbol, "1m", segmentStart, rowEnd)
					if err != nil {
						return nil, err
					}
					out = append(out, extra...)
				}
			}
			if segmentEnd > chunk.EndTime {
				rowStart := chunk.EndTime + minuteMs
				if rowStart < segmentStart {
					rowStart = segmentStart
				}
				if rowStart <= segmentEnd {
					extra, err := repo.queryKlinesRowsOpenRange(market, symbol, "1m", rowStart, segmentEnd)
					if err != nil {
						return nil, err
					}
					out = append(out, extra...)
				}
			}
			continue
		}
		rows, err := repo.queryKlinesRowsOpenRange(market, symbol, "1m", segmentStart, segmentEnd)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OpenTime < out[j].OpenTime })
	return out, nil
}

func (repo *Repository) compactKline1mMonthIfComplete(ctx context.Context, market, symbol string, month time.Time) (bool, error) {
	if !repo.kline1mMonthClosed(month) {
		return false, nil
	}
	monthStart, monthEnd := kline1mMonthBounds(month)
	var base []Kline
	chunk, found, err := loadKline1mChunk(market, symbol, monthStart)
	if err != nil {
		return false, err
	}
	if found {
		base, err = decodeKline1mChunk(chunk)
		if err != nil {
			return false, err
		}
	}
	rowRows, err := repo.queryKlinesRows(market, symbol, "1m", monthStart, monthEnd)
	if err != nil {
		return false, err
	}
	rows := mergeKline1mRows(base, rowRows)
	expected, err := expectedKlineCount("1m", monthStart, monthEnd)
	if err != nil {
		return false, err
	}
	if len(rows) != expected || !rowsCoverNaturalMonth(rows) {
		return false, nil
	}
	if err := validateContiguous1mRows(rows); err != nil {
		return false, nil
	}
	if err := repo.replaceKline1mChunk(ctx, rows); err != nil {
		return false, err
	}
	return true, nil
}

func (repo *Repository) kline1mRangeComplete(ctx context.Context, market, symbol string, start, end int64) (bool, int, error) {
	first, last, ok, err := expectedKlineBounds("1m", start, end)
	if err != nil || !ok {
		return ok, 0, err
	}
	total := 0
	for month := kline1mMonthStart(first); !month.After(kline1mMonthStart(last)); month = month.AddDate(0, 1, 0) {
		monthStart, monthEnd := kline1mMonthBounds(month)
		monthLastOpen := monthEnd - int64(time.Minute/time.Millisecond) + 1
		segmentStart, segmentEnd := first, last
		if segmentStart < monthStart {
			segmentStart = monthStart
		}
		if segmentEnd > monthLastOpen {
			segmentEnd = monthLastOpen
		}
		chunk, found, err := loadKline1mChunk(market, symbol, monthStart)
		if err != nil {
			return false, total, err
		}
		minuteMs := int64(time.Minute / time.Millisecond)
		expected := int((segmentEnd-segmentStart)/minuteMs) + 1
		segmentCount := 0
		if found {
			overlapStart := segmentStart
			if overlapStart < chunk.StartTime {
				overlapStart = chunk.StartTime
			}
			overlapEnd := segmentEnd
			if overlapEnd > chunk.EndTime {
				overlapEnd = chunk.EndTime
			}
			if overlapStart <= overlapEnd {
				segmentCount += int((overlapEnd-overlapStart)/minuteMs) + 1
			}
			if segmentStart < chunk.StartTime {
				rowEnd := chunk.StartTime - minuteMs
				if rowEnd > segmentEnd {
					rowEnd = segmentEnd
				}
				count, err := repo.countKline1mRows(market, symbol, segmentStart, rowEnd)
				if err != nil {
					return false, total, err
				}
				segmentCount += count
			}
			if segmentEnd > chunk.EndTime {
				rowStart := chunk.EndTime + minuteMs
				if rowStart < segmentStart {
					rowStart = segmentStart
				}
				count, err := repo.countKline1mRows(market, symbol, rowStart, segmentEnd)
				if err != nil {
					return false, total, err
				}
				segmentCount += count
			}
		} else {
			segmentCount, err = repo.countKline1mRows(market, symbol, segmentStart, segmentEnd)
			if err != nil {
				return false, total, err
			}
		}
		total += segmentCount
		if segmentCount != expected {
			return false, total, nil
		}
		if segmentStart == monthStart && segmentEnd == monthLastOpen && repo.kline1mMonthClosed(month) {
			if _, err := repo.compactKline1mMonthIfComplete(ctx, market, symbol, month); err != nil {
				return false, total, err
			}
		}
	}
	expected, err := expectedKlineCount("1m", start, end)
	return total == expected, total, err
}
func (repo *Repository) countKline1mRows(market, symbol string, start, end int64) (int, error) {
	if start > end {
		return 0, nil
	}
	from := " FROM market_klines_1m"
	if repo.mysql() {
		from += " FORCE INDEX (market)"
	}
	var count int
	query := "SELECT COUNT(*)" + from + " WHERE market=? AND symbol=? AND open_time>=? AND open_time<=?"
	if err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), start, end).QueryRow(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (repo *Repository) queryKlinesRows(market, symbol, interval string, start, end int64) ([]Kline, error) {
	first, last, ok, err := expectedKlineBounds(interval, start, end)
	if err != nil || !ok {
		return nil, err
	}
	return repo.queryKlinesRowsOpenRange(market, symbol, interval, first, last)
}

func (repo *Repository) queryKlinesRowsOpenRange(market, symbol, interval string, first, last int64) ([]Kline, error) {
	table, err := KlineTable(interval)
	if err != nil {
		return nil, err
	}
	from := " FROM " + table
	if repo.mysql() {
		from += " FORCE INDEX (market)"
	}
	var rows []models.MarketKline1m
	query := "SELECT market,symbol,open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_base_volume,taker_buy_quote_volume,source,source_ref,created_at,updated_at" +
		from + " WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	if _, err := orm.NewOrm().Raw(query, market, strings.ToUpper(symbol), first, last).QueryRows(&rows); err != nil {
		return nil, err
	}
	out := make([]Kline, 0, len(rows))
	for _, row := range rows {
		out = append(out, Kline{
			Market: row.Market, Symbol: row.Symbol, Interval: interval, OpenTime: row.OpenTime, CloseTime: row.CloseTime,
			Open: row.Open, High: row.High, Low: row.Low, Close: row.Close, Volume: row.Volume, QuoteVolume: row.QuoteVolume,
			TradeCount: row.TradeCount, TakerBuyBaseVolume: row.TakerBuyBaseVolume, TakerBuyQuoteVolume: row.TakerBuyQuoteVolume,
			Source: row.Source, SourceRef: row.SourceRef,
		})
	}
	return out, nil
}
