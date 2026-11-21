# BT-9：高精度 Adaptive 回测 CPU Profiling 与热路径优化

日期：2026-11-21  
状态：**本地代码与离线测试完成；真实四年 BTCUSDT 全量回测尚未复跑，不能据此宣称整轮提速倍数。**

## 一、问题及实测背景

BTCUSDT `web-20261120-030`、固定保证金 200 USDT、初始资金 10000 USDT、4x、TP/SL ROI 门槛 8%/6%，2023-01～2026-11，自适应精度连续两次：

| 指标 | ID 583 | ID 584 |
| --- | ---: | ---: |
| 全程耗时 | 1,310.189s | 1,312.161s |
| 引擎耗时 | 1,306.170s | 1,308.164s |
| 实际新 ZIP 下载 | **0** | **0** |
| 缓存 ZIP 命中 | 12,668 | 12,668 |
| 高精度秒线下钻分钟 | 11,822 | 11,822 |
| Trades 下钻秒 | 608,693 | 608,693 |
| 分钟 Trades 已解析帧复用 | 607,827 | 607,827 |
| ZIP 解析累计耗时 | 248.459s | 256.454s |
| 稀疏行情 SQL 查询耗时 | 53.725s | 53.296s |
| 净收益 | +155.597812 USDT | 完全相同 |
| 交易数 | 256 | 256 |

上述存量数据说明 **跨回测复用磁盘 ZIP 已生效**，后续时间主要是重新构建指标与平仓环境、执行逐笔回放和再次解压解析；不应将“下一轮 22 分钟”简单归因于重复下载。

## 二、Profiling 方法与证据

设备：本地 Apple M1 Pro、Go `darwin/arm64`。使用 `go test -cpuprofile` 在内存合成 K 线样本上做**可重复的小范围函数分析**，不运行真实策略下单、没有网络下载和生产 DB 写入。

- 原指标环境 benchmark：`BenchmarkBT2IntrabarIndicators`，原版 12,506 ns/op、13,320 B/op、43 allocs/op；
- 同样 benchmark、优化后：9,827 ns/op、13,320 B/op、43 allocs/op，单次耗时下降约 **21.4%**；为一次局部基准结果，非整轮回测改善率；
- 资金费范围 `BenchmarkBTProfileFundingScan`：旧 1,501 ns/op → 新 41.18 ns/op，局部约 **36.4 倍**；本项本身较短，整体收益不能按此倍数推断；
- 新增 `BenchmarkBTProfileDynamicClose`：使用与当前 BTC 策略一致的 **1h EMA/ADX/MFI/RSI/BOLL/Donchian/ATR、15m ATR、4h Supertrend** 指标组和 `Position.EntryPrice` / `kline_1h.Close[0]` 动态规则，优化后 **6,986 ns/op、6,406 B/op、46 allocs/op**。此新样本**没有相同条件的修改前数据**，不能用其与前一 benchmark 直接算提升百分比。

CPU 样本显示热点包括：

1. `historicalEnvironment.tickerStats`：原版每个秒/逐笔环境重新扫描过去 24 小时 1m K 线的 high/low/open/close；
2. `historicalEnvironment.addIndicatorsOptimized` 和 `build`：反复构造指标/环境映射及运行表达式；当前策略引用形成中 `[0]` 与 `Position`，不能直接启用“只更新 ROI”恒定环境快速路径；
3. `hasFundingInRange`：原版在每个候选 1s 区间从头线性扫描历史 Funding 列表；四年样本适合按已排序时间二分定位。

CPU Profile 二进制文件（供 `go tool pprof` 检查）：

- `doc/trade/profiles/BT-9/intrabar_before_cpu.pprof`
- `doc/trade/profiles/BT-9/intrabar_after_cpu.pprof`
- `doc/trade/profiles/BT-9/funding_before_cpu.pprof`
- `doc/trade/profiles/BT-9/funding_after_cpu.pprof`
- `doc/trade/profiles/BT-9/dynamic_close_after_cpu.pprof`

## 三、实际代码修改

### 3.1 已完成：Adaptive intrabar 24h Ticker 增量快照

- `service/backtest/ticker_intrabar_cache.go`：在同一当前 1m K 线内，缓存历史已完成 1m K 线的 24h 价格统计；每次依然用**当前传入的部分 K 线**更新 Close/High/Low/NowSymbolPercentChange。
- 基于实际 `asOf` 用二分搜索检查 24h 窗口起止；不符合预期/存在异常时间点变化时自动重建，以避免缓存跨分钟泄漏或使用未来价格。
- `service/backtest/environment.go`：仅在 `adaptiveOptimized` 的 `BuildIntrabar` 内应用上述结果。退出 intrabar 时清除当前 asOf 标记；原 1m、未优化测试引擎、策略信号及指标算法不变。
- 不改币种仓位、止盈止损规则、形成中高周期 K 线、滑点、手续费、Funding、DataHash/精度证据。

### 3.2 已完成：Funding 范围二分查找

- `service/backtest/adaptive_engine.go::hasFundingInRange`：对现有已排序历史 Funding 使用 `sort.Search` 判断 `[start,end]` 内是否存在 Funding。
- 不改变 Funding 的实际结算/记账逻辑、`advanceFunding` 时点、同秒止盈止损顺序或交易填单路径。

### 3.3 新增 Profiling 与等价性用例

- `service/backtest/adaptive_profile_test.go`：新旧 24h ticker 指标严格相等，覆盖分钟边界、回退访问、时间跨度、形成中 K 线；Funding 在 1 万个时间范围上与旧线性实现一致；新增针对性 Benchmark，便于后续 CPU Profiles。

## 四、测试与一致性

以下均已通过：

```bash
go test ./service/backtest ./service/historicalmarket ./controllers ./routers -count=2 -p=2
go test -race ./service/backtest -run 'TestProfile|TestBT2IncrementalIntrabarMatchesLegacy|TestBT2AdaptiveOptimizedResultsMatchBT1|TestBT2AdaptiveIndicatorsProduceIdenticalFullResults|TestAdaptiveV8TradeReplay|TestAdaptiveFunding' -count=1
go vet ./service/backtest ./service/historicalmarket ./controllers ./routers
go build -o /tmp/s510_highprecision_buildcheck .
git diff --check
```

检查范围包括高精度运行的环境与标准/未优化基线结果等价、资金费范围、逐笔先后顺序和已有完整结果对照。**这些测试不能代替同一 BTCUSDT 4 年真实历史样本的生产验收**；未开展 22 分钟完整回测，不能保证 256 笔真实交易与原始 `data_hash` 全部逐项匹配。

## 五、如何复现 Profiling

```bash
cd /Users/zhz/work/binance/go_binance_futures
go test ./service/backtest -run '^$' -bench '^BenchmarkBT2IntrabarIndicators$' -benchtime=2s -count=1 -cpuprofile=/tmp/bt9_intrabar_cpu.pprof
go test ./service/backtest -run '^$' -bench '^BenchmarkBTProfileDynamicClose$' -benchtime=2s -count=1 -cpuprofile=/tmp/bt9_dynamic_cpu.pprof
go test ./service/backtest -run '^$' -bench '^BenchmarkBTProfileFundingScan$' -benchtime=1s -count=1 -cpuprofile=/tmp/bt9_funding_cpu.pprof
go tool pprof -top /tmp/bt9_dynamic_cpu.pprof
```

注意：带 `-cpuprofile` 的 `go test` 可能在当前工作目录留下 `backtest.test` 测试二进制，请勿误提交；只清理明确是本次 Profiling 产生的该文件。

## 六、后续放行标准与限制

1. **先编译并重启本地回测服务** 才能让新引擎执行代码生效；本轮不重启正在运行的服务。
2. 重新运行完全相同的 BTCUSDT ID 584 条件（策略版本 `af22b6db...`、固定 200 USDT、初始 10000 USDT、4x、8%/6%、0.0005/side、5bps/side、相同区间）。对比完整 **256 笔 Trades、事件、权益曲线、Fees、Funding、NetPnL、DataHash 与 ResolutionEvidence**。结果必须严格一致，出现分歧须定位原因再发布。
3. 记录 `timing.engine_ms`、`archive_parse_ms`、`sparse_query_ms` 和实际下载次数，测量**真实整体改善**；上面的 21.4% 是 intrabar 函数微基准，不是 22 分钟缩短 21.4% 的承诺。
4. 对动态策略完整 CPU Profile 如果仍显示大量 `addIndicatorsOptimized` / 表达式计算，进一步优化时只能复用可证明相同的部分指标；`kline_1h.Close[0]` 和 `Position` 不能冻结。
5. 二次重复运行即使 ZIP 命中，也仍需约 250 秒重新解析缓存 ZIP；如需**同参数秒级复用旧回测**，必须单独设计具有可验证参数/数据/引擎指纹的结果复用功能，不能直接跳过数据变更检查。
6. 本次只更改本地回测源码、测试及本记录，不涉及现货/实盘交易、数据库迁移、`conf/app.conf`、生产数据库或前端 `dist`。

**结论：** 本轮解决了两个明确的 CPU 热点并提供可复现的 pprof 证据和等价性测试。真实 4 年端到端加速倍数仍为待验收，不能基于微基准直接宣称解决全部 22 分钟问题。
