# V3-3 Historical Backtest 测试指南

> 状态：✅ 适用于当前最终 V3-3 实现  
> 目标：验证 Schema、历史行情生成/复用、外部覆盖、防未来函数、撮合和回测结果。

## 1. 推荐验收顺序

建议按以下顺序测试：

1. 检查数据库 Version 7 与历史行情表。
2. UI 跑一次最小 BTCUSDT 回测。
3. 检查第一次是否生成 Binance REST 历史行情。
4. 相同参数再跑一次，验证本地复用。
5. 查看 Dataset Spec Hash / Data Hash。
6. 查看 Trades / Events / Equity。
7. 运行 Repository 和 Backtest 核心自动测试。

不要为了人工验证缺口算法直接删除正式数据库中的历史 Kline；缺口测试已有隔离单元测试。
## 2. 检查数据库结构

进入后端：

```bash
cd /Users/zhz/work/binance/go_binance_futures
./go_binance_futures sync db
```

预期：

```text
database version is already up to date: 7
```

MySQL 检查：

```sql
SHOW TABLES LIKE 'market_klines_%';
SHOW TABLES LIKE 'market_funding_rates';
SHOW TABLES LIKE 'market_data_import_batches';
SHOW TABLES LIKE 'agent_backtest_%';
```

应能看到 15 张 Kline interval 表，以及 Funding、Import Batch 和 Backtest 相关表。
## 3. UI 最小回测

打开：

```text
AI → 历史回测
```

建议第一次使用：

```text
Strategy Template: 选择一个简单已有策略
Symbol: BTCUSDT
Execution Interval: 5m
时间范围: 最近 1~3 天
Initial Equity: 1000
Position Size: 1
Leverage: 1
Fee Rate: 0.0005
Slippage: 5 bps
Stop Loss ROI: 0
Take Profit ROI: 0
```

提交后状态应依次进入：

```text
queued → building_dataset → running_backtest → saving_results → succeeded
```

## 4. 验证第一次是否生成 Kline

例如回测使用 `BTCUSDT / 5m`：

```sql
SELECT
    COUNT(*) AS cnt,
    FROM_UNIXTIME(MIN(open_time) / 1000) AS min_time,
    FROM_UNIXTIME(MAX(open_time) / 1000) AS max_time
FROM market_klines_5m
WHERE market = 'futures_usdt'
  AND symbol = 'BTCUSDT';
```

检查来源：

```sql
SELECT source, COUNT(*) AS cnt
FROM market_klines_5m
WHERE market = 'futures_usdt'
  AND symbol = 'BTCUSDT'
GROUP BY source;
```

第一次自动补数据通常能看到 `source = binance_rest`。
## 5. 检查 Binance 自动补数据记录

```sql
SELECT
    batch_id,
    source,
    kind,
    status,
    total_rows,
    written_rows,
    invalid_rows,
    source_ref,
    FROM_UNIXTIME(created_at / 1000) AS created_at
FROM market_data_import_batches
ORDER BY id DESC
LIMIT 30;
```

第一次回测应能看到类似：

```text
source = binance_rest
kind   = kline
source_ref = BTCUSDT:5m:<start>-<end>
```

这可以确认 Repository 实际补了哪段缺失数据。
## 6. 验证第二次是否复用本地历史数据

第一次运行完成后先记录：

```sql
SELECT MAX(id) AS max_id FROM market_data_import_batches;
```

然后使用完全相同的 Strategy、Symbol、Interval 和时间范围再次运行 Backtest。

完成后检查：

```sql
SELECT *
FROM market_data_import_batches
WHERE id > <第一次记录的 max_id>
ORDER BY id;
```

如果本地 Kline/Funding 已完整，不应再次产生同一范围的 Binance Kline 下载记录。

如果仅扩大了回测时间范围，则只应补新增的缺失区间，而不是重新下载已有数据。
## 7. 验证不再获取 Benchmark 数据

Dataset Builder 只应读取目标 Symbol 的固定 1m、Technology 实际依赖周期和 Funding，不再因为回测额外拉取 BTC/ETH/SOL/BNB。

例如目标 `ONGUSDT` 且 Technology 使用 5m/15m/1h 时，只应准备：

```text
ONGUSDT: 1m, 5m, 15m, 1h
ONGUSDT Funding
```

选择包含 `MarketCondition` 的 Strategy Template 时：若目标范围没有足够的 `market_condition_histories` 覆盖，应提示先点击“补充 MarketCondition 历史”；补充完成且覆盖连续后，应允许“获取历史数据”和正式回测。回放时必须使用当前 Bar 时刻之前最近一条历史 MarketCondition，不能读取未来小时。

## 7.1 验证 MarketCondition 历史补充

点击“补充 MarketCondition 历史”后应看到后台阶段从查询上线时间 → BTC 1h → ETH 1h → 推断 → 保存。首次执行会补齐缺失 Kline；再次执行应主要复用本地数据。

检查 `market_condition_histories` 时，同一小时若原来已有真实记录，值和记录必须保持不变；推断任务只能插入空缺小时。BTC/ETH 只有在两者 1h Kline 均存在时才生成推断值。

## 8. 验证外部数据覆盖

可通过接口导入一根测试 Kline：

```text
POST /agents/historical-market/import
```

示例请求：

```json
{
  "source": "manual_test",
  "klines": [{
    "symbol": "BTCUSDT",
    "interval": "5m",
    "open_time": 1788192000000,
    "open": 108000,
    "high": 108500,
    "low": 107900,
    "close": 108300,
    "volume": 1234.56
  }]
}
```

重复导入同一 `open_time` 且修改 OHLCV，应覆盖原值；请只对明确的测试时间点操作，避免污染正式研究数据。
## 9. 验证 DatasetSpecHash / DataHash

同样的 Dataset 规格重复运行时，应满足：

```text
DatasetID          相同
DatasetSpecHash    相同
```

如果底层历史 Kline/Funding 被覆盖，则下一次 Run 应出现：

```text
DataHash           改变
```

UI 的 Backtest Detail 会展示 `Dataset Spec Hash` 和 `Run Data Hash`。

这意味着 Dataset 只表示查询规格，而 DataHash 表示该次 Run 实际消费的当前历史行情内容。
## 10. 验证 Trades / Events / Equity

打开一个 `succeeded` Run，至少检查：

```text
Summary: Net PnL / Return / Drawdown / Win Rate / Fees / Funding
Trades: Entry/Exit Time、Price、Side、Net PnL、Exit Reason
Events: signal / order / fill / position / funding
Equity: 资金曲线与 Drawdown 曲线非空
```

重点人工验证防未来函数：如果某根 5m Bar 收盘后产生 `signal`，普通策略开/平仓的 `order/fill` 必须出现在下一根 5m Bar 的 `open_time`，不能按当前 Bar 的 open 成交。

止盈/止损输入值是杠杆后的毛 ROI Gate，不是触线强平。例如 `Leverage=10`、`Take Profit ROI=10%`，LONG Entry=100 时价格约到 101.01 已达到 10% ROI 门槛；此时系统才开始评估 `close_long`，只有该规则为 true 才产生平仓 signal，并在下一根 execution Bar open 成交。止损同理。

人工测试至少验证三点：ROI 未过门槛时即使 `close_long/close_short` 表达式为 true 也不平；ROI 越过门槛但平仓表达式为 false 时继续持有；`Stop Loss/Take Profit = 0` 时与真实交易一样视为门槛关闭。Backtest 只在 execution Bar close 采样 ROI，不使用 Bar High/Low 猜测线上 2 秒轮询是否曾瞬时触发。
## 11. 核心自动测试

Repository 数据层：

```bash
cd /Users/zhz/work/binance/go_binance_futures

go test -count=1 \
  -run 'TestRepositoryLastWriteWins|TestRepositoryReusesCompleteLocalRange|TestRepositoryFetchesOnlyInternalGap|TestKlineTableWhitelist' \
  ./service/historicalmarket
```

它们分别验证：

- 后写数据覆盖旧数据。
- 本地完整时 Source 调用次数为 0。
- 只补中间缺失区间。
- interval 动态表名只能使用白名单。
Backtest 核心语义：

```bash
go test -count=1 \
  -run 'TestSameDatasetSpecUsesLatestCanonicalMarketData|TestHistoricalEnvironmentCannotSeeFutureBars' \
  ./service/backtest
```

它们验证：

- 相同 Dataset Spec 在历史数据覆盖后保持 DatasetID/SpecHash 不变，但 DataHash 改变。
- 历史环境不能看到未来 Bar。

完整 Backtest fixture：

```bash
go test -count=1 ./service/backtest
```

覆盖 LONG、SHORT、ROI Gate、门槛未过不评估平仓、规则 false 不强平、0 门槛关闭、No Trade、Fees、Slippage、Funding 和确定性 Engine replay。
## 12. 完整发布级 Gate

需要重新做完整回归时：

```bash
go test -count=1 ./...

go test -race \
  ./service/historicalmarket \
  ./service/backtest \
  ./feature/api/binance \
  ./controllers \
  ./command \
  ./models

go build -o go_binance_futures .
```

前端：

```bash
cd /Users/zhz/work/binance/go_binance_futrues_new_ui
pnpm typecheck
pnpm build
```

最终应确认没有遗留测试服务，且不要在测试过程中删除正式历史行情来制造缺口。