# BT-1 回测准确性开发记录（待 Code Review）

> 2026-11-19。仅修改本地回测代码和测试。依据：`回测引擎优化方案-精简版.md` 及其复审报告；本文件是开发记录，不是独立审查结论。

## 已实现

1. **当前 K 线主动买入字段修复**：`service/backtest/indicator_cache.go` 的 `cachedKlinePriceSeries` 每次更新 `[0]` 时同步设置 `TakerBuyAmount` 和 `TakerBuyRatio`；成交额为 0 时比率归零。新增按连续分钟比较完整/缓存两条路径的测试。
2. **同秒止盈止损冲突修复**：`service/backtest/adaptive_engine.go` 当 1s 同时可能触达止盈、止损时，先用按 `TradeTime,TradeID` 排序的 Trades 判定先后，不再让 1s 收盘价抢先确定退出。保留其他无歧义秒现有 1s 优先快路径。
3. **缺失证据保护**：同秒双门槛歧义且 Trades 为空时，明确报错，不以 1s Close 推断先后。
4. **引擎版本**：`service/backtest/types.go` 将 Standard `backtest_engine_v7` → `v8`（`standard_1m_v2`）、Adaptive `backtest_engine_v10` → `v11`（`adaptive_intrabar_v3`）。复用现有 Run 的版本与 DataHash 字段，不改数据库结构和旧结果。
5. **价格语义确认**：实盘主循环优先币安 Mark Price WS，失败时有本地行情回退；回测 `grossROI` 与 `DetectROICandidate` 目前均使用同一套成交 K 线价作为 Mark Price 的近似。已在代码加约束注释，防止未来将不同口径混用。

## 当前不做／限制

- `service/historicalmarket/public_data.go` 有 `ArchiveKindMarkPriceKlines` 和 URL 构造，但**没有证明历史归档的真实存在、字段格式、覆盖范围及与 ROI 时序一致**。因此本次没有接入历史 Mark Price，也没有使用真实 Binance 接口；**不能声称回测 ROI 已完全等同实盘**。
- `adaptive` 开仓仍按 1m 信号、下一分钟开盘模拟成交；秒级精细化重点在持仓期间的平仓。实际约 2 秒调度、订单延迟及撮合不在本阶段。
- 若 Trades 记录非空但数据不完整，仍有历史证据一致性风险；不要把离线单测当作完整历史验证。
- 不修改实盘、配置、数据库、存储/IO 架构（这些归 BT-2 / BT-3 或后续独立议题）。

## 已执行的验证

| 检查 | 结果 |
|---|---|
| `go test ./service/backtest -count=5 -p=2` | **通过**，5 次运行 |
| `go test ./service/historicalmarket -run '^TestPublicDataArchiveURL$' -count=1` | **通过**，仅验证归档 URL 构造 |
| `go vet ./service/backtest` | **通过** |
| `git diff --check` | **通过** |
| 真实 Binance 历史 Mark Price / 全量回测 / MySQL 8 / 生产库 | **未测试** |

新增测试文件：`service/backtest/bt1_accuracy_test.go`。覆盖动态 Taker 字段、零成交额、LONG/SHORT 同秒双门槛先后和缺失 Trades 证据。

## 独立 Code Review 重点

1. 检查同秒双门槛是否正确遵守 Trades 顺序，且不影响普通无歧义秒的性能。
2. 检查 Taker 缓存所有当前字段与原始路径一致，避免修复引入新的指标缓存差异。
3. 确认引擎版本及 `resolution_model` 已区分旧结果；无需 DB Migration。
4. 价格口径限制应继续显示在后续真实历史对照报告中，不能将未验证的 Mark Price 归档写成已接入。
