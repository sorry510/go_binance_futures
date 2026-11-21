# BT-8｜固定保证金回测、实盘币种参数默认值与下单数量精度

日期：2026-11-21

## 目标

现有 1m/自适应精度回测沿用 `cash × position_size_pct × leverage`，100% 仓位会随着累计盈亏复利扩张或收缩。而实盘合约 `feature/feature.go` 使用 `symbols.usdt × symbols.leverage ÷ entryPrice`，每笔使用固定配置的 USDT 保证金，不随账户余额自动放大。

本轮新增两种可选模式：**固定 USDT 保证金（前端默认）** 和 **历史资金比例（保持兼容）**。用户给出的 20 USDT、4x、止盈门槛 8%、止损门槛 6% 来自本地 BTCUSDT 配置示例，**没有写入 Go 引擎常量**。

## 代码变更

- `service/backtest/types.go`：`RunConfig` 增加 `position_mode`（`fixed_margin` / `equity_percent`）、`fixed_margin_usdt`、`quantity_step_size`。缺失模式兼容原 `equity_percent`，非法固定保证金或 StepSize 不允许静默转换到百分比仓位。
- `service/backtest/symbol_defaults.go`：从现有 `symbols` 读取 `usdt`、`leverage`、`profit`、`loss`、`stepSize`；只读，不改变实盘配置。禁止非法 symbol。
- `controllers/agent_backtest.go`、`routers/router.go`：增加 `GET /agents/backtests/symbol-defaults?symbol=BTCUSDT`，复用当前回测 API 认证。
- `service/backtest/store.go`：新回测保存仓位模式、固定 USDT、所选杠杆与止盈止损阈值；固定模式必须从数据库取得权威 StepSize（忽略客户端提交的 `quantity_step_size`）；旧历史记录以原比例模式展示。
- `service/backtest/engine.go`：原比例模式逻辑不变。固定模式开仓名义金额 = `fixed_margin_usdt × leverage`，数量按实盘的 `utils.GetTradePrecision(quantity, symbols.stepSize)` 取整；实际保证金与开仓手续费超出账户现金余额时跳过该次开仓并留下 `order skipped` 事件。盈亏继续进入账户余额，但不会改变下一笔固定保证金。1m 和 adaptive 共用同一开仓路径。
- `models/agent_backtest.go`：只新增 `agent_backtest_runs` 的 3 个快照字段：`position_mode`、`fixed_margin_usdt`、`quantity_step_size`。不动历史交易数据。
- `appversion/version.go`、`command/db_update_test.go`：DB 版本由 27 到 **28**，增加升级与重复升级的测试，使用项目现有的 `./go_binance_futures sync db` 执行 ORM 加列，无手工 SQL、无生产数据库写入。
- `src/views/ai/backtest.vue`、`src/api/backtest.ts`、`locales/zh-CN.yaml` 和 `locales/en.yaml`：回测资金设置增加「固定保证金 / 账户资金比例」切换；固定模式展示保证金输入和名义金额估计，百分比模式保留原字段；选择币种自动读取并允许手工覆盖杠杆、固定保证金及止盈止损门槛；快捷测试**切换币种时刷新默认值**，但点击**当前币种**不会覆盖已经手动修改的金额。回测历史参数弹窗展示实际模式与固定保证金。

## 保留不变的交易语义

- 固定 20 USDT 不代表“20 USDT 初始资金”，初始资金仍独立设置，例如 1000 USDT，用于总收益率、最大回撤和可用资金判断。
- ROI 触及 `stop_loss_pct` / `take_profit_pct` **仅解锁平仓策略判断**；配置数字可更改，必须等自定义平仓规则满足才实际平仓。
- Mark Price 与历史 Kline/Trades 估价差异暂不修改（用户明确忽略）。
- 成交时间/盘口/限价部分成交与现有费率、滑点模型保持不变，不引入额外延迟模型。
- 本轮仅复用实盘 StepSize **数量精度**；历史执行价格继续使用现有 OHLC/逐笔与滑点模型，不为了“对齐 TickSize”人为改变历史执行价格。
- 忽略最多 20 个并发仓位的共享资金调度问题，维持单币、单仓回测。
- 原来的 `equity_percent` 回测名义金额、ROI、权益和历史记录不变。

## 验证

- `go test ./service/backtest -run '^Test(FixedMargin|StartFixedMargin)' -count=3`：通过。
- 覆盖：连续盈利后保证金稳定、StepSize 数量取整、账户资金不足跳过、错误配置拒绝、旧模式结果不误用固定保证金、现有 symbols 默认值、DB 参数快照和后台忽略伪造 StepSize。
- `go test ./command -run '^TestSyncDatabaseInitializesAndIsIdempotent$'`：通过，SQLite 隔离测试验证 27 -> 28 和重复同步。
- `go test ./service/backtest ./controllers ./routers ./command -count=2 -p=2`：通过。
- `go test -race ./service/backtest -run 'TestFixedMargin|TestStartFixedMargin'`：通过。
- `go vet ./service/backtest ./controllers ./routers ./command`：通过。
- `go build -o /tmp/s510_fixed_margin_buildcheck .`：通过。
- `pnpm exec vue-tsc --noEmit --skipLibCheck --pretty false` 与 `pnpm build`：通过。

## 部署

仅在本地工作树写代码，**未修改 `conf/app.conf`，未操作生产数据、未重启交易服务**。

新后端启动前必须执行项目既有 schema 升级命令 `./go_binance_futures sync db`，让本地/目标环境具有数据库 **v28** 和新增列；必须先确认目标数据库连接配置，不能直接在生产环境运行。前端 `dist` 同步到后端 `static` 时保留 `platform-config.json`。

后端升级前，正在运行的旧二进制仍没有新 API，因此即使静态资源已更新，新页面也可能因接口不存在而无法启动回测。

## 文件隔离

未修改 `service/futuresownership` 或其他 AI 正在处理的订单逻辑；未合并、提交、重置任何既有未提交改动。
