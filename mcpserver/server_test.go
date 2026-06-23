package mcpserver

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"go_binance_futures/scanner"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStreamableHTTPListsReadOnlyTop30Tool(t *testing.T) {
	handler := NewHTTPHandler(NewServer(), HTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   httpServer.URL,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() failed: %v", err)
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("tool count = %d, want 1", len(tools.Tools))
	}
	if tools.Tools[0].Name != Top30ToolName {
		t.Fatalf("tool name = %q, want %q", tools.Tools[0].Name, Top30ToolName)
	}
	if tools.Tools[0].InputSchema == nil {
		t.Fatal("tool input schema is nil")
	}
}

func TestStreamableHTTPCallsTop30Tool(t *testing.T) {
	originalScanTop30 := scanTop30
	scanTop30 = func(_ context.Context, opts scanner.PrefilterOptions) (*scanner.PrefilterResult, error) {
		if opts.Limit != 3 {
			t.Fatalf("limit = %d, want 3", opts.Limit)
		}
		return &scanner.PrefilterResult{
			Source:     "test",
			Candidates: []scanner.PrefilterCandidate{{Rank: 1, Symbol: "TESTUSDT", Score: 88}},
			Excluded:   []scanner.PrefilterExclusion{{Symbol: "DROPUSDT", Reason: "test"}},
			Meta:       map[string]any{"rest_api_used": false},
		}, nil
	}
	defer func() {
		scanTop30 = originalScanTop30
	}()

	handler := NewHTTPHandler(NewServer(), HTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "0.1.0",
	}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   httpServer.URL,
		MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      Top30ToolName,
		Arguments: map[string]any{"limit": 3},
	})
	if err != nil {
		t.Fatalf("CallTool() failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %#v", result.Content)
	}
	if result.StructuredContent == nil {
		t.Fatal("structured content is nil")
	}
}
