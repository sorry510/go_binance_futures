package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/service/binanceapiusage"
)

const leadSAPIBaseURL = "https://api.binance.com"

var ErrLeadSAPIUnauthorized = errors.New("lead SAPI credentials or permissions rejected")

type LeadTraderStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    struct {
		IsLeadTrader bool  `json:"isLeadTrader"`
		Time         int64 `json:"time"`
	} `json:"data"`
}
type LeadTradingSymbol struct {
	Symbol     string `json:"symbol"`
	BaseAsset  string `json:"baseAsset"`
	QuoteAsset string `json:"quoteAsset"`
}
type leadSymbolResponse struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Success *bool               `json:"success,omitempty"`
	Data    []LeadTradingSymbol `json:"data"`
}

// getLeadSAPI fetches signed, read-only SAPI data. HTTP body and URL/signature
// are never returned on error (or persisted in logs).
func (a *AccountClient) getLeadSAPI(ctx context.Context, baseURL, path string, target any) error {
	if a == nil || a.id != LeadAccountID || a.client == nil {
		return errors.New("lead API requires an explicit lead account client")
	}
	q := url.Values{}
	q.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli()-a.readTimeOffset(), 10))
	q.Set("recvWindow", "10000")
	toSign := q.Encode()
	mac := hmac.New(sha256.New, []byte(a.client.SecretKey))
	_, _ = mac.Write([]byte(toSign))
	q.Set("signature", hex.EncodeToString(mac.Sum(nil)))
	req, err := http.NewRequestWithContext(binanceapiusage.WithSource(ctx, "lead_trading_sapi"), http.MethodGet, strings.TrimRight(baseURL, "/")+path+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create lead SAPI request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", a.client.APIKey)
	if a.client.HTTPClient == nil {
		return errors.New("lead HTTP client is not configured")
	}
	resp, err := a.client.HTTPClient.Do(req)
	if err != nil {
		return errors.New("lead SAPI network request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrLeadSAPIUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("lead SAPI returned HTTP %d", resp.StatusCode)
	}
	body := io.LimitReader(resp.Body, 1<<20)
	if err := json.NewDecoder(body).Decode(target); err != nil {
		return errors.New("lead SAPI response could not be decoded")
	}
	return nil
}
func (a *AccountClient) readTimeOffset() int64 {
	a.signedMu.RLock()
	defer a.signedMu.RUnlock()
	return a.client.TimeOffset
}
func (a *AccountClient) LeadTraderStatus(ctx context.Context) (LeadTraderStatus, error) {
	var v LeadTraderStatus
	err := a.getLeadSAPI(ctx, leadSAPIBaseURL, "/sapi/v1/copyTrading/futures/userStatus", &v)
	if err != nil {
		return LeadTraderStatus{}, err
	}
	if v.Code != "000000" || !v.Success {
		return LeadTraderStatus{}, ErrLeadSAPIUnauthorized
	}
	return v, nil
}
func (a *AccountClient) LeadTradingSymbols(ctx context.Context) ([]LeadTradingSymbol, error) {
	var v leadSymbolResponse
	err := a.getLeadSAPI(ctx, leadSAPIBaseURL, "/sapi/v1/copyTrading/futures/leadSymbol", &v)
	if err != nil {
		return nil, err
	}
	if v.Code != "000000" || (v.Success != nil && !*v.Success) {
		return nil, ErrLeadSAPIUnauthorized
	}
	if v.Data == nil {
		return []LeadTradingSymbol{}, nil
	}
	return v.Data, nil
}
