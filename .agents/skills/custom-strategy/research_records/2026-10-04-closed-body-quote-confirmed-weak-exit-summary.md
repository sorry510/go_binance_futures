# RG14：闭合实体与主动 quote 确认弱结构退出 — 完整结论

## 当前结论与阶段进度

- 2026-10-04 北京时间12:57续接实核：RG14主95042、开95596、关17651、成本65005均已实际terminal exit0；所有已完成结果SHA再次核验一致，不重复已结束主/核验。上一goal turn为实际progress，不是计划重述或无证据等待。
- RG14 **invalidated**，没有合格可发布策略。新完整12个49月run/5579账目（含8共享AF0/RG13对照复现）；自身2527笔。四币频率达标，BTC/SOL/XRP净亏，四币负完整Sep–Aug年及负完整日历年。精确交易所资金费结算mark仍pending，未读验证币收益。
- 两个新完整JSON均已保留temp_strategy/20261004-closed-body-quote-confirmed-weak-exit/；独立族没有单独测试收益，不能把组合收益说成独立族已验证。阶段结束不等于目标完成，actualgoal仍active，不自行complete/paused/blocked。

## 单维改动及冻结执行范围

- 全部RG13四入场整对象及顺序、原9指标配置、实时Data/价格[0]、AF0原confirmed退出和ROI<=-20灾难例外保持。只给统一弱结构关闭增加closed[1]反向实体、正quote与合法buy范围、严格反向主动quote多数，不路由entry hash/族。
- 原弱结构分支仍需closed4h ADX[1]<20、实时反向严格破closed[1]高低点、ROI>=5或<=-5。外部5与5门控、8倍、现金10%保证金、双边fee0.0005和各5bps不利滑点、资金费时序/原缺mark回退保持。
- canonical-repaired-v2-funding-tail-v1 / backtest_engine_v7 / standard_1m；UTC2022-09-01inclusive到2026-10-01exclusive，1491日/213周，每币至少192笔；四Sep–Aug完整年及完整日历年同时裁决，不改年界。

| 币 | 笔数 | 次/周 | 净USDT | 相对RG13完整重撮合净变化 | PF | 最大回撤 | 平均小时 |
|---|---:|---:|---:|---:|---:|---:|---:|
| BTCUSDT | 539 | 2.531 | -182.382 | -98.780 | 0.955 | 37.23% | 24.255 |
| ETHUSDT | 646 | 3.033 | 620.251 | 292.679 | 1.112 | 32.07% | 16.709 |
| SOLUSDT | 725 | 3.404 | -633.939 | -49.589 | 0.892 | 75.80% | 11.263 |
| XRPUSDT | 617 | 2.897 | -17.104 | 26.249 | 0.996 | 59.39% | 15.168 |

每币独立初始资金及复利，不相加当一个交易组合。相对差异来自完整新撮合，不是静态删掉旧亏损交易后的收益。

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外2026-09 |
|---|---:|---:|---:|---:|---:|
| BTCUSDT | -85.887 | 55.727 | -178.666 | 62.901 | -36.458 |
| ETHUSDT | -137.586 | 35.007 | 432.842 | 279.595 | 10.394 |
| SOLUSDT | -196.785 | 230.881 | -464.300 | -230.306 | 26.571 |
| XRPUSDT | -59.945 | -390.253 | 108.254 | 207.666 | 117.174 |

年度/侧/族均为完整原重撮合的退出时间归因，不是另初始1000或删除族的独立收益。完整日历2023/24/25也每币至少一年负，不能挑有利年界过关。

## 真实验证

- 收益前10330实际Go/Expr passed、0failed（11213 exit0），包括新弱关闭量/体/多数、ROI/ADX/实时结构等值、forming taker及ratio无关、原正常确认/灾难例外、全部整对象身份。它们证明规则实现，不证明市场盈利或private/cached/live/API接收者。
- 前端installed TypeScript→isolated VM现有technology validator两个issue=null、9enabled、四规则类型/shape通过；限定302portable/623enabled closes全代码去空白0重复。入场特意与RG13相同，不宣称新alpha或入口0重复，无App/UI/build/前端修改。
- 完整会计actualexit0：12run/5579逐笔、8旧AF0/RG13共享控制完整trades/metrics/annual/source复现、errors0。路径只相对固定workspace resolve+realpath校验；仅cache_hit是公开记录的获得/复用观察，不忽略其它source或snapshot字段。
- 开95596 exit0：2243补充、179440 closed canonical字段、11215 closed4h值、6729日线、4486ATR、13458范围；实际200input/199closed原receiver、全合法quote/收缩/首次突破/方向/日线/fullExpr，0failed。使用同一个参数化RG13入口核验binary，实际输入/portable SHA/version均是RG14，正确保留rg13_入口名称；不是拿旧报告当新证据。
- 关17651 exit0：2527正常、forced-end0、old1604、added938、both15、added-only923、0failed；全部新added触发closed合法反向quote和反向实体，四个canonical/原receiver闭合字段全部match。原Position/现金/ROI/outergate、完整新Expr与oldAF0程序和独立新分支相等。
- 成本65005 exit0：5579账目/10773funding应用/3231缺mark观测分钟价回退；自身2527/5014/1527，原现金复利/活动分钟/price×quantity/gross/fee/net/结算包含规则0错，zero-activity fills0。缺mark回退不能冒充exact venue成本，仍需真实结算mark证据才能晋级。
- 2026-10-04本轮已重新读官方fundingRate文档：markPrice对应该笔资金费计价，旧URL不可用时使用官方新catalog页面。这里只验证接口定义，不证明本地所有历史mark已补齐；没有发明结算价或调整原回退。来源：[Binance funding rate history](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)。

## 机制归因与下一步

- 收益前诊断全2623旧close配对中，1160 added-only的596个没有同时反向实体/主动quote确认。完整RG14重撮合不是简单删这596个，所有现金/持仓/下一入场路径独立重算。
- 实际延长持仓并减少weak独有退出只帮助ETH及小幅改善XRP，BTC/SOL更差。ETH仍首完整年负，说明单独确认退出没有解决入场质量；不继续追网格确认阈值、不以ETH成功推广到其它币。
- 自身2243补充按原closed4h ADX20弱/强自然边界全配对：

| 币 | 弱组笔数/净 | 强组笔数/净 |
|---|---:|---:|
| BTCUSDT | 244/-62.993 | 239/-520.102 |
| ETHUSDT | 309/250.241 | 265/371.440 |
| SOLUSDT | 299/-799.938 | 334/5.288 |
| XRPUSDT | 305/-216.008 | 248/-287.566 |

- 原归因仍无一致盈利组。强SOL约+5.288但两完整年负，不能静态删weak；SHORT补充四币负，不能据此反向/删侧/挑币。所有统计都是原交易归因。
- 下一安全动作：检查已有配置是否使用直接闭合quote与价格方向的背离。可研究“主动卖额多数但价格闭合上破/实体向上，或主动买额多数但价格闭合下破/实体向下”的吸收/被动侧支持假设，明确不是从亏损推反向alpha。先完整配置防近重复、冻结一个自己的单机制/原9配置/AF0基础/统一已验证关闭，再全量四币撮合与全部门槛，不使用已阅开发结果选币或未阅验证币。RG15当前尚未生成。

## 可恢复文件身份

- 族SHA38a922b73d8569daf6b88c702256f7e277f50a11c188af84045f0a373e34eb66 / version9d12e9d6b2b2fadca4b459346e60a688f9cf8edd265db08f4e90a77dc3d479d7。
- 组合SHAb91741c9f1338f06cc9077c8eda2a32b20f305e8143d9cdc0698a7f2fbfa36bf / versiond126788c8f43777fa25d5b1d75d37a6b50ceca00bbaf15a5fa09f8375817c13d。
- 所有结果根目录：/Users/zhz/Library/Caches/go-binance-strategy-research/results/
- 20261004-rg14-development4-canonical-repaired-v2-funding-tail-v1.json：SHA 71dcef2ea5fad5e33bdca5f7e3b647021176b8491fc2258942094be972b46a73。
- 20261004-rg14-accounting-summary.json：SHA abd0e0fecb800747e4c70eacfa355b058072524ab166a277a37935841405dd54。
- 20261004-rg14-expr-checks.json：SHA b1d0ab478b911a145eb0595bfb1d46649997aec8237c94a0d82b665312d8234b。
- 20261004-rg14-canonical-actual-seed-open-signal-audit.json：SHA e80313eec0069cf3c67fe987fb2646d3fee21c2483d8c4c839ea41b1bd89e263。
- 20261004-rg14-execution-original-cost-fallback-audit.json：SHA 55fcb2262e73b4ba5595ae838a05bcc420d6570bc617420fcd7a978aec28c720。
- 20261004-rg14-original-position-close-signal-audit.json：SHA 4152409c60087d677f2b4e56a16d9db979131e95154451a0bea3b09146800c20。
- 20261004-rg14-natural-entry-regime-attribution.json：SHA70ab1d8d350e71923271ae08fbb3ac13d93b2bf7f60653d4ac0643041f7f6980。
- 协议2026-10-04-closed-body-quote-confirmed-weak-exit-protocol.md先于主10:29:12.301启动，预声明参数保持。

## 安全及恢复说明

- 本轮全部主和核验真实句柄已结束，无已知本轮live工作；恢复时不重poll95042/95596/17651/65005或重跑已完成完整主。完整目标仍未达标，最新actualget_goal active。
- conf/app.conf、productionengine/environment/indicator_cache、frontend、DB templates/assignment/enabling/orders均没修改；无App操作、仓库测试文件或材料删除。正式skill外部metrics行与其它dirty改动保持，两pending技能无行为strict-win不晋级。
- ARM最新已读仍10:04元数据：17/222、17/219、17/8和v29相同身份，不冒称为12:57最新全forward记录。
- 用户要求已执行：本次续接第一条先阶段总结。下次新续接仍先总结实际完成数/门槛/失败/文件/真实livehandle/下一步，再核goal与文件，而不是仅说继续。
