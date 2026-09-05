package mcpclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/beego/beego/v2/client/orm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go_binance_futures/agent/observability"
	"go_binance_futures/agent/security"
	"go_binance_futures/models"
)

func canonicalSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			out.WriteRune(r)
			lastDash = false
		} else if !lastDash && out.Len() > 0 {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
func canonicalToolName(serverName, toolName string) (string, error) {
	serverPart, toolPart := canonicalSegment(serverName), canonicalSegment(toolName)
	if serverPart == "" || toolPart == "" {
		return "", fmt.Errorf("MCP server/tool name cannot be normalized")
	}
	return "mcp." + serverPart + "." + toolPart, nil
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func schemaIdentity(input, output any) (string, string, string) {
	inputRaw := marshalOptionalSchema(input)
	outputRaw := marshalOptionalSchema(output)
	identity, _ := json.Marshal(map[string]string{
		"input": inputRaw, "output": outputRaw,
	})
	return inputRaw, outputRaw, sha256Hex(identity)
}

func marshalOptionalSchema(value any) string {
	if value == nil {
		return ""
	}
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return ""
	}
	return string(raw)
}

func discoveryHash(discovery Discovery) string {
	type entry struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Raw  any    `json:"raw"`
	}
	entries := make([]entry, 0, len(discovery.Tools)+len(discovery.Resources)+len(discovery.Prompts))
	for _, item := range discovery.Tools {
		if item != nil {
			entries = append(entries, entry{Type: "tool", Name: item.Name, Raw: item})
		}
	}
	for _, item := range discovery.Resources {
		if item != nil {
			entries = append(entries, entry{Type: "resource", Name: item.URI, Raw: item})
		}
	}
	for _, item := range discovery.Prompts {
		if item != nil {
			entries = append(entries, entry{Type: "prompt", Name: item.Name, Raw: item})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type == entries[j].Type {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Type < entries[j].Type
	})
	raw, _ := json.Marshal(entries)
	return sha256Hex(raw)
}
func (g *Gateway) RefreshCatalog(ctx context.Context, serverID int64) (RefreshResult, error) {
	started := time.Now()
	server, err := g.Store.GetServer(ctx, serverID)
	if err != nil {
		return RefreshResult{}, err
	}
	if _, err := ValidateEndpoint(server.Endpoint, server.AllowPrivate == 1); err != nil {
		return RefreshResult{}, err
	}
	discovery, err := g.Discover(ctx, server)
	if err != nil {
		g.markServerError(serverID, err)
		return RefreshResult{}, err
	}
	catalogHash := discoveryHash(discovery)
	if err := g.syncCatalog(ctx, server, discovery, catalogHash); err != nil {
		g.markServerError(serverID, err)
		observability.RecordChange(ctx, observability.ChangeEvent{Category: "mcp", EntityType: "mcp_server", EntityID: server.ID, EntityName: server.Name, ChangeType: "catalog_refresh", BeforeHash: server.CatalogHash, Status: "error", Detail: map[string]any{"error": err.Error()}})
		return RefreshResult{}, err
	}
	changeType := "catalog_refresh"
	if server.CatalogHash != catalogHash {
		changeType = "catalog_changed"
	}
	observability.RecordChange(ctx, observability.ChangeEvent{Category: "mcp", EntityType: "mcp_server", EntityID: server.ID, EntityName: server.Name, ChangeType: changeType, FromVersion: server.ProtocolVersion, ToVersion: discovery.ProtocolVersion, BeforeHash: server.CatalogHash, AfterHash: catalogHash, Status: "success", Detail: map[string]any{"tools": len(discovery.Tools), "resources": len(discovery.Resources), "prompts": len(discovery.Prompts)}})
	return RefreshResult{
		ProtocolVersion: discovery.ProtocolVersion,
		ServerName:      discovery.ServerName, ServerVersion: discovery.ServerVersion,
		ToolCount: len(discovery.Tools), ResourceCount: len(discovery.Resources), PromptCount: len(discovery.Prompts),
		CatalogHash: catalogHash, DurationMs: time.Since(started).Milliseconds(),
	}, nil
}

func (g *Gateway) TestConnection(ctx context.Context, serverID int64) (RefreshResult, error) {
	started := time.Now()
	server, err := g.Store.GetServer(ctx, serverID)
	if err != nil {
		return RefreshResult{}, err
	}
	if _, err := ValidateEndpoint(server.Endpoint, server.AllowPrivate == 1); err != nil {
		return RefreshResult{}, err
	}
	release, err := g.acquire(ctx, serverID)
	if err != nil {
		return RefreshResult{}, err
	}
	defer release()
	session, err := g.connect(ctx, server)
	if err != nil {
		g.markServerError(serverID, err)
		return RefreshResult{}, err
	}
	defer session.Close()
	result := RefreshResult{DurationMs: time.Since(started).Milliseconds()}
	if init := session.InitializeResult(); init != nil {
		result.ProtocolVersion = init.ProtocolVersion
		if init.ServerInfo != nil {
			result.ServerName = init.ServerInfo.Name
			result.ServerVersion = init.ServerInfo.Version
		}
	}
	now := time.Now().UTC().UnixMilli()
	params := orm.Params{
		"protocol_version": result.ProtocolVersion,
		"server_name":      result.ServerName,
		"server_version":   result.ServerVersion,
		"status":           "healthy", "last_success_at": now, "last_error": nil, "updated_at": now,
	}
	if server.AuthType == AuthOAuth2 {
		params["oauth_status"] = "authorized"
	}
	_, err = g.Store.orm().QueryTable(new(models.AgentMCPServer)).Filter("id", serverID).Update(params)
	if err != nil {
		return RefreshResult{}, err
	}
	return result, nil
}

func (g *Gateway) markServerError(serverID int64, err error) {
	now := time.Now().UTC().UnixMilli()
	message := security.RedactText(err.Error())
	if len(message) > 2000 {
		message = message[:2000]
	}
	_, _ = g.Store.orm().QueryTable(new(models.AgentMCPServer)).Filter("id", serverID).Update(orm.Params{
		"status": "error", "last_error_at": now, "last_error": message, "updated_at": now,
	})
}
func (g *Gateway) syncCatalog(ctx context.Context, server models.AgentMCPServer, discovery Discovery, catalogHash string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := g.Store.orm()
	if err := syncTools(ctx, o, server, discovery.Tools, catalogHash); err != nil {
		return err
	}
	if err := syncResources(o, server, discovery.Resources, catalogHash); err != nil {
		return err
	}
	if err := syncPrompts(o, server, discovery.Prompts, catalogHash); err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	params := orm.Params{
		"protocol_version": discovery.ProtocolVersion,
		"server_name":      discovery.ServerName,
		"server_version":   discovery.ServerVersion,
		"status":           "healthy", "last_success_at": now, "last_error": nil,
		"catalog_hash": catalogHash, "updated_at": now,
	}
	if server.AuthType == AuthOAuth2 {
		params["oauth_status"] = "authorized"
	}
	_, err := o.QueryTable(new(models.AgentMCPServer)).Filter("id", server.ID).Update(params)
	return err
}

func syncTools(ctx context.Context, o orm.Ormer, server models.AgentMCPServer, discovered []*mcp.Tool, catalogHash string) error {
	var existing []models.AgentMCPTool
	if _, err := o.QueryTable(new(models.AgentMCPTool)).Filter("server_id", server.ID).All(&existing); err != nil {
		return err
	}
	byName := make(map[string]models.AgentMCPTool, len(existing))
	for _, row := range existing {
		byName[row.RemoteName] = row
	}
	seen := map[string]bool{}
	now := time.Now().UTC().UnixMilli()
	for _, remote := range discovered {
		if remote == nil || strings.TrimSpace(remote.Name) == "" {
			continue
		}
		canonical, err := canonicalToolName(server.Name, remote.Name)
		if err != nil {
			return err
		}
		inputSchema, outputSchema, schemaHash := schemaIdentity(remote.InputSchema, remote.OutputSchema)
		row, exists := byName[remote.Name]
		previousSchemaHash := row.SchemaHash
		schemaChanged := false
		if !exists {
			row = models.AgentMCPTool{ServerID: server.ID, RemoteName: remote.Name, CanonicalName: canonical, Status: ToolUnclassified, Risk: "read", Enabled: 0, TimeoutMs: defaultToolTimeoutMs, MaxResultBytes: 128 << 10, CreatedAt: now}
		} else if row.SchemaHash != "" && row.SchemaHash != schemaHash {
			row.Status, row.Enabled = ToolNeedsReview, 0
			schemaChanged = true
		}
		row.CanonicalName, row.Description = canonical, remote.Description
		row.InputSchema, row.OutputSchema, row.SchemaHash, row.CatalogHash = inputSchema, outputSchema, schemaHash, catalogHash
		if remote.Annotations != nil {
			row.ReadOnlyHint, row.IdempotentHint = remote.Annotations.ReadOnlyHint, remote.Annotations.IdempotentHint
		}
		row.UpdatedAt = now
		if row.ID == 0 {
			id, err := o.Insert(&row)
			if err != nil {
				return err
			}
			row.ID = id
		} else if _, err := o.Update(&row); err != nil {
			return err
		}
		if schemaChanged {
			if _, err := o.QueryTable(new(models.AgentMCPPermission)).Filter("server_id", server.ID).Filter("capability_type", CapabilityTool).Filter("capability_id", row.ID).Update(orm.Params{"enabled": 0, "updated_at": now}); err != nil {
				return err
			}
			observability.RecordChange(ctx, observability.ChangeEvent{Category: "mcp", EntityType: "mcp_tool", EntityID: row.ID, EntityName: row.CanonicalName, ChangeType: "schema_changed", BeforeHash: previousSchemaHash, AfterHash: schemaHash, Status: "review_required", Detail: map[string]any{"server_id": server.ID, "server_name": server.Name}})
		}
		seen[remote.Name] = true
	}
	for _, row := range existing {
		if seen[row.RemoteName] {
			continue
		}
		if row.Enabled != 0 || row.Status != ToolDisabled {
			_, _ = o.QueryTable(new(models.AgentMCPTool)).Filter("id", row.ID).Update(orm.Params{"enabled": 0, "status": ToolDisabled, "catalog_hash": catalogHash, "updated_at": now})
		}
		_, _ = o.QueryTable(new(models.AgentMCPPermission)).Filter("server_id", server.ID).Filter("capability_type", CapabilityTool).Filter("capability_id", row.ID).Update(orm.Params{"enabled": 0, "updated_at": now})
	}
	return nil
}

func syncResources(o orm.Ormer, server models.AgentMCPServer, discovered []*mcp.Resource, catalogHash string) error {
	var existing []models.AgentMCPResource
	if _, err := o.QueryTable(new(models.AgentMCPResource)).Filter("server_id", server.ID).All(&existing); err != nil {
		return err
	}
	byURI := map[string]models.AgentMCPResource{}
	for _, row := range existing {
		byURI[row.URI] = row
	}
	seen := map[string]bool{}
	now := time.Now().UTC().UnixMilli()
	for _, remote := range discovered {
		if remote == nil || strings.TrimSpace(remote.URI) == "" {
			continue
		}
		row := byURI[remote.URI]
		if row.ID == 0 {
			row = models.AgentMCPResource{ServerID: server.ID, URI: remote.URI, CreatedAt: now}
		}
		row.Name, row.Title, row.Description, row.MIMEType, row.Size = remote.Name, remote.Title, remote.Description, remote.MIMEType, remote.Size
		if remote.Annotations != nil {
			row.LastModified = remote.Annotations.LastModified
		}
		row.CatalogHash, row.UpdatedAt = catalogHash, now
		if row.ID == 0 {
			id, err := o.Insert(&row)
			if err != nil {
				return err
			}
			row.ID = id
		} else if _, err := o.Update(&row); err != nil {
			return err
		}
		seen[remote.URI] = true
	}
	for _, row := range existing {
		if seen[row.URI] {
			continue
		}
		_, _ = o.QueryTable(new(models.AgentMCPPermission)).Filter("server_id", server.ID).Filter("capability_type", CapabilityResource).Filter("capability_id", row.ID).Delete()
		_, _ = o.QueryTable(new(models.AgentMCPResource)).Filter("id", row.ID).Delete()
	}
	return nil
}
func syncPrompts(o orm.Ormer, server models.AgentMCPServer, discovered []*mcp.Prompt, catalogHash string) error {
	var existing []models.AgentMCPPrompt
	if _, err := o.QueryTable(new(models.AgentMCPPrompt)).Filter("server_id", server.ID).All(&existing); err != nil {
		return err
	}
	byName := map[string]models.AgentMCPPrompt{}
	for _, row := range existing {
		byName[row.RemoteName] = row
	}
	seen := map[string]bool{}
	now := time.Now().UTC().UnixMilli()
	for _, remote := range discovered {
		if remote == nil || strings.TrimSpace(remote.Name) == "" {
			continue
		}
		row := byName[remote.Name]
		if row.ID == 0 {
			row = models.AgentMCPPrompt{ServerID: server.ID, RemoteName: remote.Name, CreatedAt: now}
		}
		row.Title, row.Description, row.Arguments = remote.Title, remote.Description, marshalJSON(remote.Arguments)
		row.CatalogHash, row.UpdatedAt = catalogHash, now
		if row.ID == 0 {
			id, err := o.Insert(&row)
			if err != nil {
				return err
			}
			row.ID = id
		} else if _, err := o.Update(&row); err != nil {
			return err
		}
		seen[remote.Name] = true
	}
	for _, row := range existing {
		if seen[row.RemoteName] {
			continue
		}
		_, _ = o.QueryTable(new(models.AgentMCPPermission)).Filter("server_id", server.ID).Filter("capability_type", CapabilityPrompt).Filter("capability_id", row.ID).Delete()
		_, _ = o.QueryTable(new(models.AgentMCPPrompt)).Filter("id", row.ID).Delete()
	}
	return nil
}
