# CODE_REVIEW_CM-4 · 币本位持仓/真实 Fill 周期汇总、inverse PnL、资产统计与钱包只读

> 日期：2026-11-18。工作分支 `feat/m-coin-futures`，数据库 Schema 仍为 **v27**。
>
> **本阶段只新增计算、查询和页面展示。未执行 `sync db`，未修改 `app.conf` 或实际数据库，未执行任何 Binance 下单/撤单/划转/仓位模式变更。**

## 一、需求与代码验收

| 需求 | 落地位置 | 结果 |
| --- | --- | --- |
| 独立 COIN-M Position/Order/Fill 数据源 | `service/coinmquant/ledger_summary.go`、`ledger_read.go` | PASS；只读 `coin_m_managed_*` 和 `coin_m_trade_fills`；一行一个 position_id 周期 |
| LONG/SHORT inverse PnL | `service/coinmquant/pnl.go` | PASS；`math/big.Rat` 精确计算，不将合约张数当币数量、不复用 U 本位线性算法 |
| 实际已实现盈亏 | `ledger_summary.go` | PASS；按位置关联的已记录 Fill `realized_pnl` 加总，校验 OPEN/CLOSE 方向、总开/平张数及结算币；不假设本地持仓字段一定可靠 |
| 未实现盈亏 | `feature/api/binance/coin_m_mark_price.go` + `ledger_summary.go` | PASS；专用 DAPI 公开 `GET /dapi/v1/premiumIndex` 标记价格，最新时间需处于 120 秒内；缺失则返回 `null`，不能从 `delivery_symbols.close` 偷换为成交价 |
| USD 参考换算 | `ledger_summary.go`、前端 `coinM/account.vue` | PASS；已实现部分按每笔 Fill 的实际成交价参考换算，未实现部分按当前 mark 估值；只是**估算展示**，不代表结算 USDT/USD |
| 筛选结果全量汇总 | `PositionSummaries` | PASS；汇总在分页之前完成，按 `margin_asset` 分组；不能把 BTC、ETH 直接相加 |
| 手续费和资金费 | `PositionSummary`、`AssetTotal` | PASS；手续费按原扣费资产单独分组；资金费无独立 Income 证据时明确 `funding_included=false`，不编造费用 |
| 异常证据显示 | `PositionSummary.PnLStatus` | PASS；`pending_fills`、`unverified_fills`、`invalid`、`unpriced` 和 `verified`，异常/无标记价时资产完整总盈亏为 `null` |
| 钱包只读查询 | `controllers/coin_m_quant_read.go`、`feature/api/binance/delivery.go` | PASS；JWT 验证后使用所选 COIN-M 环境的 **签名 GET /dapi/v1/balance**，8 秒超时；未配凭证或环境错误返回失败，不假装余额为 0、不回退主网 |
| 前端账户和受控交易页 | `src/views/coinM/account.vue`、`controlledTrade.vue` | PASS；钱包、持仓周期/明细、按资产全量汇总、费用、证据状态；参考 USD 开关默认关闭，必要时打开；受控交易页仍仅供查看 |
| 中文/英文 | `locales/zh-CN.yaml`、`locales/en.yaml` | PASS；结算资产、标记价格、收益口径、缺失证据提示 |
| 数据库升级 | v27 四表和既有 `delivery_symbols` | **不需要新增迁移**；CM-4 不修改 Schema，保持现有索引和 Fill 三元组唯一约束 |

## 二、关键算法与边界

### 2.1 按实际保证金币结算

设 N 是**合约张数**、C 是每张合约 USD 面值、Pe 是开仓价、Px 是对应标记/实际成交价：

- LONG 未实现盈亏（保证金币）= N × C × (1/Pe − 1/Px)
- SHORT 未实现盈亏（保证金币）= N × C × (1/Px − 1/Pe)

示例：BTCUSD_PERP，10 张、100 USD/张、50000 USD 开仓、55000 USD 标记价，LONG 浮盈约 `0.001818181818181818 BTC`。**不会**把它写成 100 USDT 的已结算收益。

已实现盈亏以真实 `coin_m_trade_fills.realized_pnl` 为权威证据。单独的手续费按 `commission_asset` 汇总，不把资金费虚构为 Fill。实际显示 USD 仅是换算估值，未实现用 mark，已实现用每笔成交的对应 `price`。

### 2.2 完整性判断

- `pnl_asset` 必须与所在 Managed 周期的 `margin_asset` 相符；Fill 的 order_id、position_id、symbol、positionSide、BUY/SELL 及 OPEN/CLOSE 意图需一致。
- 已确认开仓 Fill 张数需与 `opened_contracts` 相符，已确认平仓 Fill 张数需与 `opened_contracts - remaining_contracts` 相符。
- **部分平仓**以剩余张数和原开仓均价计算未实现盈亏。Fill 不完整、币种不匹配、数量异常时不提供完整总盈亏。
- **标记价格无效、过旧或暂时获取失败**时，将未实现盈亏、完整总盈亏留空；不会使用上一次成交价兜底。
- 新建但尚无任何成交的 `PENDING_OPEN` 周期显示为 `pending_fills`，不计入可验证收益。
- 聚合只计入已经具备完整 Fill 证据的周期；混合资产分别列示并提供缺失/异常计数。即使部分行可验证，如同一资产还有未核实记录，完整总盈亏保留 `null`。

### 2.3 钱包、ROI 和真实数据说明

- 钱包 `balance`、`availableBalance`、`crossWalletBalance`、`crossUnPnl` 均来自 DAPI 签名只读返回；**钱包不等于本地 Managed 持仓**，不自动归属任何人工持仓。
- **没有凭证、无权限或网络异常时，前端报告“钱包不可读取”，不展示假零余额。**
- v27 周期未存历史开仓初始保证金，因此暂**不计算收益率 ROI**，避免用当前策略杠杆虚构历史初始保证金；该字段需要后续在真实下单链路记录真实初始保证金后再验收。
- v27 表目前可能为空；**CM-5 尚未实现真实 userTrades / 私有 WS 导入**，所以完成 CM-4 不代表已经产生真实成交记录。
- **不是生产验收**：未在真实 MySQL 8、真实 DAPI 签名账户、COIN-M Testnet/Demo 或 ARM Linux 上执行交易链路验收。

## 三、测试记录

| Gate | 结果 |
| --- | --- |
| `go test -run '^TestCM4' -count=5 -p=2 ./service/coinmquant ./feature/api/binance ./controllers` | PASS |
| BTC 多空方向、价格上涨下跌、部分平仓、零价拒绝、精确反向计算 | PASS |
| Filter 全量汇总不依赖当前分页、按 BTC/ETH 区分结算、按 BTC/USDT 分开手续费 | PASS |
| 关闭周期按实际 Fill 成交价估值，未平仓按 DAPI mark 估值 | PASS |
| Fill 数量缺失、结算币错误、重复交易 ID、缺少 mark / 其他合约价格 | PASS |
| 纯公开 DAPI mark 接口 HTTP Mock、2 分钟有效性及主机隔离 | PASS |
| 签名余额 DAPI Mock、APIKey/签名、所选网络隔离 | PASS |
| 钱包无 JWT 请求直接返回 401，未进入 DAPI 请求 | PASS；`TestCM4WalletRequiresJWTBeforeAnyDAPIRequest` 连续运行 5 次 |
| 87 个非历史研究 Go 包 `go test -count=1 -p=2` | PASS |
| `go vet ./service/coinmquant ./controllers ./feature/api/binance ./routers` | PASS |
| `go test -race -count=2 -p=2 ./feature/api/binance ./service/coinmquant ./controllers` | PASS（macOS linker 有非致命 LC_DYSYMTAB warning） |
| `go build -o /tmp/go_binance_futures_cm4_v27 .` | PASS；没有替换运行程序 |
| 前端 `pnpm typecheck` | PASS |
| 前端 `pnpm build` + `dist/` → 后端 `static/` | PASS；最终树级 `diff -qr dist/ static/` 结果为 0 |
| `git diff --check` | PASS |

注：全部 Go 包不含独立历史 `strategy_templates/research/` 目录；该历史目录中存在已知重复函数编译问题，与 CM-4 无关。

## 四、CM-5 前仍不能宣称的能力

- **未实现** DAPI signed `userTrades` / 私有 User Data WS / 真实 Fill 写入、完全对账和自动异常恢复。
- **未实现** 真实 LONG/SHORT/CLOSE 开平仓、Algo 止盈止损、订单撤销、仓位模式修改或手动认领。
- **未实现** CM-6 策略调度/全局与逐币交易门禁的真实执行；不会因为 CM-4 页面显示收益就自行发出订单。
- **未验证** 生产 DB 的 v27 升级、真实账号权限、Testnet 和实盘交易。
- **未提供** 资金费的精确按周期 Income 归集，也不额外引入复杂审批/审计链路。

## 五、结论

**CM-4 在现有本地 Managed 账本只读展示、逆向盈亏计算、结算币分类及在线查询接口的代码与 Mock 测试范围内完成，可以进入 CM-5 开发。** 实际数据需后续 CM-5 正式 Fill 写入才会出现，不能混淆本地计算通过与实盘收益已验收。

---

## 六、首次复审后修复追踪（2026-11-18）

首次独立复审在 Fill 跨 symbol/positionSide 的归属关系中发现 F1/P1 缺陷，原报告作为历史记录保留。该缺陷已通过独立校验和永久测试修复；测试与复审证据见 [第二轮复审](CODE_REVIEW_CM-4-rereview-2.md)。真实迁移与实盘交易仍待 CM-5～CM-7 验收。
