package backtest

import (
	"context"
	"fmt"
	"math"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

// Keep one chunk per INSERT: a worst-case base64 payload is ~2.9 MiB and must fit MySQL 5.6's default 4 MiB max_allowed_packet.
const equityChunkInsertBatchSize = 1

func buildEquityStorage(runID string, points []EquityPoint, createdAt int64) ([]models.AgentBacktestEquityChunk, *models.AgentBacktestEquityPreview, error) {
	chunks := make([]models.AgentBacktestEquityChunk, 0, (len(points)+equityArchiveChunkPoints-1)/equityArchiveChunkPoints)
	for start, chunkIndex := 0, 0; start < len(points); start, chunkIndex = start+equityArchiveChunkPoints, chunkIndex+1 {
		end := start + equityArchiveChunkPoints
		if end > len(points) {
			end = len(points)
		}
		part := points[start:end]
		encoded, err := encodeEquityPayload(part)
		if err != nil {
			return nil, nil, fmt.Errorf("encode equity chunk %d: %w", chunkIndex, err)
		}
		chunks = append(chunks, models.AgentBacktestEquityChunk{
			RunID: runID, ChunkIndex: chunkIndex,
			StartSequence: part[0].Sequence, EndSequence: part[len(part)-1].Sequence,
			StartTime: part[0].BarTime, EndTime: part[len(part)-1].BarTime,
			PointCount: len(part), Encoding: equityEncoding, Compression: equityCompression,
			Checksum: int64(encoded.Checksum), Payload: encoded.Payload, CreatedAt: createdAt,
		})
	}
	if len(points) == 0 {
		return chunks, nil, nil
	}
	previewPoints := sampleEquityPoints(points, equityPreviewPoints)
	encoded, err := encodeEquityPayload(previewPoints)
	if err != nil {
		return nil, nil, fmt.Errorf("encode equity preview: %w", err)
	}
	preview := &models.AgentBacktestEquityPreview{
		RunID: runID, PointCount: len(previewPoints), Encoding: equityEncoding,
		Compression: equityCompression, Checksum: int64(encoded.Checksum),
		Payload: encoded.Payload, CreatedAt: createdAt,
	}
	return chunks, preview, nil
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

func persistPreparedEquityStorage(ctx context.Context, tx orm.TxOrmer, chunks []models.AgentBacktestEquityChunk, preview *models.AgentBacktestEquityPreview, progress func(points int)) (execMs int64, err error) {
	execStarted := time.Now()
	for start := 0; start < len(chunks); start += equityChunkInsertBatchSize {
		end := start + equityChunkInsertBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[start:end]
		if _, err := tx.InsertMultiWithCtx(ctx, equityChunkInsertBatchSize, &batch); err != nil {
			return time.Since(execStarted).Milliseconds(), err
		}
		if progress != nil {
			persistedPoints := 0
			for _, row := range batch {
				persistedPoints += row.PointCount
			}
			progress(persistedPoints)
		}
	}
	if preview != nil {
		if _, err := tx.InsertWithCtx(ctx, preview); err != nil {
			return time.Since(execStarted).Milliseconds(), err
		}
	}
	return time.Since(execStarted).Milliseconds(), nil
}
