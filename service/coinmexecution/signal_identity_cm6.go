package coinmexecution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"reflect"
)

var coinMIntervalCode = map[string]string{
	"1m": "01", "3m": "03", "5m": "05", "15m": "15", "30m": "30",
	"1h": "H1", "2h": "H2", "4h": "H4", "6h": "H6", "8h": "H8", "12h": "HC",
	"1d": "D1", "3d": "D3", "1w": "W1", "1M": "M1",
}
var coinMIntervalSpan = map[string]time.Duration{
	"1m": time.Minute, "3m": 3 * time.Minute, "5m": 5 * time.Minute,
	"15m": 15 * time.Minute, "30m": 30 * time.Minute,
	"1h": time.Hour, "2h": 2 * time.Hour, "4h": 4 * time.Hour,
	"6h": 6 * time.Hour, "8h": 8 * time.Hour, "12h": 12 * time.Hour,
	"1d": 24 * time.Hour, "3d": 72 * time.Hour, "1w": 7 * 24 * time.Hour, "1M": 31 * 24 * time.Hour,
}
var coinMKlineRef = regexp.MustCompile(`\bkline_(1m|3m|5m|15m|30m|1h|2h|4h|6h|8h|12h|1d|3d|1w|1M)\b`)

func coinMSignalInterval(coin models.DeliverySymbols) (string, error) {
	longest := "1m"
	if strings.TrimSpace(coin.Technology) != "" {
		var config technology.TechnologyConfig
		if err := json.Unmarshal([]byte(coin.Technology), &config); err != nil {
			return "", err
		}
		groups := reflect.ValueOf(config)
		for i := 0; i < groups.NumField(); i++ {
			group := groups.Field(i)
			for j := 0; j < group.Len(); j++ {
				item := group.Index(j).Interface().(technology.IndicatorConfig)
				if !item.Enable {
					continue
				}
				dur, ok := coinMIntervalSpan[item.KlineInterval]
				if !ok {
					return "", fmt.Errorf("invalid COIN-M indicator candle interval %q", item.KlineInterval)
				}
				if dur > coinMIntervalSpan[longest] {
					longest = item.KlineInterval
				}
			}
		}
	}
	for _, m := range coinMKlineRef.FindAllStringSubmatch(coin.Strategy, -1) {
		if coinMIntervalSpan[m[1]] > coinMIntervalSpan[longest] {
			longest = m[1]
		}
	}
	return longest, nil
}
func coinMEventEnd(open time.Time, interval string) time.Time {
	if open.IsZero() {
		return time.Time{}
	}
	if interval == "1M" {
		return open.AddDate(0, 1, 0)
	}
	return open.Add(coinMIntervalSpan[interval])
}
func coinMEventOpen(ctx context.Context, symbol, interval string, now time.Time) (time.Time, error) {
	if _, ok := coinMIntervalCode[interval]; !ok {
		return time.Time{}, errors.New("unknown COIN-M signal candle interval")
	}
	// Do not guess Kline boundaries, especially Binance's 3d / 1w /
	// month boundary, or a different Testnet-vs-Mainnet open timestamp.
	child, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	rows, err := binance.CoinMKlinesContext(child, symbol, interval, 2)
	if err != nil {
		return time.Time{}, err
	}
	open := time.UnixMilli(rows[len(rows)-1].OpenTime).UTC()
	if open.IsZero() || open.After(now.Add(30*time.Second)) ||
		!coinMEventEnd(open, interval).After(now.Add(-5*time.Second)) {
		return time.Time{}, errors.New("COIN-M DAPI candle open is inconsistent with signal time")
	}
	return open, nil
}
func coinMStableEntryID(environment, symbol, side, interval string, open time.Time) string {
	prefix := ""
	switch environment {
	case "mainnet":
		prefix = "cmn"
	case "testnet":
		prefix = "cmt"
	case "demo":
		prefix = "cmd"
	// A fake runner may opt into this explicit test-only prefix. The
	// runtime never calls entry with an empty/unconfigured environment.
	case "fixture":
		prefix = "cm6"
	default:
		return ""
	}
	code, ok := coinMIntervalCode[interval]
	if !ok || symbol == "" || (side != "LONG" && side != "SHORT") || open.IsZero() || open.Unix() < 0 {
		return ""
	}
	raw := strconv.FormatInt(open.UTC().Unix(), 36)
	if len(raw) > 7 {
		return ""
	}
	for len(raw) < 7 {
		raw = "0" + raw
	}
	identity := environment + "|" + symbol + "|" + side + "|" + interval + "|" + strconv.FormatInt(open.UTC().Unix(), 10)
	hash := sha256.Sum256([]byte("coinm_cm6_entry_v1|" + identity))
	return prefix + code + raw + hex.EncodeToString(hash[:11]) // exactly 34 chars
}
func coinMDecodeEntryID(id string) (string, time.Time, bool) {
	if len(id) != 34 {
		return "", time.Time{}, false
	}
	prefix := id[:3]
	if prefix != "cmn" && prefix != "cmt" && prefix != "cmd" && prefix != "cm6" {
		return "", time.Time{}, false
	}
	interval := ""
	for key, code := range coinMIntervalCode {
		if code == id[3:5] {
			interval = key
			break
		}
	}
	if interval == "" {
		return "", time.Time{}, false
	}
	seconds, err := strconv.ParseInt(id[5:12], 36, 64)
	if err != nil || seconds <= 0 {
		return "", time.Time{}, false
	}
	return interval, time.Unix(seconds, 0).UTC(), true
}
