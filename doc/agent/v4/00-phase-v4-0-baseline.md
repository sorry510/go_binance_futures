# Phase V4-0：Baseline

## 1. 目标

冻结新 V4 开始前的 Chat、Skill、选币和 Binance API 调用基线，避免后续改造把 V2/V3 已经稳定的能力破坏。

本阶段不新增业务功能。

## 2. Chat 基线

确认并固定：

- Conversation 创建、列表、标题、消息历史。
- Chat 默认使用 `general_chat`。
- 单条消息当前只选择一个 effective Skill。
- 相同 Conversation 不允许同时存在多个 running task。
- Portable/Native chat-capable Skill 列表。
- 后端 `DeleteChat` 已存在：有 running task 时拒绝；删除 conversation/message，但保留 Task/Trace 审计历史。
- 当前模型由 Runtime / Skill / Model Gateway 决定，Conversation 本身没有明确 model preference。

## 3. Skill 基线

固定现有 Agent Skills 行为：

- `SKILL.md` YAML frontmatter。
- `name / description / license / compatibility / metadata / allowed-tools`。
- name 与目录名一致。
- ZIP / 单个 `SKILL.md` 导入。
- Package Hash / immutable version / activate。
- Path traversal / symlink / package size 等安全检查。
- Portable Skill script 只可读取，不执行。

## 4. 本地选币基线

记录当前 `TradeCoin1～6`：

- 随机抽样。
- 24h 涨跌幅排序。
- QuoteVolume Top N。
- 最近交易 cooldown。

同时记录已有 `scanner.PrefilterTop30` 的本地确定性评分能力，后续 V4-3 优先复用，不新造第二套 scanner。

## 5. Binance API 基线

扫描所有 Futures / Spot / Delivery REST 调用，并按用途分类：

```text
Trading Mutation
Account/User Data
Market Data
Historical Data
Background Maintenance
```

建立 endpoint inventory，至少记录：

- 调用函数。
- Binance endpoint。
- 调用来源。
- 当前频率/循环周期。
- 已知 request weight。
- 是否已有 WS / local DB 替代数据。
- 是否可以 cache / coalesce。

重点记录已发现热点：

`StartTrade()` 已经在每轮读取 Positions / Open Orders，但每个 `submitOwnedFeatureOpen()` 在 `ensureAccountOpenSlotAvailable()` 中可能再次读取相同数据；当未启用 User Data WS 本地镜像时，这会重复触发高权重账户查询。`GetOpenOrder()` 无 Symbol 的全账户请求当前代码标注权重 40，应作为 V4-5 首要优化对象。

## 6. 输出

V4-0 已完成以下产物：

- [v4-0-baseline-report.md](./v4-0-baseline-report.md)：Chat / Skill / Selector / API 行为基线与 Gate 结果。
- [v4-0-binance-api-inventory.md](./v4-0-binance-api-inventory.md)：Futures / Spot / Delivery REST inventory、调用放大链路与 V4-5 优先级。
- 关键 Fixture / Gate 清单已记录在 baseline report。

## 7. Gate

- `go test ./...`
- Chat / Conversation / Portable Skill 相关 `-race`。
- `go build ./...`
- `git diff --check`
- 前端 `pnpm typecheck`、`pnpm build`。

## 8. 本阶段不做

- 不新增数据库行为。
- 不改 Model Routing。
- 不改选币结果。
- 不加 API limiter。
- 不修改 `app.conf`。

## 9. 完成状态

**V4-0 已完成（2026-09-19）。**

本阶段只新增/更新 V4 文档，没有修改 Go/Vue 业务代码、数据库 Schema 或 `app.conf`。后端全量测试、关键 race、build、前端 typecheck/build 和 diff check 均通过。
