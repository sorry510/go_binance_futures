package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
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

func setMCPConfigForTest(t *testing.T, key, value string) {
	t.Helper()
	previous, _ := config.String(key)
	if err := config.Set(key, value); err != nil {
		t.Fatalf("set config %s: %v", key, err)
	}
	t.Cleanup(func() {
		if err := config.Set(key, previous); err != nil {
			t.Fatalf("restore config %s: %v", key, err)
		}
	})
}

func setupMCPTestStore(t *testing.T) Store {
	t.Helper()
	setupMCPStoreOnce.Do(func() {
		_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
		orm.RegisterModel(new(models.AgentMCPServer), new(models.AgentMCPTool), new(models.AgentMCPResource), new(models.AgentMCPPrompt), new(models.AgentMCPPermission), new(models.AgentMCPSecret), new(models.AgentMCPOAuthState))
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
func TestMCPReadRetryIsFailSafe(t *testing.T) {
	attempts := 0
	value, err := withMCPReadRetry(context.Background(), func(context.Context) (string, error) {
		attempts++
		if attempts == 1 {
			return "", errors.New("upstream status 503 service unavailable")
		}
		return "ok", nil
	})
	if err != nil || value != "ok" || attempts != 2 {
		t.Fatalf("transient MCP read retry: value=%q attempts=%d err=%v", value, attempts, err)
	}

	attempts = 0
	_, err = withMCPReadRetry(context.Background(), func(context.Context) (string, error) {
		attempts++
		return "", errors.New("remote MCP tool returned invalid arguments")
	})
	if err == nil || attempts != 1 {
		t.Fatalf("application MCP error must not retry: attempts=%d err=%v", attempts, err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	attempts = 0
	_, err = withMCPReadRetry(canceled, func(context.Context) (string, error) { attempts++; return "", nil })
	if !errors.Is(err, context.Canceled) || attempts != 0 {
		t.Fatalf("canceled parent context must not call MCP: attempts=%d err=%v", attempts, err)
	}
}

func TestOAuthPublicBaseConfigAllowsLoopbackHTTPOnly(t *testing.T) {
	setMCPConfigForTest(t, oauthPublicBaseConfig, "http://127.0.0.1:3333")
	base, metadata, callback, err := oauthPublicURLs()
	if err != nil {
		t.Fatalf("loopback OAuth public base should be accepted: %v", err)
	}
	if base != "http://127.0.0.1:3333" || metadata != base+"/agents/mcp/oauth/client-metadata" || callback != base+"/agents/mcp/oauth/callback" {
		t.Fatalf("unexpected OAuth public URLs: base=%q metadata=%q callback=%q", base, metadata, callback)
	}

	if err := config.Set(oauthPublicBaseConfig, "http://203.0.113.10:3333"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := oauthPublicURLs(); err == nil {
		t.Fatal("non-loopback HTTP OAuth public base was accepted")
	}

	if err := config.Set(oauthPublicBaseConfig, "https://agent-host.example.com"); err != nil {
		t.Fatal(err)
	}
	if base, _, _, err := oauthPublicURLs(); err != nil || base != "https://agent-host.example.com" {
		t.Fatalf("HTTPS OAuth public base rejected: base=%q err=%v", base, err)
	}
}

func TestConnectionOnlyInitializesWithoutRefreshingCatalog(t *testing.T) {
	ctx := context.Background()
	store := setupMCPTestStore(t)
	_, httpServer := testMCPServer(t)
	view, err := store.SaveServer(ctx, 0, ServerInput{Name: "connection-only", Description: "fixture server description", Endpoint: httpServer.URL, Enabled: 1, AuthType: AuthNone, AllowPrivate: 1})
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
	if catalog.Server.Description != "fixture server description" {
		t.Fatalf("MCP server description was not persisted: %+v", catalog.Server)
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
	if catalog.Tools[0].TimeoutMs != defaultToolTimeoutMs {
		t.Fatalf("new MCP tool timeout = %d, want %d", catalog.Tools[0].TimeoutMs, defaultToolTimeoutMs)
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
	if _, err := oauthSource(context.Background(), string(oauthRaw), false, nil); err == nil {
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

func TestInteractiveOAuthPKCEPersistenceAndReconnect(t *testing.T) {
	ctx := context.Background()
	store := setupMCPTestStore(t)
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "oauth-fixture", Version: "1.0.0"}, nil)
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	var fixture *httptest.Server
	refreshCount := 0
	fixture = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			if r.Header.Get("Authorization") != "Bearer access-2" {
				w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+fixture.URL+`/.well-known/oauth-protected-resource/mcp"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			mcpHandler.ServeHTTP(w, r)
		case "/.well-known/oauth-protected-resource/mcp":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"resource": fixture.URL + "/mcp", "authorization_servers": []string{fixture.URL}})
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer": fixture.URL, "authorization_endpoint": fixture.URL + "/authorize", "token_endpoint": fixture.URL + "/token",
				"token_endpoint_auth_methods_supported": []string{"none"}, "response_types_supported": []string{"code"},
				"grant_types_supported": []string{"authorization_code", "refresh_token"}, "code_challenge_methods_supported": []string{"S256"},
				"client_id_metadata_document_supported": true,
			})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("token form: %v", err)
				w.WriteHeader(400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if r.Form.Get("grant_type") == "refresh_token" {
				refreshCount++
				if r.Form.Get("refresh_token") != "refresh-1" {
					t.Errorf("refresh token = %q", r.Form.Get("refresh_token"))
				}
				_, _ = w.Write([]byte(`{"access_token":"access-2","token_type":"Bearer","refresh_token":"refresh-2","expires_in":3600}`))
				return
			}
			if r.Form.Get("code") != "fixture-code" || r.Form.Get("code_verifier") == "" {
				t.Errorf("invalid code exchange: %#v", r.Form)
				w.WriteHeader(400)
				return
			}
			if r.Form.Get("resource") != fixture.URL+"/mcp" {
				t.Errorf("resource = %q", r.Form.Get("resource"))
			}
			if r.Form.Get("client_id") != fixture.URL+"/agents/mcp/oauth/client-metadata" {
				t.Errorf("client_id = %q", r.Form.Get("client_id"))
			}
			_, _ = w.Write([]byte(`{"access_token":"access-1","token_type":"Bearer","refresh_token":"refresh-1","expires_in":1}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fixture.Close()

	setMCPConfigForTest(t, oauthPublicBaseConfig, fixture.URL)
	setMCPConfigForTest(t, oauthEncryptionConfig, strings.Repeat("11", 32))
	view, err := store.SaveServer(ctx, 0, ServerInput{Name: "oauth", Endpoint: fixture.URL + "/mcp", Enabled: 1, AuthType: AuthOAuth2, AllowPrivate: 1})
	if err != nil {
		t.Fatal(err)
	}
	gateway := NewGateway(store)
	start, err := gateway.StartOAuth(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(start.AuthorizationURL, fixture.URL+"/authorize?") {
		t.Fatalf("authorization URL = %s", start.AuthorizationURL)
	}
	if !strings.Contains(start.AuthorizationURL, "code_challenge_method=S256") || !strings.Contains(start.AuthorizationURL, "resource=") {
		t.Fatalf("authorization URL missing PKCE/resource: %s", start.AuthorizationURL)
	}
	authURL, _ := url.Parse(start.AuthorizationURL)
	state := authURL.Query().Get("state")
	if state == "" {
		t.Fatal("missing OAuth state")
	}
	result, err := gateway.CompleteOAuth(ctx, state, "fixture-code", "", "", "")
	if err != nil || result.Status != "authorized" {
		t.Fatalf("complete OAuth: result=%+v err=%v", result, err)
	}
	if _, err := gateway.CompleteOAuth(ctx, state, "fixture-code", "", "", ""); err == nil {
		t.Fatal("OAuth state replay was accepted")
	}

	var secret models.AgentMCPSecret
	if err := store.orm().QueryTable(new(models.AgentMCPSecret)).Filter("server_id", view.ID).One(&secret); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(secret.Ciphertext, "access-1") || strings.Contains(secret.Ciphertext, "refresh-1") {
		t.Fatal("OAuth token stored in plaintext")
	}
	freshGateway := NewGateway(store)
	if _, err := freshGateway.TestConnection(ctx, view.ID); err != nil {
		t.Fatalf("reconnect with persisted OAuth credential: %v", err)
	}
	if refreshCount != 1 {
		t.Fatalf("refresh count = %d, want 1", refreshCount)
	}
	raw, err := ResolveManagedSecret(ctx, oauthSecretPrefix+strconv.FormatInt(view.ID, 10))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "access-2") || !strings.Contains(raw, "refresh-2") {
		t.Fatalf("rotated OAuth credential was not persisted: %s", raw)
	}
	serverRow, err := store.GetServer(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if serverRow.OAuthStatus != "authorized" || serverRow.SecretRef == "" {
		t.Fatalf("OAuth server state not authorized: %+v", serverRow)
	}
}
