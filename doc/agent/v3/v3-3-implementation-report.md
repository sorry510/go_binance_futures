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

`agent_backtest_datasets` 只保存 Dataset Spec：Symbol、Execution Interval、指标依赖 intervals、benchmark、时间范围和 warmup。

- `dataset_spec_hash`：Dataset 查询规格指纹。
- `data_hash`：本次 Run 实际读取到的 Kline/Funding 内容指纹。
- `StrategyVersion`：现有 Technology + Strategy snapshot hash。
- `EngineVersion`：Backtest Engine 版本。

原设计的 `agent_backtest_bars` 与 `agent_backtest_funding` 已取消，不再按 Dataset 重复存储行情。

## 4. Deterministic Backtest Engine

- Strategy 在 Bar close 产生 signal，最早下一根 execution Bar open 成交。
- 每个历史环境只能访问当前 close_time 以前的数据。
- 支持 LONG / SHORT、策略平仓、TP、SL、end-of-data close。
- 同 Bar 同时命中 TP/SL 时 stop-loss first。
- TP/SL 平仓后的同 Bar 不重新开仓。
- 计入双边手续费、方向滑点和 Funding。
- 生成 Trade、signal/order/fill/position Audit Event 和 Equity Curve。
- 输出收益、回撤、胜率、Profit Factor、Sharpe、Sortino、Fees、Funding、持仓时长及 Side/MarketCondition 分组。

## 5. API / UI

Backtest API：创建、列表、详情、取消、Trades、Events、Equity。外部历史数据通过 canonical import API 写入 Historical Market Repository。

Web 新增 AI → 历史回测，包含参数表单、异步进度、结果指标、Equity Curve、Trades、Audit Events、Side/MarketCondition 分组和两次成功 Run 对比。

ECharts 改为按模块引入后，Backtest production chunk 从约 1.05 MB 降至约 501 KB。

## 6. Database Version 7

实际 MySQL 已执行 `./go_binance_futures sync db`：Version `6 -> 7`，Schema Sync 成功；随后再次执行确认 Version 7 幂等。

## 7. Final Gates

- `go test -count=1 ./...`：通过。
- `go test -race ./service/historicalmarket ./service/backtest ./controllers ./command ./models ./feature/api/binance`：通过，无 data race；仅有既知 macOS linker warning。
- Historical Repository：last-write-wins、完整本地零 Source 调用、内部缺口仅补缺失区间、非法 interval 白名单拒绝均通过。
- Backtest Fixture：LONG、SHORT、TP、SL、无交易、手续费、滑点、Funding、未来 Bar 不可见、确定性 replay 均通过。
- 最新数据语义 Gate：相同 Dataset Spec 覆盖历史 Kline 后 Dataset ID / Spec Hash 不变，Run `data_hash` 改变。
- `pnpm typecheck`：通过。
- `pnpm build`：通过。
- `go build -o go_binance_futures .`：通过。
- 前端 `dist` 已同步部署到后端 `static`。
- `git diff --check` / `git diff --cached --check`：通过。
- `doc/TODO.md` 无修改。
- 未启动或遗留测试 Web Server。

## 8. V3-3 边界

没有实现 V3-4 Candidate / Paper / Active / Retired 生命周期，没有参数大规模寻优，没有 LLM 逐 K 线交易决策，也没有改变真实交易链路。
