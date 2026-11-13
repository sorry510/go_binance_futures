package binance

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/common"

	"github.com/beego/beego/v2/core/config"
)

func TestSpotQuantTestnetConfigDoesNotReuseFuturesTestnetKeys(t *testing.T) {
	fields := map[string]string{
		"binance::testnet_binance_spot_base_url": "https://testnet.binance.vision",
		"binance::testnet_spot_api_key":          "spot_hmac_key",
		"binance::testnet_spot_api_secret":       "spot_hmac_secret",
		// Distinct Futures Testnet credentials must NEVER be read.
		"binance::testnet_api_key":    "futures_only_key",
		"binance::testnet_api_secret": "futures_only_secret",
		"binance::api_key":            "mainnet_key",
		"binance::api_secret":         "mainnet_secret",
	}
	var accessed []string
	adapter, err := spotQuantTestnetAdapterFromConfig(func(name string) (string, error) {
		accessed = append(accessed, name)
		value, ok := fields[name]
		if !ok {
			return "", errors.New("unexpected configuration key")
		}
		return value, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"binance::testnet_binance_spot_base_url",
		"binance::testnet_spot_api_key",
		"binance::testnet_spot_api_secret",
	}
	if !reflect.DeepEqual(accessed, want) {
		t.Fatalf("unexpected credential fields read: %v", accessed)
	}
	if adapter.c.BaseURL != spotQuantTestnetRESTURL ||
		adapter.c.APIKey != "spot_hmac_key" ||
		adapter.c.SecretKey != "spot_hmac_secret" ||
		adapter.allowOrderMutation ||
		adapter.c.HTTPClient == nil {
		t.Fatal("Spot Testnet adapter did not isolate credentials, endpoint or read-only gate")
	}
	if adapter.c == client || client.BaseURL == spotQuantTestnetRESTURL {
		t.Fatal("Spot Testnet configuration modified the shared Spot Mainnet client")
	}
}

func TestSpotQuantTestnetWrongEndpointOrMissingKeysRejected(t *testing.T) {
	for _, tc := range []struct {
		name, url, key, secret string
	}{
		{"old user URL", "https://testnet.binance.com", "spot", "secret"},
		{"production Binance URL", "https://api.binance.com", "spot", "secret"},
		{"production lookalike", "https://testnet.binance.vision.evil.example", "spot", "secret"},
		{"path with api", "https://testnet.binance.vision/api", "spot", "secret"},
		{"plain HTTP", "http://testnet.binance.vision", "spot", "secret"},
		{"empty base", "", "spot", "secret"},
		{"empty key", spotQuantTestnetRESTURL, "", "secret"},
		{"empty secret", spotQuantTestnetRESTURL, "spot", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := newSpotQuantTestnetAdapter(tc.url, tc.key, tc.secret); err == nil {
				t.Fatalf("%s was allowed without safe testnet-only configuration", tc.name)
			}
		})
	}
	if _, err := spotQuantTestnetAdapterFromConfig(nil); err == nil {
		t.Fatal("nil config accessor accepted")
	}
}

func TestSpotQuantTestnetConfigDoesNotFallBackToFutures(t *testing.T) {
	_, err := spotQuantTestnetAdapterFromConfig(func(field string) (string, error) {
		if strings.HasSuffix(field, "testnet_spot_api_key") {
			return "", errors.New("missing separate Spot testnet key")
		}
		if strings.Contains(field, "testnet_api_key") || strings.Contains(field, "testnet_api_secret") {
			t.Fatal("must not read Futures Testnet credentials")
		}
		return "https://testnet.binance.vision", nil
	})
	if err == nil {
		t.Fatal("missing Spot Testnet key was silently replaced")
	}
}

// Opt-in only; the standard test suite never reads user credentials or makes
// requests. Validation is a local, read-only constructor check, no POST.
func TestSpotQuantLocalTestnetConfigOnlyWhenExplicitlyRequested(t *testing.T) {
	if os.Getenv("SPOT_QUANT_VALIDATE_LOCAL_CONFIG") != "1" {
		t.Skip("requires SPOT_QUANT_VALIDATE_LOCAL_CONFIG=1 (no network)")
	}
	adapter, err := NewSpotQuantTestnetAdapterFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	if adapter.c.BaseURL != "https://testnet.binance.vision" ||
		adapter.c.APIKey == "" || adapter.c.SecretKey == "" ||
		adapter.allowOrderMutation {
		t.Fatal("local Spot Testnet configuration failed the read-only gate")
	}
	if _, err := config.String("binance::testnet_api_key"); err != nil {
		t.Fatalf("Futures config remains independently accessible: %v", err)
	}
}

// Explicitly opt in with SPOT_QUANT_VALIDATE_SIGNED_ACCOUNT=1. Only GET
// /api/v3/time and signed GET /api/v3/account are performed against the
// strictly pinned Spot Testnet client. No balances, API keys or secrets
// are logged. Normal tests skip this network/account access.
func TestSpotQuantLocalTestnetSignedAccountReadOnly(t *testing.T) {
	if os.Getenv("SPOT_QUANT_VALIDATE_SIGNED_ACCOUNT") != "1" {
		t.Skip("signed Spot Testnet account GET requires explicit opt-in")
	}
	adapter, err := NewSpotQuantTestnetAdapterFromConfig()
	if err != nil {
		t.Fatalf("Spot Testnet configuration: %v", err)
	}
	if adapter.allowOrderMutation || adapter.c.BaseURL != spotQuantTestnetRESTURL {
		t.Fatal("signed GET attempted with an unsafe Spot adapter")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err := adapter.c.NewSetServerTimeService().Do(ctx); err != nil {
		t.Fatalf("Spot Testnet server-time GET failed (%T)", err)
	}
	account, err := adapter.c.NewGetAccountService().Do(ctx)
	if err != nil {
		var apiErr *common.APIError
		if errors.As(err, &apiErr) {
			t.Fatalf("Spot Testnet signed account GET rejected with Binance code=%d (not Futures Testnet credentials)", apiErr.Code)
		}
		t.Fatalf("Spot Testnet signed account GET transport failed (%T)", err)
	}
	if account == nil {
		t.Fatal("Spot Testnet signed account GET returned nil")
	}
	t.Logf("Spot Testnet signed GET /api/v3/account PASS; canTrade=%t (no balances displayed)", account.CanTrade)
}
