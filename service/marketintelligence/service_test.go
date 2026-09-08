package marketintelligence

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

type fixtureProvider struct {
	name   string
	events []EventInput
	err    error
}

func (provider fixtureProvider) Name() string { return provider.name }
func (provider fixtureProvider) Fetch(context.Context, int64) ([]EventInput, error) {
	return provider.events, provider.err
}

var serviceTestOnce sync.Once
var serviceTestErr error

func setupServiceTest(t *testing.T) Service {
	t.Helper()
	serviceTestOnce.Do(func() {
		if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
			serviceTestErr = err
			return
		}
		orm.RegisterModel(new(models.AgentMarketEvent), new(models.AgentMarketEventSource), new(models.AgentMarketFact), new(models.AgentMarketSourceStatus))
		if err := orm.RegisterDataBase("default", "sqlite3", "file:market_intelligence_test?mode=memory&cache=shared"); err != nil {
			serviceTestErr = err
			return
		}
		serviceTestErr = orm.RunSyncdb("default", true, false)
	})
	if serviceTestErr != nil {
		t.Fatal(serviceTestErr)
	}
	o := orm.NewOrm()
	for _, table := range []string{"agent_market_event_sources", "agent_market_events", "agent_market_facts", "agent_market_source_status"} {
		if _, err := o.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return Service{}
}

func TestIngestEventDeduplicatesAnnouncementAndMergesSources(t *testing.T) {
	service := setupServiceTest(t)
	now := time.Date(2026, 9, 8, 4, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	first := EventInput{Type: EventTypeAnnouncement, Category: "listing", Symbols: []string{"ABCUSDT"}, EventTime: now.Add(-time.Minute).UnixMilli(), ObservedAt: now.UnixMilli(), Source: "binance_announcement", SourceRef: "ann-1", Headline: "Binance Will List ABC (ABC)"}
	item, created, err := service.IngestEvent(context.Background(), first)
	if err != nil || !created {
		t.Fatalf("first ingest: created=%t item=%+v err=%v", created, item, err)
	}
	second := first
	second.Source = "crypto_news"
	second.SourceRef = "news-42"
	second.ObservedAt = now.Add(time.Second).UnixMilli()
	merged, created, err := service.IngestEvent(context.Background(), second)
	if err != nil || created {
		t.Fatalf("duplicate ingest: created=%t item=%+v err=%v", created, merged, err)
	}
	if merged.ID != item.ID || len(merged.Sources) != 2 {
		t.Fatalf("event was not merged: first=%+v merged=%+v", item, merged)
	}
	if merged.ObservedAt != first.ObservedAt {
		t.Fatalf("canonical observed_at must remain first observation: first=%d merged=%d", first.ObservedAt, merged.ObservedAt)
	}
	events, err := service.ListEvents(context.Background(), ListOptions{Symbol: "ABCUSDT", StartTime: now.Add(-time.Hour).UnixMilli(), EndTime: now.Add(time.Hour).UnixMilli(), Limit: 10})
	if err != nil || len(events) != 1 {
		t.Fatalf("unexpected query: events=%+v err=%v", events, err)
	}
}

func TestEventTimeObservedAtAndFreshnessRemainDistinct(t *testing.T) {
	service := setupServiceTest(t)
	eventAt := time.Date(2026, 9, 6, 2, 0, 0, 0, time.UTC)
	observedAt := eventAt.Add(48 * time.Hour)
	service.Now = func() time.Time { return observedAt }
	item, _, err := service.IngestEvent(context.Background(), EventInput{DedupKey: "late-announcement", Type: EventTypeAnnouncement, Category: "listing", Symbols: []string{"BTCUSDT"}, EventTime: eventAt.UnixMilli(), ObservedAt: observedAt.UnixMilli(), Source: "binance_announcement", SourceRef: "late", Headline: "Old announcement"})
	if err != nil {
		t.Fatal(err)
	}
	if item.EventTime == item.ObservedAt || item.Freshness != FreshnessStale || item.FreshnessMs < (47*time.Hour).Milliseconds() {
		t.Fatalf("time semantics lost: %+v", item)
	}
}

func TestProviderFailureIsDataMissingAndDoesNotBlockOtherProviders(t *testing.T) {
	service := setupServiceTest(t)
	now := time.Date(2026, 9, 8, 5, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	results := service.SyncProviders(context.Background(), now.Add(-time.Hour).UnixMilli(),
		fixtureProvider{name: "crypto_news", err: errors.New("upstream timeout")},
		fixtureProvider{name: "binance_announcement", events: []EventInput{{DedupKey: "ann-ok", Type: EventTypeAnnouncement, Category: "listing", Symbols: []string{"BTCUSDT"}, EventTime: now.UnixMilli(), SourceRef: "ok", Headline: "Announcement OK"}}},
	)
	if len(results) != 2 || results[0].Error == "" || results[1].Inserted != 1 {
		t.Fatalf("provider isolation failed: %+v", results)
	}
	snapshot, err := service.Snapshot(context.Background(), "BTCUSDT", 24*time.Hour, 20)
	if err != nil {
		t.Fatal(err)
	}
	foundMissing := false
	for _, value := range snapshot.DataMissing {
		if value == "source:crypto_news" {
			foundMissing = true
		}
	}
	if !foundMissing || len(snapshot.Events) != 1 {
		t.Fatalf("provider failure did not remain partial: %+v", snapshot)
	}
}

func TestFactQueryBySymbolAndWindow(t *testing.T) {
	service := setupServiceTest(t)
	now := time.Date(2026, 9, 8, 6, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	_, created, err := service.IngestFact(context.Background(), FactInput{Type: FactTypeFunding, Category: "derivatives", Symbol: "BTCUSDT", EventTime: now.Add(-time.Minute).UnixMilli(), Source: "binance_futures", Data: []byte(`{"rate_pct":0.01}`)})
	if err != nil || !created {
		t.Fatalf("ingest fact created=%t err=%v", created, err)
	}
	facts, err := service.ListFacts(context.Background(), ListOptions{Symbol: "BTCUSDT", StartTime: now.Add(-time.Hour).UnixMilli(), EndTime: now.UnixMilli(), Limit: 10})
	if err != nil || len(facts) != 1 || facts[0].Freshness != FreshnessFresh {
		t.Fatalf("unexpected facts: %+v err=%v", facts, err)
	}
}

func TestTimelineUsesEndTimeAsReplayFreshnessReference(t *testing.T) {
	service := setupServiceTest(t)
	eventAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return eventAt.Add(48 * time.Hour) }
	_, _, err := service.IngestEvent(context.Background(), EventInput{DedupKey: "replay-ann", Type: EventTypeAnnouncement, Category: "listing", Symbols: []string{"BTCUSDT"}, EventTime: eventAt.UnixMilli(), ObservedAt: eventAt.Add(time.Minute).UnixMilli(), Source: "binance_announcement", Headline: "Replay announcement"})
	if err != nil {
		t.Fatal(err)
	}
	replayEnd := eventAt.Add(2 * time.Hour)
	timeline, err := service.Timeline(context.Background(), "BTCUSDT", eventAt.Add(-time.Minute).UnixMilli(), replayEnd.UnixMilli(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Events) != 1 || timeline.Events[0].Freshness != FreshnessFresh {
		t.Fatalf("replay freshness used wall clock instead of end_time: %+v", timeline)
	}
	if timeline.AsOf != replayEnd.Format(time.RFC3339) {
		t.Fatalf("timeline as_of=%s want=%s", timeline.AsOf, replayEnd.Format(time.RFC3339))
	}
}

func TestParseBinanceAnnouncementPreservesEventAndObservedTime(t *testing.T) {
	published := int64(1788840000000)
	observed := published + 2500
	inner := `{"catalogId":161,"catalogName":"New Cryptocurrency Listing","publishDate":1788840000000,"title":"Binance Will List Example (ABC)","body":"listing body","disclaimer":"x"}`
	raw := []byte(`{"type":"DATA","topic":"com_announcement_en","data":` + strconv.Quote(inner) + `}`)
	input, ok, err := parseBinanceAnnouncement(raw, observed)
	if err != nil || !ok {
		t.Fatalf("parse announcement ok=%t input=%+v err=%v", ok, input, err)
	}
	if input.EventTime != published || input.ObservedAt != observed || len(input.Symbols) != 1 || input.Symbols[0] != "ABCUSDT" {
		t.Fatalf("announcement time/symbol semantics changed: %+v", input)
	}
}

func TestBinanceAnnouncementAndExternalNewsMergeIntoOneCanonicalEvent(t *testing.T) {
	service := setupServiceTest(t)
	published := int64(1788840000000)
	observed := published + 1000
	inner := `{"catalogId":161,"catalogName":"New Cryptocurrency Listing","publishDate":1788840000000,"title":"Binance Will List Example (ABC)","body":"listing body"}`
	raw := []byte(`{"type":"DATA","topic":"com_announcement_en","data":` + strconv.Quote(inner) + `}`)
	announcement, ok, err := parseBinanceAnnouncement(raw, observed)
	if err != nil || !ok {
		t.Fatalf("parse announcement ok=%t input=%+v err=%v", ok, announcement, err)
	}
	first, created, err := service.IngestEvent(context.Background(), announcement)
	if err != nil || !created {
		t.Fatalf("ingest announcement created=%t item=%+v err=%v", created, first, err)
	}
	external := announcement
	external.Source = "crypto_news"
	external.SourceRef = "story-42"
	external.ObservedAt = observed + 5000
	merged, created, err := service.IngestEvent(context.Background(), external)
	if err != nil || created {
		t.Fatalf("ingest external duplicate created=%t item=%+v err=%v", created, merged, err)
	}
	if merged.ID != first.ID || len(merged.Sources) != 2 {
		t.Fatalf("cross-source announcement was not merged: first=%+v merged=%+v", first, merged)
	}
}

func TestParseBinanceAnnouncementClassifiesAlphaListing(t *testing.T) {
	inner := `{"catalogId":200,"catalogName":"Binance Alpha","publishDate":1788840000000,"title":"Binance Alpha Will List Example (XYZ)","body":"alpha listing"}`
	raw := []byte(`{"type":"DATA","topic":"com_announcement_en","data":` + strconv.Quote(inner) + `}`)
	input, ok, err := parseBinanceAnnouncement(raw, 1788840001000)
	if err != nil || !ok || input.Type != EventTypeAlphaListing || len(input.Symbols) != 1 || input.Symbols[0] != "XYZUSDT" {
		t.Fatalf("Alpha announcement classification failed: ok=%t input=%+v err=%v", ok, input, err)
	}
}

func TestTimelineExcludesEventsNotObservedByReplayEnd(t *testing.T) {
	service := setupServiceTest(t)
	eventAt := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	observedAt := eventAt.Add(3 * time.Hour)
	service.Now = func() time.Time { return observedAt }
	_, _, err := service.IngestEvent(context.Background(), EventInput{DedupKey: "late-observation", Type: EventTypeAnnouncement, Category: "listing", Symbols: []string{"BTCUSDT"}, EventTime: eventAt.UnixMilli(), ObservedAt: observedAt.UnixMilli(), Source: "binance_announcement", Headline: "Late observed announcement"})
	if err != nil {
		t.Fatal(err)
	}
	timeline, err := service.Timeline(context.Background(), "BTCUSDT", eventAt.Add(-time.Minute).UnixMilli(), eventAt.Add(2*time.Hour).UnixMilli(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Events) != 0 {
		t.Fatalf("future-observed event leaked into replay: %+v", timeline.Events)
	}
}
