# BT-10｜回测配置本地持久化与 Adaptive 动态平仓性能优化

日期：2026-11-21  
状态：本地代码、前端生产构建、同步及离线测试完成。未重启正在运行的后端；完整四年 BTCUSDT 回测仍待用户在新后端上执行。

## 1. 重新回测为何仍约 22 分钟

本地 `go_bn_test.agent_backtest_runs` 最新 BTCUSDT 策略 `web-20261120-030` 的三次高精度结果：

| 字段 | ID 583 | ID 584 | ID 585 |
| --- | ---: | ---: | ---: |
| 状态 | succeeded | succeeded | succeeded |
| 总耗时 (ms) | 1,310,189 | 1,312,161 | **1,311,571** |
| 固定保证金 (USDT) | 200 | 200 | **20** |
| ZIP 实际下载 | 0 | 0 | **0** |
| 1s 下钻分钟 | 11,822 | 11,822 | **11,822** |
| Trades 下钻秒 | 608,693 | 608,693 | **608,693** |
| 已解析分钟 Trades 命中 | 607,827 | 607,827 | **607,827** |
| ZIP/CSV 解析 (ms) | 248,459 | 256,454 | **328,156** |
| 稀疏 SQL (ms) | 53,725 | 53,296 | **53,929** |
| 引擎总耗时 (ms) | 1,306,170 | 1,308,164 | **1,304,901** |

ZIP 持久缓存已生效，瓶颈并非网络再次下载。重复解压、逐笔成交检查和 **动态平仓指标环境重建** 占据显著时间。BT-9 的部分局部 CPU 优化未改变总耗时量级，故本轮按实际动态 CLOSE 策略规则继续精简。

该策略的 `close_long` / `close_short` 以 `Position.EntryPrice`、`ROI`、`kline_1h.Close[0]` 和已完成的高周期 `[1+]` 指标组成；此前保守的 ROI-only 判断器发现 `Position` 和 `[0]` 就拒绝环境复用，导致达到 ROI 门槛的成交继续反复构建环境。

## 2. 后端实现

### 2.1 动态 CLOSE 快路径：只更新真正变化的字段

变更：

- 新建 `service/backtest/adaptive_forming_close.go`，静态识别可安全复用的规则依赖；
- 修改 `service/backtest/adaptive_engine.go`：只在完整历史分钟回放的 Adaptive 优化分支、静态依赖分析通过、未发生分钟内 MarketCondition 切换时，将该分钟的**已完成指标**构建一次；在每次秒级 / Trades 检查更新 `ROI` 和真实当前 `kline_1h.Close[0]`，复用原有 Expr 程序、平仓规则和 Fill 时序；
- 禁止复用：`Position.MarkPrice`、`Positions`、`NetROI`、`NowPrice`、`NowTime`、`FundingRate`、`MarketCondition`、其他 `[0]` 指标、未知字段、动态索引及未能证明不变的依赖；自动回退原来的完整 `BuildIntrabar`；
- 成交时间、价格、订单数量、止盈止损、Funding 结算、滑点、费率、精度证据生成与原值一致；不要简单跳过逐笔成交，也不改变固定 USDT 保证金模式。

实测针对性微基准（Apple M1 Pro）：

| Benchmark | 优化前完整环境 | 优化后静态环境+动态更新 |
| --- | ---: | ---: |
| 当前规则形态，每次满足门槛后环境+Expr | 4,656 ns/op | **270 ns/op** |
| 内存分配 | 3,236 B/op、34 allocs/op | **55 B/op、2 allocs/op** |

这是**局部 benchmark**，不能直接宣称 4 年回测提速 17 倍；因为仍需要逐秒 Trades 扫描、历史 ZIP 解析、SQL、完成分钟和其它环境计算。

### 2.2 避免已排序 Trades 重复排序

`service/backtest/adaptive_engine.go::ensureChronologicalTrades`：验证后的逐秒成交大多按 `TradeTime/TradeID` 有序；先线性比较，若乱序再用原排序。128 笔已排序成交的样本 **715 ns/op → 171 ns/op**，不改变事件顺序。

### 2.3 可归因的结果计数

`service/backtest/types.go` 的 `ResolutionStats` 增加：

- `trade_rows_visited`：参与当前秒处理的有效逐笔成交数
- `trade_gate_passes`：触及配置 ROI 平仓门槛的成交数
- `trade_env_builds`：无法复用而实际完整执行 `BuildIntrabar` 的成交数

统计保存在已有 `resolution_stats_json`，无需数据库迁移。页面结果同时显示这三项、`engine_ms`、`archive_parse_ms`、`sparse_query_ms`。下一次真实回测可据此确认快路径实际命中和剩余热点。

## 3. 回测页面配置缓存与 200 USDT 默认值

前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui/src/views/ai/backtest.vue`、`src/api/backtest.ts`、中文/英文 locales。

- `localStorage` 中已有 `go_binance_futures:backtest:form:v1`：保存策略选择、币种、日期区间、精度、资金模式、初始资金、保证金、权益比例、杠杆、手续费、滑点及 ROI 门槛；恢复时**优先保留用户缓存**，异步加载币种默认值不得覆盖已经保存或正在修改的值；
- 新建 `go_binance_futures:backtest:page:v1`：保存历史筛选、策略名称、创建时间、分页、对比选择与详情展示偏好；
- **无有效缓存时，单笔保证金默认为 200 USDT**；已有缓存（例如 20 USDT）优先保持原值，不能无声修改用户选择。主动切换其它币种时保证金初始为 200，其它交易门槛和杠杆仍从该币种表读取；
- 仅持久化 UI 的可编辑设置，不缓存整份历史成交结果、进度或帐号敏感信息。

前端已完成 Vue 类型检查、`pnpm build` 和 `rsync` 至后端 `static/`；保留 `static/platform-config.json` 原样不变。前端源代码其它代理已有并行改动，未重置、未提交，也未改变它们的业务逻辑。

## 4. 测试和验证

- `service/backtest/adaptive_trade_sort_test.go`：将乱序、同秒同 ID 时间排序结果与原版逐笔比较，多轮执行通过；
- `service/backtest/adaptive_forming_close_test.go`：严格 whitelist 及拒绝动态不安全引用；测试多个分钟内各时点 `kline_1h.Close[0]` 和策略是否与原环境相同；
- 使用完整 Engine 做同秒先盈利后回撤、逐笔撮合路径对照，**交易、事件、权益、净收益、数据及精度证据哈希全部一致**（仅新性能计数不同）；
- 从最新 ID 585 只读取原始 BTC `technology_json` 和 `close_long/close_short`：生成临时 Go 测试，验证两条真实规则均匹配安全快路径、在真实 1h/15m/4h 指标组合下的新旧逐时点表达式结果相同；运行后删除临时测试源；
- 测试构建仅写本地临时文件，不写测试库或生产库、不启动真实交易。

## 5. 后续验收与上线边界

1. 新后端优化**不会自动加载**进运行中的旧二进制。用户确认后重新构建、部署、重启本地后端；不直接覆盖当前运行中的可执行文件。
2. 再以 ID 585 相同 BTCUSDT、相同 20 USDT 固定保证金、4x、原策略原始区间和成本重跑，对比 ID 585 **全部 Trades、Events、Equity、净收益、Funding、DataHash**；只有完全一致才可判定该版本没有改变交易语义。
3. 读取新 `trade_rows_visited`、`trade_gate_passes`、`trade_env_builds` 与 `timing.engine_ms` 的数值，确定动态缓存实际覆盖多少次，实测整轮耗时才是性能验收标准。
4. ZIP 解析目前仍是约 **5分28秒**的实际历史消耗，新优化不会自动消除它。若后续仍慢，按本轮计数证据决定是否独立设计高精度持久解压缓存或结果复用，不引入不必要的数据库模型。
5. 未改变 `app.conf`、实盘配置、生产数据库；未做任何真实下单。新前端已经同步 `static`，新后端尚未重启。

**结论**：配置缓存和默认值已完成；针对 BTC 动态 CLOSE 的局部环境构建已有约 17× 微基准改善及实际规则等价性证据。**四年端到端加速百分比仍需实跑核实，不以微基准虚报。**
