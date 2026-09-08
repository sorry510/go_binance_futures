package marketintelligence

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"go_binance_futures/binanceproxy"

	"github.com/gorilla/websocket"
)

const (
	binanceAnnouncementSource = "binance_announcement"
	binanceAnnouncementTopic  = "com_announcement_en"
	binanceAnnouncementURL    = "wss://api.binance.com/sapi/wss"
)

type BinanceAnnouncementConfig struct {
	APIKey   string
	Secret   string
	ProxyURL string
}

type binanceAnnouncementEnvelope struct {
	Type  string `json:"type"`
	Topic string `json:"topic"`
	Data  string `json:"data"`
}

type binanceAnnouncementPayload struct {
	CatalogID   int64  `json:"catalogId"`
	CatalogName string `json:"catalogName"`
	PublishDate int64  `json:"publishDate"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Disclaimer  string `json:"disclaimer"`
}

var (
	parenthesizedAsset = regexp.MustCompile(`\(([A-Za-z0-9]{2,15})\)`)
	usdtPairAsset      = regexp.MustCompile(`(?i)\b([A-Z0-9]{2,15})(?:/USDT|USDT)\b`)
)

// RunBinanceAnnouncementStream continuously consumes Binance's documented
// English announcement topic. It is intentionally isolated from the trading
// and alert loops: connection or ingest failures only update source health.
func RunBinanceAnnouncementStream(ctx context.Context, config BinanceAnnouncementConfig) {
	service := DefaultService()
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.Secret = strings.TrimSpace(config.Secret)
	if config.APIKey == "" || config.Secret == "" {
		_ = service.RecordSourceFailure(context.Background(), binanceAnnouncementSource, time.Now().UnixMilli(), fmt.Errorf("Binance announcement API credentials are not configured"))
		return
	}
	pool, err := binanceproxy.New(config.ProxyURL)
	if err != nil {
		_ = service.RecordSourceFailure(context.Background(), binanceAnnouncementSource, time.Now().UnixMilli(), err)
		return
	}
	for ctx.Err() == nil {
		err = runBinanceAnnouncementConnection(ctx, service, pool, config)
		if ctx.Err() != nil {
			return
		}
		_ = service.RecordSourceFailure(context.Background(), binanceAnnouncementSource, time.Now().UnixMilli(), err)
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runBinanceAnnouncementConnection(ctx context.Context, service Service, pool *binanceproxy.Pool, config BinanceAnnouncementConfig) error {
	endpoint, err := signedAnnouncementURL(config.Secret, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	dialer, err := pool.WebSocketDialer()
	if err != nil {
		return err
	}
	headers := http.Header{}
	headers.Set("X-MBX-APIKEY", config.APIKey)
	conn, response, err := dialer.DialContext(ctx, endpoint, headers)
	if err != nil {
		if response != nil {
			return fmt.Errorf("connect Binance announcement websocket: HTTP %d: %w", response.StatusCode, err)
		}
		return fmt.Errorf("connect Binance announcement websocket: %w", err)
	}
	defer conn.Close()
	_ = service.RecordSourceSuccess(context.Background(), binanceAnnouncementSource, time.Now().UnixMilli())

	pingDone := make(chan struct{})
	defer close(pingDone)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			}
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		input, ok, err := parseBinanceAnnouncement(raw, time.Now().UnixMilli())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if _, _, err := service.IngestEvent(ctx, input); err != nil {
			return err
		}
	}
}

func signedAnnouncementURL(secret string, timestamp int64) (string, error) {
	randomValue := make([]byte, 16)
	if _, err := rand.Read(randomValue); err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("random", hex.EncodeToString(randomValue))
	params.Set("recvWindow", "30000")
	params.Set("timestamp", fmt.Sprintf("%d", timestamp))
	params.Set("topic", binanceAnnouncementTopic)
	unsigned := params.Encode()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsigned))
	return binanceAnnouncementURL + "?" + unsigned + "&signature=" + hex.EncodeToString(mac.Sum(nil)), nil
}

func parseBinanceAnnouncement(raw []byte, observedAt int64) (EventInput, bool, error) {
	var envelope binanceAnnouncementEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return EventInput{}, false, fmt.Errorf("decode Binance announcement envelope: %w", err)
	}
	if envelope.Type != "DATA" || envelope.Topic != binanceAnnouncementTopic || strings.TrimSpace(envelope.Data) == "" {
		return EventInput{}, false, nil
	}
	var payload binanceAnnouncementPayload
	if err := json.Unmarshal([]byte(envelope.Data), &payload); err != nil {
		return EventInput{}, false, fmt.Errorf("decode Binance announcement payload: %w", err)
	}
	if payload.PublishDate <= 0 || strings.TrimSpace(payload.Title) == "" {
		return EventInput{}, false, fmt.Errorf("Binance announcement is missing publishDate or title")
	}
	innerRaw, _ := json.Marshal(payload)
	eventType := EventTypeAnnouncement
	classification := strings.ToLower(payload.CatalogName + " " + payload.Title)
	if strings.Contains(classification, "alpha") && (strings.Contains(classification, "list") || strings.Contains(classification, "launch")) {
		eventType = EventTypeAlphaListing
	}
	return EventInput{
		Type: eventType, Category: payload.CatalogName, Symbols: extractAnnouncementSymbols(payload.Title),
		EventTime: payload.PublishDate, ObservedAt: observedAt, Source: binanceAnnouncementSource,
		SourceRef: fmt.Sprintf("catalog:%d", payload.CatalogID), Headline: strings.TrimSpace(payload.Title),
		Summary: truncateText(payload.Body, 4000), Severity: SeverityInfo, Confidence: 1, Raw: innerRaw,
	}, true, nil
}

func extractAnnouncementSymbols(title string) []string {
	seen := map[string]bool{}
	appendAsset := func(asset string) {
		asset = strings.ToUpper(strings.TrimSpace(asset))
		if asset == "" || asset == "USDT" {
			return
		}
		if !strings.HasSuffix(asset, "USDT") {
			asset += "USDT"
		}
		seen[asset] = true
	}
	for _, match := range parenthesizedAsset.FindAllStringSubmatch(title, -1) {
		if len(match) > 1 {
			appendAsset(match[1])
		}
	}
	for _, match := range usdtPairAsset.FindAllStringSubmatch(strings.ToUpper(title), -1) {
		if len(match) > 1 {
			appendAsset(match[1])
		}
	}
	result := make([]string, 0, len(seen))
	for symbol := range seen {
		result = append(result, symbol)
	}
	sort.Strings(result)
	return result
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if limit > 0 && len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return value
}
