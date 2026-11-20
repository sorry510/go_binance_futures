package backtest

import (
	"reflect"
	"testing"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

func TestImportantEquitySamplingPreservesKeyEventsAndExtrema(t *testing.T) {
	const n = 100000
	points := make([]EquityPoint, n)
	for i := range points {
		v := 1000.0 + float64(i%71-35)/10 + float64(i)/10000
		points[i] = EquityPoint{Sequence: i + 1, BarTime: int64(i+1)*60000 - 1, Equity: v, Cash: 1000, DrawdownPct: float64(i % 37)}
	}
	// Short-lived spikes and troughs must survive even when uniform sampling
	// would not land on them.
	points[7311].Equity = 2000
	points[7312].Equity = 200
	points[7312].DrawdownPct = 90
	points[59878].Equity = 790
	points[59878].DrawdownPct = 57
	trades := []Trade{
		{EntryTime: int64(54322)*60000 - 5000, ExitTime: int64(54679)*60000 + 25000},
		{EntryTime: int64(59878)*60000 - 15000, ExitTime: int64(59878)*60000 + 15000},
	}
	sample := sampleImportantEquityPoints(points, trades, equityPreviewPoints)
	if len(sample) != equityPreviewPoints {
		t.Fatalf("sample len=%d want=%d", len(sample), equityPreviewPoints)
	}
	if sample[0] != points[0] || sample[len(sample)-1] != points[n-1] {
		t.Fatal("first or last equity point missing")
	}
	included := make(map[int]bool, len(sample))
	for i, p := range sample {
		included[p.Sequence-1] = true
		if i > 0 && (p.Sequence <= sample[i-1].Sequence || p.BarTime <= sample[i-1].BarTime) {
			t.Fatalf("non-monotone preview at %d: %+v %+v", i, sample[i-1], p)
		}
	}
	for _, index := range []int{7311, 7312, 59878, 54320, 54321, 54678, 54679} {
		if !included[index] {
			t.Errorf("important equity point index %d was lost", index)
		}
	}
	repeated := sampleImportantEquityPoints(points, trades, equityPreviewPoints)
	if !reflect.DeepEqual(repeated, sample) {
		t.Fatal("sampling must be deterministic")
	}
	small := sampleEquityPoints(sample, 1000)
	if len(small) != 1000 || small[0] != sample[0] || small[len(small)-1] != sample[len(sample)-1] {
		t.Fatalf("sparse preview resample invalid: count=%d", len(small))
	}
}

func TestImportantEquitySamplingLimitsAndTinyRuns(t *testing.T) {
	points := []EquityPoint{
		{Sequence: 1, BarTime: 1000, Equity: 5},
		{Sequence: 2, BarTime: 2000, Equity: 10, DrawdownPct: 1},
		{Sequence: 3, BarTime: 3000, Equity: 1, DrawdownPct: 30},
		{Sequence: 4, BarTime: 4000, Equity: 9},
		{Sequence: 5, BarTime: 5000, Equity: 7},
	}
	if got := sampleImportantEquityPoints(points, nil, 5); !reflect.DeepEqual(got, points) {
		t.Fatalf("all short-run points must be preserved: %+v", got)
	}
	if got := sampleImportantEquityPoints(points, nil, 1); len(got) != 1 || got[0] != points[0] {
		t.Fatalf("limit 1: %+v", got)
	}
	if got := sampleImportantEquityPoints(points, nil, 2); len(got) != 2 || got[0] != points[0] || got[1] != points[4] {
		t.Fatalf("limit 2: %+v", got)
	}
	if got := sampleEquityPoints([]EquityPoint{points[0], points[2], points[4]}, 2); len(got) != 2 || got[1].Sequence != 5 {
		t.Fatalf("resampling non-contiguous sequence: %+v", got)
	}
}

func TestExistingFullEquityChunksStillReadableWithoutPreview(t *testing.T) {
	setupBacktestStoreTest(t)
	const runID = "bt_legacy_chunk_fallback"
	points := []EquityPoint{
		{Sequence: 1, BarTime: 60000, Equity: 1000, Cash: 1000},
		{Sequence: 2, BarTime: 120000, Equity: 950, Cash: 900, UnrealizedPnL: 50, DrawdownPct: 5},
		{Sequence: 3, BarTime: 180000, Equity: 1050, Cash: 1050},
	}
	encoded, err := encodeEquityPayload(points)
	if err != nil {
		t.Fatal(err)
	}
	_, err = orm.NewOrm().Insert(&models.AgentBacktestEquityChunk{
		RunID: runID, ChunkIndex: 0, StartSequence: 1, EndSequence: 3,
		StartTime: 60000, EndTime: 180000, PointCount: 3,
		Encoding: equityEncoding, Compression: equityCompression,
		Checksum: int64(encoded.Checksum), Payload: encoded.Payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewManager(DatasetBuilder{}).Equity(runID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, points) {
		t.Fatalf("existing legacy archive changed: got=%+v want=%+v", got, points)
	}
	_, err = NewManager(DatasetBuilder{}).Equity(runID, 2)
	if err != nil {
		t.Fatal(err)
	}
}

// Benchmark only; not run by the ordinary test suite. Approximate four years
// of one-minute equity history and measure preview extraction in isolation.
func BenchmarkImportantEquitySamplingFourYears(b *testing.B) {
	const n = 4 * 365 * 24 * 60
	points := make([]EquityPoint, n)
	for i := range points {
		points[i] = EquityPoint{
			Sequence: i + 1, BarTime: int64(i+1) * 60000,
			Equity:      1000 + float64(i%231-115)/17 + float64(i)/120000,
			DrawdownPct: float64(i%357) / 10,
		}
	}
	trades := make([]Trade, 350)
	for i := range trades {
		j := (i + 1) * n / (len(trades) + 1)
		trades[i] = Trade{EntryTime: points[j].BarTime - 5000, ExitTime: points[j+15].BarTime}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := sampleImportantEquityPoints(points, trades, equityPreviewPoints)
		if len(result) != equityPreviewPoints {
			b.Fatalf("preview points=%d", len(result))
		}
	}
}
