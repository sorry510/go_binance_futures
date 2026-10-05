# RG26 两bar完整极值续行：完整失败与两轮对比总结

2026-10-05北京时间15:30:42，主24627、开5187、关9575、成本61228已actual terminal0；natural/phase actual exit0。唯一裁决 **invalidated**，release_qualified=false。目标仍active，当前没有满足原门槛的可用策略，没有真实活句柄。两轮RG25/RG26共24完整49月run（包含对照复跑，不是24独立收益样本）。

## 原门槛下的完整结果

AF0/RG25/RG26×BTC/ETH/SOL/XRP，12完整49月1491日213周run；本地v29 ID114再次只读语义exact。初始1000/币、8x/currentcash10%margin、outer5/5、每侧fee0.0005/不利slip5bps、原actualfunding和原missing-mark minuteClose回退不变。每币至少192笔以及四年/真实成本/跨币门槛不放宽。

| 币 | 笔数 | 次/周 | 毛收益 | 费用 | 资金费PnL | 净收益USDT | 净PF | 回撤% | 平均持有h |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| BTCUSDT | 182 | 0.854460 | 865.101251 | 180.824954 | -22.786916 | 661.489382 | 1.366598 | 18.112190 | 27.688187 |
| ETHUSDT | 205 | 0.962441 | 287.463340 | 157.647905 | -12.621272 | 117.194162 | 1.065908 | 27.313362 | 17.823496 |
| SOLUSDT | 222 | 1.042254 | -34.566187 | 153.128371 | -22.767672 | -210.462230 | 0.905799 | 39.763666 | 12.104429 |
| XRPUSDT | 154 | 0.723005 | 638.632476 | 145.604667 | -3.614837 | 489.412971 | 1.269735 | 21.834619 | 15.646104 |

BTC/XRP频率失败，SOL总净亏。更完整突破没有共同改进四币。每币频率都下降；ETH/XRP累计净改善，但不能用单币累计优势替代完整年/频率/成本/泛化。

| 币 | RG25→RG26笔数 | RG25→RG26净收益 | 差额，仅两次完整回放对比 |
| --- | --- | --- | --- |
| BTCUSDT | 191→182 | 750.160948→661.489382 | -88.671566 |
| ETHUSDT | 221→205 | 27.274771→117.194162 | 89.919391 |
| SOLUSDT | 227→222 | -226.961215→-210.462230 | 16.498984 |
| XRPUSDT | 164→154 | 300.417038→489.412971 | 188.995933 |

差额是两个完整路径真实研究结果，不是静态删407旧交易的反事实收益。改变cap/trigger会改变后续entry、占用和复利。

## 年度稳定性

| 币 | 2022-09→2023-08 | 2023-09→2024-08 | 2024-09→2025-08 | 2025-09→2026-08 | 额外2026-09 |
| --- | --- | --- | --- | --- | --- |
| BTCUSDT | 88.633931 | 146.403939 | -51.128111 | 576.777061 | -99.197438 |
| ETHUSDT | -33.079173 | -95.233948 | 85.482243 | 198.205491 | -38.180450 |
| SOLUSDT | -187.740746 | 33.443764 | -62.658072 | 44.142343 | -37.649520 |
| XRPUSDT | 183.145624 | -70.393202 | 134.367144 | 245.337738 | -3.044331 |

| 币 | 日历2023 | 日历2024 | 日历2025 | 2026 Jan–Sep |
| --- | --- | --- | --- | --- |
| BTCUSDT | 265.710528 | 26.695202 | 36.966262 | 379.128619 |
| ETHUSDT | 48.326209 | -131.027211 | 46.550335 | 204.253159 |
| SOLUSDT | -0.625971 | -227.912947 | -138.996404 | 130.575760 |
| XRPUSDT | 92.487385 | 187.390956 | -181.186552 | 392.648793 |

四币均有负完整Sep-Aug年，BTC较RG25还新增2024-09→2025-08负年。ETH2024负、SOL2023/24/25负、XRP2025负。年度按原顺序ledger平仓时间归因，并非独立初始化。移除最佳五笔的描述净额=303.104524/-193.561887/-538.520911/59.400934; 不可执行删除，不证明可贸易alpha。

## 补充家庭与自然趋势

364条新补充+399条本组合自身base=763，不静态减AF0控制组429笔。

| 币 | 补充笔数 | 毛收益 | 费用 | 资金费PnL | 净收益 | weak净 | strong净 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| BTCUSDT | 90 | 258.938397 | 89.606360 | -2.056199 | 167.275838 | 90.254622 | 77.021217 |
| ETHUSDT | 110 | -257.441364 | 84.265918 | -2.838248 | -344.545530 | -184.535048 | -160.010483 |
| SOLUSDT | 94 | -39.877468 | 65.422083 | -3.045990 | -108.345541 | -21.744699 | -86.600842 |
| XRPUSDT | 70 | -108.388388 | 64.943297 | 1.827617 | -171.504068 | -125.259227 | -46.244841 |

ETH/SOL/XRP补充毛收益已负；费用不是唯一根因。没有跨币四年共同稳定weak或strong组；ETH弱组四完整年全负，SOL/XRP弱强都不稳。不得静态过滤弱/强、只保留获利币/单边或从净亏推断直接反转可赚。

RG25完整几何证据407/131未过推进极值/57原cap不可达说明更严格确认确实是非等价新准入；RG26实际两bar极值全部通过，但仍不稳定。说明“局部突破不充分”不是唯一问题，不继续仅调整同一延续机制的阈值、cap锚或币别参数来制造通过。

## 核验范围

- 全会计12run/1995笔/八AF0-RG25完整控制snapshot、账目、metrics、data/source exact，errors=[]。两轮all账目项合计3860，包含重复控制交易，不是3860独立交易。
- 开仓364补充/29120closedfields/1820fourh/1092daily/728ATR/728unused1hEMA parity/2912priorquote/1092current activity/364strict followthrough、所有新two-bar level、闭合价量/主动多数/半实体和regime/flow合法性、Float64字面QPS与0.35cap均通过，0failed。actualInputBars200、原199closed种子复算；原函数parity不是独立指标数学，freshreceiver不是全sequential/private/live/forward。
- 最小活动ratio0.9000263728836047，最大新锚advance0.34986785837572804ATR，最大相对旧pullback锚差1.3953746355799777ATR。仅描述，不作为参数grid或收益筛选。
- 全763 normal/0强制期末、old591/added173/both1/added-only172/0失败，整个uniformRG4信号确认保持；灾难ROI<=-20唯一无确认例外。outer5/5只是评估gate，AutoStop=false；完整平仓矩阵见原RG25冻结协议及RG26协议的风险段。
- all12成本1995笔/4516actualfund结算/1224原observed-minute mark回退/0零活动及失败；RG26自身763/1675/473/0失败，最大net误差2.2737367544323206e-13。control/own/allfailed_rows=[]。
- 精确venue funding结算mark、历史深度/容量/真实滑点、原未阅AAVE/ATOM/ETC/LINK仍pending；original算术/分钟活动通过不能冒称完整真实成本或跨币合格。开发失败，不启动未阅币验证。
- RG26真实Go/Expr6610/0，frontend真实VM两issue=null/9/四type、该次限定326portable753entries0exactdup。HTTPactualexit7、3333无listener，API pending；按skill用真实Go/Expr回退进行独立历史回测，没有自动启服务/App。HTTP及synthetic不是盈利测试。
- 主/诊断源、conf、正式SKILL的SHA当前保持，主未使用overlay。没有新增DB写、分配、激活、订单、App、生产/前端/conf、新仓库测试文件、删除或新委派。QPS经验仅未审批技能草案，结构valid不是strict-win/promote。

## 下一具体研究方向（尚无新收益）

整组continuation在更完整结构确认后仍缺乏共同优势，下一替换补充开仓家庭为 **反向放量冲击失败→闭合价格与主动quote重新转向→完整结构续行**，而不是只调cap或直接反转旧亏损交易。

预先定义LONG：闭合[2]是above-prior-eight-mean量的bearish实体、sellquote多数；闭合[1]变bullish且收复至少半实体、保持低点，闭合buyquote多数且成交额仍高于此前八小时均值但低于冲击bar；current严格越两闭合bar高点与原0.35ATR cap，原0.90活动/高阶方向及daily-veto。SHORT对称。这些是另一个反向冲击失败机制，不是静态逆转旧交易，不据净亏称盈利。

只换补充家庭；wholeAF0 base/order、wholeuniformRG4关闭、原九/interval、8x/outer5/5/currentcash/cost/date和所有gate保持。0.15..2冲击body/50%多数/半实体/quote均值都是预定义已有结构，不作阈值搜索。先完整JSON保存，再新独立合法OHLC/双方向flow/unusedforming边界及wholeparent校验、真实canonical新family审计、收益前协议和AF0/RG26/新候选四币49月12完整run。HTTP仍pending；未阅币不读，不插库/发布。

## 证据文件与hash

候选：temp_strategy/20261005-impulse-pullback-two-bar-extreme/两完整JSON（失败保留）。
协议：2026-10-05-impulse-pullback-two-bar-extreme-protocol.md，SHA59ef57bc089177a2ce7645760a434872fed9c49308d3bfad7fc9275e33adf5b6。
results/20261005-rg26-*.json位于 /Users/zhz/Library/Caches/go-binance-strategy-research/，独占输出不覆盖：

- main ec3e1e225242b7d0927ae9028707634b0e7fbf6168163cde238098fffd347cf0
- accounting 6e4fb416336d3b2c99e65855dcfa0662f991ef1110db351dba975d067e8b33aa
- opening dda8e0c28e179f44dc58a9b1608d9e9e4e5abdb522c81e0433aa6d8e9524d40b
- closing 0bb627c9a3a9d2ea14e43ebc148d347c2893f19a53761cc3a7d22fd7c1711cb3
- costs cbb6bfee3b2f86678cf8d6216dddfce62802a5d2d00ad5865c96c18b6ee3fc6b
- natural 29c8d395e03cb3586894c715aaf7fab6bc4e4c25dea2e8a93003dda4229f8991
- phase de5da6c4474f7691aa3c9352a4b3353c3946d110aea23c21997e831b89bba9b3

整个研究目标尚未达成；当前阶段是progress，不是goal完成/暂停/blocked。新研究假说没有收益，不降低任何门槛。

