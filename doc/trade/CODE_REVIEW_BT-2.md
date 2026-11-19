# Code Review：BT-2 回测速度优化

> 审查对象：BT-2 工作区改动（基线 `HEAD a23b59a feat: bt-1`，BT-1 已提交）
> 审查依据：`doc/trade/BT-2-回测速度优化开发记录.md`、`doc/trade/回测引擎优化方案-精简版.md` 及复审报告
> 审查方式：隔离副本 `/tmp/phase_audit_bt2`（1220 文件 / 798 Go，SHA256 全等）+ 全套离线 Gate + 独立审计探针
> 未改用户业务代码、未提交、未跑 `sync db`、未连真实 Binance、未写生产库。

---

## 一、结论

**BT-2：✅ 源码 / 离线 Gate 通过，可进入 BT-3 开发；真实历史数据性能与端到端收益未验收。**

- 无 P0/P1 Finding。三项优化（规则分析预热、Intrabar 分钟前缀缓存、安全指标缓存复用）实现与方案 BT-2 边界一致，未发现语义变更。
- 「纯性能优化逐笔一致」硬门槛经开发方 parity 测试 + 本次独立探针双重验证：full Result JSON（Trades / Events / Equity / 精度证据 / 指标统计）字节一致。
- 性能收益实测为正向，但仅在合成内存行情下成立；不得推断端到端回测提速。
- 3 项非阻断观察项（O1～O3），纳入后续阶段或生产 Gate 处理。

**允许**：进入 BT-3（存储/IO）。**禁止**：据本轮宣称真实历史数据性能达标、生产库或实盘通过。

---

## 二、改动范围与等价性

| 文件 | 类型 | 用途 |
|---|---|---|
| `service/backtest/adaptive_engine.go` | M | `adaptiveRunState` 增加 `closeProfiles[2]` 预热；`evaluateROIClose` 改用 `state.closeProfile()` |
| `service/backtest/engine.go` | M | `disableAdaptiveOptimizations` 包内对照开关；adaptive 模式启用标准指标缓存 + 预热 profile |
| `service/backtest/environment.go` | M | `BuildIntrabar` 双路径；优化路径注入缓存 overlays 并保护 `tickerStatsCache` / `minuteCloseCurrentAsOf` / market condition 状态 |
| `service/backtest/indicator_cache.go` | M | adaptive 下动态 `[0]` 指标禁用 EMA 等快速路径，强制走 `calculateIndicatorValue` 保守计算 |
| `service/backtest/intrabar_cache.go` | A | 每候选分钟扫一次已完成 1m 形成高周期前缀；1s/Trade 仅合并部分分钟；支持 1m/1h/4h/1d/1w/1M |
| `service/backtest/bt2_parity_test.go` | A | Intrabar 环境逐字段对照 + full Result JSON 字节对照 |
| `service/backtest/bt2_performance_test.go` | A | legacy/优化路径并排 benchmark + 规则分析 benchmark |

隔离副本 7 个改动文件 SHA256 与工作区逐一全等。

---

## 三、离线 Gate（全部通过）

| 检查 | 结果 |
|---|---|
| `git diff --check` | ✅ 通过（工作区） |
| gofmt（7 个改动文件） | ✅ 无未格式化文件 |
| `go vet ./service/backtest ./service/historicalmarket` | ✅ 通过 |
| `go build ./...`（88 包） | ✅ 通过 |
| `go test -count=1` 全量 88 包 | ✅ 76 ok + 12 no test + 0 FAIL |
| `go test ./service/backtest -count=5 -p=2` | ✅ 通过（4.614s） |
| `go test -race ./service/backtest ./service/historicalmarket -count=1` | ✅ 通过（LC_DYSYMTAB warning 非致命） |

Go 1.25.0 / darwin/arm64。macOS linker warning 已确认非致命，退出码 0。

---

## 四、核心声明实证

### 4.1 逐笔一致性（硬门槛）—— ✅

**开发方测试覆盖**：`bt2_parity_test.go` 四个 full Result JSON 对照用例（current-price / roi-only / no-candidate / dynamic-current）与 completing/forming 指标两组，另含 Intrabar 环境逐字段对照（分钟前缀、跨周期、时间回退、MarketCondition 分钟内变更）。

**本次独立探针**（副本内，未污染工作区）：
- 复用多周期 fixture（1m+1h，1440 根），注入真实形态的 1s + Trades provider，构造三组规则：① 动态 `[0]` EMA/ADX + 已完成 RSI；② ROI 门槛 + 已完成 ATR；③ 纯已完成指标 + 做空。
- 结果：三组 full Result JSON 全部字节一致；探针 `-count=5` 稳定通过。
- **非空跑验证**：另设计数 provider 探针，实测产生 **570 笔交易、570 次 SecondBars 调用**——证明自适应高精度路径确实被触发，一致性结论非空比较。

**嫌疑排除**：parity 测试第 62-65 行删除 `KdjSimple` / `IsAsc` / `IsDesc` 三字段。实证这些是函数指针（`line.KdjSimple`、`utils.IsAsc/IsDesc` 均为 func），`reflect.DeepEqual` 对函数值恒为 false（另写小程序验证），删除是为规避不可比较而非掩盖数值差异。**不是 Finding。**

### 4.2 规则分析预计算 —— ✅

- `prepareCloseProfiles` 按 LONG/SHORT 各调用一次 `analyzeCloseRuleROIProfile`，`evaluateROIClose` 复用缓存结果；`profilesReady=false` 时保留 legacy 直调回退（覆盖旧测试接口）。
- 生命周期核查：`rules` 由 `StrategyJSON` 反序列化一次（engine.go:53-55），`adaptiveState` 每次 Run 新建并以同一份 rules 预热（engine.go:82/84），生产路径无缓存串味。独立探针验证 LONG/SHORT profile 分离正确、legacy 回退与直接分析一致。
- benchmark 实测：`BenchmarkBT2CachedRuleROIProfile` 142.5 ns/op vs `BenchmarkBT2RuleROIAnalysis` 6007 ns/op（约 42 倍），且缓存路径 0 allocs。开发记录「仅在初始化执行」的收益声明属实。

### 4.3 Intrabar 前缀缓存正确性 —— ✅

- `cachedIntrabarOverlays` 按 `partial.OpenTime` 变化时重建前缀，按 `asOf` 逐次合并当前部分分钟；支持 6 个周期，回退时间顺序时重建。
- 独立探针构造 600→599→600→601 的回退序列（含同分钟不同 tick），逐次与 legacy `buildIntrabarOverlays` 对照，全部一致——无陈旧前缀泄漏。
- 状态保护实证：`BuildIntrabar` 优化路径在 `builder.build` 前后保存/恢复 `overlays`、`tickerStatsCache`、`minuteCloseCurrentAsOf`、market condition 三个字段，避免分钟收盘滚动缓存污染秒内价格。开发方 parity 测试覆盖 MarketCondition 分钟内变更，本次探针亦覆盖。

### 4.4 动态指标保守策略 —— ✅

`indicator_cache.go` 在 `adaptiveOptimized` 下对 `!cacheable` 任务直接调用 `calculateIndicatorValue`，跳过当前条 EMA/RSI 等快速路径。开发记录说明「ADX 快速路径对秒级部分 K 线不能保证逐字段一致，故未启用」——这是正确的保守取舍，符合方案 BT-2「先验证缓存字段一致再复用」的要求。

### 4.5 性能实测（合成内存行情）—— ✅ 正向，但范围受限

| 指标 | Legacy | BT-2 优化 | 变化 |
|---|---:|---:|---:|
| Intrabar ns/op（同轮同配置） | 17,184 | 13,963 | 约快 18.7% |
| B/op | 28,456 | 13,443 | 约减 52.8% |
| allocs/op | 73 | 43 | 约减 41.1% |
| 规则分析 ns/op | 6,007 | 142.5 | 约 42 倍 |

本次独立实测（`-benchtime=200x -benchmem`）与开发记录方向一致。注意：开发记录表格 35.5% 的 ns/op 降幅来自其采样轮次，本次同轮重测为 18.7%；两者均为合成 benchmark，样本受系统负载影响，**应以同轮同配置对比为准**（开发记录第 23 行已作此声明）。不含下载/MySQL/ZIP/落库/真实策略全量成本。

---

## 五、观察项（非阻断）

- **O1（测试档位）**：开发方 parity 的 full Result 对照仅 4+2 个场景，虽本次独立探针补充了三组且确认非空跑，但仍是合成 fixture。方案验收第 36 行要求「固定相同代码基线、数据、策略和硬件」——建议 BT-3 交付后，在综合 Gate 用用户正式策略 + 本地固定历史样本做一次端到端 legacy/optimized 全量对照，作为长期回归基线上报。
- **O2（对照开关的测试可见性）**：`disableAdaptiveOptimizations` 是包内字段，仅同包测试可用；生产无 UI 开关（符合方案「不增加永久 UI 开关」）。记录在案，防止未来被误当配置项扩散。
- **O3（benchmark 数字口径）**：开发记录表格与本次复测的 ns/op 降幅不同（35.5% vs 18.7%），非矛盾而是采样差异。建议后续性能报告统一标注「同轮同配置」与样本数，避免跨报告直接比较百分比。

---

## 六、边界与放行

- **源码/离线/合成 benchmark 通过 ≠ 真实历史数据性能验收**：未接入真实历史行情、未跑 MySQL 8、未涉及 ZIP 解压与结果落库成本，不能推断端到端整体提速。
- BT-1 遗留限制继续有效：真实 Mark Price 归档未验证/未接入、开仓仍为 1m 信号+下分钟开盘、约 2 秒调度延迟未模拟。
- 未发现改动触及实盘交易流程、`conf/app.conf`、生产库或 `EngineVersion` / `ResolutionModel` 版本（BT-2 不改语义，版本不变符合预期）。
- **放行**：可进入 BT-3；BT-3 交付后另起独立 Code Review。

> 审查探针仅写在隔离副本，报告落盘后清理；用户工作区保持 BT-2 交付时的改动状态。
