# Phase V4-6：Finalization

## 1. 目标

完成 V4 收尾，确认新 Chat / Skill / Selector / Binance API 治理能力适合长期个人运行，没有引入重复平台或新的交易风险。

## 2. Chat / Skill 检查

确认：

- Conversation 多 Skill 行为稳定。
- Model 切换可追踪实际模型。
- Chat 删除不会留下不可访问的 UI 状态。
- Skill Draft / Published Version 边界清晰。
- Skill Studio 发布结果与上传 Import 结果一致。
- scripts 仍不会被执行。

## 3. Selector 检查

确认：

- smart local selector 确定性。
- 不调用 LLM。
- 不新增逐 Symbol Binance REST。
- 旧 selector 配置仍兼容。

## 4. Binance API 检查

确认：

- System Dashboard 可查看真实 API 用量。
- Top endpoint 和权重来源清晰。
- StartTrade 多币开仓不再线性重复高权重账户查询。
- Background/Historical 会在预算紧张时让路。
- 429/418 不形成 retry storm。
- Mutation timeout/429 不会导致 duplicate order。
- User Data WS/local snapshot 有 freshness/fallback。

## 5. System Dashboard

最终增加必要摘要：

```text
Chat / Model errors
Skill validation errors
Binance Used Weight
Binance Order Count
429 / 418
Deferred / Coalesced API calls
```

不做新的 Prometheus/Grafana 强依赖。

## 6. 文档

同步：

- 项目 README 多语言。
- V4 implementation reports。
- Skill Studio 使用说明。
- Binance API Budget 说明。

## 7. 最终 Gate

后端：

```bash
go test ./...
go test -race <V4相关包>
go vet ./...
go build ./...
git diff --check
```

前端：

```bash
pnpm typecheck
pnpm build
```

并按既有方式同步 `dist → backend/static`。

数据库 Schema 如果变化仍只能使用：

```bash
./go_binance_futures sync db
```

## 8. 项目约束

- 不修改 `app.conf`。
- 测试结束后不留下进程。
- 不回滚其它未提交修改。
- ARM 环境保持兼容。
- 个人使用优先，不增加企业级治理复杂度。

## 9. V4 最终完成状态

V4 完成后，项目应从“Agent 平台已经很完整，但日常使用和 API 资源治理不足”升级为：

```text
Chat Workspace
  ├─ Multi Skill
  ├─ Model Switch
  └─ Conversation Management

Skill Studio
  └─ Standard Agent Skills Authoring

Local Smart Selector
  └─ Deterministic / No extra REST

Binance API Governance
  ├─ Usage Observability
  ├─ Weight Budget
  ├─ Request Reuse
  └─ Priority Scheduling

```

## 10. 实施结果

- Skill Studio Draft Validate / Publish 前置校验已接入现有 Agent Observability，失败统一记录为 `skill_validation_error`。
- Agent Observability Summary 新增 `chat_start_errors`、`llm_errors` 与 `skill_validation_errors` 周期计数。
- 系统看板新增 Chat 启动错误、LLM 错误、Skill 校验错误摘要；Binance API 区域继续展示 Used Weight、Order Count、429/418、Budget 与优化命中。
- Plugin System 已从当前 V4 开发范围移除，不属于本阶段 Gate。
- 未新增数据库 Schema，未修改 `app.conf`。
- 自动化 Final Gate 已通过；长期运行观察仍待实际部署验证，详见 `v4-6-implementation-report.md`。

## 11. 长期运行观察清单

V4-6 保持 🚧，直到实际部署连续运行至少 24 小时并人工确认：

- Binance `429 / 418` 没有持续增长或 retry storm。
- `deferred_requests` 的比例符合预期，关键交易请求没有被普通后台任务长期饿死。
- `coalesced_requests` / cache / local WS 命中持续出现，说明优化路径真实生效。
- Futures / Spot WS 自动重连后恢复 freshness；User Data WS reconnect 后完成当前 generation full sync。
- `chat_start_errors`、`llm_errors`、`skill_validation_errors` 没有异常持续增长，并能从 Trace / error 分类定位原因。
- StartTrade、Notice、Rush、Funding、历史数据等实际启用功能未再次触发 API 权重异常放大。

观察通过后再把 V4-6 从 🚧 改为 ✅。
