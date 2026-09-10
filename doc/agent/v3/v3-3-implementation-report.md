# V3-3 Historical Backtest Engine 实施报告

## 1. 结果

V3-3 已完成正式 Historical Backtest Engine，并把历史行情从 Backtest 内部数据复制升级为独立的 Historical Market Repository。

核心原则：

- Backtest Engine 确定性逐 Bar 执行，不在循环中调用 LLM。
- 历史行情采用全局 canonical store，Kline 按 interval 物理分表。
- 相同时间槽的数据采用 last-write-wins，后写入数据覆盖旧 OHLCV/Funding。
- Dataset 只保存查询规格，不复制历史行情。
- 每次 Run 保存 `data_hash`，用于识别实际使用的数据内容。
- 同 Dataset Spec 重跑允许因历史数据更新而得到新结果。
- 新建 Backtest 的回放精度固定为 1m；Strategy Technology 的指标周期继续独立生效。
- 增加手动历史数据预取，Binance 历史 REST 分页请求统一限速，避免长时间范围补数据时形成突发请求。

## 2. Historical Market Repository

新增 `service/historicalmarket`，对上层隐藏物理分表和 Binance 补缺逻辑。

Kline 表：

- `market_klines_1m / 3m / 5m / 15m / 30m`
- `market_klines_1h / 2h / 4h / 6h / 8h / 12h`
- `market_klines_1d / 3d / 1w / 1mo`，其中 `1mo` 对应 Binance `1M`。

每张 Kline 表唯一键：`(market, symbol, open_time)`。Funding 使用 `market_funding_rates`，唯一键为 `(market, symbol, funding_time)`；`market_data_import_batches` 保存 Binance gap fill 和外部导入审计。

Repository 行为：

1. 先查询本地 canonical store。
2. 本地范围完整时不访问 Binance。
3. 存在缺口时只拉缺失区间。
4. 拉取结果 UPSERT 到全局表，再统一从本地返回。
5. 外部数据通过 `POST /agents/historical-market/import` 进入相同 Normalize / Validate / UPSERT 流程。
6. interval → 表名只能通过固定 whitelist 路由。

## 3. Dataset / Run

`agent_backtest_datasets` 只保存 Dataset Spec：Symbol、固定 1m Replay Interval、指标依赖 intervals、时间范围和 warmup。

- `dataset_spec_hash`：Dataset 查询规格指纹。
- `data_hash`：本次 Run 实际读取到的 Kline/Funding 内容指纹。
- `StrategyVersion`：现有 Technology + Strategy snapshot hash。
- `EngineVersion`：Backtest Engine 版本。

原设计的 `agent_backtest_bars` 与 `agent_backtest_funding` 已取消，不再按 Dataset 重复存储行情。

## 4. Deterministic Backtest Engine

- Strategy 在 Bar close 产生 signal，最早下一根 execution Bar open 成交。
- 每个历史环境只能访问当前 close_time 以前的数据。
- 支持 LONG / SHORT、策略平仓、止盈/止损 ROI Gate、end-of-data close。
- `backtest_engine_v6` 固定使用 1m 回放主时间轴，统一 `Profit/Loss = 0` 为 `1,000,000%`，并支持从 `market_condition_histories` 做无未来函数的 MarketCondition 历史回放；真实交易、模拟盘和 Backtest 共用 `DisabledFuturesROIThreshold`。旧 Run 保留各自 EngineVersion 和原结果，不回写历史记录。
- Backtest 的 ROI Gate 在 execution Bar close 采样，平仓 signal 仍按防未来函数规则在下一根 Bar open 成交；不使用 Bar High/Low 推断线上 2 秒级瞬时触发。
- 计入双边手续费、方向滑点和 Funding。
- 生成 Trade、signal/order/fill/position Audit Event 和 Equity Curve。
- 输出收益、回撤、胜率、Profit Factor、Sharpe、Sortino、Fees、Funding、持仓时长及 Side 分组；引用 MarketCondition 的策略从 `market_condition_histories` 读取历史值，不读取当前实时值。

## 5. API / UI

Backtest API：创建、列表、详情、取消、删除、Trades、Events、Equity，并新增历史数据预取创建/状态接口。预取只准备目标 Symbol 的 1m + Technology 依赖周期、Funding 和指标 warmup，不再获取 BTC/ETH/SOL/BNB benchmark；Repository 已完整的数据不会再次请求 Binance。删除 Run 时事务清理其 Trade/Event/Equity；对子表采用 `DELETE ... WHERE run_id = ?` 直接条件删除，避免 1m 长周期回测产生大量记录时 Beego ORM 展开超大主键 placeholder 列表；仅在无引用时删除 Dataset Manifest，Historical Market Repository 行情缓存不受影响。外部历史数据通过 canonical import API 写入 Historical Market Repository。

Web 新增 合约交易 → 历史回测，包含参数表单、异步进度、结果指标、Equity Curve、Trades、Audit Events、Side 分组和两次成功 Run 对比。

ECharts 改为按模块引入后，Backtest production chunk 从约 1.05 MB 降至约 501 KB。


- 回测页面新增 MarketCondition 历史补充任务：BTCUSDT/ETHUSDT 1h 从各自 Binance Futures 最早可用时间 local-first 补齐到当前已闭合小时，再基于 24h 方向/强弱/分化和 1h 波动确定性推断每小时 MarketCondition；同小时已有记录跳过不覆盖。
- Backtest Dataset 在策略实际引用 MarketCondition 时才加载历史条件，并将其计入 `data_hash`；Engine 只读取当前 Replay Bar 已经可见的最近历史值。

## 6. Database Version 8 / 9

Version 8 新增 append-only `market_condition_histories`，每次真实 MarketCondition 持久化时同时记录历史；7→8 升级还会清理现有 Strategy Template 中已废弃的 `BasicTrend` / BTC/ETH/SOL/BNB benchmark 变量。

Version 9 对应该历史表正式接入 Backtest：增加 BTC/ETH 1h 历史补充和 MarketCondition 回放语义。Version 9 不增加额外 Schema 字段，也不创建空的 `command/sql/version/9.sql`；数据库版本仍只通过 `./go_binance_futures sync db` 升级。实际 MySQL 已完成 `8 -> 9`，随后再次同步应返回 `already up to date: 9`。

## 7. Final Gates

- `go test -count=1 ./...`：通过。
- `go test -race ./service/historicalmarket ./service/backtest ./controllers ./command ./models ./feature/api/binance`：通过，无 data race；仅有既知 macOS linker warning。
- Historical Repository：last-write-wins、完整本地零 Source 调用、内部缺口仅补缺失区间、非法 interval 白名单拒绝均通过。
- 预取 Gate：强制包含 1m + Strategy 指标周期，不再获取 benchmark；预取完成后立即 Build 不产生第二次远程获取；含 MarketCondition 的策略要求目标范围历史 MarketCondition 覆盖完整，否则提示先补充。
- Binance 历史 Kline/Funding 分页请求共享全局节流器，请求起始间隔至少 300ms。
- Backtest Fixture：LONG、SHORT、杠杆 ROI Gate、门槛前不评估 close、规则 false 不强平、0 门槛关闭、无交易、手续费、滑点、Funding、未来 Bar 不可见、确定性 replay 均通过。
- 最新数据语义 Gate：相同 Dataset Spec 覆盖历史 Kline 后 Dataset ID / Spec Hash 不变，Run `data_hash` 改变。
- `pnpm typecheck`：通过。
- `pnpm build`：通过。
- `go build -o go_binance_futures .`：通过。
- 前端 `dist` 已同步部署到后端 `static`。
- `git diff --check` / `git diff --cached --check`：通过。
- `doc/TODO.md` 无修改。
- 未启动或遗留测试 Web Server。

## 8. V3-3 边界

没有实现策略 Candidate / Paper / Active / Retired 生命周期，没有参数大规模寻优，没有 LLM 逐 K 线交易决策，也没有改变真实交易链路。后续规划复审已取消 Strategy Lab：新策略直接新建 Strategy Template，历史 Backtest / Paper 依靠已保存的完整策略快照复现。
