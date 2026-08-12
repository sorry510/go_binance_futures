package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var expectedAPITools = map[string]string{
	"futures_symbols_list":            "GET /features",
	"futures_liquidation_orders_list": "GET /futures/liquidation-orders",
	"coin_notice_list":                "GET /notice/coin",
	"coin_notice_create":              "POST /notice/coin",
	"coin_notice_update":              "PUT /notice/coin/:id",
	"coin_notice_delete":              "DELETE /notice/coin/:id",
	"coin_notice_set_all_enable":      "PUT /notice/coin/enable/:flag",
	"coin_listen_list":                "GET /listen/coin",
	"coin_listen_create":              "POST /listen/coin",
	"coin_listen_update":              "PUT /listen/coin/:id",
	"coin_listen_delete":              "DELETE /listen/coin/:id",
	"coin_listen_set_all_enable":      "PUT /listen/coin/enable/:flag",
}

func TestStreamableHTTPListsOnlyAllowedTools(t *testing.T) {
	handler := NewHTTPHandler(NewServer(), HTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() failed: %v", err)
	}
	if len(tools.Tools) != len(expectedAPITools) {
		t.Fatalf("tool count = %d, want %d", len(tools.Tools), len(expectedAPITools))
	}
	for _, tool := range tools.Tools {
		if _, ok := expectedAPITools[tool.Name]; !ok {
			t.Fatalf("unexpected MCP tool: %s", tool.Name)
		}
		if tool.InputSchema == nil {
			t.Fatalf("tool %q input schema is nil", tool.Name)
		}
	}
}

func TestAPIToolDefinitionsMatchAllowlist(t *testing.T) {
	if len(apiToolDefinitions) != len(expectedAPITools) {
		t.Fatalf("API tool count = %d, want %d", len(apiToolDefinitions), len(expectedAPITools))
	}
	names := make(map[string]bool, len(apiToolDefinitions))
	for _, definition := range apiToolDefinitions {
		if definition.Name == "" || definition.Category == "" || definition.Description == "" || definition.Method == "" || definition.Path == "" {
			t.Fatalf("incomplete API tool definition: %#v", definition)
		}
		if names[definition.Name] {
			t.Fatalf("duplicate API tool name: %s", definition.Name)
		}
		operation, ok := expectedAPITools[definition.Name]
		if !ok {
			t.Fatalf("tool is not in allowlist: %s", definition.Name)
		}
		if actual := definition.Method + " " + definition.Path; actual != operation {
			t.Fatalf("tool %s operation = %q, want %q", definition.Name, actual, operation)
		}
		names[definition.Name] = true
	}
}

func TestAllowedAPIToolsExistInRouter(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	routerFile := filepath.Join(filepath.Dir(sourceFile), "..", "routers", "router.go")
	file, err := os.Open(routerFile)
	if err != nil {
		t.Fatalf("open router file: %v", err)
	}
	defer file.Close()

	routePattern := regexp.MustCompile(`web\.Router\("([^"]+)".*,\s*"([^"]+)"\)`)
	routerOperations := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "//") {
			continue
		}
		matches := routePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		for _, mapping := range strings.Split(matches[2], ";") {
			parts := strings.SplitN(mapping, ":", 2)
			if len(parts) != 2 {
				t.Fatalf("invalid route mapping %q", mapping)
			}
			routerOperations[strings.ToUpper(parts[0])+" "+matches[1]] = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan router file: %v", err)
	}

	for name, operation := range expectedAPITools {
		if !routerOperations[operation] {
			t.Fatalf("allowed MCP tool %s references missing route %s", name, operation)
		}
	}
}

func TestStreamableHTTPForwardsAllowedAPIRequestWithAuthorization(t *testing.T) {
	originalExecuteAPIRequest := executeAPIRequest
	executeAPIRequest = func(_ context.Context, definition APIToolDefinition, input APIToolInput, authorization string) (*APIResult, error) {
		if definition.Name != "futures_liquidation_orders_list" {
			t.Fatalf("tool name = %q, want futures_liquidation_orders_list", definition.Name)
		}
		if input.Query["symbol"] != "BTCUSDT" {
			t.Fatalf("symbol = %#v, want BTCUSDT", input.Query["symbol"])
		}
		if input.Query["min_notional"] != float64(10000) {
			t.Fatalf("min_notional = %#v, want 10000", input.Query["min_notional"])
		}
		if authorization != "Bearer test-token" {
			t.Fatalf("authorization = %q, want Bearer test-token", authorization)
		}
		return &APIResult{StatusCode: http.StatusOK, Body: map[string]any{"code": float64(200)}}, nil
	}
	defer func() { executeAPIRequest = originalExecuteAPIRequest }()

	httpServer := httptest.NewServer(NewHTTPHandler(NewServer(), HTTPOptions{JSONResponse: true}))
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	httpClient := &http.Client{Transport: authorizationRoundTripper{base: http.DefaultTransport, authorization: "Bearer test-token"}}
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL, HTTPClient: httpClient, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "futures_liquidation_orders_list",
		Arguments: map[string]any{"query": map[string]any{
			"symbol":       "BTCUSDT",
			"min_notional": 10000,
		}},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %#v", result.Content)
	}
}

func TestExecuteInternalAPIRequest(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", request.Method)
		}
		if request.URL.Path != "/notice/coin/12" {
			t.Fatalf("path = %q, want /notice/coin/12", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["enable"] != float64(1) {
			t.Fatalf("enable = %#v, want 1", body["enable"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
	}))
	defer httpServer.Close()

	originalBaseURL := getInternalAPIBaseURL
	getInternalAPIBaseURL = func() (string, error) { return httpServer.URL, nil }
	defer func() { getInternalAPIBaseURL = originalBaseURL }()

	result, err := executeInternalAPIRequest(context.Background(), APIToolDefinition{
		Method: http.MethodPut,
		Path:   "/notice/coin/:id",
	}, APIToolInput{
		PathParams: map[string]string{"id": "12"},
		Body:       map[string]any{"enable": 1},
	}, "Bearer test-token")
	if err != nil {
		t.Fatalf("executeInternalAPIRequest() failed: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want 200", result.StatusCode)
	}
}

type authorizationRoundTripper struct {
	base          http.RoundTripper
	authorization string
}

func (transport authorizationRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", transport.authorization)
	return transport.base.RoundTrip(request)
}

func TestResolveAPIPath(t *testing.T) {
	path, err := resolveAPIPath("/notice/coin/:id", map[string]string{"id": "BTC/USDT"})
	if err != nil {
		t.Fatalf("resolveAPIPath() failed: %v", err)
	}
	if !strings.Contains(path, "BTC%2FUSDT") {
		t.Fatalf("resolved path = %q", path)
	}
	if _, err := resolveAPIPath("/notice/coin/:id", nil); err == nil {
		t.Fatal("resolveAPIPath() should reject a missing id")
	}
}
