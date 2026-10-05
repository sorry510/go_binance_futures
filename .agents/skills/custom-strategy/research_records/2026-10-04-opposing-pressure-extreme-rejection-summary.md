# RG10 持续反向主动压力的闭合极值拒绝：完整失败研究

2026-10-04北京时间07:35:18核对，主92584、开仓35448、平仓55977、成本53961均 actual terminal exit0；会计程序亦exit0。限定进程检查无本轮残留。当前goal active，无联合门槛合格策略，不标记完整研究结束。

## 结论

RG10替换弱4h趋势补充为“此前8个闭合小时反向主动quote压力及价格运动，最新闭合小时继续该主动压力、突破极值却价格反向收回”的对称多空机制；仍保留AF0基础、原9配置及全部统一RG4退出。3482/0合成检查、真实信号及成本核验通过，只能确认实现符合事前机制，不能证明有效盈利。

AF0/RG7/RG10×BTC/ETH/SOL/XRP共12完整49月run，2064条执行账目（含共享对照，非2064个不重复历史交易）；RG10自身692笔。BTC/ETH/XRP频率失败，SOL净亏，四币均有亏损完整Sep–Aug年度及完整日历年度。联合资格invalidated，不发布、不挑币、不删方向、不放宽频率/年限/成本。

## 四开发币结果

UTC2022-09-01含至2026-10-01不含，1491天/213周，每币初始1000 USDT，至少192笔才符合0.9次/周。以下净值保留原真实funding率及缺mark分钟Close回退，未宣称完整精确成本。

| 币 | 笔数 | 次/周 | 毛收益USDT | 净收益USDT | PF | 最大回撤 | 平均持仓小时 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| BTC | 165 | 0.775 | 367.340 | 224.665 | 1.189 | 21.223% | 26.504 |
| ETH | 169 | 0.793 | 518.325 | 360.364 | 1.235 | 17.118% | 16.410 |
| SOL | 198 | 0.930 | 71.424 | -89.506 | 0.953 | 29.349% | 10.532 |
| XRP | 160 | 0.751 | 684.158 | 526.694 | 1.329 | 23.805% | 10.930 |

RG10组合与共享RG7净收益相比，BTC/ETH下降、SOL亏损缩小、XRP增加；不是四币同时改善，不能因此放行XRP或事后每币选择版本。

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外2026-09 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTC | -114.149 | 3.638 | -7.354 | 405.449 | -62.919 |
| ETH | 0.698 | -33.457 | 214.857 | 215.026 | -36.760 |
| SOL | -233.164 | 84.166 | 30.978 | 55.267 | -26.752 |
| XRP | 178.209 | -191.276 | 223.682 | 226.507 | 89.572 |

| 币 | 日历2023 | 日历2024 | 日历2025 | 2026 Jan–Sep |
| --- | ---: | ---: | ---: | ---: |
| BTC | 80.473 | -98.176 | 101.998 | 254.281 |
| ETH | 84.374 | -70.295 | 190.651 | 237.088 |
| SOL | -79.221 | -102.280 | -46.133 | 118.200 |
| XRP | -23.852 | 136.663 | -193.377 | 504.219 |

年度、侧、入口族、45月cohort均是实际退出时间归因，不能当作另一次独立初始1000的收益。四年开发数据已观察，非时间holdout；AAVE/ATOM/ETC/LINK收益仍未读取、资格待核。

## 描述性失败归因

| 币 | 补充LONG 笔/净USDT | 补充SHORT 笔/净USDT | 静态扣除最佳5笔后的净USDT |
| --- | ---: | ---: | ---: |
| BTC | 36 / -158.077 | 26 / -87.389 | -37.841 |
| ETH | 34 / -94.227 | 31 / -13.323 | 2.541 |
| SOL | 45 / -162.123 | 20 / 7.360 | -416.342 |
| XRP | 56 / -33.335 | 15 / -23.641 | 96.092 |

所有补充LONG实际净归因均负，SHORT仅SOL为正；不得从亏损直接反向交易、删侧、删族或拿剩余账目当可实现反事实收益。RG10持续主动压力与极值收回没有建立跨币稳定优势；quote多数与价格反向只是吸收代理，不识别被动挂单、钱包或未来持续反应。下一步先观察这些真实入场后的价格保持与主动压力是否延续，未来小时仅诊断；再冻结可在自己的闭合入场时读取的新机制，不能把未来字段塞回旧入场或网格搜索本轮阈值。

## 实现、源身份和真实运行核验

- 主输出12完整run，8个AF0/RG7控制的全交易、metrics、年度、source/data身份完整重现；仅既定cache_hit是采集/复用观察差异，非数据身份差异。会计2064笔0错误，snapshot/规则hash、gross/双侧fee/net/frequency/年度/侧逐项核对。
- 263个实际RG10补充入口，21040个canonical闭合OHLC/quote/QPS/taker字段、789个日线ADX/DI值，0失败。逐入口验证entry_time−1、全Expr、[2:10]有效加权反向quote多数及相反价格、闭合[1]极值sweep/reclaim、反向主动多数、放量body、已观察当前quote/live hold、日线反向排除及弱闭合ADX。ATR/ADX原函数使用真实200input/199closed种子；ATR2、4h EMA仅记录配置值一致性，不额外作为入口门槛。
- 692正常关闭、forced-end0、失败0。AF0旧关闭463、RG4弱小时附加235、同时6、附加独自229。全部入口统一完整RG4关闭；原Position/actual exit_time−1/gross mark-price-denominator ROI/外部5/5调用gate全部核对。没有hash路由或RG8强loss分支。
- 原成本2064笔/4016实际funding应用/1066缺mark分钟回退、zero_activity_fill0、算术失败0；RG10自身692笔/1314应用/379回退。当前现金10%保证金×8、下分钟Open不利5bps、双侧0.0005手续费、真实funding inclusion及原mark回退均逐笔核对。
- fresh receiver核验不是整段顺序缓存/private selector/API/live/forward一致性、独立指标数学、订单簿或容量证明；原资金费结算mark缺失仍使精确成本门槛pending。RG8固定12原始HTTP空mark证据仅样例，不补造mark、不降低真实成本要求。
- 原backtest_engine_v7/standard_1m、风险、统一关闭和source冻结不变，public-canonical-repaired-v2-funding-tail-v1/public-archive及verified-archive20261003-v2；四data hash与原helpers SHA保持，tail54/overlap7 manifest保持。
- 07:35实际hash保护conf/app.conf、engine/environment/indicator_cache、原helpers、正式1.0.8技能、原pending与独立阶段总结草案均保持；overlay virtual源真实不存在。无App、DB写入/分配/启用/下单、生产/前端改动或新仓库测试文件。

## 冻结候选、失败checker及输出

两个完整JSON保留temp_strategy/20261004-opposing-pressure-extreme-rejection/。族SHA c7bdd15bdf692f0138c6c98f95f7fc60a0ec88d2e8a7cf08115f3bb12ff1e672/version b7a63f8c9393b8e52c8b4c994a0e56dc62e33ea21ad7b9c600f717ae33a33e35；组合SHA bdf5485f178c2e909993ebc476f9774afbda7198e4312e72a14fc786f3c0fd9a/version bf556e3117b99573ac753d1e401cb49b1ca48459b28a41d1e1cd3d75f6aeef88。收益前协议2026-10-04-opposing-pressure-extreme-rejection-protocol.md保持。

初版checker29341 exit1的3314/168结果及源保留；旧expected方向和错误weighted_opposition fixture修在另一个helper，7970 exit0的3482/0，候选未改。412旧完整配置/905同侧完整程序比较无精确去空白重复，非语义/alpha证明。

所有下列大结果均在 /Users/zhz/Library/Caches/go-binance-strategy-research/results/：

| 文件 | SHA256 |
| --- | --- |
| 20261004-rg10-development4-canonical-repaired-v2-funding-tail-v1.json | 667f8d3051ef5b2c795289f3ed61a3337d8d91fa22c2831f57be6efd2fa4bf91 |
| 20261004-rg10-accounting-summary.json | 1b081c2b0404b9c5449fd664f6971d642ef62bd20334e3457a72435f29ef11e4 |
| 20261004-rg10-expr-checks-fixturefix.json | a0cdba223ff0395f0d31f8179c156a9ad28146054f048c4a1fb887e52f5b36a2 |
| 20261004-rg10-canonical-actual-seed-open-signal-audit.json | 625a86e15a4014a0e65a80361346412cb500967b7b40d63dc91f08734918a545 |
| 20261004-rg10-execution-original-cost-fallback-audit.json | 2d8439172a95500e526f96aa0a8aa1d01476cd054ad06481dfd9a382961c5340 |
| 20261004-rg10-original-position-close-signal-audit.json | 44d90537545a8bb1d335a83c72c29b461a7a50efd16e44dcfe94f3fa2c9ca362 |
| 20261004-rg10-prior-failure-path-attribution.json | 2fd20a347cdba07ba3f1882c8ee5e44f4c69ae12ab53c4b2a2a93b28acb590ca |
| 20261004-rg10-arm-metadata.json | 5d16edd10f3eb83f815c8f097c30985fdb8fccb1b8f4d9f5496a5f9a24274f87 |
| 20261004-rg10-db-v29.json | b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0 |

07:04三库只读元数据截止/222、219、8结果和v29全导出核对仍是本轮最新ARM采集；不冒称449行新forward内容已分析。临时Go/overlay/二进制仓库外。正式技能与原pending不变；阶段总结独立草案结构valid、行为四例pending/aggregate null，未委派或晋级。

下次开始第一条先总结本完成阶段、再核实际goal/进程；不poll已终止92584/35448/55977/53961，不重复已完成12主run。不因阶段完成自行complete/paused/blocked；若真实goal paused则停止不自行恢复。

