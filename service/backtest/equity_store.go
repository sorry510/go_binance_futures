package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

// New backtests store only a bounded, information-preserving chart preview.
// Historical full-resolution chunks remain readable through loadEquityArchiveSample.
func buildEquityStorage(runID string, points []EquityPoint, trades []Trade, createdAt int64) (*models.AgentBacktestEquityPreview, error) {
	if len(points) == 0 {
		return nil, nil
	}
	selected := sampleImportantEquityPoints(points, trades, equityPreviewPoints)
	encoded, err := encodeEquityPayload(selected)
	if err != nil {
		return nil, fmt.Errorf("encode equity preview: %w", err)
	}
	return &models.AgentBacktestEquityPreview{
		RunID: runID, PointCount: len(selected), Encoding: equityEncoding,
		Compression: equityCompression, Checksum: int64(encoded.Checksum),
		Payload: encoded.Payload, CreatedAt: createdAt,
	}, nil
}

func decodeStoredEquity(encoding, compression, payload string, checksum int64, pointCount int) ([]EquityPoint, error) {
	if encoding != equityEncoding || compression != equityCompression {
		return nil, fmt.Errorf("unsupported equity storage encoding=%q compression=%q", encoding, compression)
	}
	if checksum < 0 || checksum > math.MaxUint32 {
		return nil, fmt.Errorf("invalid equity checksum %d", checksum)
	}
	points, err := decodeEquityPayload(payload, uint32(checksum))
	if err != nil {
		return nil, err
	}
	if len(points) != pointCount {
		return nil, fmt.Errorf("equity point count mismatch: decoded=%d stored=%d", len(points), pointCount)
	}
	return points, nil
}

func loadEquityPreview(runID string) ([]EquityPoint, bool, error) {
	var row models.AgentBacktestEquityPreview
	err := orm.NewOrm().QueryTable(new(models.AgentBacktestEquityPreview)).Filter("run_id", runID).One(&row)
	if err == orm.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	points, err := decodeStoredEquity(row.Encoding, row.Compression, row.Payload, row.Checksum, row.PointCount)
	if err != nil {
		return nil, true, fmt.Errorf("decode equity preview: %w", err)
	}
	return points, true, nil
}

func loadEquityArchiveSample(runID string, limit int) ([]EquityPoint, bool, error) {
	var rows []models.AgentBacktestEquityChunk
	if _, err := orm.NewOrm().QueryTable(new(models.AgentBacktestEquityChunk)).
		Filter("run_id", runID).OrderBy("chunk_index").All(&rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}
	for i := range rows {
		if rows[i].ChunkIndex != i {
			return nil, true, fmt.Errorf("equity chunk index gap at %d: got %d", i, rows[i].ChunkIndex)
		}
		if rows[i].PointCount <= 0 || rows[i].StartSequence > rows[i].EndSequence || rows[i].StartTime > rows[i].EndTime {
			return nil, true, fmt.Errorf("invalid equity chunk metadata at %d", i)
		}
		if i > 0 {
			if rows[i].StartSequence != rows[i-1].EndSequence+1 {
				return nil, true, fmt.Errorf("equity chunk sequence gap between %d and %d", i-1, i)
			}
			if rows[i].StartTime <= rows[i-1].EndTime {
				return nil, true, fmt.Errorf("equity chunk time order invalid between %d and %d", i-1, i)
			}
		}
	}
	if limit <= 0 {
		limit = equityPreviewPoints
	}
	targets := equitySampleSequences(rows[0].StartSequence, rows[len(rows)-1].EndSequence, limit)
	out := make([]EquityPoint, 0, len(targets))
	targetIndex := 0
	for _, row := range rows {
		if targetIndex >= len(targets) {
			break
		}
		if targets[targetIndex] > row.EndSequence {
			continue
		}
		points, err := decodeStoredEquity(row.Encoding, row.Compression, row.Payload, row.Checksum, row.PointCount)
		if err != nil {
			return nil, true, fmt.Errorf("decode equity chunk %d: %w", row.ChunkIndex, err)
		}
		pointIndex := 0
		for targetIndex < len(targets) && targets[targetIndex] <= row.EndSequence {
			target := targets[targetIndex]
			for pointIndex < len(points) && points[pointIndex].Sequence < target {
				pointIndex++
			}
			if pointIndex >= len(points) || points[pointIndex].Sequence != target {
				return nil, true, fmt.Errorf("equity sequence %d missing from chunk %d", target, row.ChunkIndex)
			}
			out = append(out, points[pointIndex])
			targetIndex++
		}
	}
	if targetIndex != len(targets) {
		return nil, true, fmt.Errorf("equity archive sample incomplete: got=%d want=%d", targetIndex, len(targets))
	}
	return out, true, nil
}

func persistPreparedEquityStorage(ctx context.Context, tx orm.TxOrmer, preview *models.AgentBacktestEquityPreview) (execMs int64, err error) {
	started := time.Now()
	if preview != nil {
		if _, err := tx.InsertWithCtx(ctx, preview); err != nil {
			return time.Since(started).Milliseconds(), err
		}
	}
	return time.Since(started).Milliseconds(), nil
}
