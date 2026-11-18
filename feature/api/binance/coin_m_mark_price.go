package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CoinMMarkPrice is a PUBLIC DAPI mark price. Do not substitute a ticker's
// last trade price; its timestamp is mandatory for freshness checks.
type CoinMMarkPrice struct {
	Symbol     string `json:"symbol"`
	MarkPrice  string `json:"markPrice"`
	IndexPrice string `json:"indexPrice"`
	Time       int64  `json:"time"`
}

// GetCoinMMarkPricesContext performs one read-only public
// GET /dapi/v1/premiumIndex call for all COIN-M contracts. It uses the
// already isolated DAPI HTTP client (host guard + weight collection).
// Only call in response to explicit read requests, never on a 2s loop.
func GetCoinMMarkPricesContext(ctx context.Context) ([]CoinMMarkPrice, error) {
	if coinMSetupErr != nil {
		return nil, fmt.Errorf("COIN-M mark feed unavailable: %w", coinMSetupErr)
	}
	if deliveryClient == nil || deliveryClient.HTTPClient == nil ||
		coinMRuntime.RESTURL == "" || deliveryClient.BaseURL != coinMRuntime.RESTURL {
		return nil, errors.New("COIN-M DAPI mark client is unconfigured")
	}
	if ctx == nil {
		return nil, errors.New("nil COIN-M mark request context")
	}
	url := strings.TrimRight(coinMRuntime.RESTURL, "/") + "/dapi/v1/premiumIndex"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := deliveryClient.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("COIN-M premiumIndex read: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("COIN-M premiumIndex returned HTTP %d", res.StatusCode)
	}
	var marks []CoinMMarkPrice
	decoder := json.NewDecoder(io.LimitReader(res.Body, 2<<20))
	if err := decoder.Decode(&marks); err != nil {
		return nil, fmt.Errorf("COIN-M premiumIndex response: %w", err)
	}
	if len(marks) == 0 {
		return nil, errors.New("empty COIN-M mark feed")
	}
	return marks, nil
}

func CoinMMarkTimeValid(markTime, nowMillis int64) bool {
	const maxAge = int64((2 * time.Minute) / time.Millisecond)
	return markTime > 0 && nowMillis > 0 && nowMillis-markTime >= -30_000 && nowMillis-markTime <= maxAge
}
