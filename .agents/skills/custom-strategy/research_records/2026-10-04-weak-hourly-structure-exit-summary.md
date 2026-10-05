# RG4：弱趋势小时结构确认退出完整研究总结

结论：联合门槛失败。四币频率通过，但SOL/XRP净亏，四币负完整年；BTC/ETH的总净正不是稳定盈利。目标仍active，未读AAVE/ATOM/ETC/LINK验证收益，没有发布或启用。

## 完整结果

AF0/RG3/RG4×四开发币12完整49月run、3238笔，57075实际terminal exit0；8共享控制逐笔/metrics/年度/source/hash完整重现，会计0错误。UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔；原standard v7/1m、当前现金10%保证金/8倍、双边各fee0.0005/slip5bps、实际funding时点/率和原缺mark回退、外部5/5。新增关闭维度是策略变化，不是引擎或成本改变。

| 币 | 笔数 | 次/周 | gross | fee | funding | net USDT | PF | DD% | 去最佳5笔net |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BTCUSDT | 348 | 1.634 | 566.700 | 288.050 | -20.446 | 258.204 | 1.121 | 21.297 | -11.406 |
| ETHUSDT | 412 | 1.934 | 306.092 | 275.606 | -15.101 | 15.385 | 1.006 | 43.884 | -296.577 |
| SOLUSDT | 428 | 2.009 | -354.796 | 219.180 | -19.129 | -593.105 | 0.759 | 62.081 | -878.364 |
| XRPUSDT | 389 | 1.826 | 82.447 | 260.926 | -6.671 | -185.150 | 0.924 | 37.415 | -505.151 |

退出年度归因属于同一现金复利路径，不是逐年重新从1000初始化；保留四完整年和新增月份。

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 单独2026-09 |
| --- | --- | --- | --- | --- | --- |
| BTCUSDT | -77.087 | 28.735 | 35.264 | 359.881 | -88.589 |
| ETHUSDT | -131.242 | -217.428 | 179.710 | 228.359 | -44.015 |
| SOLUSDT | -360.737 | -11.451 | -68.848 | -123.976 | -28.094 |
| XRPUSDT | -91.330 | -217.074 | 218.189 | -73.465 | -21.470 |

| 币 | 日历2023 | 2024 | 2025 | 2026 Jan–Sep |
| --- | --- | --- | --- | --- |
| BTCUSDT | 72.319 | -60.587 | 158.562 | 165.973 |
| ETHUSDT | -107.688 | -147.247 | 138.502 | 230.487 |
| SOLUSDT | -145.978 | -311.070 | -119.182 | -16.403 |
| XRPUSDT | -89.322 | 75.732 | -139.576 | 68.085 |

45月cohort和所有方向/族归因保留在accounting-summary；不能把cohort、静态删除交易或某族盈亏当新初始化/新策略收益。

## 开仓、关闭与成本审计

40845开仓audit terminal exit0：1170补充/93600 canonical闭合字段，原200 input/199 closed seed的ATR/ADX/EMA、原RG3入口price/quote/taker/首次脱离/live缓冲全部符合；仍保留有意forming价格和量[0]，不读取forming taker0。

25659关闭audit terminal exit0：1576 normal/1 forced-end/0失败。使用exit_time−1已观察分钟、真实entry/quantity/fee/hash及只到signal的实际funding重建原Position/cash，原接收器gross mark-price-denominator ROI/外部gate、canonical上一闭合小时Low/High和原closed ADX窗口相符。原program true547、新branch true1039、重叠10、新branch-only1029，union1576；只是同一观测环境的条件归因，不是移除分支后的反事实利润，也不是实际私有short-circuit branch日志。

new-only按BTC/ETH/SOL/XRP分别226/279/262/262，其中正ROI66/67/52/61、负ROI160/212/210/201，弱ADX或ROI资格违规0。多数新正常关闭是在亏损资格处，但不能据此静态反向或删单。forced-end不算表达式确认。fresh receiver不证明全顺序cache/private selector/API/live/forward/独立指标数学或订单簿执行。

41271成本audit terminal exit0：3238笔独立现金复利quantity/fill/fee/7002实际资金费应用，0零活动填单或算术失败；1958 mark回退，RG4自身1577笔/2147应用/586回退。真实费率/时点覆盖不等于精确交易所结算mark，完整成本资格仍受限，不补零/制造精确mark/修生产runtime。最大账目净聚合误差2.3874235921539366e−12、gross0、fee4.440892098500626e−16。

新分支实际生效，不能把失败归因为它从未参与。它让机会和资金路径变化、部分回撤下降，但SOL/XRP仍亏及四币负年，收益和稳定性门槛没有解决；不是成功策略。

## 原始证据

全部大文件在/Users/zhz/Library/Caches/go-binance-strategy-research/results/；辅助Go/overlay在相邻verification/，完整两个候选保留temp_strategy/20261004-weak-hourly-structure-exit/，冻结协议及2806 Expr矩阵保留。

| 文件 | SHA-256 |
| --- | --- |
| 20261004-rg4-development4-canonical-repaired-v2-funding-tail-v1.json | bb15daaa139f83a3be2cfd52263841ae0a059b5974eb9a97a6c67b4becf2395d |
| 20261004-rg4-accounting-summary.json | 6cd3ce22dc5a42fd7fea361fed06ebda99a07982de87e8fb8534580aa472bc74 |
| 20261004-rg4-canonical-actual-seed-open-signal-audit.json | 9a9fedefbedf11f9f7cdb7cac3c8e03f37e7b021ef284128d236b1acedd7d530 |
| 20261004-rg4-execution-original-cost-fallback-audit.json | 5bc13b0adeb089932eaa0d0da5638e43daa226917c8532e41730ddd969129d49 |
| 20261004-rg4-original-position-close-signal-audit.json | c353d72b80fac1527f8c29fbfddd441974b35eaca0a372fc42b2a5bdb35d4bb1 |

## 下一步和保护

不继续调EMA/ROI阈值或挑币/年；下一轮预先冻结一个不同的量价确认维度：区间脱离之前8个闭合小时的累计主动quote方向，与本次闭合放量方向一致，检验是否缺少持续成交支持。先读相关旧入口/防精确重复，再保存完整RG5 JSON/矩阵/协议，在原全部门槛下全量重撮合。当前尚未生成或回测RG5，不把这个待检验假设当改善；开发失败不读未观察验证收益。

2026-10-04北京时间01:26:53实际get_goal=active，pgrep无本轮回测/审计进程。最新ARM metadata仍00:11:54.640/57.451/00:12:00.033，模板17/17/17、结果221/217/6及v29身份相同；不是444行内容新forward分析。config/engine/environment/cache/正式技能SHA相同；技能1.0.8/trusted:false，新receiver-seed/mark候选行为pending/aggregate null，未晋级。无App/DB写入/分配/启用交易/生产或前端/config/新仓库测试文件。

本阶段RG2/RG3/RG4共36完整run（含重复对照）/9516笔执行账目，不能称9516互不重复市场交易。恢复先阶段总结，再核对实际goal/handle/进程/冻结身份，不重复旧主、不自行暂停/complete/blocked。
