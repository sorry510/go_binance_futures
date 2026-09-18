package backtest

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math"
)

const (
	equityArchiveChunkPoints = 32768
	equityPreviewPoints      = 20000
	equityCodecVersion       = 1
	equityPositionSideBytes  = 16
	equityRecordBytes        = 8 + 8 + 8*4 + 1 + equityPositionSideBytes
	equityEncoding           = "binary_v1_base64"
	equityCompression        = "gzip"
)

var equityCodecMagic = [8]byte{'B', 'T', 'E', 'Q', '0', '0', '0', '1'}

type encodedEquityPayload struct {
	Payload  string
	Checksum uint32
}

func encodeEquityPayload(points []EquityPoint) (encodedEquityPayload, error) {
	raw := make([]byte, 12+len(points)*equityRecordBytes)
	copy(raw[:8], equityCodecMagic[:])
	binary.LittleEndian.PutUint32(raw[8:12], uint32(len(points)))
	offset := 12
	for _, point := range points {
		binary.LittleEndian.PutUint64(raw[offset:offset+8], uint64(int64(point.Sequence)))
		offset += 8
		binary.LittleEndian.PutUint64(raw[offset:offset+8], uint64(point.BarTime))
		offset += 8
		for _, value := range []float64{point.Equity, point.Cash, point.UnrealizedPnL, point.DrawdownPct} {
			binary.LittleEndian.PutUint64(raw[offset:offset+8], math.Float64bits(value))
			offset += 8
		}
		if len(point.PositionSide) > equityPositionSideBytes {
			return encodedEquityPayload{}, fmt.Errorf("equity position_side too long: %q", point.PositionSide)
		}
		raw[offset] = byte(len(point.PositionSide))
		offset++
		copy(raw[offset:offset+equityPositionSideBytes], point.PositionSide)
		offset += equityPositionSideBytes
	}

	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestSpeed)
	if err != nil {
		return encodedEquityPayload{}, err
	}
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return encodedEquityPayload{}, err
	}
	if err := writer.Close(); err != nil {
		return encodedEquityPayload{}, err
	}
	return encodedEquityPayload{
		Payload:  base64.StdEncoding.EncodeToString(compressed.Bytes()),
		Checksum: crc32.ChecksumIEEE(raw),
	}, nil
}

func decodeEquityPayload(payload string, checksum uint32) ([]EquityPoint, error) {
	compressed, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode equity payload base64: %w", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open equity payload gzip: %w", err)
	}
	raw, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("decompress equity payload: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close equity payload gzip: %w", closeErr)
	}
	if crc32.ChecksumIEEE(raw) != checksum {
		return nil, fmt.Errorf("equity payload checksum mismatch")
	}
	if len(raw) < 12 || !bytes.Equal(raw[:8], equityCodecMagic[:]) {
		return nil, fmt.Errorf("invalid equity payload magic")
	}
	count := int(binary.LittleEndian.Uint32(raw[8:12]))
	if len(raw) != 12+count*equityRecordBytes {
		return nil, fmt.Errorf("invalid equity payload size: points=%d bytes=%d", count, len(raw))
	}
	points := make([]EquityPoint, 0, count)
	offset := 12
	var lastSequence int64 = -1 << 63
	var lastBarTime int64 = -1 << 63
	for i := 0; i < count; i++ {
		sequence := int64(binary.LittleEndian.Uint64(raw[offset : offset+8]))
		offset += 8
		barTime := int64(binary.LittleEndian.Uint64(raw[offset : offset+8]))
		offset += 8
		values := [4]float64{}
		for j := range values {
			values[j] = math.Float64frombits(binary.LittleEndian.Uint64(raw[offset : offset+8]))
			offset += 8
		}
		sideLen := int(raw[offset])
		offset++
		if sideLen > equityPositionSideBytes {
			return nil, fmt.Errorf("invalid equity position_side length %d", sideLen)
		}
		side := string(raw[offset : offset+sideLen])
		offset += equityPositionSideBytes
		if i > 0 && (sequence <= lastSequence || barTime <= lastBarTime) {
			return nil, fmt.Errorf("equity payload is not strictly ordered at point %d", i)
		}
		points = append(points, EquityPoint{
			Sequence: int(sequence), BarTime: barTime, Equity: values[0], Cash: values[1],
			UnrealizedPnL: values[2], DrawdownPct: values[3], PositionSide: side,
		})
		lastSequence, lastBarTime = sequence, barTime
	}
	return points, nil
}

func sampleEquityPoints(points []EquityPoint, limit int) []EquityPoint {
	if limit <= 0 || len(points) <= limit {
		return append([]EquityPoint(nil), points...)
	}
	targets := equitySampleSequences(points[0].Sequence, points[len(points)-1].Sequence, limit)
	out := make([]EquityPoint, 0, len(targets))
	targetIndex := 0
	for _, point := range points {
		if targetIndex >= len(targets) {
			break
		}
		if point.Sequence == targets[targetIndex] {
			out = append(out, point)
			targetIndex++
		}
	}
	return out
}
