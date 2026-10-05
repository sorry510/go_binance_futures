# RG11 闭合小时主动压力交接与价格接受（收益前冻结）

2026-10-04北京时间07:46:00，3890 Go/Expr合成全部通过之后、RG11任何完整收益前冻结。当前goal active，RG10主及开/关/成本全部actual terminal exit0且无遗留进程；已完成12完整49月run但联合失败，报告与两个失败完整JSON保留，不重复旧主。

## 诊断与机制

新canonical描述性诊断72882 actual terminal exit0：RG10完整263补充与原主/实际入口审计/hash/四币data身份逐项一致，两个后续闭合小时0censored。第1小时115主动quote多数转为入场方向、148仍相反，实际净归因+167.309/-732.066；第2小时107/156，归因+83.123/-647.879。第1小时36失去旧拒绝边界、净归因-298.308；未失去227笔归因仍-266.448，不足以只加保持条件。第1小时27笔已经退出、第2小时80笔已退出；future market数据不能作持仓内信息。两个horizon复用263笔而非526独立入场。

诊断results/20261004-rg11-rejection-path-attribution.json SHA8121245479e9894de142973751b575a84684179ac5c87267793719fd17e33f9b。PriceResponseATR相对旧闭合trigger Close而非真实fill收益；压力“交接”只是主动quote多数方向切换，不识别钱包、订单簿或因果。实际未来分组收益不是筛选、反转或延迟策略收益。不能将未来字段放回旧RG10入场。

新假设在自己的当前入场时只读取已闭合小时：此前8小时反向价格及有效加权主动压力一致，前一闭合小时仍反向主动多数，最新闭合小时主动多数交接到候选方向并放量、收盘穿越前一小时的对向高低点。观察价格已经接受压力交接，而非RG10在主动多数仍反向时推断吸收。

完整只读v67、v80、RG1配置，未读取它们的收益：v67为强4h EMA/DMI/ADX、相邻TakerBuyRatio cross、形成价穿越最新High/Low并自有trend-only关闭；v80为三小时price/ratio背离、forming恢复和ROI-only关闭；RG1为二次新极值试探、绝对buy/sell额轮换和forming边界恢复。RG11弱closedADX、8小时有效加权压力/价格、闭合quote多数交接及自己的闭合价格接受不等于上述旧完整程序；不借用它们的退出或新增EMA2。403个限定完整可移植配置/884同侧全入口比较，无精确去空白重复；非全语义/alpha新颖证明。检索初次把technology误当数组的空结果已纠正，不用该空结果作证据。

## 候选与唯一机制替换

temp_strategy/20261004-closed-hourly-pressure-handoff/：
- 00-closed-hourly-pressure-handoff-family.json，SHA760fc46eca907ec938cc385608df98b4eea4596d84a9718c20fba7c139f8366d，version025e602a2d11e548e69a92577ece9f1063ec0afc4bd4c003eb8213826e79b5f3。
- 01-v29c-closed-hourly-pressure-handoff.json，SHA0ed2cf7e64766d714b8c1fbcb76dd0030528ab38775c9eb2e23c2ed71d41c455，versionc07fca00571c1266b128007890fa6b92617b710009119d9d6e4a5488115c6dfc。

族4/组合6启用规则，仅替换RG10两补充全程序为rg11_closed_hourly_pressure_handoff_long/short；原9 technology、AF0基础开仓整对象、完整统一RG4关闭整对象保持。不携带RG10极值sweep/reclaim要求，不追加RG9 RSI55/45、RG8强loss、新指标/周期或按币/按hash路由。

LONG：closed4h ADX<20；[2:10]8个闭合小时quote/taker合法，加权主动卖出多数、Close[2]<Open[9]，且[2]仍主动卖出严格多数。[1]quote大于此前8小时均值、合法主动买入严格多数；Close[1]>High[2]+0.10ATR1，body上涨0.15..2ATR1。SHORT全部镜像。closed日线ADX>=20且DI反向排除保持；forming[0]仅要求已观察quote>0及相对closedClose1的-0.15..+0.35ATR1保持。有意Data[0]不变，不依赖forming taker/ratio0。原配置的EMA/ATR2未新增为补充入场条件；前8极值也不是门槛。

20、8小时、0.10/0.15/0.35/2ATR、50%继承先前明示尺度，未做收益网格或放松频率。强趋势仍由原AF0判断，弱趋势由压力交接补充；不要求每种市场状态每时刻成交。

## 已实际验证与计划回测

33162 actual terminal exit0：3890 passed/0 failed。results/20261004-rg11-expr-checks.json SHA10f8526c9a371341da4243517d3d9f8b09dd6cf6b2190f1f8f02f938a0f8faf0。新增[1]/[2]quote多数合法性/严格等值cross、闭合接受High2/Low2+0.10ATR严格边界、quote加权且前小时仍反向、量纲缩放、body/live边界；forming taker/ratio及无关EMA/ATR2/QPS/前8极值独立；原dailyADX/DI/配置/base可达/全关闭矩阵及唯一替换整对象保留。真实Go/Expr与本地有序规则模型不是private selector/cached sequential/API/live/forward/独立指标数学或盈利证明。

AF0/RG10/RG11×BTC/ETH/SOL/XRP，12完整run各初始1000；UTC2022-09-01含至2026-10-01不含，1491天/213周/至少192笔每币。联合硬门槛每币>=0.9次/周、真实成本净正、四年稳定、跨币泛化全部不变。报告四完整Sep–Aug年、额外2026-09、日历2023/24/25/2026 Jan–Sep；45月exit cohort仅归因不冒充独立初始1000收益。49月开发已观察、非时间holdout；AAVE/ATOM/ETC/LINK收益未读，资格待核，开发失败不读验证收益。

原backtest_engine_v7/standard_1m，分钟观察close/下分钟Open不利5bps，当前现金10%保证金×8、每侧fee0.0005、真实funding率/时点及原缺mark分钟Close回退，外部5/5调用gate、AutoStopOrder=false、whole统一RG4退出保持；普通ROI-only不平，ROI<=-20唯一灾难例外。精确funding结算mark仍pending，RG10全4016应用/1066回退，自身1314/379，不补造mark/改成本。

冻结source public-canonical-repaired-v2-funding-tail-v1/public-archive双方/verified-archive20261003-v2，四data hash与原helpers SHA/tail54 overlap7 manifest身份保持。actual seed200input/199closed，不改递推种子/容差/默认warmup。

主输出results/20261004-rg11-development4-canonical-repaired-v2-funding-tail-v1.json。预声明：8个AF0/RG10共享控制逐笔/metrics/annual/source重现；会计、入口族/方向/集中度归因；全部真实新入口canonical[1..10]、有效前8权重/反向价格、[2]→[1]quote多数交接、closed High2/Low2价格接受/body/live/daily/actual-seed原函数与完整Expr；实际正常关闭原Position/gross mark-price-denominator ROI/outergate/whole统一RG4，forced-end单列；全原复利仓位/fill/双fee/funding inclusion及缺mark回退/minute活动另核。不把静态删组或未来归因当可实现收益。

## 边界与恢复

最新ARM仍07:04:21.904/23.659/25.402只读元数据，222/219/8结果、17模板各库，v29摘要及go_binance全导出与RG8相同；非449条新forward内容分析。配置/原引擎/环境/cache/helpers/正式1.0.8技能及两pending草案SHA保持，无App/DB写入/分配/启用/下单/生产/前端/conf/app.conf/新仓库测试文件。临时Go/overlay/二进制/大输出仓库外，virtual实际不得创建，完整JSON失败也留temp_strategy。

实际启动后记录真实handle，只poll同handle不因timeout另起。阶段完成不自行complete/paused/blocked；下次第一条先本阶段总结再核真实goal/进程，paused停不自恢复。阶段总结技能草案结构valid/四行为pending/aggregate null，未委派/strictwin/promote。

