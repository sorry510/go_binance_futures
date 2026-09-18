# Phase V4-0：Real Trading Baseline

## 1. 目标

冻结 V3 完成后的真实交易行为，为后续 Position Guardian、Live Ledger 和仓位管理能力建立可靠回归基线。

本阶段不新增真实交易功能。

## 2. 冻结范围

重点冻结：

- Trade Proposal → Risk → Approval → Execution。
- Ownership Claim / Managed Order / Managed Position。
- Binance Hedge Mode 下 LONG / SHORT 的方向语义。
- Entry、Stop Loss、Take Profit、Full Close。
- Partial Fill。
- Restart / Reconcile。
- 人工减仓后 `managed_qty` 收缩。
- 人工加仓保持 unmanaged，不扩大 `managed_qty`。
- Unknown Exchange Result → `reconcile_required`，禁止盲目重试。
- Protection cleanup。
- Owner 隔离：`auto_strategy / new_coin_rush / notice_auto_order / funding_rate / agent_trade`。

## 3. 自动化 Fixture

优先复用现有：

- `service/futuresownership`
- `service/agenttrade`
- `feature`

已有测试，不重复建设第二套测试体系。

重点确认：

```text
Open LONG → Close LONG
Open SHORT → Close SHORT
Entry → Stop
Entry → TP
Partial Fill
Manual Reduce → managed_qty shrink
Manual Add → managed_qty unchanged
Restart/Reconcile
Sibling Protection Cleanup
Unknown Submit Result → fail closed
```

## 4. Testnet Baseline

继续复用 V3 已建立的 Binance Futures Testnet E2E。

至少确认：

- 开多 → 平多。
- 开空 → 平空。
- TP。
- SL。
- 人工部分减仓后系统只处理剩余 managed quantity。

本阶段不为了基线测试真实 Mainnet。

## 5. 输出

- V4-0 Baseline Report。
- 当前 Ownership / Agent Trade 核心行为清单。
- 后续 V4 Phase 不允许无意破坏的回归 Gate。

## 6. 验收 Gate

- `go test ./...`
- `go test -race` 覆盖 `futuresownership / agenttrade / feature` 关键包。
- `go build ./...`
- `git diff --check`
- 前端 `pnpm typecheck`、`pnpm build`。
- 必要的 Testnet E2E 通过。

## 7. 本阶段不做

- 不做回测。
- 不新增数据库业务模型。
- 不增加 Position Action。
- 不修改真实交易 Policy。
- 不开放多空双开。
- 不开放同方向加仓。
