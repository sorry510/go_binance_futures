# RG11 闭合小时主动压力交接与价格接受：完整失败研究

结果截至2026-10-04北京时间08:02:09。主85411、开78858、关47768、成本7854及会计均actual terminal exit0。目标尚未完成；本轮无联合门槛合格策略，不发布、不标记complete或主动paused/blocked。

## 结论与阶段总量

RG11四开发币频率均>=0.9次/周，BTC/ETH/XRP组合总净正、SOL净亏；ETH四个完整Sep–Aug年度净正，但日历2024仍亏，其余三币有亏损完整Sep–Aug年度。四币都有亏损完整日历年度，四年稳定/跨币联合资格失败。精确funding结算mark资格仍pending。合成及逐笔核验成功不等于盈利或发布通过。

本当前恢复阶段RG10+RG11新增24完整49月run、4186执行账目（含重复共享对照，不是4186个不重复市场交易）、四份完整JSON、两份收益前协议、全部合成/会计/入口/关闭/成本证据及两项新描述性诊断保存。16个共享对照完整重现；其中RG10作为RG11对照再次核对不是新独立策略证据。没有可发布版本。

## RG11 四币完整结果

UTC2022-09-01含至2026-10-01不含，1491天/213周，各初始1000 USDT，频率门槛每币至少192笔；原真实funding率及缺mark分钟Close回退均计入下表净值，不声称完整精确成本。

| 币 | 笔数 | 次/周 | 毛USDT | 净USDT | PF | 最大回撤 | 平均持仓小时 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | 233 | 1.094 | 587.151 | 373.287 | 1.233 | 16.348% | 22.614 |
| ETH | 270 | 1.268 | 719.206 | 452.728 | 1.202 | 21.084% | 13.157 |
| SOL | 282 | 1.324 | -37.764 | -245.292 | 0.889 | 38.973% | 8.541 |
| XRP | 216 | 1.014 | 287.182 | 105.153 | 1.059 | 30.687% | 10.201 |

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外2026-09 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTC | -121.088 | 108.322 | 1.192 | 406.550 | -21.690 |
| ETH | 38.160 | 22.721 | 192.402 | 295.641 | -96.196 |
| SOL | -271.792 | 142.446 | -124.758 | 5.448 | 3.363 |
| XRP | 86.397 | -176.504 | 114.529 | 84.656 | -3.926 |

| 币 | 日历2023 | 日历2024 | 日历2025 | 2026 Jan–Sep |
| --- | ---: | ---: | ---: | ---: |
| BTC | 136.166 | -21.124 | 121.965 | 264.947 |
| ETH | 151.608 | -22.871 | 191.820 | 223.304 |
| SOL | -18.428 | -176.985 | -119.651 | 84.516 |
| XRP | 1.932 | 113.026 | -276.229 | 247.352 |

年度/方向/入口族/45月cohort是actual exit-time归因，非独立初始化1000的另一收益；开发49月已观察，非time holdout。ETH改善不能事后只部署ETH或切换统计年度。AAVE/ATOM/ETC/LINK收益未读、资格待核。

## 描述性经验：压力交接仍无跨币优势

原RG10未来小时诊断支持提出新“自己的闭合入场压力交接”假设，但RG11必须完整新撮合，而不能继承事后分组盈利。此次实现无偏离，完整收益仍失败。

| 币 | 补充LONG笔/净USDT | 补充SHORT笔/净USDT | 静态扣除最佳5笔后组合净USDT |
| --- | ---: | ---: | ---: |
| BTC | 60 / -17.447 | 71 / -152.206 | 86.889 |
| ETH | 87 / -165.533 | 79 / 110.754 | 63.812 |
| SOL | 91 / -194.499 | 58 / -101.939 | -559.321 |
| XRP | 73 / -151.906 | 55 / -206.101 | -279.363 |

四币补充LONG均负、补充SHORT仅ETH正；不能删方向、按币选版本、反转亏损或静态删组当反事实收益。RG11实际574补充、427基础，不是用总1001减AF0独立429即可认定572补充：新增持仓改变后续基本入口可达性，BTC/XRP各少一个基础入口，现金路径也改变。

新单一自然强度诊断：latest closed directional net aggressive quote，是否严格大于previous8 closed opposing net quote/8。实际完整574入口，源主/原入口SHA及全交易身份核对、无future字段或阈值网格。代码曾因生成调用嵌套反引号解析失败，失败调用未写文件；改普通拼接后程序actual exit0，未改候选。

| 币 | 强于旧反向小时均值：笔/净归因 | 未强于：笔/净归因 |
| --- | ---: | ---: |
| BTC | 120 / -184.387 | 11 / 14.734 |
| ETH | 154 / -62.084 | 12 / 7.305 |
| SOL | 117 / -172.811 | 32 / -123.627 |
| XRP | 104 / -284.811 | 24 / -73.196 |

该强度组四币仍负，不据此生成一个加此过滤即可盈利的RG12。以上是旧真实账目的描述性归因，不是filtered/backtested/new-initialized/counterfactual收益，也不证明相反比较可交易。研究数据已观察，adaptive hypothesis不冒充未见验证。下一方向转向已有4h趋势中的回撤恢复：保留v29基础/原9配置/统一退出和全部门槛，先核相关旧完整配置，再冻结一个可在自己闭合入场时观察的机制；RG12候选当前尚未生成/测试收益。

## 实际实现与成本核验

- 12完整run/2122账目、8个AF0/RG10共享控制的全trades/metrics/annual/source/data identity重现，会计0错误。source唯一既定cache_hit差异只表采集或复用，非数据身份。全部规则snapshot/hash、gross/fee/net/frequency/年度/侧核对。
- 574实际补充/45920canonical闭合OHLC/quote/QPS/taker字段/1722日线值，失败0。exact entry_time−1/完整Expr/原输入；独立前8有效加权反向quote与价格、[2]反向→[1]同向严格主动多数、[1]量高于mean、closed Close1穿越High2或Low2±0.10ATR1、body/live hold、日线排除/closed弱ADX通过。200input/199closed原函数种子通过；ATR2/EMA及8小时extremes仅诊断，不新增为入场门槛。
- 1000正常退出/1 forced-end/0失败，AF0旧分支485/RG4弱小时新增519/both4/added-only515。完整统一RG4关闭原Position、actual exit_time−1、gross/current-mark分母ROI及外部5/5 gate核验。BTC最后一笔补充LONG为forced-end，净-3.7947189109258015，只作end_of_data删失/费用账目，不是表达式关闭证明；其他真实正常关闭逐笔核。
- 全2122笔/4019实际funding应用/1097缺mark分钟回退、0零活动fill/0算术失败；RG11自身1001笔/1612应用/450回退。原当前现金10%保证金×8、下分钟Open不利5bps、双侧0.0005fee、真实funding时点/normal entry inclusive exit exclusive/forced-end inclusive和原mark回退逐筆核对；精确结算mark仍欠，不能补零或造价格。
- fresh receiver不是全顺序缓存/private-selector/API/live/forward/独立指标数学/订单簿/容量证明；压力交接不是钱包或因果识别。均未修改生产、源、缓存、seed或容差。
- 原backtest_engine_v7/standard_1m、whole统一退出、AutoStopOrder=false、风险和所有门槛不变。四币public-canonical-repaired-v2-funding-tail-v1/verified-archive20261003-v2/data hash/helper/tail54 overlap7 manifest冻结保持。

## 保存证据与下一次恢复

两个完整RG11 JSON保留temp_strategy/20261004-closed-hourly-pressure-handoff/。族SHA760fc46eca907ec938cc385608df98b4eea4596d84a9718c20fba7c139f8366d/version025e602a2d11e548e69a92577ece9f1063ec0afc4bd4c003eb8213826e79b5f3；组合SHA0ed2cf7e64766d714b8c1fbcb76dd0030528ab38775c9eb2e23c2ed71d41c455/versionc07fca00571c1266b128007890fa6b92617b710009119d9d6e4a5488115c6dfc。唯一替换RG10两补充，原9配置/base及full统一RG4整对象不变；合成3890/0，不是市场盈利证明。收益前protocol2026-10-04-closed-hourly-pressure-handoff-protocol.md保持。

大输出目录 /Users/zhz/Library/Caches/go-binance-strategy-research/results/：

| 文件 | SHA256 |
| --- | --- |
| 20261004-rg11-development4-canonical-repaired-v2-funding-tail-v1.json | cbef8385a4312b517ac40fd2923bbba3d35e8a1758756b77f043f0109fe03b50 |
| 20261004-rg11-accounting-summary.json | 398ccb4937f9bd3d648a9abb7df2e3d52449bef4378349fc33e2429074c42876 |
| 20261004-rg11-expr-checks.json | 10f8526c9a371341da4243517d3d9f8b09dd6cf6b2190f1f8f02f938a0f8faf0 |
| 20261004-rg11-canonical-actual-seed-open-signal-audit.json | 2ceeba72fd35163f759e697ae221cf8a32b1d4db654cf3798cbc702886fb7430 |
| 20261004-rg11-execution-original-cost-fallback-audit.json | 56119706d94cdb8b56d1b3e231a93408850174a923b3bc17fba8e52b7abfc5e0 |
| 20261004-rg11-original-position-close-signal-audit.json | a498543b18e3b7a3273b7ba9fe07688afe37d432c9188f2789cbdb2a8114ae26 |
| 20261004-rg11-rejection-path-attribution.json | 8121245479e9894de142973751b575a84684179ac5c87267793719fd17e33f9b |
| 20261004-rg12-closed-pressure-strength-attribution.json | 2d2ba426866c344a14217e8f38d8fa2698600d75103b505c07b9ee953ce17a96 |

最新ARM仍07:04只读元数据/v29 full export，不冒称449条新forward结果内容已分析。无App/DB写入/分配/启用/下单/生产/前端/config/新仓库测试文件。临时Go/overlay/二进制/大输出仓库外，虚拟诊断源真实未创建；其他dirty研究保留。正式1.0.8技能/trusted:false及原pending、独立阶段总结草案不变；四行为案例仍pending/aggregate null，无委派/strictwin/promote。

08:05:23实际get_goal active，限定本轮主/审计/合成/路径诊断进程无残留；config/engine/environment/indicator_cache/helpers/正式技能/两个pending草案/四完整候选SHA均保持，三个overlay虚拟源真实不存在。恢复遵循2026-10-03-resume-checkpoint.md最新状态：下次第一条先阶段总结、再核实际goal及进程。不得poll已经terminal的85411/78858/47768/7854，或重复旧完整主回测。阶段完成不自行complete/paused/blocked；真实goal若paused则停止不自恢复。
