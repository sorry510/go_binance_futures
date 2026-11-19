# BNB 手续费历史估算（只读展示）

日期：2026-11-19

## 背景

服务器 `go_bn_oracle1` 历史只读调查：50 个 closed/incomplete 周期都有 BNB 手续费，已记录的开仓和平仓 Fill 总量一致，但 `verifiedSettlementFacts` 不允许把 BNB 手续费直接按 USDT 处理。真实结算状态与订单执行状态不能混为一谈。

## 方案

- 仅对 `status=closed`、`settlement_status=incomplete`、币对后缀 USDT 的 Managed 周期应用。
- 重用 `summarize` 的 Managed 订单与 Fill 身份、REST 核实、开/平仓数量及订单状态证明。历史正式核实 T0 可以不存在：估值只用于展示，独立执行证明不会提升官方核实状态。
- 仅当除未正式结算外 **没有其它成交数据错误**，且所有非零手续费资产只为 BNB/USDT 时，才允许 BNB 费用估算。
- **2026-11-19 最终简化：** 使用已有订单历史查询中的本地 `symbols` 表 `symbol='BNBUSDT'`、`type='USDT'` 的 `close` 当前价格，同一批请求的所有历史 BNB 手续费统一按此价格折算：`estimated_fee_usdt = Σ(commission[BNB] × 本地 BNB 当前价) + Σ(commission[USDT])`。复用原本读取 `symbols` 的 SQL，额外带上 BNBUSDT，不调用任何历史 K 线、网络 API，不读写历史行情表，也不进行日级缓存。新价格写入本地 `symbols` 后重新查询即可更新估算值。
- 后端返回 `bnb_fee_estimated`、`estimated_fee_usdt`（包含同周期 USDT 原费）、`estimated_net_pnl`、`bnb_fee_price_usdt` 与本地价格时间；已平仓参考盈亏、总盈亏、名义收益率和筛选汇总采用估算扣费值。前端绿色「BNB估算」及本地使用价格说明。
- 若本地 BNBUSDT 当前价缺失、无效、非正数或报价资产不为 USDT，**不假设汇率**，保留原 `incomplete` 显示。非 BNB 周期保持旧算法；人工覆盖完整性优先于估算。

## 不变的边界

- 服务器 `go_bn_oracle1` 不执行 UPDATE、INSERT、DELETE、DDL、迁移、历史补算。
- `settlement_status` 保持 `incomplete`，`SettlementProcessedAt`、`SettlementPNLSign`、`pnl_asset` 不回填；`NetPnL`（官方已核实字段）仍为空。
- `settleVerifiedFlat` 与 `applySettledRisk` 不改变，估算值不影响真实连续亏损、资金控制或交易执行。
- UF-5 「已核实实盘策略效果」仍然只接收 `settled` 周期，不纳入估算收益；订单页的筛选参考盈亏则可以显示估算结果。
- UF-3 默认列表已区分执行异常与财务待核实。BNB 估算仅说明交易执行证据正常，不能伪造已完成财务结算。
- 不等同于 Binance 实际 BNB 扣费汇率；采用当前价估算历史手续费会随 BNB 价格变化而浮动，用户明确接受该近似口径。

## 本地测试

- `TestBNBFeeEstimateClosedCycleWithoutMutatingSettlement`：正常 BNB 周期估算手续费、净收益、详情和筛选列表；仍非官方核实。
- `TestBNBFeeEstimateNeverHidesMissingFillsOrPrices`：缺价格、REST 未核实不允许判为估算正常。
- `TestBNBFeeEstimateWorksWithoutCertifiedHistoryT0`：即使 T0 缺失，仍可做标识明确的只读估算，且真实结算状态不变。
- `TestBNBFeeEstimateUsesCurrentLocalBNBPrice`：本地价格由 600→650 时，历史手续费估算 4.8→5.2 USDT，结算状态不变。
- `TestBNBFeeEstimateMissingLocalQuoteRemainsIncomplete`：没有本地 BNB 价格时不虚构正常结算。
- `go test -count=1 -p=2 ./service/orderhistory ./service/futuresownership ./service/tradefills ./controllers`：通过。
- `go vet ./service/orderhistory ./controllers`：通过。
- `pnpm exec vue-tsc --noEmit`、`pnpm build`：通过；构建产物同步到后端 `static`。

## 部署预期

仅本地代码修改。部署后，已核实完整且本地 `symbols` 中存在有效 BNBUSDT 现价的已关闭周期，会显示绿色「BNB估算」和扣费后的参考盈亏；原始数据库依然为 `incomplete`。不会在此流程中尝试自动修改那 50 个服务器历史周期。
