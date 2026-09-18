package backtest

import "testing"

func TestEquityChunkPayloadFitsMySQLDefaultPacket(t *testing.T) {
	points := make([]EquityPoint, equityArchiveChunkPoints)
	state := uint64(0x9e3779b97f4a7c15)
	next := func() float64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return float64(state%1000000000000) / 1000003.0
	}
	for i := range points {
		side := ""
		if i%3 == 1 {
			side = "LONG"
		} else if i%3 == 2 {
			side = "SHORT"
		}
		points[i] = EquityPoint{Sequence: i + 1, BarTime: int64(i+1) * 60000, Equity: next(), Cash: next(), UnrealizedPnL: next(), DrawdownPct: next(), PositionSide: side}
	}
	encoded, err := encodeEquityPayload(points)
	if err != nil {
		t.Fatal(err)
	}
	const safePayloadLimit = 3 * 1024 * 1024
	if len(encoded.Payload) >= safePayloadLimit {
		t.Fatalf("full equity chunk payload=%d bytes, must stay below %d to leave room inside MySQL 4 MiB max_allowed_packet", len(encoded.Payload), safePayloadLimit)
	}
}
