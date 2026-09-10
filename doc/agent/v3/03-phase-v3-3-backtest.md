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

Dataset 不再复制历史 Bar/Funding，只保存查询规格：Symbol、固定 1m Replay Interval、策略依赖 intervals、benchmark、起止时间和 warmup 范围。

- `dataset_spec_hash`：查询规格指纹，相同规格复用同一个 Dataset ID。
- `data_hash`：某次 Backtest Run 实际读取到的 Kline/Funding 内容指纹。
- 同一个 Dataset 以后重跑允许因全局历史数据被更新而产生不同结果；新的结果以最新 canonical 数据为准。
- 若结果发生变化，可用不同 `data_hash` 明确判断是否由历史数据变化导致。
- Strategy 快照仍绑定 `StrategyVersion`，执行逻辑绑定 `EngineVersion`。

## 执行与时间语义

- 新建 Backtest 的回放主时间轴固定为 `1m`，API/UI 不再允许选择 Execution Interval；Technology 中各指标仍按自身 `kline_interval` 读取历史 Kline。
- Engine 运行期间不访问 Binance、实时 WebSocket、当前 `symbols` 或当前系统 MarketCondition。
- 每个执行 Bar 只允许访问 `close_time <= 当前 Bar close_time` 的历史数据。
- Bar close 产生策略 signal，最早在下一根 execution Bar open 成交，防止同 Bar 未来函数。
- 当前 Engine 采用**单仓位模式**：同一时刻最多只有一个 `Position`，已有仓位时只评估对应的 `close_long` / `close_short`，不会继续评估新的 `long` / `short` 开仓规则。
- 因此必须先平仓后才能再次开仓；V3-3 不支持加仓、金字塔、多笔并行持仓、LONG/SHORT 同时持有或持仓中直接反手。
- 策略平仓信号在 Bar close 产生、下一根 Bar open 平仓；该下一根 Bar 收盘时已为空仓，因此可以产生新的开仓 signal，并最早在再下一根 Bar open 成交。
- `StopLossPct` / `TakeProfitPct` 表示**杠杆后的持仓 ROI 触发门槛**，与真实交易/模拟盘的 `nowProfit` 语义一致，不是标的价格直接涨跌百分比。三者统一使用 `ROI = unrealizedPnL / (abs(quantity) * markPrice) * leverage * 100`；手续费和 Funding 不参与这个毛 ROI 触发判断。
- 对自定义 Strategy Template，ROI 门槛不是独立的‘触线强平单’：ROI 仍处于 `(-StopLossPct, +TakeProfitPct)` 区间时不评估 `close_long` / `close_short`；越过门槛后才评估对应平仓规则，规则为 true 才产生平仓 signal。`0` 与真实交易一致表示门槛基本关闭（内部统一等价 `1,000,000%`）。
- Backtest 在 execution Bar close 计算 ROI 并评估平仓规则，满足后最早下一根 Bar open 成交；不会用当前 Bar High/Low 模拟线上 2 秒级轮询中的瞬时触发，因此高低点只在 Bar 内短暂越线但收盘恢复时可能与实时执行不同，这是历史回放粒度差异，不是未来函数。
- 最后一根 Bar 仍有仓位时以 `end_of_data` 确定性平仓。
- `backtest_major_regime_v1` 仅用于回测历史分组，不冒充线上全市场 Market Regime。

## API / UI

- `GET/POST /agents/backtests`
- `GET /agents/backtests/:id`
- `DELETE /agents/backtests/:id`：删除 Run 及其 Trade/Event/Equity；无其它 Run 引用时同时删除轻量 Dataset Manifest，不删除全局历史行情缓存。
- `POST /agents/backtests/prefetch`：按 Strategy Template + Symbol + 时间范围预取回测需要的 1m、指标周期 Kline、benchmark 1m 与 Funding；包含 warmup。
- `GET /agents/backtests/prefetch/:jobId`：查询预取进度和补齐结果。
- `POST /agents/backtests/:id/cancel`
- `GET /agents/backtests/:id/trades|events|equity`
- `POST /agents/historical-market/import`：外部 Kline/Funding canonical 导入。
- Web 新增 AI → 历史回测：创建任务、手动“获取历史数据”、进度、结果详情、Equity Curve、Trades、Audit Events、分组指标和两次结果简单对比。

## 验收 Gate

- LONG、SHORT、止盈/止损 ROI Gate、无交易固定 Fixture；必须验证‘越过 ROI 门槛才评估平仓规则、规则 false 不强平、0 表示门槛关闭’。
- 手续费、Funding、滑点均有单元测试。
- 明确验证未来 Bar 不可见。
- 同一输入内存 Dataset/Strategy/Engine 重放必须确定性。
- 相同 Dataset Spec 在全局 Kline 被覆盖后：Dataset ID / Spec Hash 不变，Run `data_hash` 必须变化。
- 完整本地历史范围不得请求 Binance；内部缺口只请求缺失区间。
- Backtest 不修改真实策略、旧模拟盘状态或真实 Binance 账户。

## 本阶段不做

- 不做遗传算法/大规模参数优化。
- 不让 LLM 在回测循环里逐 K 线做决策。
- 不做 Candidate/Active/Promote 生命周期；个人项目采用“新策略新建 Strategy Template”，历史 Run 依靠完整 Snapshot 保持可复现。
- 不做历史 Kline Revision/Watermark；历史数据采用 latest canonical value。
