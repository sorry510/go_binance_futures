package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	xproxy "golang.org/x/net/proxy"
)

const maxErrorBodyBytes = 8 * 1024

type httpTransport struct {
	client         *http.Client
	proxyEnabled   bool
	proxyURLMasked string
	proxyDialed    atomic.Bool
}

type ProxyDiagnostics struct {
	Enabled   bool   `json:"enabled"`
	Dialed    bool   `json:"dialed"`
	URLMasked string `json:"url_masked,omitempty"`
}

func (transport *httpTransport) ProxyDiagnostics() ProxyDiagnostics {
	if transport == nil {
		return ProxyDiagnostics{}
	}
	return ProxyDiagnostics{Enabled: transport.proxyEnabled, Dialed: transport.proxyDialed.Load(), URLMasked: transport.proxyURLMasked}
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (err *HTTPError) Error() string {
	return fmt.Sprintf("llm request failed with HTTP %d: %s", err.StatusCode, err.Body)
}

func newHTTPTransport(cfg Config) (*httpTransport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport has unexpected type")
	}
	transport := base.Clone()
	proxyURL := strings.TrimSpace(cfg.ProxyURL)
	result := &httpTransport{proxyEnabled: proxyURL != "", proxyURLMasked: maskProxyURL(proxyURL)}

	if proxyURL != "" {
		transport.Proxy = nil
		if err := validateProxyURL(proxyURL); err != nil {
			return nil, err
		}
		parsed, _ := url.Parse(proxyURL)
		var auth *xproxy.Auth
		if parsed.User != nil {
			password, _ := parsed.User.Password()
			auth = &xproxy.Auth{User: parsed.User.Username(), Password: password}
		}
		baseDialer := &net.Dialer{Timeout: cfg.Timeout, KeepAlive: 30 * time.Second}
		dialer, err := xproxy.SOCKS5("tcp", parsed.Host, auth, baseDialer)
		if err != nil {
			return nil, fmt.Errorf("initialize LLM SOCKS5 proxy: %w", err)
		}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			result.proxyDialed.Store(true)
			if contextDialer, ok := dialer.(xproxy.ContextDialer); ok {
				return contextDialer.DialContext(ctx, network, address)
			}
			return dialer.Dial(network, address)
		}
	}

	result.client = &http.Client{Timeout: cfg.Timeout, Transport: transport}
	return result, nil
}

func (transport *httpTransport) postJSON(ctx context.Context, endpoint string, payload interface{}, destination interface{}, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode llm request: %w", err)
	}

	requestURL := strings.TrimSpace(endpoint)
	if !strings.HasPrefix(requestURL, "http://") && !strings.HasPrefix(requestURL, "https://") {
		return fmt.Errorf("llm api_url must be an absolute HTTP URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create llm request: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := transport.client.Do(req)
	if err != nil {
		return fmt.Errorf("send llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errorBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		if readErr != nil {
			return fmt.Errorf("read llm error response: %w", readErr)
		}
		return &HTTPError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(errorBody))}
	}

	if err := json.NewDecoder(resp.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode llm response: %w", err)
	}
	return nil
}

func requestModel(request Request, fallback string) string {
	if strings.TrimSpace(request.Model) != "" {
		return strings.TrimSpace(request.Model)
	}
	return fallback
}

func requestMaxTokens(request Request, fallback int) int {
	if request.MaxTokens > 0 {
		return request.MaxTokens
	}
	return fallback
}

func requestTemperature(request Request, fallback *float64) *float64 {
	if request.Temperature != nil {
		return request.Temperature
	}
	return fallback
}
