package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/agent/permission"
	"go_binance_futures/models"
)

type Store struct{ Alias string }

func (s Store) orm() orm.Ormer {
	if strings.TrimSpace(s.Alias) != "" {
		return orm.NewOrmUsingDB(s.Alias)
	}
	return orm.NewOrm()
}

func (s Store) ListServers(ctx context.Context) ([]ServerView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.AgentMCPServer
	if _, err := s.orm().QueryTable(new(models.AgentMCPServer)).OrderBy("id").All(&rows); err != nil {
		return nil, err
	}
	out := make([]ServerView, 0, len(rows))
	for _, row := range rows {
		out = append(out, serverView(row))
	}
	return out, nil
}

func (s Store) GetServer(ctx context.Context, id int64) (models.AgentMCPServer, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentMCPServer{}, err
	}
	var row models.AgentMCPServer
	if err := s.orm().QueryTable(new(models.AgentMCPServer)).Filter("id", id).One(&row); err != nil {
		return row, err
	}
	return row, nil
}

func (s Store) SaveServer(ctx context.Context, id int64, input ServerInput) (ServerView, error) {
	if err := ctx.Err(); err != nil {
		return ServerView{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	input.AuthType = strings.TrimSpace(input.AuthType)
	if input.AuthType == "" {
		input.AuthType = AuthNone
	}
	if input.Name == "" {
		return ServerView{}, fmt.Errorf("MCP server name is required")
	}
	if _, err := ValidateEndpoint(input.Endpoint, input.AllowPrivate == 1); err != nil {
		return ServerView{}, err
	}
	switch input.AuthType {
	case AuthNone, AuthBearer, AuthOAuth2, AuthCustomHeader:
	default:
		return ServerView{}, fmt.Errorf("unsupported MCP auth_type %q", input.AuthType)
	}
	if input.AuthType == AuthCustomHeader && !validHeaderName(strings.TrimSpace(input.CustomHeader)) {
		return ServerView{}, fmt.Errorf("invalid MCP custom header name")
	}
	now := time.Now().UTC().UnixMilli()
	row := models.AgentMCPServer{ID: id, Name: strings.TrimSpace(input.Name), Endpoint: strings.TrimSpace(input.Endpoint), Enabled: input.Enabled, AuthType: strings.TrimSpace(input.AuthType), SecretRef: strings.TrimSpace(input.SecretRef), CustomHeader: strings.TrimSpace(input.CustomHeader), AllowPrivate: input.AllowPrivate, UpdatedAt: now}
	o := s.orm()
	if id == 0 {
		row.CreatedAt = now
		row.Status = "disconnected"
		newID, err := o.Insert(&row)
		if err != nil {
			return ServerView{}, err
		}
		row.ID = newID
	} else {
		existing, err := s.GetServer(ctx, id)
		if err != nil {
			return ServerView{}, err
		}
		if row.SecretRef == "" {
			row.SecretRef = existing.SecretRef
		}
		row.CreatedAt, row.ProtocolVersion, row.ServerName, row.ServerVersion = existing.CreatedAt, existing.ProtocolVersion, existing.ServerName, existing.ServerVersion
		row.Status, row.LastSuccessAt, row.LastErrorAt, row.LastError, row.CatalogHash = existing.Status, existing.LastSuccessAt, existing.LastErrorAt, existing.LastError, existing.CatalogHash
		if _, err := o.Update(&row); err != nil {
			return ServerView{}, err
		}
	}
	return serverView(row), nil
}

func serverView(row models.AgentMCPServer) ServerView {
	hasSecret := strings.TrimSpace(row.SecretRef) != ""
	row.SecretRef = ""
	return ServerView{AgentMCPServer: row, HasSecret: hasSecret}
}

func (s Store) DeleteServer(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := s.orm()
	for _, model := range []any{new(models.AgentMCPPermission), new(models.AgentMCPTool), new(models.AgentMCPResource), new(models.AgentMCPPrompt)} {
		if _, err := o.QueryTable(model).Filter("server_id", id).Delete(); err != nil {
			return err
		}
	}
	if _, err := o.QueryTable(new(models.AgentMCPServer)).Filter("id", id).Delete(); err != nil {
		return err
	}
	return nil
}

func (s Store) Catalog(ctx context.Context, id int64) (Catalog, error) {
	server, err := s.GetServer(ctx, id)
	if err != nil {
		return Catalog{}, err
	}
	o := s.orm()
	var tools []models.AgentMCPTool
	_, err = o.QueryTable(new(models.AgentMCPTool)).Filter("server_id", id).OrderBy("canonical_name").All(&tools)
	if err != nil {
		return Catalog{}, err
	}
	var resources []models.AgentMCPResource
	_, err = o.QueryTable(new(models.AgentMCPResource)).Filter("server_id", id).OrderBy("name").All(&resources)
	if err != nil {
		return Catalog{}, err
	}
	var prompts []models.AgentMCPPrompt
	_, err = o.QueryTable(new(models.AgentMCPPrompt)).Filter("server_id", id).OrderBy("remote_name").All(&prompts)
	if err != nil {
		return Catalog{}, err
	}
	var permissions []models.AgentMCPPermission
	_, err = o.QueryTable(new(models.AgentMCPPermission)).Filter("server_id", id).OrderBy("skill_name", "capability_type", "capability_id").All(&permissions)
	if err != nil {
		return Catalog{}, err
	}
	return Catalog{Server: serverView(server), Tools: tools, Resources: resources, Prompts: prompts, Permissions: permissions}, nil
}

func (s Store) UpdateTool(ctx context.Context, id int64, input ToolUpdateInput) (models.AgentMCPTool, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentMCPTool{}, err
	}
	if input.Risk != permission.RiskRead && input.Risk != permission.RiskWrite && input.Risk != permission.RiskTrade {
		return models.AgentMCPTool{}, fmt.Errorf("invalid MCP tool risk %q", input.Risk)
	}
	if input.Enabled != 0 && input.Enabled != 1 {
		return models.AgentMCPTool{}, fmt.Errorf("MCP tool enabled must be 0 or 1")
	}
	if input.TimeoutMs < 0 || input.CacheTTLms < 0 || input.MaxResultBytes < 0 {
		return models.AgentMCPTool{}, fmt.Errorf("MCP tool runtime limits cannot be negative")
	}
	if input.TimeoutMs > 120000 {
		return models.AgentMCPTool{}, fmt.Errorf("MCP tool timeout_ms cannot exceed 120000")
	}
	if input.CacheTTLms > 86400000 {
		return models.AgentMCPTool{}, fmt.Errorf("MCP tool cache_ttl_ms cannot exceed 86400000")
	}
	if int64(input.MaxResultBytes) > defaultMaxResponseBytes {
		return models.AgentMCPTool{}, fmt.Errorf("MCP tool max_result_bytes cannot exceed %d", defaultMaxResponseBytes)
	}
	var row models.AgentMCPTool
	o := s.orm()
	if err := o.QueryTable(new(models.AgentMCPTool)).Filter("id", id).One(&row); err != nil {
		return row, err
	}
	if input.Enabled == 1 {
		var schema map[string]any
		if strings.TrimSpace(row.InputSchema) == "" || json.Unmarshal([]byte(row.InputSchema), &schema) != nil || schema["type"] != "object" {
			return row, fmt.Errorf("MCP tool %q cannot be enabled without a valid object input_schema", row.CanonicalName)
		}
	}
	row.Risk = string(input.Risk)
	row.Enabled = input.Enabled
	row.Idempotent = input.Idempotent
	row.TimeoutMs = input.TimeoutMs
	row.CacheTTLms = input.CacheTTLms
	row.MaxResultBytes = input.MaxResultBytes
	row.UpdatedAt = time.Now().UTC().UnixMilli()
	if row.Enabled == 1 {
		row.Status = ToolGranted
	} else {
		row.Status = ToolDisabled
	}
	if _, err := o.Update(&row); err != nil {
		return row, err
	}
	return row, nil
}

func (s Store) SavePermission(ctx context.Context, input PermissionInput) (models.AgentMCPPermission, error) {
	if err := ctx.Err(); err != nil {
		return models.AgentMCPPermission{}, err
	}
	input.SkillName = strings.TrimSpace(input.SkillName)
	input.CapabilityType = strings.TrimSpace(input.CapabilityType)
	if input.SkillName == "" || input.ServerID <= 0 || input.CapabilityID <= 0 {
		return models.AgentMCPPermission{}, fmt.Errorf("skill_name, server_id and capability_id are required")
	}
	if input.Enabled != 0 && input.Enabled != 1 || input.AutoLoad != 0 && input.AutoLoad != 1 {
		return models.AgentMCPPermission{}, fmt.Errorf("MCP permission flags must be 0 or 1")
	}
	if err := s.validateCapability(ctx, input.ServerID, input.CapabilityType, input.CapabilityID); err != nil {
		return models.AgentMCPPermission{}, err
	}
	if input.CapabilityType == CapabilityPrompt && input.AutoLoad == 1 {
		prompt, err := s.PromptByID(ctx, input.CapabilityID)
		if err != nil {
			return models.AgentMCPPermission{}, err
		}
		var arguments []struct {
			Required bool `json:"required"`
		}
		if strings.TrimSpace(prompt.Arguments) != "" {
			if err := json.Unmarshal([]byte(prompt.Arguments), &arguments); err != nil {
				return models.AgentMCPPermission{}, fmt.Errorf("decode MCP prompt arguments: %w", err)
			}
		}
		for _, argument := range arguments {
			if argument.Required {
				return models.AgentMCPPermission{}, fmt.Errorf("MCP prompt with required arguments cannot be auto-loaded")
			}
		}
	}
	now := time.Now().UTC().UnixMilli()
	o := s.orm()
	var row models.AgentMCPPermission
	err := o.QueryTable(new(models.AgentMCPPermission)).Filter("skill_name", input.SkillName).Filter("server_id", input.ServerID).Filter("capability_type", input.CapabilityType).Filter("capability_id", input.CapabilityID).One(&row)
	if err != nil && err != orm.ErrNoRows {
		return row, err
	}
	if err == orm.ErrNoRows {
		row = models.AgentMCPPermission{SkillName: input.SkillName, ServerID: input.ServerID, CapabilityType: input.CapabilityType, CapabilityID: input.CapabilityID, CreatedAt: now}
	}
	row.Enabled, row.AutoLoad, row.UpdatedAt = input.Enabled, input.AutoLoad, now
	if row.ID == 0 {
		id, err := o.Insert(&row)
		row.ID = id
		return row, err
	}
	_, err = o.Update(&row)
	return row, err
}

func (s Store) validateCapability(ctx context.Context, serverID int64, capabilityType string, capabilityID int64) error {
	if _, err := s.GetServer(ctx, serverID); err != nil {
		return fmt.Errorf("MCP server %d not found: %w", serverID, err)
	}
	var model any
	switch capabilityType {
	case CapabilityTool:
		model = new(models.AgentMCPTool)
	case CapabilityResource:
		model = new(models.AgentMCPResource)
	case CapabilityPrompt:
		model = new(models.AgentMCPPrompt)
	default:
		return fmt.Errorf("unsupported MCP capability_type %q", capabilityType)
	}
	count, err := s.orm().QueryTable(model).Filter("id", capabilityID).Filter("server_id", serverID).Count()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("MCP %s capability %d does not belong to server %d", capabilityType, capabilityID, serverID)
	}
	return nil
}

func (s Store) GrantedToolNames(ctx context.Context, skillName string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var perms []models.AgentMCPPermission
	if _, err := s.orm().QueryTable(new(models.AgentMCPPermission)).Filter("skill_name", strings.TrimSpace(skillName)).Filter("capability_type", CapabilityTool).Filter("enabled", 1).All(&perms); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(perms))
	for _, p := range perms {
		var tool models.AgentMCPTool
		if err := s.orm().QueryTable(new(models.AgentMCPTool)).Filter("id", p.CapabilityID).Filter("server_id", p.ServerID).Filter("enabled", 1).Filter("status", ToolGranted).One(&tool); err == nil {
			names = append(names, tool.CanonicalName)
		}
	}
	return names, nil
}

func (s Store) ActiveTools(ctx context.Context) ([]models.AgentMCPTool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.AgentMCPTool
	_, err := s.orm().QueryTable(new(models.AgentMCPTool)).Filter("enabled", 1).Filter("status", ToolGranted).OrderBy("canonical_name").All(&rows)
	return rows, err
}

func (s Store) ToolByID(ctx context.Context, id int64) (models.AgentMCPTool, error) {
	var row models.AgentMCPTool
	err := s.orm().QueryTable(new(models.AgentMCPTool)).Filter("id", id).One(&row)
	return row, err
}

func (s Store) ToolByCanonical(ctx context.Context, name string) (models.AgentMCPTool, error) {
	var row models.AgentMCPTool
	err := s.orm().QueryTable(new(models.AgentMCPTool)).Filter("canonical_name", strings.TrimSpace(name)).One(&row)
	return row, err
}

func (s Store) GrantedContextPermissions(ctx context.Context, skillName string) ([]models.AgentMCPPermission, error) {
	var rows []models.AgentMCPPermission
	_, err := s.orm().QueryTable(new(models.AgentMCPPermission)).Filter("skill_name", strings.TrimSpace(skillName)).Filter("enabled", 1).Filter("capability_type__in", CapabilityResource, CapabilityPrompt).All(&rows)
	return rows, err
}

func (s Store) ResourceByID(ctx context.Context, id int64) (models.AgentMCPResource, error) {
	var row models.AgentMCPResource
	err := s.orm().QueryTable(new(models.AgentMCPResource)).Filter("id", id).One(&row)
	return row, err
}
func (s Store) PromptByID(ctx context.Context, id int64) (models.AgentMCPPrompt, error) {
	var row models.AgentMCPPrompt
	err := s.orm().QueryTable(new(models.AgentMCPPrompt)).Filter("id", id).One(&row)
	return row, err
}
