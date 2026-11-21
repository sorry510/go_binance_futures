# go_bn_oracle1 实盘策略效果无数据：原因与只读展示修复

调查日期：2026-11-21。只对服务器 go_bn_oracle1 执行 SELECT，使用只读事务并回滚，未修改服务器数据库或实际交易。

## 一、服务器只读事实（UTC 2026-11-21 09:55）

| 指标 | 最近7天 | 最近30天 |
| --- | ---: | ---: |
| Main/auto_strategy 已平仓周期 | 39 | 70 |
| 正式 settled | 0 | 0 |
| incomplete | 39 | 70 |
| 非零 BNB 手续费的周期 | 39 | 70 |
| OPEN/CLOSE Fill 数量精确匹配 | 39 | 70 |
| 全部 Fill 经 REST 核实 | 39 | 70 |

Main USDM Fill 共 215 条，未核实 0 条；symbols 中 BNBUSDT 当前 close=526.20 USDT。数据库结果仅说明基础事实，展示资格仍有完整 Managed 订单/Fill 归属等检查。

## 二、空白原因

旧版实盘策略效果 UF-5 只统计正式 settled 的 Main auto_strategy 周期，其完整核实门槛排除所有 closed+incomplete 的 BNB 交易，故真实成交存在但统计面板空白。此前订单页已有 BNB 手续费当前价估算，尚未接入 UF-5。

## 三、本地修复

- 官方认证桶 assets、verified_cycles 仍只纳入原 settled 且满足所有认证条件的周期，原有安全门槛不变。
- 新增只读估算桶 estimated_assets、estimated_cycles，独立展示按币种、方向、原始规则 Hash 统计及交易数、胜率、PF、估算净收益和累计估算净收益回撤。
- 对 Main auto_strategy 已平仓、incomplete 且归属与成交数量精确匹配、Fill 经 REST 核实的 BNB 周期复用 applyBNBFeeEstimate，统一按照本地 symbols.BNBUSDT.close 当前价格折算 BNB 手续费；不访问交易所或历史 K 线。
- 无合法 BNB 价格、Fill 缺失、身份冲突、REST 未核实等情况均不得进入估算桶。全部资格检查都在后端进行。
- UI 明确将已核实与 BNB 估算分离，显示价格及行情时间，不再显示为空。
- 未修改 settled 认证交易的胜率、PF、PnL 统计；估算不写服务器数据库、不参与真实风控，也不回填 settlement_status、pnl_asset 或结算计数器。
- 仍以实际 closed_at 计算最近 7/30 天窗口；估算值会随本地 BNB 当前价变化。

## 四、回归覆盖

新增 service/orderhistory/performance_bnb_estimate_test.go，覆盖估算成功、无 T0、价格变化、无报价、REST 未核实以及正式结算与估算隔离；原 UF-5 测试仍需全部通过。执行前后端测试和前端构建，构建产物同步后端本地 static。未部署服务器，需后续发布新代码方能看到更新。
