# Phase V4-7：Finalization

## 1. 目标

完成 V4 收尾，确认新 Chat/Skill/Selector/API/Plugin 能力适合长期个人运行，没有引入重复平台或新的交易风险。

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

## 5. Plugin 检查

确认：

- Plugin 只是 Packaging / Capability Group，不形成第二套 Runtime。
- Skill 继续走 Portable Skill Runtime。
- MCP 继续走现有 MCP Client。
- Permission / RiskTrade 边界不被 Plugin 绕过。

## 6. System Dashboard

最终增加必要摘要：

```text
Chat / Model errors
Skill validation errors
Binance Used Weight
Binance Order Count
429 / 418
Deferred / Coalesced API calls
Plugin health
```

不做新的 Prometheus/Grafana 强依赖。

## 7. 文档

同步：

- 项目 README 多语言。
- V4 implementation reports。
- Skill Studio 使用说明。
- Binance API Budget 说明。
- Plugin compatibility 说明。

## 8. 最终 Gate

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

## 9. 项目约束

- 不修改 `app.conf`。
- 测试结束后不留下进程。
- 不回滚其它未提交修改。
- ARM 环境保持兼容。
- 个人使用优先，不增加企业级治理复杂度。

## 10. V4 最终完成状态

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

Plugin Package
  └─ Skill + MCP compatibility
```
