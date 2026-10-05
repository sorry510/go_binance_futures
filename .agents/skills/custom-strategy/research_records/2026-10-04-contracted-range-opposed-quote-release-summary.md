# RG15：收缩后闭合价格突破与主动 quote 背离 — 完整结论

## 当前结论

- 2026-10-04北京时间13:19，主71006、开37407、关52545、成本93142均actual terminal exit0；12完整49月run/3727账目，8AF0/RG13共享控制逐笔/metrics/年度/source复现、会计errors0。自身675笔。RG15 **invalidated**，未达完整目标。
- BTC/ETH净正但每币0.9次/周失败；SOL/XRP频率达标却净亏。四币负完整Sep–Aug年及负完整日历年。精确资金费结算mark仍pending，未观察验证币收益未读；不选币/侧/年度或降低成本/频率换过关。
- 完整族和组合JSON都保留temp_strategy/20261004-contracted-range-opposed-quote-release/；独立族没有单独回测，不能拿组合收益证明族。当前actualgoal active，阶段结束不自行complete/paused/blocked。

## 单维机制及固定执行

- 原RG13两补充只改变closed[1]合法主动quote多数方向：LONG在价格/交易仍向上时buy*2<quote；SHORT仍向下时buy*2>quote。不是交易方向反转，吸收/被动支持仅假设，不是订单簿或因果证明。
- 原收缩/首次突破/volume/ATR1和2/实体/实时保留/weak或强EMA-DI方向/日线反向排除、原9配置、AF0全基础、wholeuniform RG4多空关闭整对象完全不变；没有叠加RG14退出确认、forming taker0或hash路由。
- 原UTC2022-09-01inclusive至2026-10-01exclusive，1491日/213周，每币至少192笔；原8倍/现金10%margin/双边0.0005fee和各5bps不利slippage/真实资金费时序与缺mark分钟回退/外部5与5门控/AutoStop=false/canonical数据hash/actual200input199closed种子保持。

| 币 | 笔数 | 次/周 | 净USDT | PF | 最大回撤 | 平均小时 |
|---|---:|---:|---:|---:|---:|---:|
| BTCUSDT | 129 | 0.606 | 624.278 | 1.521 | 16.01% | 32.388 |
| ETHUSDT | 136 | 0.638 | 564.557 | 1.363 | 19.13% | 18.421 |
| SOLUSDT | 198 | 0.930 | -103.373 | 0.949 | 35.09% | 12.420 |
| XRPUSDT | 212 | 0.995 | -15.814 | 0.992 | 45.62% | 12.364 |

| 币 | 完整2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外2026-09 |
|---|---:|---:|---:|---:|---:|
| BTCUSDT | -16.597 | 88.723 | 164.435 | 460.002 | -72.284 |
| ETHUSDT | 62.350 | -72.102 | 296.476 | 297.046 | -19.213 |
| SOLUSDT | -225.335 | 0.354 | -7.426 | 163.470 | -34.435 |
| XRPUSDT | 209.540 | -424.730 | 118.725 | 54.541 | 26.111 |

每币独立初始资金/复利，不将此表加总当可交易组合；年度为完整原交易退出时间归因，非另初始资金的年度独立收益。

## 实际验证和错误保留

- 2057首次合成checker actualexit1：4670pass/288fail，顺序字符串替换抵消导致模型预期仍旧同向。只另存精确原子方向块checker_v2；5067 actualexit0：4958pass/0fail，candidate字节SHA从未改动，首次失败完整输出/源及v2全部保留。
- 真实Go/Expr矩阵和local ordered model检验新合法oppositequote与严格半数、所有不变入场/日线/4h矩阵/实体live/ATR/收缩/新鲜边界及unused字段独立、原normal confirmed/灾难例外/fullbase/fullclose对象。非private/cached sequential/API/live/forward或盈利证明。
- 真实前端validator VM两issue=null/9enabled/四类型/shape，304限定portable/689启用完整entry identity去空白0dup，排除research/诊断名/>128KiB，非全语义或alpha新颖性。无App/前端build或修改。
- 开37407 actualexit0：266补充/21280闭合8字段/1330四小时/798日线/532ATR/1596范围/actual原200input种子，全新LONG-SHORT价量相反majority及合法quote/收缩/首次释放/fullExpr/方向/时间/数据hash0failed。专用新helper的FlowDiverges独立比较，未拿旧同向FlowConfirms报告冒充。
- 关52545 actualexit0：675 normal、forced-end0、old546/added130/both1/added-only129、0failed；实际参数化原完整RG4程序用新input/SHA/version核所有原Position/现金/ROI/outergate/canonical前hour高低/closed4hADX，不是旧报告或RG14关闭。
- 成本93142 actualexit0：3727账目/7183funding应用/2054缺mark观测分钟价回退；自身675/1424/350。原实际资金复利/qty/fills/双fee/gross/net/包含规则0failed、zero-activity fills0。精确结算mark仍缺，不冒称exact venue计价。

## 全配对归因与下一机制

| 币 | 补充LONG笔数/净 | 补充SHORT笔数/净 | 补充总gross/净 |
|---|---:|---:|---:|
| BTCUSDT | 20/84.234 | 14/-157.191 | -35.812/-72.957 |
| ETHUSDT | 14/6.545 | 19/30.699 | 70.121/37.245 |
| SOLUSDT | 45/-184.258 | 27/-36.919 | -147.403/-221.176 |
| XRPUSDT | 81/-213.305 | 46/-185.610 | -301.312/-398.915 |

- BTC/ETH的正组合收益主要由保留基础贡献；新补充只有ETH正，BTC/SOL/XRP补充gross就已负，不是只要减费用可解决。总675不是675减AF0独立429来推补充246；真实identity配对补充266、组合基础409。
- 原ADX20自然weak/strong全部266配对，不挑阈值；无一致正组，侧/year/退出组仅原归因，不能静态删除、反向或重新初始1000称新净收益。
- 对突破之后的相反主动quote，未有直接失败突破/区间收回价格响应要求；本次结果不支持把它本身视为吸收后延续证据。
- 下一步可研究一个完整的新价格响应机制：闭合hour先严格越过收缩区间反方向边界再严格收回、相反主动quote多数与放量保持，衡量从极值的收回位移而非强制同向实体。先读取相关原完整配置防近重复、保存完整候选/自测/收益前协议，再完整四币/控制/原成本撮合；不把旧266归因删组当新收益，不调网格或读未观察验证收益。RG16尚未生成。

## 可恢复文件

- 族SHA2efa79d95b65673a5c388af8f4b6bcb353bf2f44bcec358cfc4bbe25448a72db/version7d27ad2b691ded8db941081c6050822584f87fc96af382cb2a298c32e8046f99；组合SHA8dba1d67bca704c9a093828e416b11d168aff8e2981b05dc05f255dedf506fc4/version12fd54f7b9f5bf3db8529edc9d811556c8be22e1f054d982df76235e9b6b1254。
- 结果根目录：/Users/zhz/Library/Caches/go-binance-strategy-research/results/
- 20261004-rg15-development4-canonical-repaired-v2-funding-tail-v1.json SHAd0af3a08bf5ad80cfe5b333aabc0c8f79da1a182f32a84bc5b68d547ea29c689。
- 20261004-rg15-accounting-summary.json SHAff918e55c00cf7bb70d880be981e1ac10082638ce424007732cd675cc3285e4a。
- 20261004-rg15-expr-checks-v2.json SHAfc0af20badc4743c095e01978d14e2f7e3ae43150748d5da7715408b27539a6f。
- 20261004-rg15-canonical-actual-seed-open-signal-audit.json SHA7421669e2716ef26328d38638a2a8441213ea5f1385cb0d89d72dcfdeb1617e8。
- 20261004-rg15-execution-original-cost-fallback-audit.json SHAaf881af31b3739ec117747e3b8d13e12fdeea875e089c070fd7e46de03b0503e。
- 20261004-rg15-original-position-close-signal-audit.json SHAa970e268a2041587b49f45462c9b07dbd8b9ae773f22ad7d9f311ca311b997fa。
- natural-entry-regime-attribution哈希：e74160471e16369d252327196425f9eb1b88026f8915625da8dd9e256a7ec631  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261004-rg15-natural-entry-regime-attribution.json。
- protocol2026-10-04-contracted-range-opposed-quote-release-protocol.md先于主13:08:00.216启动；全部错误和中间完整JSON保留，没有删除材料。

## 安全及恢复

- 全本轮真实主/核验均结束，下一次不能重poll这些句柄/重复旧完整主；先阶段总结后实核goal/live/file。上一turn为实质progress，不是状态重述或无证据wait；完整目标未达标，继续安全研究而不自暂停/complete/blocked。
- 未操作App、DB写/分配/启用/订单/production/前端/config或新增仓库测试文件；conf/app.conf及正式skill外部metrics行/其它dirty改动保持，两个SkillMax草案行为gate未过不晋级。未读AAVE/ATOM/ETC/LINK收益，最新已读ARM仍10:04元数据，不冒称最新forward全量。
