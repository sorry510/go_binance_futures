# go_bn_oracle1 历史结算误报警调查与本地修复

调查日期：2026-11-21

## 服务器数据库只读证据

本次只用 MySQL SELECT + ReadOnly 事务读取 go_bn_oracle1，未修改服务器数据库。

- UNIUSDT：周期 233，main/auto_strategy/SHORT，closed + incomplete，managed_qty=0；Managed 订单 2、Fill 3、REST 未核实 0、BNB 扣费 Fill 3。
- LITEUSDT：周期 243，main/auto_strategy/SHORT，closed + incomplete，managed_qty=0；Managed 订单 2、Fill 2、REST 未核实 0、BNB 扣费 Fill 1。
- 截至调查时 main/auto_strategy 共有 70 个 closed + incomplete 历史周期。

本地 managed_qty=0 不等于交易所最新零仓位证明；不可直接修改正式结算或风控。

## 根因

ReconcileOwner 持续回查已平仓、财务未结算的历史周期。交易所仓位快照不一定保留旧币种/方向；每轮额外精确快照查询限制为最多 6 个，因此部分历史周期无法证明新的零仓位。旧逻辑把这一财务待核实原因作为每 15 分钟一次、每个周期一次的报警 Warning，造成重复误报。

即使后续取得快照，非 USDT 的 BNB 手续费仍不能作为正式 USDT 结算证据，历史周期依旧是 incomplete。这是财务估值限制，不说明开平仓失败。

## 本地修复

1. 精确限制为 Main / auto_strategy、closed、incomplete、managed_qty=0。对原因为 symbol-scoped fresh position snapshot missing or ambiguous 的历史快照待核实，只保留 aged 不完整计数与已有汇总 Info，不再发出报警 Warning。
2. 对全部订单和 Fill 的 REST 核验、身份、数量平衡都通过，且非结算资产手续费只有 BNB 并确有正数 BNB 扣费的周期，正式结算仍被拒绝，但改为明确的 BNB FX 待核实分类，不再发出误报警。
3. 对真实仓位非零、待处理委托、缺少 REST 验证、成交缺失/不匹配、未知手续费资产以及 Lead/其他 Owner，保留报警和原有 15 分钟节流。
4. 仍然拒绝把当前价格估算用于正式结算、损失连亏计数、仓位调整。未调用交易所交易接口、修改服务器数据、重启服务器。

## 测试覆盖

- TestHistoricalClosedSnapshotPendingIsTrackedWithoutRepeatedAlarm：两种纯历史财务待核实原因无误报警、完整观察计数；真正异常、非目标账户和 Owner 仍有报警。
- TestVerifiedBNBFeeIsFinancialBacklogOnlyAfterFullFillProof：所有 REST/订单身份/成交数量证明通过才可能降级 BNB 警告；正式结算状态不变、风险配置不变；其他错误不降级。
- 全量 service/futuresownership、关联服务与控制器回归，以及 Go Race、go vet、构建复查。

仅本地变更；正式服务器必须部署新程序后报警行为才会变化。
