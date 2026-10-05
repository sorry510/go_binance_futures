# RG12 强四小时趋势内量价回撤恢复（收益前冻结）

2026-10-04北京时间08:24:28之后，3926项真实Go/Expr合成检查全部通过、开/关审计工具实际编译exit0之后、任何RG12完整收益前冻结。当前目标active；本次新续接开始已先给RG10/RG11阶段总结，不重复已完成主回测。

## 诊断与假设

RG10极值拒绝和RG11弱趋势主动压力交接均联合失败。RG11四币频率通过，但SOL总净亏、四币亏损完整日历年；574补充的单一闭合净主动额强度诊断中，强于前八小时反向均额的四币分组净归因全部负。该分组只是原交易描述，不能据此直接加过滤、选币、删侧或反转，RG12不使用这个诊断阈值。

新假设将同一闭合压力交接与价格接受放在已有强四小时方向内：八小时价格和加权有效主动额先反向回撤，再在自己的最新闭合小时同时恢复。不是弱趋势猜反转，也不是向旧入场填入未来小时。

本次完整只读v34、4h趋势1h回归v5、强趋势精选回踩v12、4h趋势ATR回踩v12、v67配置，未读取它们的收益。v67已有强四小时方向和相邻占比cross；其他配置已有EMA/RSI/ATR回踩。因此本轮是可证伪的趋势内回撤机制对照，不宣称全新alpha。限定扫描排除所有research路径、诊断名称及超过128KiB文件，共296完整portable/665同侧启用全入口去空白比较、0精确重复；非全局语义或盈利优势证明。

## 完整候选及唯一变化

temp_strategy/20261004-strong-trend-pullback-pressure-handoff/：
- 00-strong-trend-pullback-pressure-handoff-family.json，SHA9b6603a91eda1d9bd9af5cfd3c7c1c6652471d20190478c6217b3243cad1afcc，version1f021fea0c50618deedae0f2e4ac5ddb2fb54fc90438bc948fb4cb7a27e4552d。
- 01-v29c-strong-trend-pullback-pressure-handoff.json，SHAffb7b20784944fb14f38e835d11fa01827c1c1b9f3b4e0f7e60ed4bb3971d5fb，version22ec73f30484962cbbedd3a43d232645ffa27039dda82c894d67f41e65e0d13e。

族4/组合6条启用规则。只替换RG11两补充的regime全条件及名称：closed4h ADX[1]>=20，LONG closed EMA20[1]>EMA50[1]且PlusDI[1]>MinusDI[1]；SHORT全部镜像。原9配置、AF0基础开仓完整对象、whole统一RG4关闭完整对象与所有其余补充条件保持。没有EMA2、新指标、周期、斜率或强度网格；不按币或入场hash路由。

保留的LONG确认：[2:10]八闭合小时quote/taker合法、加权主动卖出严格多数且Close2<Open9；[2]仍主动卖出多数。[1]合法主动买入严格多数、quote大于此前均值、Close1>High2+0.10ATR1、同向body0.15..2ATR1。SHORT镜像。closed日线ADX>=20且DI反向排除保持。forming[0]只使用已观察quote>0与相对closedClose1的-0.15..+0.35ATR保持，不读取forming taker/ratio0；Data[0]实时设计不修改。前八极值、ATR2及QPS不是新增补充门槛。

该研究候选只在已确认强四小时方向中补充；弱/冲突状态可选择不交易，不冒称所有市场状态都有开仓信号。20、8小时、50%与ATR尺度继承已声明口径，不放松0.9次/周门槛。

## 已完成的实现核验

75589 actual terminal exit0：3926 passed/0 failed，results/20261004-rg12-expr-checks.json SHAbb6045910db19039a92eff7658ea159dadadb9e00f6c240cdf0e77a29b4d0e6a。覆盖强ADX20严格边界、closed EMA/DI方向及等值，forming0反向独立；原量价交接/有效加权历史/body/live/daily/整对象/基础可达与完整统一关闭矩阵保留。真实Go/Expr及本地有序模型不是私有selector、全顺序缓存、API/live/forward、独立指标数学或盈利证明。

1659/74031实际编译exit0，JS syntax checks exit0。开审计新扩展原fresh receiver导出closed4h DI，按实际200input/199closed canonical种子重算ADX/DI/EMA并核强方向资格；不拿原weak门槛冒充。虚拟诊断source实际不存在，所有Go/overlay/二进制/大结果保留仓库外。

verification源SHA：rg12_expr_checks.go=1a5030e70bd23ec46e045f5a6dc0cc3d4db679ee2216d03a6de877a53f596488；rg12_closed_signal_audit.go=ec288fafb47d88243350bb51eca028d3f5dbd6f83ec98e705f8c934f78e27d2a；rg12_signal_receiver_bridge.go=8faa247f4cbe657fc12d2a050b8edd8a92dea76975a25ef66d8d6adf135ca08c；rg12_close_signal_audit.go=cf7136ba5b2c49da66650f013f748ea4f6b0d896855f91a5148a058406b01cf8；rg12_accounting_audit.mjs=ad549ebe8607d401282726b7f98b76974bdb18782d77a2ee81412ead7f815b8a；overlay=d4e6483be2cfa7d6f8642e3cd3dd640e360def1a0a0b80eeabf40f88d947b880。

## 预声明完整回测与审计

AF0/RG11/RG12×BTC/ETH/SOL/XRP，共12完整run各初始1000。同UTC2022-09-01含至2026-10-01不含，1491天/213周/每币最少192笔。每币>=0.9次/周、真实成本净正、四年稳定、跨币泛化联合硬门槛不变。四完整Sep–Aug年、额外2026-09与日历2023/24/25/2026 Jan–Sep全部报告；不挑有利年度。45月exit cohort仅归因，不冒充独立初始1000收益。开发四币49月已观察，非时间holdout；AAVE/ATOM/ETC/LINK验证收益未读、资格待核，开发失败不读。

保持原backtest_engine_v7/standard_1m、观察分钟close/下一分钟Open不利5bps、当前现金10%保证金×8、每侧fee0.0005、真实funding率/时点及原缺mark分钟Close回退、外部5/5调用gate、AutoStopOrder=false；统一RG4完整关闭不变，普通ROI-only不平，ROI<=-20为唯一灾难例外。ROI是gross/(absqty*currentMark)*8*100，不是净保证金收益。

资金费历史官方文档将markPrice定义为相应funding结算标记价；定义不是历史完整覆盖保证。[Binance原文](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/market-data#Get-Funding-Rate-History)。此前12同目标项目source及原HTTP样例mark为空证据保持；本轮继续单列实际缺mark回退，精确交易所结算成本资格pending，不补造mark、减fee或换成本引擎。

冻结source public-canonical-repaired-v2-funding-tail-v1，execution/indicator均public-archive、verified-archive20261003-v2。四data hash和54追加/7重叠funding tail manifest 78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547保持。原pv5 replay/data helper SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 / a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63。

主输出results/20261004-rg12-development4-canonical-repaired-v2-funding-tail-v1.json。主后：8旧AF0/RG11控制逐笔/trades/metrics/annual/source完整重现（仅cache_hit是获取/复用观察）；全部会计/方向/族/集中度归因；每个真实补充原receiver/canonical小时80字段、4h ADX/DI/EMA五值及强方向、daily三值、前八有效加权反向量价/[2]→[1]多数交接/price接受/body/live/fullExpr；全部正常原Position/gross ROI/outergate/完整关闭与forced-end单列；所有12run原复利仓位/fill/双fee/funding纳入及缺mark回退/分钟活动。描述性删除/分组不是可实现策略收益。

## 当前状态与边界

56921真实ARM只读程序exit0，08:09:28.326/30.843/33.110三个库各17模板、222/219/8结果；v29 id68/45/46，go_binance完整JSON逐字与RG10旧导出相同SHAb511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0。元数据SHA269acc194a7175c60ff8430e1c9b777f24a529d109e5c40cfcbe8bd68322c4e5。仅metadata/v29身份核对，不冒称449条新forward内容分析。

conf/app.conf与原engine/environment/indicator_cache/正式技能SHA保持，无App/UI、DB写入/分配/启用/下单、生产/前端/config/新仓库测试文件；其他dirty研究不动。初次生成调用嵌套反引号解析失败发生于任何工具动作前，改普通拼接后成功，没有删失败结果或改候选。

启动后只poll真实同handle至actual terminal；timeout不重复启动。阶段完成不自行complete/paused/blocked。下次有恢复授权时第一条先给阶段总结并核真实goal/进程，paused停不自行恢复。正式技能1.0.8及两pending草案不变；行为评估pending/aggregate null，不委派、不称strictwin或晋级。
