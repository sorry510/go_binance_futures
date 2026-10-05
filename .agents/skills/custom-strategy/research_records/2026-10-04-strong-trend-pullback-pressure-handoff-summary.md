# RG12 强四小时趋势内量价回撤恢复：完整完成，联合失败

2026-10-04北京时间08:39:30全部主/会计/开/关/成本已actual terminal exit0。本续接新增12完整49月run/2331执行账目（含8个共享对照重复执行，不是2331独立市场交易）；目标仍active，没有达到全部要求的可发布策略。

## 当前结论

RG12没有通过：XRP 182笔/0.854次每周低于192笔/0.9；SOL总净亏；四币均有负完整Sep–Aug年。BTC三个完整日历2023/24/25净正，但第一个完整Sep–Aug年仍负，不能换年度定义声称四年稳定。ETH/SOL/XRP还有负完整日历年。精确funding结算mark成本资格仍pending；开发失败，不读取AAVE/ATOM/ETC/LINK验证收益、不挑币或方向、不放松费用和频率。

实现检查通过不等于策略有效：完整strong4h ADX/EMA/DI和量价规则确实按预期运行，损失不能归因于此轮表达式或数据下标写错。既有强趋势概念和相邻flow cross早已存在，本轮是趋势内回撤确认的可证伪对照，不宣称全新alpha。

## 同口径四币总结果

每个完整run各初始1000 USDT；当前现金10%保证金×8，原手续费、5bps不利滑点、真实funding率/时点和缺mark原分钟Close回退保持。同UTC2022-09-01含至2026-10-01不含，1491天/213周；所有原9指标、AF0基础入口和whole统一RG4关闭完整对象不变。

| 币 | 笔数 | 次/周 | 净USDT | PF | 最大回撤% | 平均持仓小时 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | 213 | 1.000 | 86.499 | 1.045 | 23.250 | 34.916 |
| ETHUSDT | 242 | 1.136 | 347.368 | 1.137 | 26.647 | 21.108 |
| SOLUSDT | 264 | 1.239 | -268.207 | 0.902 | 41.454 | 12.987 |
| XRPUSDT | 182 | 0.854 | 228.433 | 1.115 | 27.421 | 17.852 |

与完整重现的RG11相比，BTC/ETH/SOL总净更差，XRP总净改善但频率失败。不能只看四币组合总净或挑BTC/ETH/XRP来通过跨币要求。BTC/ETH持仓34.916/21.108小时较RG11的22.614/13.157延长；是实际持仓统计，不是删除新入场的反事实收益。

## 年度覆盖

下表是原连续复利执行账目的exit-time归因，不是逐年重新初始1000的独立回测。

| 币 | 2022-09至2023-08 | 2023-09至2024-08 | 2024-09至2025-08 | 2025-09至2026-08 | 额外2026-09 |
| --- | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | -133.940 | 8.052 | 25.699 | 265.690 | -79.001 |
| ETHUSDT | -85.068 | 5.909 | 166.802 | 301.954 | -42.230 |
| SOLUSDT | -185.829 | -49.980 | 9.714 | -3.460 | -38.652 |
| XRPUSDT | 20.119 | -150.323 | 120.550 | 248.999 | -10.911 |

| 币 | 日历2023 | 日历2024 | 日历2025 | 2026-01至09（非全年） |
| --- | ---: | ---: | ---: | ---: |
| BTCUSDT | 11.935 | 2.052 | 66.129 | 87.952 |
| ETHUSDT | 32.729 | -155.468 | 244.918 | 295.068 |
| SOLUSDT | -89.856 | -161.839 | -86.484 | 15.661 |
| XRPUSDT | 107.186 | 1.004 | -167.084 | 335.467 |

2023-01后的45个月只能作exit cohort归因，不冒充独立45月本金/收益；四币49月开发已经观察，不是时间holdout。没有用有利币、侧、周期或年份替代联合裁决。

## 新补充族归因

组合实际901笔=538补充+363基础，而非901减AF0独立429来认472补充。主顺序、持仓占用、资本路径和统一退出均可能影响入场，原交易删组或相减不能恢复另一策略收益。

| 币 | 方向 | 笔数 | Gross | 双侧fee | Funding | Net |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| BTCUSDT | SHORT | 72 | -277.291 | 55.735 | 10.026 | -323.000 |
| BTCUSDT | LONG | 57 | -29.229 | 43.180 | -14.267 | -86.675 |
| ETHUSDT | SHORT | 77 | 57.972 | 61.651 | 7.923 | 4.243 |
| ETHUSDT | LONG | 74 | -28.454 | 60.360 | -12.018 | -100.832 |
| SOLUSDT | LONG | 70 | -190.504 | 47.870 | -4.721 | -243.095 |
| SOLUSDT | SHORT | 80 | -165.575 | 55.134 | -4.305 | -225.014 |
| XRPUSDT | LONG | 63 | -98.181 | 49.782 | -6.952 | -154.915 |
| XRPUSDT | SHORT | 45 | -158.714 | 34.934 | 4.035 | -189.613 |

新补充LONG四币gross和net全负；SHORT只有ETH净约4.243微正。BTC/SOL/XRP补充gross合计已经负，ETH补充gross约29.518仍不足覆盖费用，因此不能靠减费用解释或修复全部问题。静态without-best-five四币均负（-202.599/-51.714/-578.590/-130.645），仅表明收益集中，不代表可实现删除策略，更不授权反转、删侧或选币。

## 实际完成的核验

- 主48358 actual terminal exit0：AF0/RG11/RG12×BTC/ETH/SOL/XRP，共12完整run/2331账目。8旧AF0/RG11控制trades/metrics/annual/source全部完整重现，仅cache_hit为获取/复用记录，身份不改。会计node actual exit0/errors0；全部价格/数量/双fee/净恒等式、方向/策略hash/年总数/净/频率校验通过。
- 75589 actual terminal exit0：3926 Go/Expr合成0失败，强ADX20、closed EMA/DI严格方向及forming反向独立、前八有效加权反向压力/价格、[2]→[1]多数交接、High2/Low2+0.10ATR严格接受、body/live/daily边界、原9配置/base/whole关闭矩阵与唯一regime替换整对象通过。不是私有selector/API/live/forward/独立数学或盈利验证。
- 开80712 actual terminal exit0：全部538真实补充、43040闭合[1..10] OHLC/quote/QPS/taker canonical字段、2690原closed4h ADX/PlusDI/MinusDI/EMA20/EMA50数值、1614日线值0失败；actual200input/199closed种子重算及strong方向、前八有效加权量价/交接/价格接受/body/live/fullExpr全部通过。四小时DI/EMA是真实新增审计门槛，不复用weak判断冒充。
- 关13286 actual terminal exit0：901正常关闭/0 forced-end/0失败，全部原Position/gross current-mark-price-denominator ROI/outergate/whole统一RG4通过；old862/added41/both2/added-only39。按币added-only22/9/0/8。只是同环境条件归因，不是私有短路日志或删除分支反事实。
- 成本52037 actual terminal exit0：全2331笔/5040真实funding应用/1466原观察分钟mark回退，0零活动fill/0算术失败。自身901/2335/748；证明原复利仓位、下一分钟Open不利fill、双fee、funding时点包含和原回退算术一致，不证明全结算精确mark覆盖。
- 真实前端src/utils/technology.ts经既有TypeScript在隔离Node VM调用，两完整JSON均issue=null/9启用指标/规则对象类型通过；没有App/UI、安装、修改或完整frontend build/交互声明。

fresh receiver核验不是全顺序缓存/私有selector/实盘API/forward/独立指标数学/容量或订单簿证明。Data[0]实时设计有意保留，新开仓不依赖forming taker/ratio0，退出仍原forming价格和指标逻辑。

## 完整候选和身份

两完整JSON保留temp_strategy/20261004-strong-trend-pullback-pressure-handoff/：
- 00-strong-trend-pullback-pressure-handoff-family.json SHA9b6603a91eda1d9bd9af5cfd3c7c1c6652471d20190478c6217b3243cad1afcc，version1f021fea0c50618deedae0f2e4ac5ddb2fb54fc90438bc948fb4cb7a27e4552d。
- 01-v29c-strong-trend-pullback-pressure-handoff.json SHAffb7b20784944fb14f38e835d11fa01827c1c1b9f3b4e0f7e60ed4bb3971d5fb，version22ec73f30484962cbbedd3a43d232645ffa27039dda82c894d67f41e65e0d13e。

收益前协议2026-10-04-strong-trend-pullback-pressure-handoff-protocol.md保存。限定296完整portable/665同侧启用全入口去空白比较0精确重复；排除research/诊断名/>128KiB，不是全局语义/alpha证明。只替换RG11补充regime为strong4h ADX>=20与同向closed EMA20/50及DI；所有其余量价/基础/全关闭/成本门槛不变。

结果均在/Users/zhz/Library/Caches/go-binance-strategy-research/results/；所有源helper/overlay/二进制在verification/：

- 20261004-rg12-development4-canonical-repaired-v2-funding-tail-v1.json：SHA 8c0087690cdedd43a5a8af9af2ebb781e9058350352168a7761d5898a4f15b63。
- 20261004-rg12-accounting-summary.json：SHA 5b571d711a7837bc5da4e109af3b5064d718f6044c34bc93264eb8ea0192b35b。
- 20261004-rg12-expr-checks.json：SHA bb6045910db19039a92eff7658ea159dadadb9e00f6c240cdf0e77a29b4d0e6a。
- 20261004-rg12-canonical-actual-seed-open-signal-audit.json：SHA a32c0a98ef23a077bed0548ee5a91021825d39a3788084b497de6a42c355238a。
- 20261004-rg12-execution-original-cost-fallback-audit.json：SHA c4fac3d002bbeb36e6efc80c74a3ed2861ac11f20627f4b58cc30dbc52f3e5de。
- 20261004-rg12-original-position-close-signal-audit.json：SHA ec3c8a1cb999a4413d3933e2327d93cfd66b1516e3c3aa88aad636df9778f88b。

四币canonical DataHash、原helpers、public-archive双方/verified-archive20261003-v2和funding tail54新增/7重叠manifest身份保持，详见收益前协议/原数据合同。官方文档将funding history markPrice定义为对应结算标记价，而非覆盖承诺。[Binance文档](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)。此前12同目标SDK/原HTTP空mark证据保留，不补造结算价、调低fee或改成本引擎。

## 研究结论和下一步

单纯让4h EMA/DI/ADX同向并不能证明“八小时下跌后的一小时压力交接”是有效趋势回撤恢复。新LONG各币gross负，须先检查恢复位置：是否仍在既有1h EMA20的逆向一侧，而只穿越前一小时High/Low；同时保持实际价量身份。下一步只做已观察入场的canonical位置描述，不把未来数据放回入场，不将静态分组当筛选/反转策略收益。

可预声明后续替代机制：用已有1h EMA20的自身闭合收盘恢复、同向合法主动quote多数和放量确认取代固定八小时反向累积及单小时极值接受，继续保留基础/九指标/whole关闭/风险/费用/频率/年度/跨币全部约束；这仍是待验证假设，RG13尚未生成/冻结/测试，不以描述性盈利分组决定阈值。开发失败不看AAVE/ATOM/ETC/LINK收益。

## 边界与恢复

本轮ARM最新08:09只读元数据及v29身份（各17模板、222/219/8结果），不是449条新forward完整内容分析；conf/app.conf与原engine/environment/indicator_cache/正式技能SHA保持。无App/UI、DB写入/分配/启用/下单、生产/前端/config/新仓库测试文件，其他dirty研究不动，全部失败完整JSON保持。初次嵌套反引号解析失败在任何工具动作前，普通拼接修复后成功，不改候选。

正式技能1.0.8和两pending草案保持，行为评估pending/aggregate null，未委派/strictwin/promote。阶段完成不是目标完成，不自行complete/paused/blocked。有明确恢复授权时下一次第一条先本阶段总结、再核真实goal/进程；paused停不自行恢复。已结束句柄不能再poll或重启旧主。

## 后续诊断实际结果与最新下一步（08:57更新）

78009 actual terminal exit0：上述小时EMA20位置诊断完成，全部538原入场配对/1076闭合EMA值/0失败。结果results/20261004-rg13-hourly-recovery-attribution.json SHA1b333814ae1af0387ece40cd81ed9169fac6a6b6c9e5dc9a019e1f63d0f1490c；RG13字样仅诊断标签，不是新策略完整回测。自身入场已恢复均线的freshRecovery组BTC/ETH/SOL/XRP53/63/69/38笔，原净归因-306.190/-18.930/-226.028/-190.839，四币均负；latestAccepted组也四币均负。因此不简单追加EMA20恢复过滤，不据静态组收益生成反转/删侧或筛选候选；没有生成该过滤的完整JSON。这不证明所有其它EMA策略无效，仅否定把这个旧组直接认作改进依据。

已完整只读v11/v89/压缩假突破v2/v54相关配置而不读收益。最新待验证假设转向此前相邻等长四小时区间相对收缩，再闭合价格对最近四小时范围新鲜突破、合法同向主动quote多数与放量。保留原指标/基础/完整关闭/风险/成本/全部发布门槛，weak与strong方向允许由原技术数组判断；不用失败均线过滤、不增加BOLL/KC/其它指标，也不借旧关闭。RG13完整族/组合仍未生成、收益未读，须先真实矩阵、差异/身份核对、收益前冻结再完整撮合。

08:57:15实际get_goal active；限定研究进程无残留、config/engine/environment/indicator_cache/正式技能与两pending草案SHA保持、三个virtual实际不存在、git diff --check通过。所有失败JSON/源和大证据保留，未观察验证币收益不读，无App/DB/生产/前端/config/新仓库测试文件；用户其它dirty研究不动。下次先给本最新阶段总结，再核真实状态，不重复结束句柄或将阶段结束冒充目标达成。
