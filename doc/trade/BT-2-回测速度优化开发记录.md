# BT-2 回测速度优化开发记录（待 Code Review）

> 日期：2026-11-19｜基线：`a23b59a`（BT-1 已提交）｜范围：本地 Go 回测代码。未访问生产数据库、未修改 `conf/app.conf`、未调用 Binance 交易接口。

## 已完成

1. **规则分析只计算一次**：`service/backtest/adaptive_engine.go` 的 `adaptiveRunState.prepareCloseProfiles()` 在策略快照初始化时分别解析 LONG / SHORT 退出规则，候选分钟复用 `analyzeCloseRuleROIProfile` 的结果；直接调用旧测试接口时保留分析回退。
2. **Intrabar 分钟前缀缓存**：新增 `service/backtest/intrabar_cache.go`，每个候选分钟只扫描一次已完成 1m K 线形成高周期前缀；每个 1s / Trade 仅与当前部分分钟合并。支持 `1m / 1h / 4h / 1d / 1w / 1M`、跨周期和回退时间顺序；缓存仅属于单次回测。
3. **复用安全指标缓存**：`adaptive` 开启现有标准模式的 `[1+]` 已完成指标缓存、K 线数据缓存及 Scratch；不重复实现指标公式。动态 `[0]` 指标仍调用原始 `calculateIndicatorValue`（实测当前 ADX 快速路径对秒级部分 K 线不能保证逐字段一致，故**没有启用**）。
4. **保持回归对照入口**：`Engine.disableAdaptiveOptimizations` 为包内测试开关；生产默认启用优化，不新增 UI 配置。
5. **不改语义**：`service/backtest/types.go` 的 BT-1 引擎和精度模型版本不变；BT-2 只降低计算开销，不改成交时间、手续费、滑点、止盈止损判定、资金费、Trades 优先级或持仓约束。

## 性能实测（合成内存行情，不代表完整回测）

平台：Apple M1 Pro、Go 1.25.0、darwin/arm64。使用相同指标配置与 4,800 根 1m 行情，`go test -bench`（按分钟内动态 ADX + 其他已完成指标访问测试）。

| 指标 | BT-1 原路径（Legacy） | BT-2 优化路径 | 变化 |
|---|---:|---:|---:|
| Intrabar 单次耗时（同轮 2s 基准） | **20,747 ns/op** | **13,376 ns/op** | **约快 35.5%** |
| 每次内存分配 | **28,456 B/op** | **13,320 B/op** | **减少约 53.2%** |
| 每次分配次数 | **73 allocs/op** | **43 allocs/op** | **减少约 41.1%** |

先前 BT-1 原路径独立采样约 18,796–19,753 ns/op（3 次）；BT-2 对照重复采样受系统负载影响，速度结果应以本表**同轮同配置**为准。策略 ROI 正则分析旧路径约 5.7–6.1 μs/次，现仅在回测初始化执行；该收益只发生于存在候选分钟的回测。

测试使用本地合成数据，**没有**包括历史行情下载、MySQL 查询、ZIP 解压、真实策略全量回测、结果落库等成本，不能推断端到端整体提速 35.5%。

## 回归与验收

- `go test ./service/backtest -count=5 -p=2`：**通过**。
- `go test ./service/backtest ./service/historicalmarket -count=1`：**通过**。
- `go vet ./service/backtest ./service/historicalmarket`：**通过**。
- `git diff --check`：**通过**。
- 新增 `service/backtest/bt2_parity_test.go`：逐字段比较原始/优化 Intrabar 环境，包含动态 `[0]` 指标、高周期边界、时间回退及 MarketCondition 分钟内变更；完整 Result JSON（Trades、Events、Equity、精度证据和指标统计）按 BT-1 路径与优化路径比较，要求字节完全一致。
- 新增 `service/backtest/bt2_performance_test.go`：原路径/优化路径并排基准及 ROI 分析基准。

## Review 重点与限制

1. **输出一致性是硬门槛**：新增测试覆盖有限的合成场景，建议独立审查时再用用户正式策略和本地固定历史样本做端到端前后对照。
2. **动态指标禁止近似替代**：ADX 在早期对比中出现秒级局部值不同，已回退到原计算；若未来想启用当前值快速路径，必须先补精确一致性证明。
3. **缓存仅限单回测**：高周期前缀与指标缓存没有跨实例共享，不改变线程隔离或持久化。磁盘/ZIP/MySQL 的读写优化归 BT-3。
4. **真实 Mark Price 和实盘调度差异仍保留 BT-1 限制**：BT-2 未尝试修正价格数据来源或模拟成交时点。

**结论：BT-2 本地开发与离线 Gate 完成，待独立 Code Review；未进行真实历史数据性能验收。**
