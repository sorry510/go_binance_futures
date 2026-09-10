# V3-3 Historical Backtest 流程说明

> 状态：✅ 已完成  
> 用途：长期查阅 V3-3 历史行情与正式回测的真实执行流程。

## 1. 总体架构

```text
Binance Futures REST / 外部历史数据
                  ↓
      Historical Market Repository
                  ↓
       全局历史行情数据库
                  ↓
           Dataset Builder
                  ↓
       Deterministic Engine
                  ↓
 Trade / Event / Equity / Metrics
```

V3-3 分为两个核心模块：

- `service/historicalmarket`：历史行情统一存储、查询、缺口补齐、外部导入和覆盖。
- `service/backtest`：构建 Dataset、逐 Bar 回放策略、计算成交和指标。

Backtest Engine 不直接访问 Binance，也不知道 Kline 实际存在哪张物理表。
## 2. Kline 的物理存储

Kline 按 interval 物理分表：

```text
market_klines_1m      market_klines_3m
market_klines_5m      market_klines_15m
market_klines_30m     market_klines_1h
market_klines_2h      market_klines_4h
market_klines_6h      market_klines_8h
market_klines_12h     market_klines_1d
market_klines_3d      market_klines_1w
market_klines_1mo     # 对应 Binance 1M
```

上层只能通过 interval 白名单路由，例如：

```text
5m → market_klines_5m
1M → market_klines_1mo
```

每张 Kline 表的业务唯一键为：

```text
market + symbol + open_time
```

因此同一 interval、同一市场、同一 Symbol、同一开盘时间只保留一条 canonical Kline。
## 3. 历史 Kline 如何产生

一次 Backtest 提交后，Dataset Builder 会先读取 Strategy Template 的 `technology` 和 `strategy`。

假设：

```text
Symbol: ONGUSDT
Execution Interval: 5m
Technology 使用: 1m / 5m / 15m / 1h
```

则需要：

```text
ONGUSDT: 1m, 5m, 15m, 1h
BTCUSDT: 5m
ETHUSDT: 5m
SOLUSDT: 5m
BNBUSDT: 5m
ONGUSDT Funding
```

其中 BTC/ETH/SOL/BNB 是固定 benchmark，使用 Execution Interval。

Repository 对每个需求先查询本地表，并计算应有的 Bar 时间点；本地完整则直接复用，不调用 Binance。
## 4. 本地缺口如何补齐

例如 5m 数据应存在：

```text
10:00 ✅
10:05 ✅
10:10 ❌
10:15 ❌
10:20 ✅
```

Repository 只生成缺口：

```text
10:10 ~ 10:19:59.999
```

然后调用 `BinanceSource` → `GetHistoricalKlines()` 分页读取 Binance Futures REST。

Binance 单页最多读取 1000 根，系统持续翻页直到目标 `end_time`；只接受 `CloseTime <= end_time` 的已闭合 Bar。

返回数据统一转换为 Historical Market Kline，再走 `Repository.Import()` 写入全局表。
## 5. 外部历史数据导入

统一入口：

```text
POST /agents/historical-market/import
```

支持同时导入 Kline / Funding，单次最多 50,000 行。

Kline 可以省略 `market` 和 `close_time`：

- `market` 默认 `futures_usdt`。
- `close_time` 根据 `interval + open_time` 自动计算。
- `quote_volume`、`trade_count`、taker volume 可以为 0。

所有来源最终都进入同一个 Repository，因此 Binance REST 和外部导入使用完全相同的校验、表路由和 UPSERT 逻辑。
## 6. 数据覆盖规则

历史行情采用 **last-write-wins**：同一个唯一键再次写入时直接覆盖 OHLCV、成交量、来源等字段。

MySQL 使用：

```text
INSERT ... ON DUPLICATE KEY UPDATE
```

不保存 Revision，也不要求 Dataset 永久绑定旧行情版本。

因此如果外部数据后来修正了一根 Kline：

```text
close: 100 → 105
```

下一次 Backtest 会使用 105。旧 Run 的结果不会被修改，但同一个 Dataset Spec 重新运行允许得到不同结果。
## 7. Funding 与 Warmup

Funding 不按 interval 分表，统一存储在：

```text
market_funding_rates
```

唯一键：

```text
market + symbol + funding_time
```

Dataset Builder 默认额外读取每个指标周期前 200 根 Warmup Bar：

```text
DefaultWarmupBars = 200
```

Warmup 数据会进入全局历史行情仓库并参与指标计算，但不会计入正式回测收益区间。
## 8. Dataset 与 Hash 语义

Dataset 现在是“历史数据查询规格”，不是 Kline 副本。

`dataset_spec_hash` 由以下规格生成：

```text
market / symbol / execution_interval
需要的 indicator intervals
benchmark symbols
start_time / end_time / warmup_start_time
```

相同规格会得到相同 `DatasetID` 和 `DatasetSpecHash`。

`data_hash` 则对本次 Run 实际读取到的全部 Kline + Funding 计算 SHA256。

因此允许：

```text
Run A: DatasetSpecHash = AAA, DataHash = 111
覆盖一根历史 Kline
Run B: DatasetSpecHash = AAA, DataHash = 222
```

这用于审计“结果变化是否来自历史数据变化”。
## 9. Backtest Engine 执行时序

Engine 进入运行阶段后不再访问 Binance、WebSocket、当前 `symbols` 或当前 MarketCondition。

核心时序：

```text
Bar close
  ↓
计算当时可见指标/环境
  ↓
执行策略表达式
  ↓
signal
  ↓
下一根 Execution Bar open 成交
```

因此不能用某根 Bar 的收盘信息按同一根 Bar 的开盘价成交。

自定义策略的止盈/止损参数采用与真实交易和模拟盘一致的**杠杆毛 ROI Gate**语义，而不是价格涨跌幅，也不是独立的强制保护单：

```text
ROI = unrealizedPnL / (abs(quantity) * markPrice) * leverage * 100

ROI 仍在 (-Loss, +Profit)
  → 不评估 close_long / close_short

ROI <= -Loss 或 ROI >= +Profit
  → 评估对应 close_long / close_short
  → 规则 true 才产生 close signal
  → 下一根 Execution Bar open 成交
```

`Profit/Loss = 0` 时，与真实交易保持一致，按 10000% 的内部门槛处理，等价于日常行情下关闭该 ROI Gate。例：Entry=100、Leverage=10、Profit=10%，LONG 在价格约 101.01 时毛 ROI 已接近/达到 10%；但这只代表允许评估 `close_long`，并不代表一定立即平仓。

线上真实交易约每 2 秒按实时 Mark Price 检查门槛，模拟盘按当前行情检查；Backtest 只能在 execution Bar close 检查。因此某根历史 Bar 的 High/Low 曾短暂越过门槛、但 Close 又回到门槛内时，Backtest 不会假设线上一定成交。这是历史数据粒度造成的执行模型差异。

### 9.1 单仓位模式

V3-3 Engine 当前采用**单仓位模式**，内部只有一个当前 `Position`：

```text
空仓
  ↓
评估 long / short
  ↓
开仓成交
  ↓
持仓期间只评估对应的 close_long / close_short
  ↓
平仓
  ↓
恢复空仓后才允许再次评估 long / short
```

因此当前明确不支持：

- 同方向加仓或金字塔加仓；
- 多笔并行持仓；
- LONG 与 SHORT 同时持有；
- 持仓期间因反向开仓 signal 直接反手。

策略平仓的典型时序：

```text
Bar A close: 产生 close_long signal
Bar B open : 平掉 LONG，position = nil
Bar B close: 已为空仓，可产生新的 long / short signal
Bar C open : 新仓最早在这里成交
```

若在 Bar B 内由 TP/SL 触发保护性平仓，则 Bar B 明确禁止重新开仓，避免仅凭 OHLC 无法确定 Bar 内先后顺序时产生不真实的“平仓后立即重开”。
## 10. 回测结果持久化

每次 Run 主要写入：

```text
agent_backtest_runs
agent_backtest_trades
agent_backtest_events
agent_backtest_equity_points
```

审计事件包括：

```text
signal / order / fill / position / funding
```

Metrics 包括 Net PnL、Return、Max Drawdown、Win Rate、Profit Factor、Sharpe、Sortino、Trade Count、Fees、Funding、Average Holding Time，并按 LONG/SHORT 与 MarketCondition 分组。

历史 Run 可通过 `DELETE /agents/backtests/:id` 删除。删除会在事务中清理对应 Trade / Event / Equity；如果 Dataset Manifest 已没有其它 Run 引用则一并删除 Manifest，但 `market_klines_*`、`market_funding_rates` 等全局 Historical Market Repository 数据不会删除。

## 11. 一句话流程

```text
Backtest 提出数据需求 → Repository 先查本地 → 缺口才调 Binance REST → Import/UPSERT 写全局历史仓库 → Dataset Builder 读取当前最新数据 → 计算 DatasetSpecHash/DataHash → Engine 逐 Bar 回放 → 保存 Trade/Event/Equity/Metrics
```
