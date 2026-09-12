# V3-5 Trade Ownership & Safety 实施报告

## 1. 结果

V3-5 已完成真实 Binance Futures 的 Ownership 隔离与 Agent Managed Position 生命周期。

核心原则已经落地：

- Observation 可以读取整个账户；Mutation 必须先确认 ownership。
- 没有 managed record 的仓位/订单一律视为 `unmanaged`。
- 系统升级不会根据账户现状或旧 `order` 表自动认领历史仓位。
- `managed_qty` 与 Binance 聚合 `account_qty` 分离。
- 人工加仓不会扩大系统管理数量；人工减仓会收缩 managed quantity。
- 同一 `(symbol, position_side)` 首版只允许一个 active managed owner。

本阶段没有引入 Portfolio Risk、复杂 Reservation、加仓、减仓或多段 TP。

## 2. Ownership Foundation

新增独立账本：

- `futures_managed_positions`
- `futures_managed_orders`

支持 owner：`auto_strategy`、`new_coin_rush`、`notice_auto_order`、`funding_rate`、`agent_trade`。

订单统一保存 `client_order_id`、`exchange_order_id`、intent、requested/fill quantity、source_ref 与状态；只有真实 fill 才增加 managed quantity。

## 3. StartTrade 与其它真实交易模块

`StartTrade` 现在同时保留两类视图：全账户仓位/挂单只用于风险和冲突判断；`owner=auto_strategy` 的 managed 数据才允许平仓或 timeout cancel。

平仓前再次读取 Binance 当前仓位，并强制 `close_qty = min(managed_qty, account_qty)`。`FutureExcludeSymbols` 已移除，手工仓位、其它 owner 仓位和来源不明仓位依靠 Ownership 隔离，不会被 `auto_strategy` 自动认领、平仓或撤单。

其它真实 Futures 入口已统一 Owner 化：

- `new_coin_rush`：开仓登记为 `new_coin_rush`。
- `notice_auto_order`：Entry、Stop、TP 均登记为 `notice_auto_order`。
- `funding_rate`：开仓登记为 `funding_rate`。
- `agent_trade`：Entry、Stop、TP、Close 均登记为 `agent_trade`。

所有生产 Futures Buy/Sell/TP/SL 已收口到 Ownership-aware Executor；Cancel 只能由传入的 expected owner 取消自己的 managed order。

旧 `GoTestOrder()` 真实下单调试旁路已删除。

## 4. Restart / Reconcile

新增统一 Reconciler，并在配置 Binance API Key 时于程序启动执行一次只读 Ownership Reconcile。

Reconcile 只遍历本地已经登记的 managed order/position：账户里多出的手工仓位不会创建 ownership。下单超时会优先按 clientOrderId 查询交易所，无法确认则进入 `reconcile_required`，禁止盲目重发。

人工加仓保持 unmanaged；人工减仓会把 managed quantity 下调；账户数量归零则关闭 managed position。

## 5. Agent Managed Position Lifecycle

V2 Controlled Trade 的 Entry 后续已补成完整首版闭环：

```text
Approved Proposal → Entry → Managed Position → Stop → optional TP → Close → protection cleanup
```

Entry 成交后必须确认 Protective Stop。固定 Reconcile 仍无法确认 Stop 时，Proposal 进入 `protection_failed`，Entry Execution 仍保留真实成交事实，并发送 Web Notification。

TP 首版只使用 Trading Plan 的第一个目标。TP 建立失败记录 warning，但不会否定已经建立成功的 Stop。

Close 仅允许处理 `owner=agent_trade` 的 managed position；先撤 TP、保留 Stop，真实 Close 确认成交后再清理 Stop。Close 结果不确定时不会提前删除 Stop。

Stop/TP 自己成交导致 managed position 关闭时，会识别为 `closed` 并清理 sibling protection order。

项目使用 Binance Hedge Mode；该模式不接受 `reduceOnly` 参数，因此保护/平仓使用明确 `positionSide` + opposite side + 固定 managed quantity。没有使用 `closePosition=true`，避免 Close-All 把同方向人工加仓一起处理。

## 6. API / UI

`合约交易 → 仓位与受控交易` 新增 Ownership 视图，展示账户仓位 owner、account qty、managed qty、source、最近 Reconcile，以及 managed orders 的 intent/clientOrderId/Binance Order ID。

Proposal 详情展示自己的 managed position、Stop/TP；只有存在 `agent_trade` managed position 时才提供“关闭此系统仓位”。`unmanaged` 与其它 owner 不提供越权操作按钮。

新增 Ownership 查看/手动 Reconcile API 和 Agent managed position Close API。合约交易开启警告文案已更新为“只管理本系统创建并登记的仓位/订单”。

## 7. Database Version 10

V3-5 将数据库版本从 9 升级到 10，通过现有 `./go_binance_futures sync db` / `RunSyncdb` 创建 Ownership 表，不增加独立 SQL migration 文件。

> 合并说明（2026-09-12）：上述版本号是 V3-5 独立分支的实施记录。V3-4 与 V3-5 合并到 `feat/ai-agent-v3` 后，当前统一数据库版本提升为 **v12**，确保已运行任一分支旧 schema 的数据库都会再次执行 ORM Schema 同步。

实际 MySQL 已完成 `9 -> 10`；再次执行 sync 返回 `database version is already up to date: 10`，幂等验证通过。

## 8. Final Gates

- `go test -count=1 ./...`：通过。
- `go test -race -count=1 ./service/futuresownership ./service/agenttrade ./feature ./command`：通过；仅有既知 macOS linker warning。
- `go build`：通过。
- Ownership Fake Broker：部分成交、超时恢复、未知结果 fail-closed、Stop/TP、Stop 失败、Close、Restart Recovery、Sibling Protection Cleanup 均通过。
- 生产 Futures mutation scan：没有绕过 Ownership 的 Buy/Sell/TP/SL 入口。
- `pnpm typecheck`：通过。
- `pnpm build`：通过。
- 前端 `dist` 已同步部署到后端 `static`。
- MySQL Version 10 sync 与二次幂等 sync：通过。
- `git diff --check` / `git diff --cached --check`：通过。

模拟策略达到连续盈利阈值后，不再自动开启真实交易或打开 LONG/SHORT；改为 Web Notification，真实交易仍要求网页人工确认。

## 9. V3-5 边界

- 当前 Ownership slot 冲突保护采用 Service 校验 + 进程内互斥，部署前提为本项目当前的**单后端实例**个人使用模式。若未来同时运行多个后端实例/独立交易进程，必须先升级为数据库级互斥/锁定机制，再允许多实例真实交易。
本阶段不做多个 owner 共持同一方向、Agent 加仓/减仓/反手、多段 TP、机构级 Portfolio Risk、复杂订单路由或自动批准真实交易。V3-6 可以在这个 Ownership-safe Execution 基础上继续建设 Opportunity Watch。
