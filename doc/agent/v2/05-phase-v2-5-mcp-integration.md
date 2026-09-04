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

Tool 名使用 canonical identity。LLM 若仅改变远端 Tool basename 的大小写/`-`/`_` 等形式，Runtime 只在“当前 Skill 已授权且已注册”的候选中做唯一匹配并恢复 canonical name；0 个或多个候选时 fail-closed，不猜 Provider，并进入一轮 `tool_name` repair feedback，明确返回允许的精确 canonical names。该兼容层不能绕过 Skill grant、Risk Policy 或 Registry。

Resource / Prompt 不注册为 Tool，而是进入 V2-2 Context Engine。远端 Prompt 固定带 `EXTERNAL_MCP_PROMPT` 边界声明，只能作为不可信外部 Context，不能覆盖 System Policy、Risk、Permission 或 Budget。

## 3. Registry 与治理模型

新增 ORM 模型：

- `agent_mcp_servers`
- `agent_mcp_tools`
- `agent_mcp_resources`
- `agent_mcp_prompts`
- `agent_mcp_permissions`
- `agent_mcp_secrets`：OAuth credential AES-GCM ciphertext；不保存 token 明文
- `agent_mcp_oauth_states`：短期 OAuth state/PKCE transaction，加密并带过期时间

没有新增 `command/sql/version/2.sql`；仍由现有 `orm.RunSyncdb` 管理结构。SQLite legacy upgrade regression 已验证：

- 旧 Agent Task 数据存在时可安全创建 V2-5 表；
- 已有 `agent_mcp_servers` 数据时可 additive 增加 OAuth 字段和两张 OAuth 存储表，原 Server 行保持可读。

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
- `oauth2` interactive Authorization Code + PKCE

Interactive OAuth 流程：

```text
MCP initialize
  -> 401 WWW-Authenticate Bearer challenge
  -> RFC 9728 Protected Resource Metadata
  -> Authorization Server Metadata
  -> Client ID Metadata Document（优先）/ DCR（兼容旧 Server）
  -> PKCE S256 + random state
  -> Browser Authorization
  -> public callback
  -> Authorization Code exchange
  -> encrypted token persistence
  -> MCP reconnect
```

OAuth state 只在数据库保存 SHA-256 hash；PKCE verifier、client registration state 和 OAuth token 使用 AES-GCM 加密。callback 完成后 state 立即消费，重复 callback 会被拒绝。

Token refresh 继续经过安全 HTTP Client；如果授权服务器旋转 access/refresh token，新 credential 会重新加密写回数据库，进程重启后仍可继续使用。

Secret resolver 支持：

```text
env:VARIABLE_NAME       # Bearer / custom header / 兼容的预置 OAuth credential
mcpdb:oauth:<serverID>  # 系统内部加密 OAuth credential；API 不接受用户直接提交
```

API/UI 不返回 `secret_ref` 或任何 credential value，只返回 `has_secret` 与非敏感 OAuth 状态。`custom_header` 仅保存 Header **名称**（例如 `X-API-Key`），Header 值仍来自 SecretRef。

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
- 新 MCP Tool 默认总执行预算为 60 秒；MCP initialize 与 HTTP 单请求限制 30 秒，Response Header 限制 25 秒
- 仅本地确认 `risk=read + idempotent=true` 的 Tool 对 timeout / 临时网络故障自动重试 1 次；write/trade 永不自动重试
- 当前项目已将现有 V2-5 早期默认的 `timeout_ms=10000` Tool 一次性迁移为 `60000`；不覆盖其它管理员自定义 timeout
- 连续失败 circuit breaker
- **MCP transport 不继承 HTTP_PROXY / HTTPS_PROXY**，避免代理绕过目标 IP SSRF 校验
- OAuth metadata、DCR、token exchange、refresh 使用受限 HTTP Client；不会继承系统 HTTP proxy
- OAuth `token_url` 使用同一套 HTTPS、DNS/IP 和 redirect 限制
- 浏览器 `authorization_endpoint` 只返回给管理员 UI 导航，后端不会服务端抓取授权页面
- OAuth callback 使用一次性 state + PKCE，防止 CSRF / code interception / replay

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
- OAuth 状态展示
- OAuth “授权 / 重新授权”
- OAuth 授权 popup 完成后自动刷新 Server 状态

Task Center Tool Trace 新增展示：

- MCP Provider Ref
- Protocol Version
- Catalog Hash
- Schema Hash

Bearer/custom-header Secret 输入只接受 `env:` 受管引用，编辑时留空仅在 Auth Type 和 Endpoint 不变时保留已有凭据。切换 Auth Type 或 OAuth Endpoint 不会静默复用旧 token。

OAuth2 Server 不要求用户手工填写 token；保存后通过“授权”按钮完成浏览器 OAuth。

## 11. HTTP API

```text
GET    /agents/mcp/servers
POST   /agents/mcp/servers
PUT    /agents/mcp/servers/:id
DELETE /agents/mcp/servers/:id
GET    /agents/mcp/servers/:id/catalog
POST   /agents/mcp/servers/:id/test
POST   /agents/mcp/servers/:id/refresh
POST   /agents/mcp/servers/:id/oauth/start
GET    /agents/mcp/oauth/client-metadata
GET    /agents/mcp/oauth/callback
PUT    /agents/mcp/tools/:id
POST   /agents/mcp/permissions
```

Server/Tool/Catalog 变化会热同步到默认 Tool Registry，不要求重启后端。Skill grant 在每个新 Task 启动时动态读取。

OAuth 路由权限：

- `/servers/:id/oauth/start` 必须登录，只有管理员 UI 能启动授权；
- `/oauth/client-metadata` 与 `/oauth/callback` 是 OAuth Provider/浏览器必须访问的公开精确路由；
- 不放开其它 `/agents/mcp/*` API。

## 12. Interactive OAuth 部署配置与 Binance 官方 MCP

Interactive OAuth 配置统一放在 `conf/app.conf` 的 `[mcp]` 段，不再读取环境变量：

```ini
[mcp]
oauth_public_base_url = "https://your-agent-host.example.com"
oauth_encryption_key = "<32-byte AES key, hex or base64>"
```

建议生成生产 key：

```bash
openssl rand -hex 32
```

`oauth_public_base_url` 默认必须是 OAuth Provider 能访问的公网 HTTPS 地址；仅 `localhost` / `127.0.0.1` / `::1` 允许 HTTP 用于本地开发。`conf/app.conf.example` 已提供可直接启动的本地回环配置，但该地址不能用于 Binance 远端授权。Client ID Metadata Document 和 callback 自动固定为：

```text
${oauth_public_base_url}/agents/mcp/oauth/client-metadata
${oauth_public_base_url}/agents/mcp/oauth/callback
```

Binance 官方 Agentic MCP 已按真实 metadata 验证兼容：

```text
Name: Binance Agentic
Endpoint: https://agent.binance.com/mcp/agentic
Auth Type: oauth2
Allow Private: off
```

使用流程：

```text
保存 Server
  -> 授权
  -> Binance 浏览器登录/权限确认
  -> callback 自动加密保存 token
  -> 测试连接
  -> 刷新能力目录
  -> 对 Tool 做本地 Risk/Enable/Skill Grant
```

Binance 当前 challenge/metadata 组合为 RFC 9728 Protected Resource Metadata + Authorization Code + PKCE S256 + Client ID Metadata Document；不应通过 Binance API Key 或手工 Bearer token 绕过该流程。



### MCP Server 显示信息与 canonical name

`agent_mcp_servers` 额外保存管理员维护的 `description TEXT`，用于说明该 MCP Server 的业务用途；该描述不参与鉴权、Catalog Hash 或权限判断。

当前 canonical Tool 命名仍使用：

```text
mcp.<normalized Server.Name>.<normalized remote tool name>
```

因此修改 `Server.Name` 后，下一次刷新能力目录会同步改变该 Server 下 Tool 的 canonical name。Skill 授权按 `capability_id/tool_id` 保存，不会仅因改名丢失；历史 Task、Evidence、Trace 中已经冻结的旧 canonical name不会回写。管理页面必须明确提示这一点。

### Agent Runtime 超时预算

接入远端 MCP 后，一个 Agent run 可能包含多个 LLM round 与多个远端只读 Tool。默认 Agent Runtime 与 market-regime Scheduler 的单任务总预算统一为 **5 分钟**；单个 MCP Tool 仍受自己的 `timeout_ms` 独立约束。这样不会用总任务 timeout 代替 Tool timeout，也避免正常 MCP 多轮任务因 3 分钟预算过紧而被误判为 timeout。

### MCP 接入后的 AI 延迟与 Evidence 兼容

V2-5 上线后验证过以下生产问题，已作为实现约束修复：

- `symbol_analysis` Final Validator 必须接受本轮真实成功调用并产生 Structured Evidence 的 `mcp.*` source；未知或未实际调用的 source 仍 fail-closed。
- MCP Skill Tool Grant 查询不得按 Tool 数量逐条访问数据库。`GrantedTools` 使用 Permission + `id__in` Tool 两次批量查询，Tool Catalog Context 复用批量结果。
- MCP Resource/Prompt Context 使用批量 Server/Resource/Prompt 查询，不允许按授权数量产生 N+1 公网数据库往返。
- 新 Agent Task 创建使用 Store `Create` 快路径：已知新 Task 直接 INSERT，不先查存在，不对空 Event 做 COUNT，也不在 HTTP 返回前重新 SELECT Task/Event。
- 公网 MySQL 配置 connect/read/write timeout、连接池大小、max lifetime 与 max idle time，降低 stale connection 复用导致的 `unexpected EOF`。
- Web AI API 不再继承旧的 10 秒全局等待：普通 Agent/AI API 为 30 秒，MCP connect/OAuth 为 45 秒，Catalog refresh 为 120 秒；这些是 HTTP 等待预算，不替代 Runtime/Tool timeout。

一次实际 `symbol_analysis` 回归中，native context、Coinfuty 与 CoinGecko MCP Tool 均成功，最终 MCP Evidence 通过 Validator 并完成 Task；历史失败任务确认原先的 `context deadline exceeded` 是错误 Evidence 拒绝触发额外 repair 后撞到总 Runtime timeout，而不是 MCP Tool 本身失败。

## 13. 自动回归覆盖

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
- Interactive OAuth 401 challenge / Protected Resource Metadata / Authorization Server Metadata
- Client ID Metadata Document client identity
- PKCE S256 / random state / one-time callback consumption
- Authorization Code token exchange
- AES-GCM OAuth token/state persistence，DB 不出现 token 明文
- 新 Gateway / 进程重启式 credential restore
- refresh token rotation 后加密写回
- 未授权 OAuth Server fail-closed
- OAuth public callback/metadata 与 protected start route
- 已有 MCP Server 行的 additive ORM upgrade
- MCP Tool/Resource/Prompt grant 批量查询，避免 N+1
- 新 Task Create fast path，不同步 read-back
- `symbol_analysis` MCP Structured Evidence Final Validation

V2-4 Eval/Replay 全量回归继续通过。

## 14. Phase Gate

- [x] 至少一个 Streamable HTTP MCP Server 可通过真实协议 E2E 发现 Tool/Resource/Prompt
- [x] 新 Tool 默认不可调用，必须完成本地治理和 Skill grant
- [x] MCP Tool 统一经过 V2-3 Tool Runtime
- [x] MCP Tool failure 可通过结构化 error taxonomy 返回 Agent
- [x] MCP Prompt 不能进入 system trust boundary
- [x] Secret 不进入 Prompt / Task / Event / API response
- [x] OAuth Authorization Code + PKCE + CIMD/DCR 可持久化跨请求执行
- [x] OAuth token/state 加密落库，state 单次消费，refresh rotation 持久化
- [x] SSRF、redirect、timeout、size、concurrency、circuit breaker 有边界
- [x] Schema drift 自动撤销 Tool 与 Skill 权限
- [x] MCP identity 进入 Tool Trace / Tool Catalog Hash
- [x] Web 管理能力完成
- [x] Go Release baseline 与 MCP SDK 要求一致
- [x] 最终 `go test ./...` 通过
- [x] 最终核心 Race Gate 通过
- [x] 最终前端 typecheck/build 通过
- [x] Binance 官方 OAuth start-flow compatibility probe 通过

V2-5 已完成；下一阶段严格进入 **V2-6 Imported / Portable Skills**。
