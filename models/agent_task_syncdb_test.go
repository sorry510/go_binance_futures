package models

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func testStringPointer(value string) *string { return &value }

func testStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

const legacyAgentTaskDDL = `CREATE TABLE agent_tasks (
	id varchar(64) primary key,
	skill varchar(64),
	conversation_id varchar(64),
	status varchar(32),
	stage varchar(64),
	progress integer,
	input_json text,
	result_json text,
	error text,
	round integer,
	max_rounds integer,
	provider varchar(64),
	model varchar(128),
	input_tokens integer,
	output_tokens integer,
	total_tokens integer,
	created_at integer,
	started_at integer,
	updated_at integer,
	completed_at integer
)`

const legacyMCPServerDDL = `CREATE TABLE agent_mcp_servers (
	id integer primary key autoincrement,
	name varchar(64),
	endpoint varchar(512),
	enabled integer,
	auth_type varchar(32),
	secret_ref varchar(255),
	custom_header varchar(128),
	allow_private integer,
	protocol_version varchar(32),
	server_name varchar(128),
	server_version varchar(64),
	status varchar(32),
	last_success_at integer,
	last_error_at integer,
	last_error text,
	catalog_hash varchar(64),
	created_at integer,
	updated_at integer
)`

const legacyAgentSkillDDL = `CREATE TABLE agent_skills (
	id integer primary key autoincrement,
	name varchar(96) unique,
	display_name varchar(128),
	description text,
	enabled integer,
	created_at integer,
	updated_at integer,
	deleted integer
)`

const legacyAgentConversationDDL = `CREATE TABLE agent_conversations (
	id varchar(64) primary key,
	skill varchar(64),
	status varchar(32),
	created_at integer,
	updated_at integer,
	closed_at integer
)`

const legacyAgentConversationMessageDDL = `CREATE TABLE agent_conversation_messages (
	id integer primary key autoincrement,
	conversation_id varchar(64),
	task_id varchar(64),
	sequence integer,
	role varchar(32),
	content text,
	created_at integer
)`

const legacyAgentTaskEventDDL = `CREATE TABLE agent_task_events (
	id integer primary key autoincrement,
	task_id varchar(64),
	sequence integer,
	stage varchar(64),
	progress integer,
	round integer,
	message text,
	skill varchar(64),
	tool varchar(128),
	status varchar(32),
	duration_ms integer,
	event_time integer
)`

func TestAgentTaskSyncdbUpgradesExistingSQLiteRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	if err := orm.RegisterDataBase("default", "sqlite3", path); err != nil {
		t.Fatal(err)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyAgentTaskDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyAgentTaskEventDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyMCPServerDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyAgentSkillDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyAgentConversationDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyAgentConversationMessageDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_conversations (id, skill, status, created_at, updated_at, closed_at) VALUES ('legacy-conv', 'strategy_builder', 'active', 1700000000000, 1700000000000, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_conversation_messages (conversation_id, task_id, sequence, role, content, created_at) VALUES ('legacy-conv', 'legacy-task', 1, 'user', 'legacy message', 1700000000000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_skills (name, display_name, description, enabled, created_at, updated_at, deleted) VALUES ('symbol_analysis', 'Symbol', 'legacy native', 1, 1700000000000, 1700000000000, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_mcp_servers
		(name, endpoint, enabled, auth_type, allow_private, status, created_at, updated_at)
		VALUES ('legacy-mcp', 'https://example.com/mcp', 1, 'none', 0, 'disconnected', 1700000000000, 1700000000000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_tasks
		(id, skill, status, stage, progress, round, max_rounds, created_at, updated_at)
		VALUES ('legacy-task', 'symbol_analysis', 'succeeded', 'completed', 100, 2, 8, 1700000000000, 1700000000000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_task_events
		(task_id, sequence, stage, progress, round, message, status, event_time)
		VALUES ('legacy-task', 1, 'completed', 100, 2, 'done', 'success', 1700000000000)`); err != nil {
		t.Fatal(err)
	}

	orm.RegisterModel(new(AgentTask), new(AgentTaskEvent), new(AgentConversation), new(AgentConversationMessage), new(AgentSkill), new(AgentSkillVersion), new(AgentSkillPermission), new(AgentMCPServer), new(AgentMCPTool), new(AgentMCPResource), new(AgentMCPPrompt), new(AgentMCPPermission), new(AgentMCPSecret), new(AgentMCPOAuthState), new(AgentAlertPipelineTrace), new(AgentMemory), new(LLMConfig), new(LLMRouterSetting))
	if err := orm.RunSyncdb("default", false, false); err != nil {
		t.Fatalf("RunSyncdb must upgrade an existing database with rows: %v", err)
	}

	requireAgentColumns(t, db, "agent_tasks", []string{
		"runtime_version", "skill_version", "prompt_version", "prompt_hash",
		"model_config_id", "final_model_config_id", "route_candidates_json", "route_reason", "route_fallback_json", "input_contract_version", "output_contract_version",
		"skill_source", "skill_source_version", "execution_mode", "plan_json",
		"steps_json", "checkpoint_json", "resume_count", "tool_catalog_hash", "skill_package_hash",
	})
	requireAgentColumns(t, db, "agent_task_events", []string{
		"step_id", "step_type", "error_type", "checkpoint",
	})
	for _, table := range []string{"agent_skill_versions", "agent_skill_permissions", "agent_mcp_servers", "agent_mcp_tools", "agent_mcp_resources", "agent_mcp_prompts", "agent_mcp_permissions", "agent_mcp_secrets", "agent_mcp_oauth_states", "agent_alert_pipeline_traces", "agent_memories", "llm_configs", "llm_router_settings"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("RunSyncdb did not create V2-5 table %s", table)
		}
	}

	requireAgentColumns(t, db, "agent_conversations", []string{"title"})
	requireAgentColumns(t, db, "agent_conversation_messages", []string{"skill"})
	var legacyConversationSkill, legacyConversationTitle string
	if err := db.QueryRow(`SELECT skill, COALESCE(title, '') FROM agent_conversations WHERE id='legacy-conv'`).Scan(&legacyConversationSkill, &legacyConversationTitle); err != nil {
		t.Fatalf("legacy conversation became unreadable after V2-7 upgrade: %v", err)
	}
	if legacyConversationSkill != "strategy_builder" || legacyConversationTitle != "" {
		t.Fatalf("legacy conversation changed unexpectedly: skill=%q title=%q", legacyConversationSkill, legacyConversationTitle)
	}

	requireAgentColumns(t, db, "agent_skills", []string{"type", "active_version_id"})
	requireAgentColumns(t, db, "agent_skill_versions", []string{"skill_id", "package_hash", "version", "metadata_json", "requested_tools_json", "validation_status", "source", "package_path"})
	requireAgentColumns(t, db, "agent_skill_permissions", []string{"skill_id", "version_id", "requested_name", "resolved_name", "risk", "status", "granted"})
	requireAgentColumns(t, db, "agent_mcp_tools", []string{
		"remote_name", "canonical_name", "input_schema", "output_schema", "schema_hash",
		"risk", "enabled", "idempotent_hint", "idempotent", "timeout_ms", "cache_ttl_ms", "max_result_bytes",
	})
	requireAgentColumns(t, db, "agent_mcp_servers", []string{
		"oauth_status", "oauth_issuer", "oauth_expires_at", "description",
	})
	var legacySkillType string
	var legacyActiveVersion int64
	if err := db.QueryRow(`SELECT type, active_version_id FROM agent_skills WHERE name='symbol_analysis'`).Scan(&legacySkillType, &legacyActiveVersion); err != nil {
		t.Fatalf("legacy AgentSkill row became unreadable after V2-6 upgrade: %v", err)
	}
	if legacySkillType != "native" || legacyActiveVersion != 0 {
		t.Fatalf("legacy AgentSkill defaults changed unexpectedly: type=%q active=%d", legacySkillType, legacyActiveVersion)
	}

	var legacyMCPName string
	if err := db.QueryRow(`SELECT name FROM agent_mcp_servers WHERE name='legacy-mcp'`).Scan(&legacyMCPName); err != nil {
		t.Fatalf("legacy MCP server row became unreadable after OAuth upgrade: %v", err)
	}
	if legacyMCPName != "legacy-mcp" {
		t.Fatalf("legacy MCP server row changed during sync: %q", legacyMCPName)
	}

	requireAgentColumns(t, db, "llm_configs", []string{"router_candidate", "structured_output", "native_tool_calling", "reasoning", "long_context", "json_reliability", "max_context_tokens", "cost_class", "latency_class"})
	for _, column := range []string{"plan_json", "steps_json", "checkpoint_json", "route_candidates_json", "route_reason", "route_fallback_json"} {
		if notNull := sqliteColumnNotNull(t, db, "agent_tasks", column); notNull {
			t.Fatalf("%s must stay nullable for additive SQLite upgrades", column)
		}
	}

	var status string
	var resumeCount int
	if err := db.QueryRow(`SELECT status, resume_count FROM agent_tasks WHERE id='legacy-task'`).Scan(&status, &resumeCount); err != nil {
		t.Fatalf("legacy task row became unreadable: %v", err)
	}
	if status != "succeeded" || resumeCount != 0 {
		t.Fatalf("legacy task changed during sync: status=%q resume_count=%d", status, resumeCount)
	}
	var eventCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_task_events WHERE task_id='legacy-task'`).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 {
		t.Fatalf("legacy events lost during sync: %d", eventCount)
	}

	o := orm.NewOrmUsingDB("default")
	legacy := AgentTask{ID: "legacy-task"}
	if err := o.Read(&legacy); err != nil {
		t.Fatalf("beego ORM must read NULL V2 text fields as empty strings: %v", err)
	}
	if testStringValue(legacy.PlanJSON) != "" || testStringValue(legacy.StepsJSON) != "" || testStringValue(legacy.CheckpointJSON) != "" {
		t.Fatalf("unexpected legacy V2 state: plan=%q steps=%q checkpoint=%q", testStringValue(legacy.PlanJSON), testStringValue(legacy.StepsJSON), testStringValue(legacy.CheckpointJSON))
	}

	fresh := AgentTask{
		ID: "fresh-task", Skill: "symbol_analysis", Status: "succeeded", Stage: "completed",
		ExecutionMode: "react", PlanJSON: testStringPointer(`{"summary":"ok"}`), StepsJSON: testStringPointer(`[{"step_id":"step-001"}]`),
		CheckpointJSON: testStringPointer(`{"safe":true}`), ResumeCount: 1, RuntimeVersion: "2.0.0",
		SkillVersion: "1.0.0", PromptVersion: "1.0.0", PromptHash: strings.Repeat("a", 64),
		ModelConfigID: 7, InputContractVersion: "symbol_analysis_input_v1",
		OutputContractVersion: "trading_plan_v1", SkillSource: "native", SkillSourceVersion: "v1",
		ToolCatalogHash: strings.Repeat("b", 64), SkillPackageHash: strings.Repeat("c", 64),
		CreatedAt: 1700000000001, UpdatedAt: 1700000000001,
	}
	if _, err := o.Insert(&fresh); err != nil {
		t.Fatalf("insert current V2 task after upgrade: %v", err)
	}
	got := AgentTask{ID: fresh.ID}
	if err := o.Read(&got); err != nil {
		t.Fatalf("read current V2 task after upgrade: %v", err)
	}
	if got.ExecutionMode != "react" || got.ResumeCount != 1 || got.ToolCatalogHash != strings.Repeat("b", 64) || got.SkillPackageHash != strings.Repeat("c", 64) || !strings.Contains(testStringValue(got.CheckpointJSON), "safe") {
		t.Fatalf("V2 fields did not round-trip after upgrade: %+v", got)
	}
}

func requireAgentColumns(t *testing.T, db *sql.DB, table string, required []string) {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		found[name] = true
	}
	for _, name := range required {
		if !found[name] {
			t.Fatalf("RunSyncdb did not add %s.%s", table, name)
		}
	}
}

func sqliteColumnNotNull(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return notNull != 0
		}
	}
	t.Fatalf("column %s.%s not found", table, column)
	return false
}
