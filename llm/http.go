package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxErrorBodyBytes = 8 * 1024

type httpTransport struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

type HTTPError struct {
	StatusCode int
	Body       string
}

func (err *HTTPError) Error() string {
	return fmt.Sprintf("llm request failed with HTTP %d: %s", err.StatusCode, err.Body)
}

func newHTTPTransport(cfg Config) (*httpTransport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid llm proxy_url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &httpTransport{
		client: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		headers: cfg.Headers,
	}, nil
}

func (transport *httpTransport) postJSON(ctx context.Context, endpoint string, payload interface{}, destination interface{}, headers map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode llm request: %w", err)
	}

	requestURL := endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		requestURL = transport.baseURL + "/" + strings.TrimLeft(endpoint, "/")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create llm request: %w", err)
	}
	for key, value := range transport.headers {
		req.Header.Set(key, value)
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
