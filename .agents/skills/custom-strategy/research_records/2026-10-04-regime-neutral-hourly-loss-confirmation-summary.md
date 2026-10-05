# RG8完整研究总结：扩大确认止损未解决年度与跨币亏损

2026-10-04北京时间03:16再次核对实际goal为active。主17550、开1530、成本65843、关80853、项目资金费探测42283、原始HTTP探测91378均已实际terminal exit0；限定进程检查无RG8残留。12完整49月run/2441执行账目，包含8个共享对照；会计errors0、8个AF0/RG7对照逐笔和完整统计重现。裁决invalidated，仅指不满足本研究联合发布要求，不能把阶段完成标成完整目标完成。

## 当前结论

| 币 | 笔数 | 周频 | 净USDT | PF | 回撤% | 四完整Sep–Aug净USDT |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| BTCUSDT | 267 | 1.254 | 94.212 | 1.060 | 21.060 | 61.334 / 25.723 / -13.100 / 75.664 |
| ETHUSDT | 274 | 1.286 | 133.428 | 1.085 | 23.340 | -105.654 / -42.987 / 151.061 / 205.212 |
| SOLUSDT | 288 | 1.352 | -302.163 | 0.845 | 45.096 | -122.224 / 126.296 / -176.475 / -109.459 |
| XRPUSDT | 240 | 1.127 | -143.748 | 0.901 | 27.010 | -59.282 / -133.218 / 92.296 / -10.805 |

四币交易频率均达到每币0.9次/周，但SOL/XRP总净为负，四币均有亏损完整年度，跨币与年度联合失败。相对RG7的净USDT467.110/369.168/-255.125/67.948，四币总净均下降；不是“止损更早必然更安全”。不能按币选择RG7或RG8、剔除短方向、删除亏损交易或降低192笔门槛。开发已失败，不读取AAVE/ATOM/ETC/LINK未观察验证收益。

| 币 | 日历2023/24/25/2026 Jan–Sep净 | 追加2026-09净 | 去最佳5笔净 | 补充LONG数/净 | 补充SHORT数/净 |
| --- | --- | ---: | ---: | --- | --- |
| BTCUSDT | 190.925 / -137.025 / 32.607 / 56.322 | -55.409 | -159.990 | 73/-167.199 | 64/14.228 |
| ETHUSDT | 51.971 / -138.676 / 161.442 / 145.408 | -74.203 | -209.073 | 79/189.412 | 65/-202.309 |
| SOLUSDT | 181.722 / -309.097 / -206.785 / -39.985 | -20.302 | -674.614 | 45/-94.282 | 75/-255.262 |
| XRPUSDT | -137.022 / 164.870 / -207.865 / 31.628 | -32.739 | -473.115 | 37/-46.753 | 89/-230.490 |

四币日历年同样有负年；不能改年度定义换取“四年稳定”。年度/入口族/45月cohort及去最佳交易仅是exit-time描述性归因，不是独立初始化收益或可实现的静态删组反事实。RG8自身1069笔，基础542/补充527；基础和补充都有仓位占用与复利交互，必须全撮合对照。SOL gross=-72.873本身已负，不是只减少手续费就可修复。

## 本轮单一变化与退出证据

此前新RG7退出路径归因逐笔关联943条真实账目及close证据，0身份错误。基础ETH/SOL/XRP分别16/40/21笔达到原ROI<=-20灾难例外；补充弱结构关闭组BTC/ETH/SOL/XRP分别118/120/101/104笔、净-348.343/-300.443/-446.490/-321.427。该归因仅用于提出假设，不证明删组、提前止损的因果收益或私有短路执行日志。

RG8完全保留RG7开仓整对象、原9指标、原盈利关闭与原弱ADX关闭，只增加两侧统一的强趋势确认止损：闭合4h ADX>=20、ROI<=-5，且当前已观察小时价格严格反向突破上一闭合Low/High。强趋势盈利端未增加新分支；普通ROI-only仍false，<=-20是唯一ROI-only例外。不依赖opening hash区分基础与补充。补充名称仍以rg7_开头是原对象身份保留，不是错误版本路由。

1069正常close/forced-end0/failed0；完整旧RG7程序true607、新加强loss分支true468、both6、added-only462。原Position、exit_time-1分钟、gross mark-price-denominator ROI、外部调用gate、完整新旧程序与独立闭合ADX/严格Low-High结构判断均通过。新分支确实参与关闭，但完整回测没有改善四币净收益；这些分支真值不能被当成删除分支后的反事实收益。

## 三项核验及成本资格

4738真实Go/Expr合成检查0失败，覆盖原入口、ADX20边界、ROI-5/+5边界、严格价格突破和任意opening hash；静态检查确认原9配置及四开仓对象完全不变。379旧portable/777同侧完整close程序没有精确去空白重复，不是语义或alpha新颖性证明。收益前协议先保存，不用已观察收益回改候选。

全部527补充入口/42160 canonical闭合小时字段/failed0；实际原receiver 200input/199closed的原函数ATR、4h ADX/EMA与闭合日线ADX/DI重算通过，日线三字段额外1581值比较不混入42160字段计数。所有入口完整Expr为true。fresh快照不证明全顺序缓存、私有selector、API/live/forward、独立指标数学或订单簿容量。

全部2441笔/3708真实funding应用/961原观察结算分钟Close mark回退，原复利仓位、下分钟fill、双边费、资金费时序算术failed0、zero_liquidity_fill0；RG8自身1069笔/1006应用/274回退。真实资金费率与时间完整，不等于交易所精确结算mark完整，精确成本资格仍未补齐。

本轮固定选择已验证RG7成本缺mark行的每币first/middle/last共12例：现有项目public source单次有界查询12次，随后同12目标原始HTTP单次查询12次，均精确匹配时间与rate。原始12响应均HTTP200、字段markPrice存在但值为空字符串，positive mark0、abortedfalse；保留原始UTF-8正文和逐条SHA。这排除了这些样例的SDK丢字段解释，但不是全部历史mark无法获取的证明，不能据此改成本、补零或伪造结算mark。没有重试、代理轮换、数据缓存修改或收益重跑。

官方GET /fapi/v1/fundingRate将markPrice定义为对应资金费结算的标记价；字段定义不保证所有历史记录均有值。[Binance官方资金费历史接口](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)。完整成本资格仍pending，原引擎回退行为照实单列。

## 冻结身份与文件

UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔；四开发币49月全部已观察，不是时间holdout。原backtest_engine_v7/standard_1m、分钟close观察/下分钟Open成交、可用现金10%保证金/8倍、每侧fee0.0005/slip5bps、真实funding及原缺mark回退保持。外部profit/loss5/5是调用资格，AutoStopOrder=false。Data[0]有意形成价格/活动保持，不新增forming taker0依赖。

完整族和组合位于temp_strategy/20261004-regime-neutral-hourly-loss-confirmation/：族SHA bda4685b2ebeb9c8a8ffb7cb0c60c59e5a1176080e3ccc2c26574fa06d705692/version a1533adb221458fcc35fc1d86227b98a1ae622d5461ae633782b213148f49bb9；组合SHA fd95c6d59bda9992c00d7515d435895788f011c78182c9a68c84b1a2c0783060/version 1d5bb8e281e919f4519f4e84749381713c7ae12127ab8e69e637d9d3b187d5a6。收益前protocol2026-10-04-regime-neutral-hourly-loss-confirmation-protocol.md保持。

下列输出均在仓库外/Users/zhz/Library/Caches/go-binance-strategy-research/results/，原中间失败文件不删除：

- 20261004-rg8-development4-canonical-repaired-v2-funding-tail-v1.json SHA b4affac212f26124adc8b76fa01a206058ddbfbb4c434c4669af42fb6c6954d3
- 20261004-rg8-accounting-summary.json SHA da6218c8dd45032f0fba4019d536da4f388858b5d071a193933658f203ff4c06
- 20261004-rg8-expr-checks.json SHA 99a9a40a338d6fa49fb3a6c1567ec7bc2b8355106631817637eeecdb39bf4dbc
- 20261004-rg8-canonical-actual-seed-open-signal-audit.json SHA 7c86dfb78165205615ebde1bdd677800ad66f7756858e43875d8ffe5b0f76fcb
- 20261004-rg8-execution-original-cost-fallback-audit.json SHA fec01c74b517c966a521722fa67c9c5e3b6cd13f4d827b7438389eb60836a0a4
- 20261004-rg8-original-position-close-signal-audit.json SHA ee3f03e369e0d591d69c5513bb082d0ffd0cf79d856c8523ab9be255f5b47a4d
- 20261004-rg7-exit-path-attribution.json SHA a4d69d755ca835e73afe00d79e9e3305af74ef0280511acddcae5afe6ea5f236
- 20261004-rg8-missing-funding-mark-probe.json SHA bf825cee9273c8ee61b7bdc2b99c467ecfb3c29e92daf3814e75434621bfbf9e
- 20261004-rg8-raw-missing-funding-mark-probe.json SHA a6e91a7ebf71deaf55b8506f79e16795804b16b1bfcf2ce20dde1d5e2cebd99a

## 当前已可用及下一步

RG8完整失败候选、冻结协议、12完整回测、会计/开仓/关闭/成本核验和12例原始资金费HTTP证据已可用于复核；没有可发布或已验证盈利策略。下一轮RG9回到RG7完整退出，只给补充弱趋势入口添加已有闭合1h RSI动量支持（LONG>=55、SHORT<=45），阈值沿用v29基础入口而不是收益搜索。先保留JSON、合成闭合/forming独立边界、重复程序检查与收益前协议，再全撮合AF0/RG7/RG9×四币同49月。现在RG9仅是假设，尚未生成或测试。

本轮ARM元数据截止北京时间02:43:13.230/15.818/18.155：模板17/17/17、结果221/218/7，v29三库摘要相同、go_binance完整parsed导出与既有相同；不是446条新forward内容分析。未操作App、写DB、分配/启用策略、下单，未改生产/前端/conf/app.conf或新增仓库测试文件。正式技能1.0.8/trusted:false不变；receiver-seed/funding-mark技能行为评估仍pending/aggregate null，未晋级。本报告不是strict-win技能评估。
