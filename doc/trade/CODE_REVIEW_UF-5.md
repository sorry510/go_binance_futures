# CODE_REVIEW_UF-5 · 已核实的 Main 实盘策略效果统计

> 日期：2026-11-19。后端基线：`master` / `d2c2f28 fix: update trade`。关联 [UF 开发方案](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **验收范围：源码、SQLite 隔离回归、Go/前端构建。未连接生产 MySQL 8/ARM/Binance 主网或 Demo 进行真单验收。**
> 该阶段为只读统计，不参与交易、风险控制、Owner 认领、订单状态修复或结算写入。

## 一、交付范围

| 文件 | 改造 |
| --- | --- |
| `service/orderhistory/performance_uf5.go` | 新增 `Store.Performance`，使用 Main / `auto_strategy` 已关闭周期的 `closed_at` 7/30 天窗口；每批最多 100 周期读取关联 Managed 订单和真实 REST Fill；复用既有 `summarize` 审计证明（不通过 `project` 的历史人工展示覆盖、行情价和估值层），再加强精确 OPEN/CLOSE Fill 数量相等、已结算和真实时间顺序校验；仅纳入真实 verified 且同资产样本 |
| `controllers/orders.go`、`routers/router.go` | JWT 单操作员授权的 `GET /orders/performance`；仅能查询 Main 自动策略；严格的 `days`、币种、LONG/SHORT 和规则 Hash 输入校验；错误消息不泄露数据库和 Binance 参数 |
| `src/api/order.ts` | TS 类型和 `getVerifiedFuturesPerformance` 只读接口 |
| `src/views/order/PerformancePanel.vue`、`src/views/order/index.vue` | 现有「合约订单」中增加默认收起的统计面板：7/30 天选择、精确币种、LONG/SHORT、原始规则 Hash 筛选；展示已核实与排除样本、逐结算资产指标、币种/方向/规则归类以及排除原因；只在展开并主动查询时调用接口 |
| `locales/zh-CN.yaml`、`locales/en.yaml` | 中英文统计标题、定义、指标口径、未知规则和排除原因 |
| `service/orderhistory/performance_uf5_test.go`、`controllers/orders_performance_uf5_test.go` | 全量隔离证据矩阵、分批大于 100 周期、授权/非法参数、精确数字和资产隔离回归 |
| `doc/trade/开发方案-UF-U本位合约交易六项精简优化.md`、中英文 README | 阶段进度与使用说明 |

## 二、数据权威与统计口径

1. **Owner 和账户固定：** `account_id='main'`、`owner='auto_strategy'`。Lead、手动/Agent、新币抢购、资金费率、COIN-M 和 Spot 不参与，也不能通过 API 参数切换到其他 Owner/账户。
2. **结算事实：** 必须同时具备 `status='closed'`、`settlement_status='settled'`、已记录结算时间、余额为零、开/平真实 Fill 严格数量匹配、订单身份一致、全部 Fill `rest_verified=1`、有效盈亏/手续费数据，以及经核实一致的结算资产。已结算时间必须不晚于本次报告时间。
3. **历史认证 T0：** 保留已有 `BINANCE_ORDER_HISTORY_T0_MAIN_MS` 安全边界。没有批准的 T0 时，周期可计入排除理由，但不能被纳入「已核实」收益。不会根据当前日期自动猜测 T0 或私自改生产配置。
4. **利润：** 毛收益 = 所有真实 Fill 的 `realized_pnl` 之和，手续费 = 同结算资产的 `commission` 之和，**净收益 = 毛收益 − 手续费**。不对 BNB 等未知手续费币种编造 FX 价格；不同币种结算资产 **USDT / USDC 分别统计，绝不直接求和**。不包含资金费用；不以当前 `symbols.close` 估值替代已核实盈亏。
5. **样本数和指标：** 一个已结算周期算一笔，所有成交的部分 Fill 聚合为同一笔；胜率 = 净盈利周期数 / 已核实周期数。盈亏恰为 0 单独记为零收益；PF = 所有正净盈利之和 / 所有负净亏损绝对值之和，没有负收益样本时返回 `null`，不报告伪造无穷大的 PF。平均单笔净收益 = 已核实净收益 / 已核实周期数。
6. **回撤：** `settled_net_pnl_drawdown` 是按真实 `closed_at`、周期 ID 升序累加净收益后的峰谷**金额差**（初始峰值为 0）。**不是账户权益回撤、不是最大回撤百分比、也不含浮动收益。**
7. **策略版本：** 仅解析严格 `auto_strategy:<symbol>:<64位 SHA-256>` 格式的 `source_ref`；不符合时归为 `unknown`，不能伪装成当前模板。外部只返回原始规则 Hash，不泄露内部 `source_ref`。可以按同一个 Hash 跨币种汇总，同时保留单币种细分。
8. **日期口径：** 7/30 日采用 UTC 当前时间减去日历天，`closed_at >= start` 且 `closed_at <= now`；不是开仓日、历史列表当前分页、或上次编辑时间。报告响应附带 UTC 毫秒的窗口边界和生成时间。
9. **明确排除：** `legacy`、未结算、前 T0、成交不完整、非 REST 核实、资产不明/混合、身份不符、结算冲突、金额无效均独立累计排除原因数；未达标周期不进入胜率、PF 或资产合并净收益。人工历史状态/完整性覆盖**不改变**纳入资格。
10. **只读性能：** 先取账户/Owner/日期范围的周期，分批读取关联 Order/Fill 后使用内存 `big.Rat` 精确汇总。前端不会自动 2 秒轮询；UF-6 再处理订单历史性能和性能基准，不新增冗余表或生产索引迁移。

## 三、测试证据

| 永久测试 | 覆盖 |
| --- | --- |
| `TestUF5SettledPartialFeesMultipleAssetsSidesRulesAndDrawdown` | LONG/SHORT、盈利/亏损/零收益、精确费用、USDT/USDC 不合并、PF、胜率、净收益回撤、同一规则 Hash 跨币种统计、Main/Lead/Owner 隔离 |
| `TestUF5PartialOpenCloseFillsExactAndManualDisplayOverrideIgnored` | 一次 OPEN 的两次部分 Fill + 两个 CLOSE、手续费 0.8、真实净收益 109.2，人工修改历史状态与完整性后核实统计依然保持底层数字 |
| `TestUF5ExclusionReasonsDontTurnIncompleteIntoProfit` | Legacy、Incomplete、BNB/空手续费资产、REST 未验证、Fill 缺失、伪造 clientOrderId、结算收益符号、非零仓位、前 T0、实际关闭时间/结算处理时间不一致及未来结算时间 |
| `TestUF5ZeroFeeWithoutCurrencyIsAllowedOnlyWhenActuallyZero` | REST 明确零手续费可无手续费资产，非零但资产未知必须排除 |
| `TestUF5DateScopeUsesRealClosedAtAndExactBoundary` | 7/30 天 `closed_at` 边界，窗口开始包含、窗口结束包含、未来结束排除、按开仓日期不错误筛选 |
| `TestUF5FullLossNoSampleAndNeverInventPF` | 空样本不伪造 PF/净利，全部亏损时 PF=0，回撤金额正确 |
| `TestUF5FilterValidationAndHistoryActivationFailClosed` | 错误天数、币种、方向、Hash、取消查询、未激活/非法 T0 不得伪造通过 |
| `TestUF5SourceRefStrictRuleExtraction` | 只有准确 Owner/币种/64 位散列才输出规则 Hash |
| `TestUF5PreservesAcrossSumsAndNoDatabaseMutation` | 连续统计读取同一个底层交易簿，不新增/更新真实 Fill |
| `TestUF5BatchOver100CyclesNoMissingOrDoubleCount` | 105 周期的多批次查询，不少算/重复，校验金额、周期数和胜率 |
| `TestUF5RejectExactOpenCloseMismatchEvenWithinDisplayTolerance` | 故意制造极小 OPEN/CLOSE 数量差；即使历史页面容差可显示，也不能进入已核实实盘统计 |
| `TestUF5PerformanceJWTAndUntrustedFiltersRejectedBeforeDatabase` | 没有 JWT 为 401，天数/币种/方向/Hash 非法为 400，过滤参数验证在 DB 查询前完成 |

## 四、工程 Gate

| Gate | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestUF5' ./service/orderhistory ./controllers` | **PASS** |
| 五个 Go 关键包重复 5 次 | **PASS** |
| 88 个非历史研究 Go 包 | **PASS** |
| `go vet` 88 包 | **PASS** |
| 五个关键包 `go test -race -count=2 -p=2` | **PASS**（macOS 非致命 `LC_DYSYMTAB` 链接器警告） |
| `go build -o /tmp/go_binance_futures_uf5 .` | **PASS** |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| `dist/` 与后端 `static/` 文件一致 | **PASS** |
| `gofmt`、Prettier、`git diff --check` | **PASS** |

> 工程历史目录 `strategy_templates/research/` 有先前独立研究重复源码，按照 UF 原定流程排除；其余完整 Go 工程包全部测试、Vet 和编译。

## 五、未执行事项及阶段结论

- 真实 MySQL 8、ARM Linux、真实 REST 私有账户 Fill / 结算流程、USDT/USDC 实际生产记录均没有在本次进行验证；不宣称生产已完成上线。
- 结算和交易分支完全保持原样；不修改 `conf/app.conf`、不运行 `sync db`、不写生产数据库，不发送交易订单、不重启服务或提交 Git。
- 报告在请求时以数据库读取结果生成，可能与交易进行中下一次刷新结果不同；不加分布式缓存或假设事务内不存在后台交易数据变化。
- UF-5 在现有订单页单独放统计面板，旧的筛选全量收益（包含浮动估值）功能保持不动。两个指标的概念和纳入范围**不同**，不可直接对比为同一收益。
- **UF-5 所有源码、离线回归及前后端构建 Gate 已通过，可进入 UF-6 订单历史查询性能优化。生产环境验证尚未完成。**
