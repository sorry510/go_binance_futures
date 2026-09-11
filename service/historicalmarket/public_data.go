package historicalmarket

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_binance_futures/binanceproxy"

	"golang.org/x/sync/singleflight"
)

const (
	SourceBinancePublicData  = "binance_public_data"
	DefaultPublicDataBaseURL = "https://data.binance.vision"

	ArchivePeriodDaily   = "daily"
	ArchivePeriodMonthly = "monthly"

	ArchiveKindKlines          = "klines"
	ArchiveKindTrades          = "trades"
	ArchiveKindMarkPriceKlines = "markPriceKlines"
)

var ErrPublicDataArchiveNotFound = errors.New("binance public data archive not found")
var ErrPublicDataChecksumMismatch = errors.New("binance public data checksum mismatch")

type PublicDataArchiveSpec struct {
	Kind     string
	Period   string
	Symbol   string
	Interval string
	Date     time.Time
}

type PublicDataArchive struct {
	Spec        PublicDataArchiveSpec
	URL         string
	ChecksumURL string
	Path        string
	SHA256      string
	Size        int64
	FromCache   bool
}

type PublicDataTrade struct {
	Market        string  `json:"market"`
	Symbol        string  `json:"symbol"`
	Source        string  `json:"source"`
	SourceRef     string  `json:"source_ref"`
	ArchiveSHA256 string  `json:"archive_sha256"`
	TradeID       int64   `json:"trade_id"`
	TradeTime     int64   `json:"trade_time"`
	Price         float64 `json:"price"`
	Quantity      float64 `json:"quantity"`
	QuoteQuantity float64 `json:"quote_quantity"`
	IsBuyerMaker  bool    `json:"is_buyer_maker"`
}

type PublicDataClientConfig struct {
	BaseURL      string
	ProxyURL     string
	CacheDir     string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	HTTPClient   *http.Client
}

type PublicDataClient struct {
	baseURL      string
	httpClient   *http.Client
	cacheDir     string
	ownsCacheDir bool
	maxRetries   int
	retryBackoff time.Duration

	mu          sync.Mutex
	archives    map[string]PublicDataArchive
	unavailable map[string]struct{}
	group       singleflight.Group
}

func NewPublicDataClient(config PublicDataClientConfig) (*PublicDataClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultPublicDataBaseURL
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	client := config.HTTPClient
	if client == nil {
		pool, err := binanceproxy.New(config.ProxyURL)
		if err != nil {
			return nil, err
		}
		transport := http.RoundTripper(http.DefaultTransport)
		if pool.Enabled() {
			transport = pool.HTTPClient().Transport
		}
		client = &http.Client{Transport: transport, Timeout: timeout}
	}
	cacheDir := strings.TrimSpace(config.CacheDir)
	ownsCacheDir := false
	if cacheDir == "" {
		var err error
		cacheDir, err = os.MkdirTemp("", "binance-public-data-*")
		if err != nil {
			return nil, fmt.Errorf("create public data cache dir: %w", err)
		}
		ownsCacheDir = true
	} else if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return nil, fmt.Errorf("create public data cache dir: %w", err)
	}
	retryBackoff := config.RetryBackoff
	if retryBackoff <= 0 {
		retryBackoff = 100 * time.Millisecond
	}
	maxRetries := config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries == 0 {
		maxRetries = 2
	}
	return &PublicDataClient{
		baseURL: baseURL, httpClient: client, cacheDir: cacheDir, ownsCacheDir: ownsCacheDir,
		maxRetries: maxRetries, retryBackoff: retryBackoff, archives: map[string]PublicDataArchive{}, unavailable: map[string]struct{}{},
	}, nil
}

func (client *PublicDataClient) Close() error {
	if client == nil || !client.ownsCacheDir || client.cacheDir == "" {
		return nil
	}
	return os.RemoveAll(client.cacheDir)
}

func (client *PublicDataClient) ArchiveURL(spec PublicDataArchiveSpec) (string, error) {
	relative, _, err := publicDataArchivePath(spec)
	if err != nil {
		return "", err
	}
	return client.baseURL + "/" + relative, nil
}

func publicDataArchivePath(spec PublicDataArchiveSpec) (string, string, error) {
	symbol := strings.ToUpper(strings.TrimSpace(spec.Symbol))
	if !validPublicDataToken(symbol) {
		return "", "", fmt.Errorf("invalid public data symbol %q", spec.Symbol)
	}
	period := strings.TrimSpace(spec.Period)
	if period != ArchivePeriodDaily && period != ArchivePeriodMonthly {
		return "", "", fmt.Errorf("unsupported public data period %q", spec.Period)
	}
	if spec.Date.IsZero() {
		return "", "", fmt.Errorf("public data archive date is required")
	}
	date := spec.Date.UTC()
	datePart := date.Format("2006-01-02")
	if period == ArchivePeriodMonthly {
		datePart = date.Format("2006-01")
	}
	kind := strings.TrimSpace(spec.Kind)
	switch kind {
	case ArchiveKindTrades:
		filename := fmt.Sprintf("%s-trades-%s.zip", symbol, datePart)
		return strings.Join([]string{"data", "futures", "um", period, kind, symbol, filename}, "/"), filename, nil
	case ArchiveKindKlines, ArchiveKindMarkPriceKlines:
		interval := strings.TrimSpace(spec.Interval)
		if !validPublicDataToken(interval) {
			return "", "", fmt.Errorf("invalid public data interval %q", spec.Interval)
		}
		filename := fmt.Sprintf("%s-%s-%s.zip", symbol, interval, datePart)
		return strings.Join([]string{"data", "futures", "um", period, kind, symbol, interval, filename}, "/"), filename, nil
	default:
		return "", "", fmt.Errorf("unsupported public data archive kind %q", spec.Kind)
	}
}

func validPublicDataToken(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
			continue
		}
		return false
	}
	return true
}

func (client *PublicDataClient) FetchArchive(ctx context.Context, spec PublicDataArchiveSpec) (PublicDataArchive, error) {
	if client == nil {
		return PublicDataArchive{}, fmt.Errorf("public data client is nil")
	}
	archiveURL, err := client.ArchiveURL(spec)
	if err != nil {
		return PublicDataArchive{}, err
	}
	if archive, ok := client.cachedArchive(archiveURL); ok {
		archive.FromCache = true
		return archive, nil
	}
	if client.archiveUnavailable(archiveURL) {
		return PublicDataArchive{}, fmt.Errorf("%w: %s", ErrPublicDataArchiveNotFound, archiveURL)
	}
	// singleflight intentionally shares the first downloader's context for the
	// in-flight archive. If that owner is cancelled, concurrent waiters receive
	// the same failure and may retry on their next request; partial files are
	// removed by downloadArchive, so a cancelled leader cannot poison the cache.
	result := client.group.DoChan(archiveURL, func() (interface{}, error) {
		if archive, ok := client.cachedArchive(archiveURL); ok {
			archive.FromCache = true
			return archive, nil
		}
		if client.archiveUnavailable(archiveURL) {
			return PublicDataArchive{}, fmt.Errorf("%w: %s", ErrPublicDataArchiveNotFound, archiveURL)
		}
		archive, err := client.downloadArchive(ctx, spec, archiveURL)
		if err != nil {
			if errors.Is(err, ErrPublicDataArchiveNotFound) {
				client.markArchiveUnavailable(archiveURL)
			}
			return PublicDataArchive{}, err
		}
		client.mu.Lock()
		client.archives[archiveURL] = archive
		client.mu.Unlock()
		return archive, nil
	})
	select {
	case <-ctx.Done():
		return PublicDataArchive{}, ctx.Err()
	case item := <-result:
		if item.Err != nil {
			return PublicDataArchive{}, item.Err
		}
		return item.Val.(PublicDataArchive), nil
	}
}

func (client *PublicDataClient) archiveUnavailable(archiveURL string) bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	_, ok := client.unavailable[archiveURL]
	return ok
}

func (client *PublicDataClient) markArchiveUnavailable(archiveURL string) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.unavailable[archiveURL] = struct{}{}
}

func (client *PublicDataClient) cachedArchive(archiveURL string) (PublicDataArchive, bool) {
	client.mu.Lock()
	defer client.mu.Unlock()
	archive, ok := client.archives[archiveURL]
	if !ok {
		return PublicDataArchive{}, false
	}
	if _, err := os.Stat(archive.Path); err != nil {
		delete(client.archives, archiveURL)
		return PublicDataArchive{}, false
	}
	return archive, true
}

func (client *PublicDataClient) downloadArchive(ctx context.Context, spec PublicDataArchiveSpec, archiveURL string) (PublicDataArchive, error) {
	_, filename, err := publicDataArchivePath(spec)
	if err != nil {
		return PublicDataArchive{}, err
	}
	key := sha256.Sum256([]byte(archiveURL))
	target := filepath.Join(client.cacheDir, hex.EncodeToString(key[:8])+"-"+filename)
	part := target + ".part"
	actual, size, err := client.downloadFile(ctx, archiveURL, part)
	if err != nil {
		_ = os.Remove(part)
		return PublicDataArchive{}, err
	}
	checksumURL := archiveURL + ".CHECKSUM"
	checksumBody, err := client.downloadSmall(ctx, checksumURL)
	if err != nil {
		_ = os.Remove(part)
		if errors.Is(err, ErrPublicDataArchiveNotFound) {
			return PublicDataArchive{}, fmt.Errorf("checksum missing for %s: %w", archiveURL, err)
		}
		return PublicDataArchive{}, err
	}
	expected, checksumFilename, err := parsePublicDataChecksum(string(checksumBody))
	if err != nil {
		_ = os.Remove(part)
		return PublicDataArchive{}, fmt.Errorf("parse checksum %s: %w", checksumURL, err)
	}
	if checksumFilename != "" && filepath.Base(checksumFilename) != filename {
		_ = os.Remove(part)
		return PublicDataArchive{}, fmt.Errorf("checksum file names %q, expected %q", checksumFilename, filename)
	}
	if !strings.EqualFold(expected, actual) {
		_ = os.Remove(part)
		return PublicDataArchive{}, fmt.Errorf("%w: expected %s got %s for %s", ErrPublicDataChecksumMismatch, expected, actual, archiveURL)
	}
	if err := os.Rename(part, target); err != nil {
		_ = os.Remove(part)
		return PublicDataArchive{}, fmt.Errorf("finalize public data archive: %w", err)
	}
	return PublicDataArchive{Spec: spec, URL: archiveURL, ChecksumURL: checksumURL, Path: target, SHA256: actual, Size: size}, nil
}

func (client *PublicDataClient) downloadFile(ctx context.Context, sourceURL, target string) (string, int64, error) {
	var lastErr error
	for attempt := 0; attempt <= client.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return "", 0, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return "", 0, err
		}
		response, err := client.httpClient.Do(request)
		if err != nil {
			lastErr = err
		} else if response.StatusCode == http.StatusNotFound {
			response.Body.Close()
			return "", 0, fmt.Errorf("%w: %s", ErrPublicDataArchiveNotFound, sourceURL)
		} else if response.StatusCode >= 500 {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			lastErr = fmt.Errorf("download %s: HTTP %d", sourceURL, response.StatusCode)
		} else if response.StatusCode < 200 || response.StatusCode >= 300 {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			return "", 0, fmt.Errorf("download %s: HTTP %d", sourceURL, response.StatusCode)
		} else {
			file, createErr := os.Create(target)
			if createErr != nil {
				response.Body.Close()
				return "", 0, createErr
			}
			hash := sha256.New()
			written, copyErr := io.Copy(io.MultiWriter(file, hash), response.Body)
			closeErr := file.Close()
			response.Body.Close()
			if copyErr == nil && closeErr == nil {
				return hex.EncodeToString(hash.Sum(nil)), written, nil
			}
			if copyErr != nil {
				lastErr = copyErr
			} else {
				lastErr = closeErr
			}
			_ = os.Remove(target)
		}
		if attempt < client.maxRetries {
			if err := client.waitRetry(ctx, attempt); err != nil {
				return "", 0, err
			}
		}
	}
	return "", 0, fmt.Errorf("download %s failed after retries: %w", sourceURL, lastErr)
}

func (client *PublicDataClient) downloadSmall(ctx context.Context, sourceURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= client.maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.httpClient.Do(request)
		if err != nil {
			lastErr = err
		} else if response.StatusCode == http.StatusNotFound {
			response.Body.Close()
			return nil, fmt.Errorf("%w: %s", ErrPublicDataArchiveNotFound, sourceURL)
		} else if response.StatusCode >= 500 {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			lastErr = fmt.Errorf("download %s: HTTP %d", sourceURL, response.StatusCode)
		} else if response.StatusCode < 200 || response.StatusCode >= 300 {
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			return nil, fmt.Errorf("download %s: HTTP %d", sourceURL, response.StatusCode)
		} else {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
			response.Body.Close()
			if readErr == nil {
				return body, nil
			}
			lastErr = readErr
		}
		if attempt < client.maxRetries {
			if err := client.waitRetry(ctx, attempt); err != nil {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("download %s failed after retries: %w", sourceURL, lastErr)
}

func (client *PublicDataClient) waitRetry(ctx context.Context, attempt int) error {
	delay := client.retryBackoff * time.Duration(1<<attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parsePublicDataChecksum(value string) (string, string, error) {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) == 0 || len(fields[0]) != sha256.Size*2 {
		return "", "", fmt.Errorf("invalid SHA256 checksum")
	}
	if _, err := hex.DecodeString(fields[0]); err != nil {
		return "", "", fmt.Errorf("invalid SHA256 checksum: %w", err)
	}
	filename := ""
	if len(fields) > 1 {
		filename = strings.TrimPrefix(fields[1], "*")
	}
	return strings.ToLower(fields[0]), filename, nil
}

func (client *PublicDataClient) ParseKlines(ctx context.Context, archive PublicDataArchive, start, end int64) ([]Kline, error) {
	rows := make([]Kline, 0, 64)
	err := readPublicDataCSV(ctx, archive.Path, func(record []string, rowNumber int) error {
		if len(record) < 12 {
			return fmt.Errorf("kline CSV row %d has %d columns", rowNumber, len(record))
		}
		openTime, err := parseArchiveInt64(record[0])
		if err != nil {
			if rowNumber == 1 {
				return nil
			}
			return fmt.Errorf("kline CSV row %d open time: %w", rowNumber, err)
		}
		openTime = normalizeArchiveMillis(openTime)
		if start > 0 && openTime < start {
			return nil
		}
		if end > 0 && openTime > end {
			return errStopCSV
		}
		closeTime, err := parseArchiveInt64(record[6])
		if err != nil {
			return fmt.Errorf("kline CSV row %d close time: %w", rowNumber, err)
		}
		closeTime = normalizeArchiveMillis(closeTime)
		open, err := parseArchiveFloat(record[1])
		if err != nil {
			return err
		}
		high, err := parseArchiveFloat(record[2])
		if err != nil {
			return err
		}
		low, err := parseArchiveFloat(record[3])
		if err != nil {
			return err
		}
		closePrice, err := parseArchiveFloat(record[4])
		if err != nil {
			return err
		}
		volume, err := parseArchiveFloat(record[5])
		if err != nil {
			return err
		}
		quoteVolume, err := parseArchiveFloat(record[7])
		if err != nil {
			return err
		}
		tradeCount, err := parseArchiveInt64(record[8])
		if err != nil {
			return err
		}
		takerBase, err := parseArchiveFloat(record[9])
		if err != nil {
			return err
		}
		takerQuote, err := parseArchiveFloat(record[10])
		if err != nil {
			return err
		}
		sourceRef := archive.URL + "#sha256=" + archive.SHA256
		rows = append(rows, Kline{
			Market: MarketFuturesUSDT, Symbol: strings.ToUpper(archive.Spec.Symbol), Interval: archive.Spec.Interval,
			Source: SourceBinancePublicData, SourceRef: sourceRef, OpenTime: openTime, CloseTime: closeTime,
			Open: open, High: high, Low: low, Close: closePrice, Volume: volume, QuoteVolume: quoteVolume,
			TradeCount: tradeCount, TakerBuyBaseVolume: takerBase, TakerBuyQuoteVolume: takerQuote,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	SortKlines(rows)
	return rows, nil
}

func (client *PublicDataClient) ParseTrades(ctx context.Context, archive PublicDataArchive, start, end int64) ([]PublicDataTrade, error) {
	rows := make([]PublicDataTrade, 0, 1024)
	err := readPublicDataCSV(ctx, archive.Path, func(record []string, rowNumber int) error {
		if len(record) < 6 {
			return fmt.Errorf("trade CSV row %d has %d columns", rowNumber, len(record))
		}
		tradeID, err := parseArchiveInt64(record[0])
		if err != nil {
			if rowNumber == 1 {
				return nil
			}
			return fmt.Errorf("trade CSV row %d id: %w", rowNumber, err)
		}
		tradeTime, err := parseArchiveInt64(record[4])
		if err != nil {
			return fmt.Errorf("trade CSV row %d time: %w", rowNumber, err)
		}
		tradeTime = normalizeArchiveMillis(tradeTime)
		if start > 0 && tradeTime < start {
			return nil
		}
		if end > 0 && tradeTime > end {
			return errStopCSV
		}
		price, err := parseArchiveFloat(record[1])
		if err != nil {
			return err
		}
		quantity, err := parseArchiveFloat(record[2])
		if err != nil {
			return err
		}
		quoteQuantity, err := parseArchiveFloat(record[3])
		if err != nil {
			return err
		}
		buyerMaker, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(record[5])))
		if err != nil {
			return fmt.Errorf("trade CSV row %d maker flag: %w", rowNumber, err)
		}
		sourceRef := archive.URL + "#sha256=" + archive.SHA256
		rows = append(rows, PublicDataTrade{
			Market: MarketFuturesUSDT, Symbol: strings.ToUpper(archive.Spec.Symbol), Source: SourceBinancePublicData,
			SourceRef: sourceRef, ArchiveSHA256: archive.SHA256, TradeID: tradeID, TradeTime: tradeTime,
			Price: price, Quantity: quantity, QuoteQuantity: quoteQuantity, IsBuyerMaker: buyerMaker,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TradeTime == rows[j].TradeTime {
			return rows[i].TradeID < rows[j].TradeID
		}
		return rows[i].TradeTime < rows[j].TradeTime
	})
	return rows, nil
}

var errStopCSV = errors.New("stop public data CSV scan")

func readPublicDataCSV(ctx context.Context, archivePath string, visit func([]string, int) error) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("open public data ZIP: %w", err)
	}
	for _, item := range reader.File {
		if item.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(item.Name), ".csv") {
			continue
		}
		stream, err := item.Open()
		if err != nil {
			return err
		}
		csvReader := csv.NewReader(stream)
		csvReader.FieldsPerRecord = -1
		csvReader.ReuseRecord = true
		rowNumber := 0
		for {
			if rowNumber%1024 == 0 {
				if err := ctx.Err(); err != nil {
					stream.Close()
					return err
				}
			}
			record, err := csvReader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				stream.Close()
				return fmt.Errorf("read public data CSV: %w", err)
			}
			rowNumber++
			if err := visit(record, rowNumber); err != nil {
				stream.Close()
				if errors.Is(err, errStopCSV) {
					return nil
				}
				return err
			}
		}
		return stream.Close()
	}
	return fmt.Errorf("public data ZIP contains no CSV file")
}

func parseArchiveInt64(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")), 10, 64)
}

func parseArchiveFloat(value string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func normalizeArchiveMillis(value int64) int64 {
	if value > 100_000_000_000_000 {
		return value / 1000
	}
	return value
}
