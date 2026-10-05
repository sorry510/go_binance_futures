# RG23：闭合收盘恢复确认的完整研究总结

## 结论和门槛

独立 verdict `invalidated`，`release_qualified=false`。12完整49月回测和全部后审计结束，八共享AF0/RG21控制完整复现，所有实际开平仓和原成本算术通过，但BTC/ETH/XRP每周频率不足。BTC/ETH/XRP有负完整Sep–Aug年，SOL完整四年为正但日历2025净负。未达到频率、真实成本完整证据、四年稳定、跨币泛化的联合目标，不发布/选币/读未阅验证币。

相对RG21唯一改动两个补充strict实时确认价位从闭合High/Low到闭合Close[1]；其他原价量/同向closed body/closed passive quote/全部基础与full统一RG4关闭/原九指标不变。原极值锚定0.35ATR上限没变，不能把它称收盘锚定0.35ATR限制。较早确认是更弱条件，不等于已证明更好的可成交价格。

## 原实验合同

UTC2022-09-01含至2026-10-01不含，49月1491天213周、每币至少192笔；每币初始1000USDT、当前可用现金10%保证金、8倍杠杆，外部门槛profit/loss5/5、双边各0.0005手续费和5bps不利滑点、真实资金费率，原缺mark使用分钟Close回退仍明确pending精确覆盖。正常退出ROI必须结合方向/动量/活跃实体或小时结构确认，仅ROI≤−20为灾难无信号例外。日期/币/风险/费用/频率/四年门槛未改变，原AAVE/ATOM/ETC/LINK未读。

## 完整逐币结果

| 币 | 笔 | 次/周 | 毛PnL | 净PnL | 净PF | 最大回撤% |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | 162 | 0.760563 | 464.793429 | 305.883403 | 1.218163 | 14.297275 |
| ETHUSDT | 176 | 0.826291 | 480.752125 | 321.636181 | 1.193059 | 18.664099 |
| SOLUSDT | 202 | 0.948357 | 1240.139798 | 973.578028 | 1.326934 | 23.050258 |
| XRPUSDT | 168 | 0.788732 | 714.853808 | 550.321143 | 1.301450 | 27.981691 |

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外2026-09 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | -29.547649 | 21.761875 | 34.848096 | 340.273319 | -61.452238 |
| ETHUSDT | -82.109256 | 9.300729 | 181.900688 | 228.773795 | -16.229776 |
| SOLUSDT | 8.075035 | 501.065530 | 142.469026 | 334.893261 | -12.924823 |
| XRPUSDT | 258.225035 | -293.042838 | 211.149184 | 302.731974 | 71.257788 |

| 币 | 日历2023 | 日历2024 | 日历2025 | 2026 Jan–Sep | 去最佳5单描述 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | 82.818955 | 19.336361 | 65.346798 | 189.771021 | 14.737349 |
| ETHUSDT | 26.054783 | -13.160879 | 95.470820 | 296.163207 | -37.896824 |
| SOLUSDT | 393.710918 | 127.975468 | -23.449584 | 464.937388 | 446.907723 |
| XRPUSDT | 140.909344 | 107.946094 | -163.127761 | 453.171250 | 99.680290 |

全部年份是完整一条复利路径的退出时点账目归因，不是逐年重新初始化收益。去最佳单仅集中度描述，禁止称静态删单后可实现利润。

## 结构归因

| 币 | 补充方向 | 笔 | 毛PnL | 手续费 | 资金费PnL | 净PnL |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | SHORT | 35 | -95.532951 | 29.526303 | 2.446141 | -122.613113 |
| BTCUSDT | LONG | 31 | -59.907858 | 26.926386 | -5.530226 | -92.364470 |
| ETHUSDT | SHORT | 41 | 50.626879 | 32.499780 | 0.884087 | 19.011186 |
| ETHUSDT | LONG | 32 | -35.261971 | 25.420807 | -5.710575 | -66.393354 |
| SOLUSDT | SHORT | 31 | 358.354523 | 35.835627 | 0.596326 | 323.115222 |
| SOLUSDT | LONG | 46 | 659.151771 | 51.071944 | -1.411890 | 606.667937 |
| XRPUSDT | LONG | 49 | 112.389886 | 45.124275 | -3.371974 | 63.893636 |
| XRPUSDT | SHORT | 36 | -100.548244 | 34.120728 | 3.505869 | -131.163104 |

301实际补充BTC/ETH/SOL/XRP66/73/77/85，补充净−214.977583/−47.382168/+929.783159/−67.269467。自身708笔=301补充+407组合内基础；单仓及复利路径变化不能按旧基础笔数或静态加总推断反事实。

| 币 | 原4h regime | 笔 | 补充净 | 四完整年净归因 |
| --- | --- | ---: | ---: | --- |
| BTCUSDT | weak | 35 | -125.231066 | -43.788939 / -21.861746 / -43.886016 / -12.268468 |
| BTCUSDT | strong | 31 | -89.746517 | 5.392650 / -44.021532 / -7.022020 / -44.095615 |
| ETHUSDT | weak | 43 | -53.318614 | -49.405004 / 44.515301 / 30.417265 / -78.846176 |
| ETHUSDT | strong | 30 | 5.936446 | -39.304980 / -32.693391 / -33.623283 / 111.558099 |
| SOLUSDT | weak | 46 | 83.074881 | 16.500258 / -98.657725 / 110.920311 / 1.258824 |
| SOLUSDT | strong | 31 | 846.708278 | 155.509042 / 561.493664 / -28.461506 / 158.167079 |
| XRPUSDT | weak | 48 | -98.995148 | 11.075961 / -38.154619 / -16.904323 / -55.012166 |
| XRPUSDT | strong | 37 | 31.725680 | 5.991826 / -78.425124 / 34.390106 / 27.348472 |

旧strict极值在新实际交易上拒绝的较早价格cohort如下，所有138笔保留配对：

| 币 | 较早价格cohort笔数 | 净归因 | 四完整年净归因 |
| --- | ---: | ---: | --- |
| BTCUSDT | 31 | -178.000358 | -50.154278 / -28.016013 / -21.334456 / -83.852240 |
| ETHUSDT | 32 | -32.996553 | 2.426723 / -24.194986 / -23.602910 / 12.374620 |
| SOLUSDT | 38 | 83.455219 | 140.929018 / 47.242290 / -82.138152 / -22.577937 |
| XRPUSDT | 37 | -180.066403 | -10.855916 / -135.026427 / -46.294833 / -30.309628 |

不存在共同稳定weak/strong组；较早确认没有跨币共同改善证据。cohort是原实际交易描述，不是删除组/换价/提前成交后的新策略收益；不是交易方向反转依据。不能为每币挑不同流向或过滤盈利年。

## 验证完整范围

- Go/Expr7890/0失败：整个父程序只改两个锚点、独立数值模型、零影线等价/正影线新可达区间、body/flow/current activity/极值cap/其他原条件及全部统一退出矩阵。前端两对象9enabled/fourtypes/shape issue=null；限定320portable735entry去空白精确重复0，不等于语义或alpha新颖。
- 本地v29 ID114唯一准确名称，read-only repeatable-read0writes再次语义一致。API六条逐条code200/passfalse actual69071 terminal0，只证明固定mock当前快照编译/运行，不是历史LONG/SHORT/8x/forward或盈利。
- 会计12run/1770笔/errors0，八共享控制全快照/逐笔/metrics/年度/source exact。
- 全301补充开仓、24080闭合字段、1505四小时/903日线/602ATR/1806range/903当前量/301严格收盘续行字段、全closed body与原全部条件0失败。当前累计量比例最小0.9000472889994341，signed closed body最小0.0057417178159178805ATR；strict收盘推进最小0.0016254177305556029ATR。旧极值163通过/138拒绝、旧收盘cap180通过/121拒绝、range153收缩/148非收缩仅描述，未删任何交易。
- 全708正常退出、0强制期末，旧539/新增171/同时2/新增独有169、0失败；原whole退出及实际原仓位/ROI/outer gates/小时结构已验证。
- All12成本1770笔/4136结算/1201分钟Close mark回退/0零活动/0算术失败；自身708/1576/497/0失败，最大自身净算术误差1.9895196601282805e-13，control/own/all failed rows全部空。原模型一致不代表精确结算mark或历史订单簿成交保证。
- 主60359、开90012、关62607、成本53181、两mark探测9200/61379及全部前置句柄均实际terminal0，不重poll或重复主回测。

## 官方成本源新证据，不替换当前成本

[Binance官方Mark Price月档下载程序](https://raw.githubusercontent.com/binance/binance-public-data/master/python/download-futures-markPriceKlines.py)给出独立monthly markPriceKlines及CHECKSUM；[官方资金费/Mark K线文档](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data)区分特定资金费结算markPrice与分钟OHLC。

实际首末月四币八个CHECKSUM全部HTTP200且格式/文件名有效，预算8/attempt8/0abort，没有宣称完整档案内容。随后只取BTC2022-09一档：HTTP200/SHA256/ZIP CRC/43200连续分钟与正且一致OHLC通过，原首个缺结算mark时间1662364800013落在其分钟内，mark OHLC open/high19742.26610332、low19736.39095718、close19738.71354941。这不是13ms时点的特定结算mark，`exact_funding_settlement_mark_recovered=false`。未换source/funding/成本/引擎/策略，无PnL重算或holdout读取。下一可以仅做明确假设下的独立边界/敏感度，不得把任意Open/Close当精确结算值。

## 完整文件和身份

两完整失败JSON仍在temp_strategy/20261004-body-close-reclaim-followthrough/，familySHA dbe12701cf25e705e4b1a2d2b88858dd3a75e359cfa11f2a85b9f58f6f2f824d/version2bdd0cdfbfeeacb44cbd37a9c1f409b261fa078e99e82270630c261b1c2eb94f；comboSHA7bb7ed3e951b7230045a7a33e5c13481f0af8cdf48b74d233eb2bc3964968eac/version7a4edb349b072fa8cf6a99d5200a41e0a77f9ba0a9a657b41f544ef98525a3a3。

收益前protocolSHA816f1faa07cedf3bb01d4feeeb429fb940120fc72cbe747ccd6e7712081ade75，API SHA9fdb5b3cc50802053ef7eaad1d795f6666c39138d715bb5c3297e51d64674245。全部cache结果：

- study: `20261004-rg23-development4-canonical-repaired-v2-funding-tail-v1.json` SHA `f374b80f3a5cd6e2891b8b4f187259cf176b4a2981a3a61c3340980f308dd847`
- accounting: `20261004-rg23-accounting-summary.json` SHA `bd97c8aef7b9a00862043e5e48fe93e069b39ac509758f1f1125035a8436a204`
- expr: `20261004-rg23-expr-checks.json` SHA `6dcc61ae985d692ce537e1507621fe7b67863d8d0819ac082e72b303d72e1ca3`
- opening: `20261004-rg23-canonical-actual-seed-open-signal-audit.json` SHA `2d0fd86d0b3e6bcc2ddbfbb0585aa35dd82b0de35fb1e021cd5507763d1dd81f`
- closing: `20261004-rg23-original-position-close-signal-audit.json` SHA `b7b58a51355f4c8eb5df9b6bf41243d9f5b3b0a3683c4001e0f117f1bccbdb63`
- costs: `20261004-rg23-execution-original-cost-fallback-audit.json` SHA `5ea768e7330273e9fd17289b4573dba33a9ce37f216c5abbe55ed41822439114`
- natural: `20261004-rg23-natural-entry-regime-attribution.json` SHA `f40ecf30dc3b152e472ed93e0a887e3525db21ab4934b61de822a406c26408c4`
- phase SHA `8924f7322fc8de487c20320aa0e8932f645e3a7461329e842de2a425291fede1`
- Mark CHECKSUM availability SHA `bbad62cea1b6d7f36c6828209f269410dae394e414d310b0e93c7e13dcfdd943`
- first-month Mark sample SHA `d46ac9920d2e26a211c7cd1e8bc956088652b94802c54ffb2d0d30381aecab28`
- sample archive SHA `cceb950dfd6858865cd76f0ccf2637a930f274e75ae031e17bb4575b4e681427`，保留cache mark-price-probes/20261004-first-month/BTCUSDT-1m-2022-09.zip，不覆盖已有price/funding数据。

生产/前端/conf/app.conf/正式skill不改，无新模板写库/绑定/激活/下单，无App UI、新仓库测试文件、失败资料删除、新委派或技能晋级。此前明确批准的RG18 ID124不是新研究版本授权或盈利证明。

## 下一固定方向

较早价格确认已检验且不能泛化。下一从RG21父（不是静态删除RG23较早交易）保持原strict High/Low确认和所有原其他条件，只改变扫边收回的结构参照：当前与previous reclaim从最近四根小时range边缘改为已有闭合1h EMA20，沿用原0.10ATR/ATR_prev缓冲与同向实体/量价条件。不新增指标或调缓冲网格。先完整JSON、真实接收者闭合EMA1/2对齐、独立可达性/whole父对象与退出检查、前端/收益前协议，再AF0/RG21/新候选×四币49月完整重撮合。原门槛与未阅币不变；goal active，仍无可发布策略。

