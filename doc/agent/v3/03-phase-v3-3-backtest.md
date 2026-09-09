# Phase V3-3：Historical Backtest Engine

> 状态：✅ 已完成

## 目标

实现正式历史回测，不再把固定场景 Strategy Experiment 当作历史回测；历史行情统一进入可复用的 Historical Market Repository。

## 核心能力

- 基于历史 K 线和 Funding 驱动策略逐 Bar 确定性执行。
- 支持 indicator warmup，严格禁止未来函数。
- 支持 LONG / SHORT、策略平仓、止盈、止损。
- 计入双边手续费、Funding 和可配置滑点。
- 记录 signal / order / fill / position、Trade 和 Equity Curve。
- 输出 Net PnL、Return、Max Drawdown、Win Rate、Profit Factor、Sharpe、Sortino、Trade Count、Fees、Funding、Average Holding Time，并按 LONG/SHORT 和 MarketCondition 分组。

## Historical Market Repository

- Kline 按 interval 物理分表：`market_klines_1m`、`3m`、`5m`、`15m`、`30m`、`1h`、`2h`、`4h`、`6h`、`8h`、`12h`、`1d`、`3d`、`1w`、`1mo`（对应 Binance `1M`）。
- 上层只能通过 `service/historicalmarket` 访问，不能自行拼动态表名。
- 每张 Kline 表唯一键为 `(market, symbol, open_time)`；相同槽位采用 last-write-wins UPSERT，永远相信最新导入/拉取结果。
- Funding 统一存入 `market_funding_rates`，唯一键为 `(market, symbol, funding_time)`。
- `market_data_import_batches` 记录 Binance 补缺和外部导入审计。
- Repository 优先读取本地；本地范围完整时不调用 Binance，缺数据时只请求缺口并写回全局缓存。
- 支持外部 Kline/Funding 通过 canonical import API 写入同一仓库。

## Dataset 与可追踪性

Dataset 不再复制历史 Bar/Funding，只保存查询规格：Symbol、Execution Interval、策略依赖 intervals、benchmark、起止时间和 warmup 范围。

- `dataset_spec_hash`：查询规格指纹，相同规格复用同一个 Dataset ID。
- `data_hash`：某次 Backtest Run 实际读取到的 Kline/Funding 内容指纹。
- 同一个 Dataset 以后重跑允许因全局历史数据被更新而产生不同结果；新的结果以最新 canonical 数据为准。
- 若结果发生变化，可用不同 `data_hash` 明确判断是否由历史数据变化导致。
- Strategy 快照仍绑定 `StrategyVersion`，执行逻辑绑定 `EngineVersion`。

## 执行与时间语义

- Engine 运行期间不访问 Binance、实时 WebSocket、当前 `symbols` 或当前系统 MarketCondition。
- 每个执行 Bar 只允许访问 `close_time <= 当前 Bar close_time` 的历史数据。
- Bar close 产生策略 signal，最早在下一根 execution Bar open 成交，防止同 Bar 未来函数。
- 当前 Engine 采用**单仓位模式**：同一时刻最多只有一个 `Position`，已有仓位时只评估对应的 `close_long` / `close_short`，不会继续评估新的 `long` / `short` 开仓规则。
- 因此必须先平仓后才能再次开仓；V3-3 不支持加仓、金字塔、多笔并行持仓、LONG/SHORT 同时持有或持仓中直接反手。
- 策略平仓信号在 Bar close 产生、下一根 Bar open 平仓；该下一根 Bar 收盘时已为空仓，因此可以产生新的开仓 signal，并最早在再下一根 Bar open 成交。
- 已有仓位的 TP/SL 可由当前 Bar high/low 触发；同 Bar TP/SL 同时命中时使用保守的 stop-loss first。
- TP/SL 保护性平仓后的同一 Bar 不重新开仓，最早从后续 Bar 再判断开仓。
- 最后一根 Bar 仍有仓位时以 `end_of_data` 确定性平仓。
- `backtest_major_regime_v1` 仅用于回测历史分组，不冒充线上全市场 Market Regime。

## API / UI

- `GET/POST /agents/backtests`
- `GET /agents/backtests/:id`
- `POST /agents/backtests/:id/cancel`
- `GET /agents/backtests/:id/trades|events|equity`
- `POST /agents/historical-market/import`：外部 Kline/Funding canonical 导入。
- Web 新增 AI → 历史回测：创建任务、进度、结果详情、Equity Curve、Trades、Audit Events、分组指标和两次结果简单对比。

## 验收 Gate

- LONG、SHORT、止盈、止损、无交易固定 Fixture。
- 手续费、Funding、滑点均有单元测试。
- 明确验证未来 Bar 不可见。
- 同一输入内存 Dataset/Strategy/Engine 重放必须确定性。
- 相同 Dataset Spec 在全局 Kline 被覆盖后：Dataset ID / Spec Hash 不变，Run `data_hash` 必须变化。
- 完整本地历史范围不得请求 Binance；内部缺口只请求缺失区间。
- Backtest 不修改真实策略、旧模拟盘状态或真实 Binance 账户。

## 本阶段不做

- 不做遗传算法/大规模参数优化。
- 不让 LLM 在回测循环里逐 K 线做决策。
- 不做 Candidate/Active/Promote 生命周期；留给 V3-4。
- 不做历史 Kline Revision/Watermark；历史数据采用 latest canonical value。
