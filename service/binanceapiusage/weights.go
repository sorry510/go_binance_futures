package binanceapiusage

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// EstimateWeight is deliberately best-effort. Binance response headers remain
// authoritative for current account/IP usage; estimates exist only to rank
// heavy endpoints inside this process.
func EstimateWeight(product, method, path string, query url.Values) int64 {
	product = strings.ToLower(strings.TrimSpace(product))
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)

	switch product {
	case "futures":
		return estimateFuturesWeight(method, path, query)
	case "delivery":
		return estimateDeliveryWeight(method, path, query)
	case "spot":
		return estimateSpotWeight(method, path, query)
	default:
		return 1
	}
}

func estimateFuturesWeight(method, path string, query url.Values) int64 {
	switch path {
	case "/fapi/v1/klines":
		limit := queryInt(query, "limit")
		switch {
		case limit > 1000:
			return 10
		case limit >= 500:
			return 5
		case limit >= 100:
			return 2
		default:
			return 1
		}
	case "/fapi/v1/depth":
		limit := queryInt(query, "limit")
		switch {
		case limit > 500:
			return 20
		case limit > 100:
			return 10
		case limit > 50:
			return 5
		default:
			return 2
		}
	case "/fapi/v1/openOrders":
		if strings.TrimSpace(query.Get("symbol")) == "" {
			return 40
		}
		return 1
	case "/fapi/v1/allOrders":
		return 5
	case "/fapi/v1/income":
		return 30
	case "/fapi/v2/account", "/fapi/v2/positionRisk", "/fapi/v3/positionRisk":
		return 5
	case "/fapi/v1/ticker/24hr":
		if strings.TrimSpace(query.Get("symbol")) == "" {
			return 40
		}
		return 1
	case "/fapi/v1/premiumIndex":
		if strings.TrimSpace(query.Get("symbol")) == "" {
			return 10
		}
		return 1
	case "/fapi/v1/order", "/fapi/v1/algoOrder", "/fapi/v1/leverage", "/fapi/v1/marginType",
		"/fapi/v1/time", "/fapi/v1/exchangeInfo", "/fapi/v1/listenKey":
		return 1
	default:
		return 1
	}
}

func estimateDeliveryWeight(method, path string, query url.Values) int64 {
	if strings.HasSuffix(path, "/openOrders") && strings.TrimSpace(query.Get("symbol")) == "" {
		return 40
	}
	if strings.HasSuffix(path, "/klines") {
		return estimateFuturesWeight(method, "/fapi/v1/klines", query)
	}
	return 1
}

func estimateSpotWeight(method, path string, query url.Values) int64 {
	switch path {
	case "/api/v3/klines":
		return 2
	case "/api/v3/depth":
		limit := queryInt(query, "limit")
		switch {
		case limit > 1000:
			return 250
		case limit > 500:
			return 50
		case limit > 100:
			return 25
		default:
			return 5
		}
	case "/api/v3/openOrders":
		if strings.TrimSpace(query.Get("symbol")) == "" {
			return 80
		}
		return 6
	case "/api/v3/ticker/24hr":
		if strings.TrimSpace(query.Get("symbol")) == "" {
			return 80
		}
		return 2
	case "/api/v3/exchangeInfo":
		return 20
	default:
		return 1
	}
}

func queryInt(query url.Values, key string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(query.Get(key)))
	return value
}

func RequestType(product, method, path string) string {
	product = strings.ToLower(strings.TrimSpace(product))
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete {
		return "read"
	}
	switch product {
	case "futures":
		switch path {
		case "/fapi/v1/order", "/fapi/v1/algoOrder", "/fapi/v1/leverage", "/fapi/v1/marginType":
			return "trade"
		}
	case "delivery":
		if strings.HasSuffix(path, "/order") || strings.HasSuffix(path, "/leverage") || strings.HasSuffix(path, "/marginType") {
			return "trade"
		}
	case "spot":
		if path == "/api/v3/order" {
			return "trade"
		}
	}
	return "read"
}
