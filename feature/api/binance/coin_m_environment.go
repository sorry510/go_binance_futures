package binance

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/adshao/go-binance/v2/delivery"
)

// CoinM endpoints are selected from fixed official hosts. The optional
// BINANCE_COIN_M_ENVIRONMENT only supports mainnet, testnet, and demo.
// Demo requires binance.testnet=true and separate Demo credentials; it never
// borrows the mainnet key or the USD-M testnet endpoint override.
type coinMRuntimeConfig struct {
	Environment string
	RESTURL     string
	WSURL       string
	APIKey      string
	APISecret   string
}

var coinMRuntime coinMRuntimeConfig
var coinMSetupErr error

func resolveCoinMRuntime(testnet bool, requested, mainKey, mainSecret, testKey, testSecret, demoKey, demoSecret string) (coinMRuntimeConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(requested))
	if mode == "" {
		if testnet {
			mode = "testnet"
		} else {
			mode = "mainnet"
		}
	}
	switch mode {
	case "mainnet":
		if testnet {
			return coinMRuntimeConfig{}, errors.New("COIN-M mainnet is forbidden when binance.testnet=true")
		}
		return coinMRuntimeConfig{"mainnet", delivery.BaseApiMainUrl, delivery.BaseWsMainUrl, mainKey, mainSecret}, nil
	case "testnet":
		if !testnet {
			return coinMRuntimeConfig{}, errors.New("COIN-M testnet requires binance.testnet=true")
		}
		return coinMRuntimeConfig{"testnet", delivery.BaseApiTestnetUrl, delivery.BaseWsTestnetUrl, testKey, testSecret}, nil
	case "demo":
		if !testnet {
			return coinMRuntimeConfig{}, errors.New("COIN-M demo requires binance.testnet=true")
		}
		return coinMRuntimeConfig{"demo", delivery.BaseApiDemoURL, delivery.BaseWsDemoURL, demoKey, demoSecret}, nil
	default:
		return coinMRuntimeConfig{}, fmt.Errorf("unsupported BINANCE_COIN_M_ENVIRONMENT %q", mode)
	}
}

func (c coinMRuntimeConfig) hasCredentials() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.APISecret) != ""
}

// A request must stay on exactly the configured DAPI host and cannot be
// silently redirected or routed to FAPI, SAPI, Spot or another environment.
// The guard also applies to every http.Client redirect attempt.
type coinMRESTGuard struct {
	base   http.RoundTripper
	origin string
}

func newCoinMRESTGuard(base http.RoundTripper, restURL string) (*coinMRESTGuard, error) {
	endpoint, err := url.Parse(restURL)
	if err != nil || endpoint == nil || endpoint.Scheme == "" || endpoint.Host == "" || endpoint.User != nil || endpoint.Path != "" {
		return nil, fmt.Errorf("invalid COIN-M REST endpoint")
	}
	if base == nil {
		base = http.DefaultTransport
	}
	return &coinMRESTGuard{base: base, origin: endpoint.Scheme + "://" + endpoint.Host}, nil
}

func (g *coinMRESTGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if g == nil || req == nil || req.URL == nil {
		return nil, errors.New("COIN-M REST endpoint is unavailable")
	}
	if req.URL.Scheme+"://"+req.URL.Host != g.origin {
		return nil, errors.New("COIN-M request rejected: cross-environment REST host")
	}
	path := req.URL.Path
	if !(strings.HasPrefix(path, "/dapi/") || strings.HasPrefix(path, "/futures/data/")) {
		return nil, errors.New("COIN-M request rejected: not a DAPI path")
	}
	return g.base.RoundTrip(req)
}

// Called once during package bootstrap, before COIN-M WS starts. Avoid any
// SDK global mutations after bootstrap: delivery.Ws* uses package-level URLs.
func configureCoinMRuntime(mode coinMRuntimeConfig) (*delivery.Client, error) {
	if mode.Environment == "" {
		return nil, errors.New("COIN-M environment is unconfigured")
	}
	delivery.UseTestnet = mode.Environment == "testnet"
	delivery.UseDemo = mode.Environment == "demo"
	client := delivery.NewClient(mode.APIKey, mode.APISecret)
	client.BaseURL = mode.RESTURL
	return client, nil
}

func validateCoinMWSConfiguration(mode coinMRuntimeConfig) error {
	if mode.Environment == "" || mode.WSURL == "" {
		return errors.New("COIN-M WS environment is unconfigured")
	}
	if delivery.UseTestnet != (mode.Environment == "testnet") || delivery.UseDemo != (mode.Environment == "demo") {
		return errors.New("COIN-M WS environment differs from REST environment")
	}
	var expected string
	switch mode.Environment {
	case "mainnet":
		expected = delivery.BaseWsMainUrl
	case "testnet":
		expected = delivery.BaseWsTestnetUrl
	case "demo":
		expected = delivery.BaseWsDemoURL
	default:
		return errors.New("unknown COIN-M WS environment")
	}
	if mode.WSURL != expected || !strings.HasPrefix(expected, "wss://") {
		return errors.New("COIN-M WS endpoint mismatch")
	}
	return nil
}

// Public market data does not need credentials; private account reads do.
// No private endpoint may use empty credentials or fall back to mainnet.
func coinMPrivateReadiness() error {
	if coinMSetupErr != nil {
		return fmt.Errorf("COIN-M setup unavailable: %w", coinMSetupErr)
	}
	if deliveryClient == nil || !coinMRuntime.hasCredentials() {
		return errors.New("COIN-M account credentials unavailable for selected environment")
	}
	if deliveryClient.BaseURL != coinMRuntime.RESTURL {
		return errors.New("COIN-M client endpoint mismatch")
	}
	return nil
}

// CoinMEnvironment identifies the REST/WS context for DAPI weight accounting.
func CoinMEnvironment() string {
	if coinMSetupErr != nil {
		return ""
	}
	return coinMRuntime.Environment
}
