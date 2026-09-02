# Phase V2-5：第三方 HTTP MCP Client 接入

## 目标

本 Phase 只建设 **MCP Client/Host** 能力：`go_binance_futures` 主动连接第三方提供的标准 HTTP MCP Server，并把远端 Tool、Resource、Prompt 纳入现有 Agent Runtime。

本 Phase **不建设、不升级、不验收本项目自身的 MCP Server**；现有 `mcpserver/` 与 `/mcp` 对外服务不属于 V2-5 范围，除非未来另开独立任务处理。

核心链路：

```text
Agent Runtime
    |
Tool Runtime / Context Engine
    |
MCP Client Gateway
    |
Streamable HTTP
    |
Third-party MCP Server
```

任何第三方 MCP 能力都不能绕过 Tool Runtime、Permission、Budget、Trace 和 Security。

## 协议与 Transport 范围

首版只支持远程 `Streamable HTTP` MCP Server。目标协议跟进 MCP `2026-07-28`，同时尽量通过官方 Go SDK 的协议探测/协商兼容较旧的标准 MCP Server。

当前项目使用 `github.com/modelcontextprotocol/go-sdk v0.4.0`，实现本 Phase 前应先验证并升级到满足目标 Client 协议能力的稳定版本。升级只服务于 Client Gateway，不要求同步重构现有 MCP Server。

明确暂不做：

- 不把本项目对外 `/mcp` Server 纳入本 Phase。
- 首版不支持本地 `stdio` MCP Server。
- 首版不主动实现旧 HTTP+SSE transport；如真实第三方服务仍只有 SSE，再独立增加 legacy adapter。
- 不允许 LLM 动态指定任意 MCP URL。

## Remote MCP Registry

新增第三方 MCP 连接注册表。建议实体包含：

```text
id
name
endpoint
enabled
auth_type
secret_ref
protocol_version
server_name
server_version
status
last_success_at
last_error_at
last_error
catalog_hash
created_at
updated_at
```

`endpoint` 必须由管理员显式配置。Agent 只能引用已注册的 Server ID/Name，不能把用户输入直接当 URL 发起连接。

Secret 不写入 Prompt、Task Input、Task Event 和普通日志；数据库只保存 `secret_ref` 或受保护的凭据引用。

## 鉴权

按第三方服务实际能力支持：

1. `none`：公开 MCP Server。
2. `bearer`：固定 Bearer Token。
3. `oauth2`：按 MCP 标准授权流程接入远程授权服务器。
4. `custom_header`：只作为受管兼容模式，Header 名和值由管理员配置并进入 Secret 管理，不允许 Skill/LLM 动态生成。

OAuth Token 的获取、刷新和失效处理属于 MCP Client Gateway，不进入 Skill Prompt。

## Client Gateway

建议新增：

```text
agent/mcpclient/
├── client.go
├── registry.go
├── connector.go
├── auth.go
├── catalog.go
├── adapter.go
├── health.go
└── security.go
```

Runtime 不直接依赖 MCP SDK；MCP 协议细节全部封装在 `mcpclient` 内。

## Discovery

连接第三方 Server 后发现其 Capability，并同步：`tools`、`resources`、`prompts`。

远端能力保存 Catalog Snapshot 与 hash，用于审计、权限审批和 Replay。Catalog 刷新不能自动扩大 Agent 权限。

远端 Server 新增 Tool 时：

```text
discovered -> unclassified -> disabled -> admin review -> granted
```

删除或修改 Tool Schema 时，应使相关授权进入 `needs_review` 或重新校验状态，避免旧权限静默套用到新语义。

## Tool 映射

MCP Tool 统一映射到 Tool Runtime，canonical name 使用命名空间：

```text
mcp.<server-name>.<tool-name>
```

例如 `mcp.coinalyze.get-liquidations`。MCP Tool 仍转换为统一 `ToolDescriptor` 和 `ToolResultEnvelope`，继续使用 JSON Schema、timeout、result size、error taxonomy、cache、Permission/Risk、Budget、Trace/Evidence。

远端 Tool annotation、description、readOnlyHint 等只能作为分类参考，**最终 Risk Level 必须由本系统决定**。

## Resource 接入

MCP Resource 不直接拼进 System Prompt，而是进入 Context Engine：

```text
MCP Resource -> MCP Client Gateway -> ContextBlock
             -> freshness / size / sensitivity check
             -> Agent Context
```

需要记录 `server_id`、`uri`、`mime_type`、`as_of`、`content_hash`、cache hint。Skill 只能读取被授权 Server/Resource 范围。

## Prompt 接入

MCP Prompt 视为外部模板资源，而不是系统级指令：

- 不允许覆盖 Runtime System Policy。
- 不允许扩大 Tool Permission。
- 不允许修改 Risk Policy、Budget 或 Skill trust。
- 标记来源为 `external_mcp_prompt`。
- 进入 Context 时按外部不可信内容处理。

## HTTP、网络与 SSRF 安全

因为这是服务端主动连接第三方 HTTP 地址，必须显式处理 SSRF：

- Endpoint 只能由管理员创建/修改。
- 默认要求 HTTPS；localhost/私网地址只有管理员显式允许时开放。
- DNS 解析后的实际 IP 仍执行网络策略校验，防止 DNS rebinding。
- 禁止自动跟随到未授权 Host 的重定向。
- 每个 Server 配置 connect/read/overall timeout。
- 限制响应体大小、并发数和单 Task 调用次数。
- 远端故障进入熔断/退避，不阻塞行情、WebSocket、交易主循环。

## Cache 与 Catalog 刷新

对于支持 cache hint 的标准 MCP Server，优先遵循 Server 返回 TTL/cache scope，同时受本系统最大 TTL 限制。

至少缓存 Tool/Resource/Prompt catalog。实时 Tool 执行结果是否缓存仍由 Tool Runtime 决定。

## 多轮输入 / Confirmation

如果远端 MCP 调用返回需要额外用户输入或确认的标准多轮结果，Gateway 将其转换为本系统统一 `waiting_input` / Approval 状态。

远端 Server 不能直接控制前端，也不能自行批准 write/trade 行为；输入完成后由 Runtime 决定是否继续原 MCP 调用。

## Web 管理

新增 `AI -> MCP`，定位为 **第三方 MCP 连接管理**：

- Server Name / Endpoint / Enabled。
- Auth 类型与 Secret 状态。
- Test Connection。
- 协议版本、Server Info、Capabilities。
- Tools / Resources / Prompts Catalog。
- Tool Risk 分类和授权。
- Skill -> MCP Tool 授权关系。
- 最近连接状态、错误、耗时。
- Refresh Catalog / Disable / Kill Switch。

页面不提供“将本项目暴露成 MCP Server”的 V2 配置。

## 数据模型建议

新增 `agent_mcp_servers`、`agent_mcp_tools`、`agent_mcp_permissions`。Resource/Prompt Catalog 可按数量决定是否独立表；普通结构变化继续使用 ORM Model + `orm.RunSyncdb`。

## 与 Agent Skills 的关系

标准 Skill 可以在 `allowed-tools` 中申请 `mcp.<server>.<tool>`，但只代表请求。实际可用必须同时满足：Server enabled + Tool healthy/classified + system permission + Skill allowlist + Runtime budget。

因此导入 Skill 绝不会自动安装、连接或授权任意第三方 MCP Server。

## Eval / Replay

MCP Tool 必须支持 Fixture 化。离线 Eval 不调用真实第三方服务器，而是回放规范化后的 `ToolResultEnvelope`。

Task 保存 `mcp_server_id`、`protocol_version`、`catalog_hash`、tool canonical name、tool schema hash，确保第三方 Server 后续变化后历史 Task 仍可解释。

## 验收

- [ ] 可配置并连接至少一个第三方 Streamable HTTP MCP Server。
- [ ] 能发现并显示远端 Tools、Resources、Prompts。
- [ ] MCP Tool 通过统一 Tool Runtime 执行，而不是 Runtime 直接调用 SDK。
- [ ] MCP Resource 通过 Context Engine 注入。
- [ ] 新发现 Tool 默认不可被 Skill 调用。
- [ ] Bearer/OAuth 等 Secret 不进入 Agent Context 或普通日志。
- [ ] Endpoint 有 SSRF、防重定向和网络范围保护。
- [ ] 第三方 MCP 故障不会影响核心行情和交易循环。
- [ ] MCP Tool 调用可追踪到 Server/Catalog/Schema 版本。
- [ ] Offline Eval 可使用 MCP Fixture，不依赖真实第三方服务。

## Definition of Done

V2-5 完成后，第三方 HTTP MCP Server 对 Runtime 来说只是另一种受治理的 Tool/Context Provider。Agent 不需要知道 HTTP、OAuth 或 MCP 协议细节，也不能因为接入外部 MCP 而获得新的权限绕过路径。
