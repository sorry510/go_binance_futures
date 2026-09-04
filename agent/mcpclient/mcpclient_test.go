package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/toolruntime"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/models"
)

var setupMCPStoreOnce sync.Once
var setupMCPStoreErr error

func setupMCPTestStore(t *testing.T) Store {
	t.Helper()
	setupMCPStoreOnce.Do(func() {
		_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		orm.RegisterModel(new(models.AgentMCPServer), new(models.AgentMCPTool), new(models.AgentMCPResource), new(models.AgentMCPPrompt), new(models.AgentMCPPermission))
		setupMCPStoreErr = orm.RegisterDataBase("default", "sqlite3", "file:mcp_e2e?mode=memory&cache=shared")
	})
	if setupMCPStoreErr != nil {
		t.Fatal(setupMCPStoreErr)
	}
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatal(err)
	}
	return Store{Alias: "default"}
}
func testMCPServer(t *testing.T) (*mcp.Server, *httptest.Server) {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "fixture-mcp", Version: "1.0.0"}, nil)
	registerFixtureTool(server, false)
	server.AddResource(&mcp.Resource{URI: "memo://market", Name: "market-note", MIMEType: "text/plain"}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: "memo://market", MIMEType: "text/plain", Text: "market resource payload"}}}, nil
	})
	server.AddPrompt(&mcp.Prompt{Name: "analyst-note", Description: "external analyst note"}, func(context.Context, *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: "external prompt payload"}}}}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return server, httpServer
}

func registerFixtureTool(server *mcp.Server, changed bool) {
	input := map[string]any{"type": "object", "properties": map[string]any{"symbol": map[string]any{"type": "string"}}}
	if changed {
		input["required"] = []string{"symbol"}
	}
	output := map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}, "required": []string{"ok"}}
	server.AddTool(&mcp.Tool{Name: "lookup", Description: "lookup fixture", InputSchema: input, OutputSchema: output, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{StructuredContent: map[string]any{"ok": true}}, nil
	})
}
func TestConnectionOnlyInitializesWithoutRefreshingCatalog(t *testing.T) {
	ctx := context.Background()
	store := setupMCPTestStore(t)
	_, httpServer := testMCPServer(t)
	view, err := store.SaveServer(ctx, 0, ServerInput{Name: "connection-only", Endpoint: httpServer.URL, Enabled: 1, AuthType: AuthNone, AllowPrivate: 1})
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(store)
	result, err := gateway.TestConnection(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion == "" || result.ServerName != "fixture-mcp" || result.ServerVersion != "1.0.0" {
		t.Fatalf("unexpected connection result: %+v", result)
	}
	if result.ToolCount != 0 || result.ResourceCount != 0 || result.PromptCount != 0 || result.CatalogHash != "" {
		t.Fatalf("connection test unexpectedly refreshed catalog: %+v", result)
	}
	catalog, err := store.Catalog(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tools) != 0 || len(catalog.Resources) != 0 || len(catalog.Prompts) != 0 {
		t.Fatalf("connection test persisted catalog entries: %+v", catalog)
	}
	if catalog.Server.Status != "healthy" || catalog.Server.ProtocolVersion == "" {
		t.Fatalf("connection test did not update server health: %+v", catalog.Server)
	}
}

func TestStreamableHTTPDiscoveryGovernanceAndRuntime(t *testing.T) {
	ctx := context.Background()
	store := setupMCPTestStore(t)
	server, httpServer := testMCPServer(t)
	view, err := store.SaveServer(ctx, 0, ServerInput{Name: "fixture", Endpoint: httpServer.URL, Enabled: 1, AuthType: AuthNone, AllowPrivate: 1})
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(store)
	refresh, err := gateway.RefreshCatalog(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refresh.ProtocolVersion == "" || refresh.ToolCount != 1 || refresh.ResourceCount != 1 || refresh.PromptCount != 1 {
		t.Fatalf("unexpected discovery: %+v", refresh)
	}
	catalog, err := store.Catalog(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Tools) != 1 || catalog.Tools[0].Status != ToolUnclassified || catalog.Tools[0].Enabled != 0 {
		t.Fatalf("new tool was not disabled/unclassified: %+v", catalog.Tools)
	}
	toolRow, err := store.UpdateTool(ctx, catalog.Tools[0].ID, ToolUpdateInput{Risk: permission.RiskRead, Enabled: 1, Idempotent: true, TimeoutMs: 5000, CacheTTLms: 1000, MaxResultBytes: 65536})
	if err != nil {
		t.Fatal(err)
	}
	for _, permission := range []PermissionInput{
		{ServerID: view.ID, SkillName: "fixture_skill", CapabilityType: CapabilityTool, CapabilityID: toolRow.ID, Enabled: 1},
		{ServerID: view.ID, SkillName: "fixture_skill", CapabilityType: CapabilityResource, CapabilityID: catalog.Resources[0].ID, Enabled: 1, AutoLoad: 1},
		{ServerID: view.ID, SkillName: "fixture_skill", CapabilityType: CapabilityPrompt, CapabilityID: catalog.Prompts[0].ID, Enabled: 1, AutoLoad: 1},
	} {
		if _, err := store.SavePermission(ctx, permission); err != nil {
			t.Fatal(err)
		}
	}
	remoteTools, err := gateway.ActiveRemoteTools(ctx)
	if err != nil || len(remoteTools) != 1 {
		t.Fatalf("active tools: %v %+v", err, remoteTools)
	}
	registry := agenttools.NewRegistry()
	if err := registry.Register(remoteTools[0]); err != nil {
		t.Fatal(err)
	}
	runtime, err := toolruntime.New(toolruntime.Config{Registry: registry, Policy: permission.AllowReadOnly(), ContextEngine: contextengine.New()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Execute(ctx, toolruntime.ExecuteRequest{SkillName: "fixture_skill", AllowedTools: map[string]bool{toolRow.CanonicalName: true}, ToolName: toolRow.CanonicalName, Arguments: json.RawMessage(`{"symbol":"BTCUSDT"}`)})
	if err != nil || result.ToolError != nil {
		t.Fatalf("MCP ToolRuntime execute: err=%v toolErr=%v", err, result.ToolError)
	}
	if result.Trace.SourceType != toolruntime.SourceMCP || result.Trace.ProtocolVersion == "" || result.Trace.CatalogHash == "" || result.Trace.SchemaHash == "" {
		t.Fatalf("MCP trace identity missing: %+v", result.Trace)
	}
	contextResources, err := gateway.ContextResourcesForSkill(ctx, "fixture_skill")
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := contextengine.New().LoadResources(ctx, contextResources, nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, block := range blocks {
		joined += block.Content + "\n"
	}
	if !strings.Contains(joined, "market resource payload") || !strings.Contains(joined, "EXTERNAL_MCP_PROMPT") || !strings.Contains(joined, "external prompt payload") ||
		!strings.Contains(joined, "MCP_TOOL_CATALOG") || !strings.Contains(joined, toolRow.CanonicalName) {
		t.Fatalf("MCP context not loaded safely: %s", joined)
	}

	server.AddTool(&mcp.Tool{Name: "needs-input", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			InputRequests: mcp.InputRequestMap{"confirm": &mcp.ElicitParams{Message: "Confirm?"}},
			RequestState:  "fixture-confirmation",
		}, nil
	})
	_, err = gateway.CallTool(ctx, view.ID, "needs-input", json.RawMessage(`{}`))
	var inputRequired *InputRequiredError
	if !errors.As(err, &inputRequired) || inputRequired.RequestState != "fixture-confirmation" {
		t.Fatalf("MCP input-required result was not surfaced: %T %v", err, err)
	}
	if got := toolruntime.TypeOf(err); got != toolruntime.ErrorInputRequired {
		t.Fatalf("input-required error type = %q", got)
	}

	registerFixtureTool(server, true)
	if _, err := gateway.RefreshCatalog(ctx, view.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := store.ToolByID(ctx, toolRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != ToolNeedsReview || updated.Enabled != 0 {
		t.Fatalf("schema change did not revoke grant: %+v", updated)
	}
	updatedCatalog, err := store.Catalog(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, grant := range updatedCatalog.Permissions {
		if grant.CapabilityType == CapabilityTool && grant.CapabilityID == updated.ID && grant.Enabled != 0 {
			t.Fatalf("schema change retained stale Skill grant: %+v", grant)
		}
	}
}

func TestEndpointAndSecretSafety(t *testing.T) {
	if _, err := ValidateEndpoint("http://example.com/mcp", false); err == nil {
		t.Fatal("plain HTTP endpoint unexpectedly allowed")
	}
	if err := validateResolvedIP(net.ParseIP("127.0.0.1"), false); err == nil {
		t.Fatal("loopback endpoint unexpectedly allowed")
	}
	if err := validateResolvedIP(net.ParseIP("10.0.0.1"), false); err == nil {
		t.Fatal("private endpoint unexpectedly allowed")
	}
	t.Setenv("MCP_TEST_SECRET", "do-not-leak-this-token")
	secret, err := ResolveEnvironmentSecret(context.Background(), "env:MCP_TEST_SECRET")
	if err != nil || secret != "do-not-leak-this-token" {
		t.Fatalf("resolve secret: %v", err)
	}
	view := serverView(models.AgentMCPServer{SecretRef: "env:MCP_TEST_SECRET"})
	raw, _ := json.Marshal(view)
	if strings.Contains(string(raw), "MCP_TEST_SECRET") || strings.Contains(string(raw), secret) {
		t.Fatalf("server view leaked secret reference/value: %s", raw)
	}
	t.Setenv("MCP_MISSING_SECRET", "")
	_, err = ResolveEnvironmentSecret(context.Background(), "env:MCP_MISSING_SECRET")
	if err == nil || strings.Contains(err.Error(), "MCP_MISSING_SECRET") {
		t.Fatalf("secret resolution error leaked reference: %v", err)
	}

	oauthRaw, _ := json.Marshal(OAuthCredential{
		AccessToken: "expired", RefreshToken: "refresh", ClientID: "client",
		TokenURL: "http://127.0.0.1/token",
	})
	if _, err := oauthSource(context.Background(), string(oauthRaw), false); err == nil {
		t.Fatal("OAuth private/plain HTTP token endpoint unexpectedly allowed")
	}

	_, outputSchema, _ := schemaIdentity(map[string]any{"type": "object"}, nil)
	if outputSchema != "" {
		t.Fatalf("missing MCP output schema encoded as %q", outputSchema)
	}
}

func TestBearerCredentialIsSentAsAuthorizationHeader(t *testing.T) {
	const token = "managed-bearer-token"
	t.Setenv("MCP_BEARER_E2E", token)
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _, err := buildHTTPClient(context.Background(), models.AgentMCPServer{
		Endpoint: server.URL, AuthType: AuthBearer,
		SecretRef: "env:MCP_BEARER_E2E", AllowPrivate: 1,
	}, ResolveEnvironmentSecret)
	if err != nil {
		t.Fatal(err)
	}
	roundTripper, ok := client.Transport.(authRoundTripper)
	if !ok {
		t.Fatalf("unexpected MCP transport %T", client.Transport)
	}
	baseTransport, ok := roundTripper.base.(*http.Transport)
	if !ok || baseTransport.Proxy != nil {
		t.Fatalf("MCP transport unexpectedly uses an HTTP proxy: %T", roundTripper.base)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if got := <-received; got != "Bearer "+token {
		t.Fatalf("authorization header = %q", got)
	}
}
