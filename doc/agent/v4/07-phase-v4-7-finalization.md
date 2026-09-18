# Phase V4-7：Finalization

## 1. 目标

完成 V4 收尾，确保真实交易生命周期能力适合长期个人运行，不把系统演变成复杂机构交易平台。

## 2. 最终检查

### Position Safety

确认：

- 所有真实 Futures Mutation 仍经过 Ownership-aware Executor。
- Position Guardian 不认领 unmanaged position。
- Protection Repair 不会 over-protect。
- Reduce / Multi-TP 不会超过 managed qty。
- `reconcile_required` 一律 fail-closed。

### Live Ledger

确认：

- PnL / Fee / Funding 可重复对账。
- 重启/Reconcile 不重复记账。
- manual/unmanaged 不污染 managed trade。

### AI Position Manager

确认：

- AI 只能提出结构化 Action Proposal。
- Deterministic Validation 与用户确认仍然存在。
- AI/LLM 故障不会影响 Stop、Guardian、Reconcile。

### Optional Hedge

如果 V4-6 未实施，文档明确记录为“保留关闭”。

如果实施，则完成双向 Testnet E2E。

## 3. System Dashboard

复用现有系统看板，只增加必要摘要：

- Unprotected Managed Position。
- Protection Degraded。
- `reconcile_required`。
- Ledger Reconcile Error。
- Position Action Pending / Failed。

不新建第二套监控系统。

## 4. 数据清理

现有 `cleanup logs` 不能删除：

- Managed Trade Ledger。
- Managed Position / Order。
- Agent Trade Proposal / Execution / Audit。
- Position Action Proposal / Audit。

真实交易生命周期数据属于长期交易记录，不按普通日志清理。

## 5. 最终自动验证

后端至少执行：

```bash
go test ./...
go test -race ./service/futuresownership ./service/agenttrade <V4新增真实交易包>
go vet ./...
go build ./...
git diff --check
```

前端：

```bash
pnpm typecheck
pnpm build
```

然后按既有约定同步 `dist` 到后端 `static`。

数据库升级仍只能通过：

```bash
./go_binance_futures sync db
```

## 6. 最终 Testnet Gate

至少覆盖：

- Open/Close LONG。
- Open/Close SHORT。
- Stop / TP。
- Guardian Repair。
- Manual Partial Reduce 后 Protection Resize。
- System Partial Reduce。
- Multi-TP。
- Restart/Reconcile。
- Ledger PnL/Fee 可对账。
- 如果启用 V4-6，再覆盖 LONG/SHORT 同币双开。

## 7. 项目约束

- 不修改 `app.conf`。
- 测试后不留下测试进程。
- 不回滚无关工作区修改。
- ARM 环境必须可运行。
- 个人使用优先，不增加企业级审批和治理复杂度。

## 8. V4 完成标准

V4 完成后，系统应具备完整的真实交易生命周期：

```text
发现机会
→ 分析
→ Proposal
→ Risk
→ Approval
→ Entry
→ Guardian
→ Ledger
→ Partial/Multi-TP 管理
→ AI Position Review
→ Deterministic Action Validation
→ Exit
→ 最终真实 PnL 复盘
```

同时保持：

- AI 不直接交易。
- Ownership 不越权。
- 不做同方向加仓。
- 不依赖任何新的 Backtest 功能。
