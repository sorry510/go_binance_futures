# RG9：闭合小时动量支持价量脱离（首次收益前冻结）

2026-10-04北京时间03:21:55，3662真实Go/Expr检查0失败后，任何RG9完整收益前冻结。实际goal active。RG8完整12run/2441账目和开/关/成本/会计已完成并联合失败，不重复原主，也不把失败RG8退出作为本轮父版本。

## 完整候选与单一变化

- temp_strategy/20261004-closed-hourly-momentum-flow-escape/00-closed-hourly-momentum-flow-escape-family.json：SHA2ff3a486f541a31396ca4c23ba3fd535d5c0158facde0485c2c240ac731602fb，version89823f32abecce632b0c805a7bd87f9cde6328ab98ce918f09ad3628b4f89c07。
- 同目录01-v29c-closed-hourly-momentum-flow-escape.json：SHA10587889b393dd4e25aef21fbb533f4f2d0ae250ae701103fb855caffa3a1628，version5aa9b5be8a1fe791fa09ff8a1b75278a38cda0f14aa377f93efeea35d200c762。

族4/组合6启用规则，原9技术配置/1h4h1d、AF0基础开仓、完整RG7（即RG4统一结构）关闭整对象完全不变。仅在两个RG7弱趋势补充入口的全部既有条件末尾添加已有闭合1h RSI14方向确认：LONG Data[1]>=55，SHORT Data[1]<=45；名称改为rg9_closed_hourly_momentum_flow_escape_long/short。55/45沿用v29基础入口原阈值，不进行收益参数网格。日线反向排除、8小时闭合有效主动quote和价格支持、4h EMA方向、fresh逃离/放量/body/形成价格保持均保留。

假设：价量脱离没有足够闭合小时方向动量时，可能只是弱趋势反复扰动。本轮检验这一维度，不声称已证因果；可能损失交易频率而失败，不能用换币/删方向/降0.9门槛救回。强4h区域仍由原AF0处理，弱4h区域由补充处理，市场状态全部有判定逻辑，不是每种状态必须交易。

仅只读temp_strategy和strategy_templates完整配置，排除新目录后409旧portable/895同侧完整入口比较，无精确去空白重复；不是语义等价、概念新颖或alpha证明，没有读取这些旧配置对应收益。原Data[0]形成价格/活动有意保留，新RSI条件只读闭合[1]，不添加forming taker0/OpenStrategyHash路由、新指标/周期/变量或系统代码改动。

## 合成检查与统一关闭

75730实际terminal exit0，verification/rg9_expr_checks.go/results/20261004-rg9-expr-checks.json：3662 passed/0 failed，结果SHA5227727a31a262679c57c7267d2823f6f6f4c4a1ab15de20b730058f3b36b729。RSI闭合0/44.999999/45/45.000001/50/54.999999/55/55.000001/100和forming0/100交叉、两侧边界均覆盖；原日线/4h方向/主动quote有效与加权/价格/body/live/基础可达矩阵保持。静态断言唯一新增条件、原配置与所有关闭整对象一致。不是3662市场盈利样本、private selector、全顺序cached、API/live/forward证明。

普通ROI+5/-5/+16/+28/-12而无独立确认仍false。原RG4：ROI±5外+闭合4h ADX<20+形成小时价格严格反向破上一闭合Low/High为正常关闭；结构等值/ADX>=20/ROI内区间不触发该分支，旧确认退出保留。原>=16趋势失败或动量+反向冲击、<=-12趋势/动量失败、>=28动量/反向冲击不变；ROI<=-20为唯一ROI-only灾难例外。全入口统一退出，不按hash或新字段区别，也不引入RG8强趋势loss分支。

## 冻结完整撮合与硬门槛

AF0/RG7/RG9×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT共12完整run，各初始1000；UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔。每币>=0.9次/周、真实成本净正、四年稳定、跨币泛化联合门槛不放宽。报告四完整Sep–Aug、额外2026-09、日历2023/24/25/2026 Jan–Sep，45月exit cohort只是归因非独立1000初始化。开发49月已观察非时间holdout；AAVE/ATOM/ETC/LINK收益未观察，资格待核，开发失败不读验证收益。

原backtest_engine_v7/standard_1m、已观察分钟close/下分钟Open、当前可用现金10%保证金/8倍、每侧fee0.0005/slip5bps、真实funding率/时间和原缺mark已观察分钟Close回退、外部profit/loss5/5、AutoStopOrder=false及-20例外均不变。普通外部5/5是调用gate而非强制平仓。RG8成本3708应用/961回退与12原始HTTP空mark样例不意味着精确成本资格完成，本轮逐笔再单列实际应用/回退，不补零/造mark/改原engine。

完整来源public-canonical-repaired-v2-funding-tail-v1、public-archive/public-archive及verified-archive20261003-v2，每次重算source/CRC/full data hash/prefix/tail身份。四币SHA依次c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457、7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371、64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818、8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。原replay/data helper SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8/a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63；资金费suffix54/overlap7 manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547。原指标种子/200输入窗口/默认warmup/容差不改。

## 预声明核验、保护与恢复

主输出results/20261004-rg9-development4-canonical-repaired-v2-funding-tail-v1.json。主后验证全部账目恒等、8个AF0/RG7同窗口对照逐笔/metrics/年度/source完全重现、年度/方向/入口族/集中度。全部补充入口再核canonical闭合价量、原9指标fresh receiver及已有日线排除；新增闭合RSI[1]与原CalculateRSI在真实receiver input−1（实际200input/199closed）canonical小时价格重算的[0]对应值、55/45门槛逐笔核对。使用原指标函数，不冒充独立数学；forming[0]不替代closed[1]。正常关闭复用原Position/gross mark-price-denominator ROI/外部gate/完整RG4结构核验，forced-end单列。全部逐笔现金复利qty/fills/fee/funding及缺mark回退再审计，不将静态删除交易当反事实收益。

禁止App/DB写入/策略分配/启用/下单/生产/前端/conf/app.conf改动或新仓库测试文件。所有Go/overlay及大输出在仓库外，虚拟诊断源实际不得创建。最新ARM元数据仍02:43:13.230/15.818/18.155，17/17/17模板、221/218/7结果、三库v29相同；不是446行新forward内容分析。正式技能1.0.8/trusted:false不变，receiver-seed/funding-mark候选新行为评估pending/aggregate null、未晋级。

全部中间完整JSON与失败保留。实际启动后保存真实handle，只poll同handle，不因观察超时重启。下次继续前先阶段总结，再核实际goal/真实进程；实际paused停止，不自行恢复/暂停/因阶段结束标complete或blocked。
