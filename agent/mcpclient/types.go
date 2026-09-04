package mcpclient

import (
	"encoding/json"
	"time"

	"go_binance_futures/agent/permission"
	"go_binance_futures/models"
)

const (
	AuthNone         = "none"
	AuthBearer       = "bearer"
	AuthOAuth2       = "oauth2"
	AuthCustomHeader = "custom_header"

	CapabilityTool     = "tool"
	CapabilityResource = "resource"
	CapabilityPrompt   = "prompt"

	ToolUnclassified = "unclassified"
	ToolDisabled     = "disabled"
	ToolGranted      = "granted"
	ToolNeedsReview  = "needs_review"

	defaultToolTimeoutMs int64 = 60000
	maxToolTimeoutMs     int64 = 120000
)

type ServerInput struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Endpoint     string `json:"endpoint"`
	Enabled      int    `json:"enabled"`
	AuthType     string `json:"auth_type"`
	SecretRef    string `json:"secret_ref,omitempty"`
	CustomHeader string `json:"custom_header,omitempty"`
	AllowPrivate int    `json:"allow_private"`
}

type ServerView struct {
	models.AgentMCPServer
	HasSecret bool `json:"has_secret"`
}
type ToolUpdateInput struct {
	Risk           permission.RiskLevel `json:"risk"`
	Enabled        int                  `json:"enabled"`
	Idempotent     bool                 `json:"idempotent"`
	TimeoutMs      int64                `json:"timeout_ms"`
	CacheTTLms     int64                `json:"cache_ttl_ms"`
	MaxResultBytes int                  `json:"max_result_bytes"`
}

type PermissionInput struct {
	ServerID       int64  `json:"server_id"`
	SkillName      string `json:"skill_name"`
	CapabilityType string `json:"capability_type"`
	CapabilityID   int64  `json:"capability_id"`
	Enabled        int    `json:"enabled"`
	AutoLoad       int    `json:"auto_load"`
}

type Catalog struct {
	Server      ServerView                  `json:"server"`
	Tools       []models.AgentMCPTool       `json:"tools"`
	Resources   []models.AgentMCPResource   `json:"resources"`
	Prompts     []models.AgentMCPPrompt     `json:"prompts"`
	Permissions []models.AgentMCPPermission `json:"permissions"`
}

type RefreshResult struct {
	ProtocolVersion string `json:"protocol_version"`
	ServerName      string `json:"server_name"`
	ServerVersion   string `json:"server_version"`
	ToolCount       int    `json:"tool_count"`
	ResourceCount   int    `json:"resource_count"`
	PromptCount     int    `json:"prompt_count"`
	CatalogHash     string `json:"catalog_hash"`
	DurationMs      int64  `json:"duration_ms"`
}

type OAuthCredential struct {
	AccessToken     string    `json:"access_token"`
	TokenType       string    `json:"token_type,omitempty"`
	RefreshToken    string    `json:"refresh_token,omitempty"`
	Expiry          time.Time `json:"expiry,omitempty"`
	TokenURL        string    `json:"token_url,omitempty"`
	ClientID        string    `json:"client_id,omitempty"`
	ClientSecret    string    `json:"client_secret,omitempty"`
	TokenAuthMethod string    `json:"token_auth_method,omitempty"`
	Scopes          []string  `json:"scopes,omitempty"`
}

func marshalJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
