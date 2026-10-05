# RG7完整研究总结：日线冲突排除改善总收益，仍不具备发布资格

2026-10-04北京时间02:37:26核对。主80258/开8469/成本89047/关41023均实际 terminal exit0，12完整49月run/2440执行账目（含共享对照），8个AF0/RG6旧控制完整重现，会计errors0。目标实际get_goal active；没有满足全部门槛的可用策略，裁决invalidated（仅对本研究联合发布要求）。阶段完成不等于目标完成，不自行complete/paused/blocked。

## 当前结论

| 币 | 笔数 | 周频 | 净USDT | PF | 回撤% | 四完整Sep–Aug净USDT |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| BTCUSDT | 234 | 1.099 | 467.110 | 1.268 | 18.686 | 21.703 / 69.600 / 23.153 / 442.427 |
| ETHUSDT | 245 | 1.150 | 369.168 | 1.192 | 27.548 | -151.673 / -57.148 / 284.791 / 356.181 |
| SOLUSDT | 252 | 1.183 | -255.125 | 0.885 | 33.779 | -250.557 / 145.573 / -38.786 / -76.665 |
| XRPUSDT | 212 | 0.995 | 67.948 | 1.039 | 30.519 | 30.798 / -178.746 / 113.045 / 80.360 |

四币频率均>=0.9，但SOL总净亏；ETH/SOL/XRP仍有负完整Sep–Aug年，BTC虽该分组四期为正，日历2024仍负，不能挑有利年度定义来判定四年稳定。相对RG6，四币总净均提高；这只能支持在当前已观察开发数据上该候选整体表现有所改善，不能证明过滤的普遍因果或验证币泛化。没有剔除SOL/短方向、按币挑版本或降低192笔门槛。

| 币 | 日历2023/24/25/2026 Jan–Sep净 | 追加2026-09净 | 去最佳5笔净 | 补充LONG数/净 | 补充SHORT数/净 | 平均持仓小时 |
| --- | --- | ---: | ---: | --- | --- | ---: |
| BTCUSDT | 208.879 / -86.276 / 205.998 / 213.840 | -89.774 | 168.474 | 72/-197.844 | 64/63.016 | 21.268 |
| ETHUSDT | 7.152 / -87.747 / 267.794 / 320.343 | -62.983 | -9.309 | 79/136.924 | 65/-114.043 | 14.477 |
| SOLUSDT | 29.778 / -198.263 / -120.308 / 28.484 | -34.690 | -585.973 | 45/-102.584 | 75/-228.539 | 9.722 |
| XRPUSDT | -92.380 / 144.532 / -237.670 / 249.840 | 22.490 | -299.048 | 37/-52.867 | 88/-233.273 | 11.873 |

年度/方向/入口族/45月cohort仅按exit-time归因，不是独立初始化、删组/去最佳交易的可实现反事实PnL。补充SHORT在ETH/SOL/XRP均亏，补充LONG在BTC/SOL/XRP均亏；基础入口贡献与仓位占用/复利交互不能用静态删交易替代。SOL全部gross -60.681本身已负，不是只减手续费可补救。XRP频率只有212笔（门槛192），进一步叠加过滤的频率余量有限。

## 三项核验与未完成资格

全部525补充入口/42000 canonical闭合小时字段/failed0；原9配置 fresh receiver 的实际200input/199closed原函数ATR、4h ADX和EMA重算通过。新增闭合日线ADX/PlusDI/MinusDI三字段逐笔同199闭合日线原函数重算一致（额外1575数值比较，不包含在42000小时字段计数中），日线反向排除全部通过。此前8小时quote有效性/严格加权主动方向、Close[2]/Open[9]价格同向、原分钟形成价格保持/活动均通过。fresh快照不证明全顺序缓存/private selector/API/live/forward/独立指标数学或订单簿容量。

正常close943/forced-end0/failed0；old true498/RG4 added448/both3/added-only445。原Position、exit_time−1已观察分钟、gross mark-price-denominator ROI、外部门槛与完整原关闭程序，以及独立小时结构确认核对通过。分支归因不是移除分支后的反事实收益，也非生产短路日志。

全部2440笔/4404真实funding应用/1131原观察结算分钟Close mark回退，零活动成交0/算术failed0；RG7自身943笔/1609应用/419回退。历史真实率和结算时间完整不等于交易所精确mark；精确成本资格未补齐，不补零/伪造mark/改原engine。AAVE/ATOM/ETC/LINK未观察收益未读取，其历史资格仍待核；开发失败不进入验证币。

## 冻结条件与文件身份

UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔；49月开发数据已观察，不是时间holdout。原backtest_engine_v7/standard_1m、观察分钟close/下分钟Open、当前可用现金10%保证金/8倍、双边各fee0.0005/slip5bps、真实funding与原缺mark回退不变。外部profit/loss5/5仅调用资格，AutoStopOrder=false，普通平仓仍需市场确认，−20是唯一ROI-only灾难例外，退出统一不依赖hash。Data[0]有意形成价格/活动仍保留，不新增forming taker0引用。

RG7仅对RG6补充入口增加闭合日线ADX>=20且DI相反的排除；原9技术配置/基础入口/全部RG4关闭对象完全不变。完整族与组合：temp_strategy/20261004-daily-opposition-guarded-flow-escape/，族SHA7503b4692dc81bc7f36ace594e48f677e413743db05a5271aaa732d148da19f1/versiona9b995d8760765fe97278be03a3fbdd63879e688594ce8eb9b756aeab032e72d，组合SHA5988a0332dbd3583b538eee6ebc3fe1298483836a38d3b3a451a34857864c9d9/version4e2a42e65aa8a5a10e84f26a9b845e22a7de9cb3e3b923d12a6eb23e598d146e。

收益前protocol2026-10-04-daily-opposition-guarded-flow-escape-protocol.md保留；3446合成Expr检查0失败，377旧portable/1638同侧完整程序比较无精确去空白重复，非语义/alpha/市场盈利证明。缓存source/CRC/完整data hash及funding-tail prefix/suffix每次按原要求核对；四币输入SHA、回测/data helper和官方suffix54/overlap7证据与RG5/6协议相同。

以下输出均在仓库外缓存 /Users/zhz/Library/Caches/go-binance-strategy-research/results/：

- 20261004-rg7-development4-canonical-repaired-v2-funding-tail-v1.json SHA 4d08086d10e01889fba564fcfeb09de1394370e695108df996fea2a6f6621847
- 20261004-rg7-accounting-summary.json SHA 13ed53bafb1f56228dba1d7e42964ef11fd0ba25326689348a1c233f8e35f288
- 20261004-rg7-expr-checks.json SHA de4dd159b7cd65b20db120c161fbbb2a2d904dc72ed3dd821f814dfbab8c2f80
- 20261004-rg7-canonical-actual-seed-open-signal-audit.json SHA 73a05750506eeb7ce6f81625744f385dbdd7e191f023f49091935e902bd44371
- 20261004-rg7-execution-original-cost-fallback-audit.json SHA d476e9686f311275164110e27327c62a8e5f16c5b81d1be93b26f66745ef4267
- 20261004-rg7-original-position-close-signal-audit.json SHA a05050c922443bae103c103fcc55e511a720b0daae89d71859f39ab99742ed67

## 本阶段完成与下一次恢复

本次新增RG5/RG6/RG7三轮，每轮12完整49月run，共36run/8219执行账目（包含反复验证的共享对照，不是8219互不重复历史交易）。六份完整失败候选保留在temp_strategy对应三个目录；每轮收益前协议/Expr矩阵/会计/开仓/成本/关闭核验均完成。不重复已终止主回测或因timeout重启旧句柄。

下一次先给RG5/6/7阶段性总结，核实际goal和真实进程，再执行新研究。下一步先分解补充族的持仓时间、ROI/结构关闭路径与逐笔收益期望，确认是否存在无趋势区追价/退出尺度不匹配；再预声明一个价格与交易量机制变化、检查旧完整程序避免重复，冻结完整JSON和合成矩阵后才跑全撮合。当前RG8未生成/测试，不把待检验机制当已证优势，也不继续为了总净任意叠加过滤或放宽年度/跨币/费用/0.9门槛。

ARM只读元数据截止01:39:58.217/01:40:03.394/01:40:08.651：模板17/17/17、结果221/218/7，三库v29摘要相同、go_binance全导出语义与既有快照相同。这不是446条新forward内容分析。禁止App/DB写入/策略分配/启用/下单，无生产/前端/config/新仓库测试文件改动。

02:37:26再次实际核goal active；限定进程检查无本阶段回测/审计进程，所有真实session已terminal。conf/app.conf、engine.go、environment.go、service/backtest/indicator_cache.go、正式SKILL和六个候选SHA不变，git diff --check通过，虚拟诊断源实际不存在；其他用户dirty研究保留。正式技能1.0.8/trusted:false不变，receiver-seed/funding-mark新行为评估仍pending/aggregate null未晋级，本研究记录不冒充strict-win技能评估。

