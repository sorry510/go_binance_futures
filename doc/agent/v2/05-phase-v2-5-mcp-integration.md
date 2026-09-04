# V2-5：第三方 HTTP MCP Integration ✅

## 1. 阶段状态

V2-5 已完成。范围限定为 **Agent Host 主动连接第三方 Streamable HTTP MCP Server**；本项目已有的 `/mcp` Server 不在本阶段改造范围内，Imported Skill 仍留给 V2-6。

运行时身份：

```text
runtime_version = 2.4.0
checkpoint_state = runtime_state_v5
MCP Go SDK = v1.7.0
Go baseline = 1.25.x
```

Go 1.25 是官方 MCP Go SDK v1.7.0 的最低工具链要求；Release workflow 已同步到 Go 1.25。

## 2. 已实现架构

```text
AI Skill
  -> Runtime dynamic allowlist / Context resource provider
  -> V2-3 Tool Runtime
  -> MCP Remote Tool Adapter
  -> MCP Gateway
  -> Streamable HTTP MCP Server
```

MCP Tool 不存在独立旁路，仍执行统一的 Schema、Risk、Permission、Budget、Timeout、Cache、Evidence 和 Trace。

Resource / Prompt 不注册为 Tool，而是进入 V2-2 Context Engine。远端 Prompt 固定带 `EXTERNAL_MCP_PROMPT` 边界声明，只能作为不可信外部 Context，不能覆盖 System Policy、Risk、Permission 或 Budget。

## 3. Registry 与治理模型

新增 ORM 模型：

- `agent_mcp_servers`
- `agent_mcp_tools`
- `agent_mcp_resources`
- `agent_mcp_prompts`
- `agent_mcp_permissions`

没有新增 `command/sql/version/2.sql`；仍由现有 `orm.RunSyncdb` 管理结构。SQLite legacy upgrade regression 已验证旧 Agent Task 数据存在时可以安全创建 V2-5 新表。

新发现 Tool 默认：

```text
status = unclassified
enabled = 0
```

Tool 必须经过两层本地治理后才可调用：

```text
Tool 本地分类/启用
+ Skill -> MCP Tool grant
+ Runtime global Risk Policy
```

远端 `readOnlyHint` / `idempotentHint` 只保存为参考信息。缓存、并行和 safe checkpoint 使用管理员本地确认的 `risk` 与 `idempotent`，不信任远端 annotation。

当 Input/Output Schema hash 变化时：

```text
Tool -> needs_review
Tool -> disabled
已有 Skill grant -> disabled
```

管理员必须重新审核 Tool 并重新授权 Skill，Catalog refresh 不会静默扩大权限。

## 4. MCP Gateway

Gateway 基于官方 `github.com/modelcontextprotocol/go-sdk/mcp`：

- `StreamableClientTransport`
- protocol negotiation
- `tools/list` + pagination
- `resources/list` + pagination
- `prompts/list` + pagination
- `tools/call`
- `resources/read`
- `prompts/get`

支持的鉴权模式：

- `none`
- `bearer`
- `custom_header`
- `oauth2` managed credential

OAuth2 支持已受管的 access token，以及具备 refresh token / token URL / client identity 时的 refresh。Agent Runtime **不执行浏览器交互式 OAuth 授权流程**。

Secret 只保存 `secret_ref`；当前默认 resolver 支持 `env:VARIABLE_NAME`。API/UI 只返回 `has_secret`，不会返回引用名或凭据值；Secret 解析错误也不会泄漏环境变量名称。

## 5. 网络安全边界

第三方 MCP outbound 具备：

- 默认要求 HTTPS
- `allow_private=1` 才允许 HTTP / localhost / 私网
- endpoint 不允许 URL userinfo / fragment
- 每次 Dial 重新 DNS resolve 并检查实际 IP
- 阻断 loopback/private/unspecified/link-local/multicast（除非显式 allow_private）
- redirect 仅允许相同 scheme/host，且限制跳转次数
- connect/header/overall timeout
- response size 上限
- 每 Server 最大 4 个并发请求
- 连续失败 circuit breaker
- **MCP transport 不继承 HTTP_PROXY / HTTPS_PROXY**，避免代理绕过目标 IP SSRF 校验
- OAuth refresh `token_url` 使用同一套 endpoint、DNS/IP 和 redirect 限制

## 6. Schema 与 Tool Runtime

Tool Runtime JSON Schema validator 使用 Draft 2020-12。

MCP Tool Descriptor/Trace 包含：

```text
canonical_name
source_type=mcp
risk
idempotent
timeout/cache/max_result_bytes
provider_ref
protocol_version
catalog_hash
schema_hash
```

远端没有 `outputSchema` 时按“无 Output Schema”处理，不再错误保存为 JSON `null`。

Tool result 继续转换为 V2-2 Evidence / ContextBlock，并接受 Result size、freshness 和 sensitive redaction 规则。

## 7. Resource / Prompt Context

Skill 可以分别授权 MCP Resource / Prompt，并配置：

```text
auto_load = 1 -> activation
auto_load = 0 -> on_demand
```

Resource 的远端 `lastModified` 会记录到 ContextBlock `as_of`，但默认保持 `freshness=unknown`；不会仅凭远端声明擅自判定为 fresh。

带必填参数的 Prompt 不允许配置 auto-load，避免 BuildContext 阶段无参数调用。

## 8. Multi-Round-Trip / Approval 边界

官方 SDK 的自动 Multi-Round-Trip middleware 已显式关闭。Server 返回 `input_required` 时，Gateway 会保留 `request_state` / input requests，并通过统一 Tool error taxonomy 返回：

```text
error_type = input_required
```

V2-5 **不会自动接受远端 elicitation/confirmation，也不会让 MCP Server 自动获得用户权限**。当前 Agent 可以根据该结构化错误降级或结束。持久化 `waiting_input`、Web 人工确认和确认后续跑属于后续统一 Approval / Execution 能力，不在本阶段伪装为已完成。

## 9. Runtime 动态接入

Runtime 新增动态 provider：

- MCP Tool allowlist provider
- MCP Context resource provider

Native `Skill.Tools()` 保持静态契约；数据库 MCP grant 在 Task 启动时动态追加。实际 Native + MCP Tool Catalog 会进入 `tool_catalog_hash`，provider 查询失败时 Task 启动直接失败，不会使用不完整目录继续运行。

Resume 继续遵守冻结身份校验；Runtime 2.3.x / `runtime_state_v4` checkpoint 不跨版本恢复。

## 10. Web 管理

新增 `AI -> MCP`：

- MCP Server CRUD
- Test Connection
- Refresh Catalog
- Tools / Resources / Prompts 查看
- Tool Risk / Enable / Idempotent / Timeout / Cache / Max Result Bytes
- Skill -> Tool/Resource/Prompt 授权
- Resource/Prompt auto-load

Task Center Tool Trace 新增展示：

- MCP Provider Ref
- Protocol Version
- Catalog Hash
- Schema Hash

Secret 输入只接受受管引用，编辑时留空保留已有引用。

## 11. HTTP API

```text
GET    /agents/mcp/servers
POST   /agents/mcp/servers
PUT    /agents/mcp/servers/:id
DELETE /agents/mcp/servers/:id
GET    /agents/mcp/servers/:id/catalog
POST   /agents/mcp/servers/:id/test
POST   /agents/mcp/servers/:id/refresh
PUT    /agents/mcp/tools/:id
POST   /agents/mcp/permissions
```

Server/Tool/Catalog 变化会热同步到默认 Tool Registry，不要求重启后端。Skill grant 在每个新 Task 启动时动态读取。

## 12. 自动回归覆盖

真实 E2E 使用官方 MCP Go SDK Server + `httptest` Streamable HTTP Handler，覆盖：

- protocol discovery
- Tool/Resource/Prompt catalog
- 新 Tool 默认禁用
- Tool Runtime 实际远端调用
- Resource/Prompt Context
- MCP Trace identity
- Schema change 自动撤权
- Skill grant 自动撤权
- `input_required` 原样上浮
- Bearer Authorization Header 实际发送
- Secret API/错误信息不泄漏
- private/loopback endpoint guard
- OAuth token endpoint guard
- no-output-schema Tool compatibility

V2-4 Eval/Replay 全量回归继续通过。

## 13. Phase Gate

- [x] 至少一个 Streamable HTTP MCP Server 可通过真实协议 E2E 发现 Tool/Resource/Prompt
- [x] 新 Tool 默认不可调用，必须完成本地治理和 Skill grant
- [x] MCP Tool 统一经过 V2-3 Tool Runtime
- [x] MCP Tool failure 可通过结构化 error taxonomy 返回 Agent
- [x] MCP Prompt 不能进入 system trust boundary
- [x] Secret 不进入 Prompt / Task / Event / API response
- [x] SSRF、redirect、timeout、size、concurrency、circuit breaker 有边界
- [x] Schema drift 自动撤销 Tool 与 Skill 权限
- [x] MCP identity 进入 Tool Trace / Tool Catalog Hash
- [x] Web 管理能力完成
- [x] Go Release baseline 与 MCP SDK 要求一致
- [x] `go test ./...` 通过
- [x] 核心 Race Gate 通过
- [x] 前端 typecheck/build 通过

V2-5 完成后，下一阶段严格进入 **V2-6 Imported / Portable Skills**。
