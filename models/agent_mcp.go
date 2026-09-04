package models

type AgentMCPServer struct {
	ID              int64  `orm:"column(id);auto" json:"id"`
	Name            string `orm:"column(name);size(64);unique" json:"name"`
	Description     string `orm:"column(description);type(text);null" json:"description,omitempty"`
	Endpoint        string `orm:"column(endpoint);size(512)" json:"endpoint"`
	Enabled         int    `orm:"column(enabled);default(1);index" json:"enabled"`
	AuthType        string `orm:"column(auth_type);size(32);default(none)" json:"auth_type"`
	SecretRef       string `orm:"column(secret_ref);size(255)" json:"-"`
	CustomHeader    string `orm:"column(custom_header);size(128)" json:"custom_header,omitempty"`
	AllowPrivate    int    `orm:"column(allow_private);default(0)" json:"allow_private"`
	ProtocolVersion string `orm:"column(protocol_version);size(32)" json:"protocol_version,omitempty"`
	ServerName      string `orm:"column(server_name);size(128)" json:"server_name,omitempty"`
	ServerVersion   string `orm:"column(server_version);size(64)" json:"server_version,omitempty"`
	Status          string `orm:"column(status);size(32);default(disconnected);index" json:"status"`
	LastSuccessAt   int64  `orm:"column(last_success_at);index" json:"last_success_at,omitempty"`
	LastErrorAt     int64  `orm:"column(last_error_at);index" json:"last_error_at,omitempty"`
	LastError       string `orm:"column(last_error);type(text);null" json:"last_error,omitempty"`
	CatalogHash     string `orm:"column(catalog_hash);size(64)" json:"catalog_hash,omitempty"`
	OAuthStatus     string `orm:"column(oauth_status);size(32)" json:"oauth_status,omitempty"`
	OAuthIssuer     string `orm:"column(oauth_issuer);size(512)" json:"oauth_issuer,omitempty"`
	OAuthExpiresAt  int64  `orm:"column(oauth_expires_at);index" json:"oauth_expires_at,omitempty"`
	CreatedAt       int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt       int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPServer) TableName() string { return "agent_mcp_servers" }

type AgentMCPTool struct {
	ID             int64  `orm:"column(id);auto" json:"id"`
	ServerID       int64  `orm:"column(server_id);index" json:"server_id"`
	RemoteName     string `orm:"column(remote_name);size(192);index" json:"remote_name"`
	CanonicalName  string `orm:"column(canonical_name);size(255);unique" json:"canonical_name"`
	Description    string `orm:"column(description);type(text);null" json:"description,omitempty"`
	InputSchema    string `orm:"column(input_schema);type(text);null" json:"input_schema,omitempty"`
	OutputSchema   string `orm:"column(output_schema);type(text);null" json:"output_schema,omitempty"`
	SchemaHash     string `orm:"column(schema_hash);size(64)" json:"schema_hash"`
	CatalogHash    string `orm:"column(catalog_hash);size(64)" json:"catalog_hash"`
	Status         string `orm:"column(status);size(32);default(unclassified);index" json:"status"`
	Risk           string `orm:"column(risk);size(16);default(read)" json:"risk"`
	Enabled        int    `orm:"column(enabled);default(0);index" json:"enabled"`
	ReadOnlyHint   bool   `orm:"column(read_only_hint);default(false)" json:"read_only_hint"`
	IdempotentHint bool   `orm:"column(idempotent_hint);default(false)" json:"idempotent_hint"`
	Idempotent     bool   `orm:"column(idempotent);default(false)" json:"idempotent"`
	TimeoutMs      int64  `orm:"column(timeout_ms);default(10000)" json:"timeout_ms"`
	CacheTTLms     int64  `orm:"column(cache_ttl_ms);default(0)" json:"cache_ttl_ms"`
	MaxResultBytes int    `orm:"column(max_result_bytes);default(131072)" json:"max_result_bytes"`
	CreatedAt      int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt      int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPTool) TableName() string { return "agent_mcp_tools" }

type AgentMCPResource struct {
	ID           int64  `orm:"column(id);auto" json:"id"`
	ServerID     int64  `orm:"column(server_id);index" json:"server_id"`
	URI          string `orm:"column(uri);type(text)" json:"uri"`
	Name         string `orm:"column(name);size(192)" json:"name"`
	Title        string `orm:"column(title);size(255)" json:"title,omitempty"`
	Description  string `orm:"column(description);type(text);null" json:"description,omitempty"`
	MIMEType     string `orm:"column(mime_type);size(128)" json:"mime_type,omitempty"`
	Size         int64  `orm:"column(size)" json:"size,omitempty"`
	LastModified string `orm:"column(last_modified);size(64)" json:"last_modified,omitempty"`
	CatalogHash  string `orm:"column(catalog_hash);size(64)" json:"catalog_hash"`
	CreatedAt    int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt    int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPResource) TableName() string { return "agent_mcp_resources" }

type AgentMCPPrompt struct {
	ID          int64  `orm:"column(id);auto" json:"id"`
	ServerID    int64  `orm:"column(server_id);index" json:"server_id"`
	RemoteName  string `orm:"column(remote_name);size(192);index" json:"remote_name"`
	Title       string `orm:"column(title);size(255)" json:"title,omitempty"`
	Description string `orm:"column(description);type(text);null" json:"description,omitempty"`
	Arguments   string `orm:"column(arguments_json);type(text);null" json:"arguments_json,omitempty"`
	CatalogHash string `orm:"column(catalog_hash);size(64)" json:"catalog_hash"`
	CreatedAt   int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt   int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPPrompt) TableName() string { return "agent_mcp_prompts" }

type AgentMCPPermission struct {
	ID             int64  `orm:"column(id);auto" json:"id"`
	SkillName      string `orm:"column(skill_name);size(64);index" json:"skill_name"`
	ServerID       int64  `orm:"column(server_id);index" json:"server_id"`
	CapabilityType string `orm:"column(capability_type);size(16);index" json:"capability_type"`
	CapabilityID   int64  `orm:"column(capability_id);index" json:"capability_id"`
	Enabled        int    `orm:"column(enabled);default(0);index" json:"enabled"`
	AutoLoad       int    `orm:"column(auto_load);default(0)" json:"auto_load"`
	CreatedAt      int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt      int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPPermission) TableName() string { return "agent_mcp_permissions" }

type AgentMCPSecret struct {
	ID         int64  `orm:"column(id);auto" json:"id"`
	ServerID   int64  `orm:"column(server_id);unique" json:"server_id"`
	Kind       string `orm:"column(kind);size(32)" json:"kind"`
	Ciphertext string `orm:"column(ciphertext);type(text)" json:"-"`
	CreatedAt  int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt  int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentMCPSecret) TableName() string { return "agent_mcp_secrets" }

type AgentMCPOAuthState struct {
	ID         int64  `orm:"column(id);auto" json:"id"`
	ServerID   int64  `orm:"column(server_id);index" json:"server_id"`
	StateHash  string `orm:"column(state_hash);size(64);unique" json:"-"`
	Ciphertext string `orm:"column(ciphertext);type(text)" json:"-"`
	ExpiresAt  int64  `orm:"column(expires_at);index" json:"expires_at"`
	CreatedAt  int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentMCPOAuthState) TableName() string { return "agent_mcp_oauth_states" }
