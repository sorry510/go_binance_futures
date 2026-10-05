# RG9完整研究总结：闭合RSI支持仍不足以解决弱趋势突破亏损

2026-10-04北京时间03:35:05核对。主33263/开40894/关5518/成本15914全部actual terminal exit0，限定进程检查无本阶段残留。12完整49月run/2283执行账目，包含8个AF0/RG7共享对照，完整逐笔与统计重现，会计errors0。实际goal仍active；联合发布裁决invalidated，没有满足每币0.9次/周、真实成本、四年稳定和跨币泛化的可用策略。不把阶段完成标成完整目标完成，也不自行暂停或阻塞目标。

## 当前结论

| 币 | 笔数 | 周频 | 净USDT | PF | 回撤% | 四完整Sep–Aug净USDT |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| BTCUSDT | 226 | 1.061 | 424.125 | 1.250 | 18.955 | -3.015 / 70.465 / 15.878 / 443.687 |
| ETHUSDT | 236 | 1.108 | 372.974 | 1.201 | 28.230 | -173.571 / -42.704 / 266.236 / 386.171 |
| SOLUSDT | 245 | 1.150 | -217.897 | 0.901 | 31.062 | -245.819 / 158.307 / -27.987 / -65.975 |
| XRPUSDT | 204 | 0.958 | 80.190 | 1.047 | 29.244 | 18.899 / -171.191 / 117.302 / 92.431 |

四币频率均>=0.9，但SOL净亏，四币均有亏损完整Sep–Aug年及亏损日历年。与RG7相比BTC总净下降，ETH/SOL/XRP小幅提高；不是可以挑盈利币种或按币挑版本的证据。BTC原RG7四完整年为正，RG9第一期转负；不能用总净收益掩盖年度退化。XRP204笔对192门槛仅12笔余量，不继续任意叠加过滤再放宽频率。

| 币 | 日历2023/24/25/2026 Jan–Sep净 | 追加2026-09净 | 去最佳5笔净 | 补充LONG数/净 | 补充SHORT数/净 | 平均持仓小时 |
| --- | --- | ---: | ---: | --- | --- | ---: |
| BTCUSDT | 184.581 / -87.779 / 200.347 / 202.787 | -102.890 | 131.087 | 70/-217.856 | 58/55.067 | 21.204 |
| ETHUSDT | -13.496 / -74.457 / 256.307 / 342.993 | -63.158 | -1.763 | 75/109.615 | 60/-76.358 | 14.989 |
| SOLUSDT | 29.450 / -181.600 / -107.487 / 36.557 | -36.424 | -549.772 | 45/-104.709 | 68/-189.428 | 9.897 |
| XRPUSDT | -83.912 / 142.209 / -221.324 / 252.704 | 22.748 | -286.300 | 34/-49.931 | 83/-227.315 | 12.069 |

补充LONG仍在BTC/SOL/XRP亏损，补充SHORT在ETH/SOL/XRP亏损；不能静态删方向或删亏损交易当改进回测。SOL gross=-24.724本身负，不是只减手续费可救回。ETH/SOL/XRP去最佳5笔净为负，收益集中仍未解决。年度、方向、入口族、去最佳交易及45月cohort都是exit-time描述性归因，非独立初始化收益、删组/删交易或分支移除的可实现反事实。

## 候选变化与合成检查

RG9从RG7而非失败RG8派生，原9技术配置、原AF0基础入口、全部统一RG4关闭整对象不变。只给两个弱趋势补充全条件末尾增加已有闭合1h RSI14：LONG Data[1]>=55、SHORT Data[1]<=45，名称改rg9_closed_hourly_momentum_flow_escape_long/short。55/45来自v29原基础入口，不是收益阈值搜索；既有闭合8小时价格与quote支持、4h EMA方向、日线反向排除、首次脱离/放量/body/live保持均保留，没有新指标/周期/变量或forming taker0/hash路由。

75730 actual terminal exit0，3662 Go/Expr合成0失败。闭合RSI0/44.999999/45/45.000001/50/54.999999/55/55.000001/100与forming0/100交叉，含两侧包容边界、原入口/主动quote/日线/方向/价格/body/live矩阵、基础可达与完整关闭。静态全对象断言唯一新增条件及原配置/基础/关闭完全身份。只读temp_strategy+strategy_templates的409旧portable/895同侧完整程序比较无精确去空白重复，未读取相应旧收益；不是语义、alpha、private/API/live/forward或盈利证明。收益前协议2026-10-04-closed-hourly-momentum-flow-escape-protocol.md已保存。

## 三项实际核验

全部493补充入口/39440 canonical闭合小时字段/0失败。新增RSI493次数值比较另计，实际receiver Data[1]与原CalculateRSI在200input对应199个canonical闭合小时价格的结果[0]一致，55/45门槛全部true。日线ADX/PlusDI/MinusDI另有1479个原函数数值比较；200input/199closed ATR、4h ADX/EMA与全程序身份全部通过。使用原指标函数重算，不冒充独立数学；fresh快照不证明整个顺序cache、私有selector、API/live/forward或订单簿容量。没有改变默认warmup、递推种子或容差来制造通过。

正常close911/forced-end0/failed0；old AF0 true496、原RG4弱结构added417、both2、added-only415。原Position、exit_time−1已观察分钟、gross mark-price-denominator ROI、外部调用gate、完整原统一关闭、canonical上一闭合Low/High及闭合4h ADX重算通过。RG8强趋势loss分支没有带入；分支归因非私有短路日志或去分支反事实收益。

全部2283笔/4280真实funding应用/1092原观察结算分钟Close mark回退，当前现金复利10%保证金/8倍qty、原下分钟adverse fill、双边fees与资金费时序算术0失败、zero_liquidity_fill0。RG9自身911笔/1578应用/405回退。完整真实rate和结算时间不等于精确结算mark覆盖，精确成本资格仍pending。本轮沿用RG8固定12个原始HTTP空mark样例证据，不新请求或假造mark、不补零/改engine/改缓存成本。

## 冻结条件与完整文件身份

UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔；49月四开发币已观察，非时间holdout。原backtest_engine_v7/standard_1m、观察分钟close/下分钟Open、初始1000/币、当前可用现金10%保证金/8倍、每侧fee0.0005/slip5bps、真实funding和原缺mark回退不变。外部profit/loss5/5为调用gate、AutoStopOrder=false，普通ROI-only仍false，ROI<=−20是唯一ROI-only灾难例外；有意forming价格/活动Data[0]保持。

来源public-canonical-repaired-v2-funding-tail-v1、public-archive/public-archive、verified-archive20261003-v2，每次原要求全source/CRC/data hash及funding prefix/tail核对。数据/helpers/config/engine/environment/实际indicator_cache.go/正式SKILL SHA实际复核不变。AAVE/ATOM/ETC/LINK验证收益未观察且历史资格待核，开发失败不读取。

完整族和组合在temp_strategy/20261004-closed-hourly-momentum-flow-escape/：族SHA2ff3a486f541a31396ca4c23ba3fd535d5c0158facde0485c2c240ac731602fb/version89823f32abecce632b0c805a7bd87f9cde6328ab98ce918f09ad3628b4f89c07；组合SHA10587889b393dd4e25aef21fbb533f4f2d0ae250ae701103fb855caffa3a1628/version5aa9b5be8a1fe791fa09ff8a1b75278a38cda0f14aa377f93efeea35d200c762。全部中间失败文件保留，族单独收益本轮未计划，不能把组合回测称族单独验证。

以下结果均在仓库外 /Users/zhz/Library/Caches/go-binance-strategy-research/results/：

- 20261004-rg9-development4-canonical-repaired-v2-funding-tail-v1.json SHA 1588cda83f58950aaaafd8a0cf7479b39ff28f877aad8e3220ccf2072853e6d7
- 20261004-rg9-accounting-summary.json SHA e2452de21767dd5d86978c6a9972a08525a5081e93ccca95901bf3617c403d0f
- 20261004-rg9-expr-checks.json SHA 5227727a31a262679c57c7267d2823f6f6f4c4a1ab15de20b730058f3b36b729
- 20261004-rg9-canonical-actual-seed-open-signal-audit.json SHA aef46e58e3e7514b5030cbc36fc0e1d3dff93d0e082957c63d2cef2636932d1c
- 20261004-rg9-execution-original-cost-fallback-audit.json SHA 7280f1b9ef7e5ebc48c30103cf1412cf114aa47b00aadab563a2425d09668318
- 20261004-rg9-original-position-close-signal-audit.json SHA 7d9fed221137c33174bddb8450c1ab2251938b9680bd12a17cb747bcdaa22981

诊断Go、编译产物和overlay均在仓库外verification，两个virtual目标rg9_signal_diagnostic_bridge.go/rg4_close_diagnostic_bridge.go实际不存在。没有新增仓库测试文件或修改生产/前端/conf/app.conf。

## 阶段总结与下一步

本轮新增RG8/RG9两轮，共24完整49月run/4724执行账目，包含反复验证的共享对照，不是4724不重复历史交易；四份完整失败候选、两份收益前协议、全部Expr/会计/开仓/关闭/费用证据可用于复核。RG8全趋势确认止损使四币总净下降，RG9闭合RSI虽保留交易频率却未解决年度和跨币失败，两者均不能发布。原RG5/6/7和RG2/3/4完成阶段另存，不重复回测。

下一步先描述性核对弱趋势脱离的“后续闭合小时重回原区间”、已观察主动quote与价格反应是否分歧；这些后续结果只能用于诊断，不能作为同一入场的未来数据。再检查相关旧完整量价吸收/极值拒绝配置避免重复，冻结一个互补机制及独立确认矩阵后全撮合，原v29基础/风险/统一关闭与全部联合门槛保持。当前RG10未生成、未冻结或测试，不把新机制当已有优势，也不继续通过挑币/删方向/调费用/改年度来过关。

ARM最新只读元数据仍北京时间02:43:13.230/15.818/18.155：模板17/17/17、结果221/218/7，三库v29摘要相同、go_binance全导出parsed与既有相同；不是446条新forward内容分析。无App/DB写入/策略分配/启用/下单；其他用户dirty改动保持。正式技能1.0.8/trusted:false不变，新receiver-seed/funding-mark行为评估pending/aggregate null、未晋级，不把这些审计或报告当strict-win技能评估。

下次开始前先给本阶段总结，重新核实际goal及真实进程；全部真实句柄已经terminal，不poll旧句柄或重启旧主。实际paused时停止且不自行恢复，阶段结束不自动complete/paused/blocked。
