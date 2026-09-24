# V4-6 Finalization Implementation Report

## 1. 范围调整

原 V4-6 Plugin System 暂缓，不再属于当前 V4。原 V4-7 Finalization 重编号为 V4-6，V4 当前以 Finalization 结束。

## 2. 本阶段完成项

### Chat / Skill

- 保留 V4-1 已完成的 Conversation 多 Skill、删除和模型切换行为。
- 保留 V4-2 Draft → Validate → Publish 的 immutable version 边界。
- Skill Studio Draft 的 Validate 和 Publish 前置校验接入现有 Agent Observability。
- Skill 校验失败记录 `type=skill_validation`、`error_type=skill_validation_error`，不新增统计表；持久化与日志层统一经过 `security.RedactText`，当前解析器错误不保存 Draft 正文。
- scripts 仍不执行。

### Selector

- 复核 V4-3 相关测试：selector 保持确定性，不调用 LLM，不增加逐 Symbol REST。
- 真实交易与测试运行继续复用同一候选选择逻辑。

### Binance API Governance

- V4-4 / V4-5 的 Usage、Exchange response headers、Budget、request reuse、singleflight、WS/local cache 与优先级调度继续作为正式实现。
- Binance API 用量区域 5 秒静默刷新，手动刷新只刷新该区域。
- Exchange response-header counters 与 Budget 使用同一 75 秒 stale 语义，避免旧 Used Weight 永久显示。
- Futures / Spot 市场 WS 为强制基础设施，不再存在基础 WS 总开关。

### System Dashboard

新增/确认以下摘要：

- Chat start errors：消息在 Task 创建前失败时记录 `type=chat_message_start` / `error_type=chat_start_error`。
- LLM errors。
- Skill Studio validation errors。
- Binance Used Weight / Order Count。
- 429 / 418。
- Global API Budget。
- Deferred / Coalesced / Cache / local WS hits。

Agent Summary 的 `chat_start_errors`、`llm_errors` 与 `skill_validation_errors` 都按当前看板时间窗口计算。顶部卡片属于重点摘要；下方 `errors[]` 仍是按 `error_type` 的分类明细，两者是不同视图而不是重复累计。

## 3. 数据库与配置

- 无新 Schema。
- 无需执行 `./go_binance_futures sync db`。
- 未修改 `app.conf`。

## 4. 验证

针对性验证：

```bash
go test ./agent/app ./agent/conversation ./agent/portableskill ./agent/observability ./scanner ./service/binanceapiusage ./feature/api/binance ./spot/api/binance -count=1
```

通过。

最终后端 Gate：

```bash
go test -count=1 ./...
go test -count=1 -race ./agent/app ./agent/conversation ./agent/portableskill ./agent/observability ./scanner ./service/binanceapiusage ./feature/api/binance ./spot/api/binance
go vet ./...
go build ./...
git diff --check
```

全部通过。macOS race 链接阶段仍会输出已知 `malformed LC_DYSYMTAB` warning，但没有 test failure 或 race report。

最终前端 Gate：

```bash
pnpm typecheck
pnpm build
```

全部通过，并已按项目约定同步 `dist/` 到后端 `static/`。

## 5. 当前结论

V4-6 的代码、文档与自动化 Gate 已完成；真实长期运行观察尚未完成，因此 Phase 仍标记为进行中。长期观察标准为连续运行至少 24 小时，检查 429/418、deferred/coalesced、WS 重连/full sync 以及 Chat/Model/Skill 错误趋势。当前实现范围为：

```text
Chat Workspace
Skill Studio
Local Smart Selector
Binance API Observability
Binance API Budget & Optimization
Finalization
```

Plugin Package 暂缓，后续如重新规划应作为独立新阶段重新评估，而不是当前 V4 的完成条件。
