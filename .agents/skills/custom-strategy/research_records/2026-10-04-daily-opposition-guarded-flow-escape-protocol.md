# RG7：日线反向强度排除（首次收益前冻结）

2026-10-04北京时间02:17:53，3446 Go/Expr检查0失败后，RG7任何完整收益之前冻结。目标active。RG6主21928/开80315/成本43265/关87284均actual terminal exit0：12run/2635账目及全部审计已完成但年度/跨币/精确成本失败；不重复原主。

## 完整候选、唯一变化、假设

temp_strategy/20261004-daily-opposition-guarded-flow-escape/：
- 族00-daily-opposition-guarded-flow-escape-family.json SHA7503b4692dc81bc7f36ace594e48f677e413743db05a5271aaa732d148da19f1 /versiona9b995d8760765fe97278be03a3fbdd63879e688594ce8eb9b756aeab032e72d。
- 组合01-v29c-daily-opposition-guarded-flow-escape.json SHA5988a0332dbd3583b538eee6ebc3fe1298483836a38d3b3a451a34857864c9d9 /version4e2a42e65aa8a5a10e84f26a9b845e22a7de9cb3e3b923d12a6eb23e598d146e。

族4/组合6启用规则。原9技术配置/1h4h1d、AF0基础入口、RG6所有原条件与完整RG4关闭对象相同。只增加日线反向环境排除：闭合日线ADX[1]>=20且LONG的MinusDI[1]>PlusDI[1]，或SHORT的PlusDI[1]>MinusDI[1]时禁止该补充入口。ADX<20、DI相等或同向不拦截。20沿用原v29日线短侧强度门槛，不来自搜索收益网格。强4h区域仍由原AF0基础入口处理，弱4h区域仍用已声明闭合8小时价量/首次突破/放量/4h方向/body/live价格保持，新增日线冲突限定；不是所有市场都必须开单。

假设来自RG6弱方向净结果跨币不一致、价格确认未解决稳定性：当前补充逻辑没有直接约束强日线冲突，可能把大周期反向里的局部运动当持续趋势。本轮检验该维度，不声称已证此因果或通过静态删账目得到收益。没有新指标/周期/系统变量、删空/币别选择/风险成本调整；有意保留forming价格/活动[0]，不读forming taker0/hash路由。

377旧portable/1638同侧完整程序比较无精确去空白重复，不等于日线过滤概念全新、语义去重或alpha优势。未观察币及旧相关程序收益未读。

## 矩阵与关闭

91483 actual terminal exit0，verification/rg7_expr_checks.go/results/20261004-rg7-expr-checks.json：3446/0。新增日线ADX{0,19.999999,20,25,50}×DI多/等/空×两侧，故意设置forming日线相反强势来验证仅闭合[1]；保留原quote合法性/加权多数/价格进展/基础可达/入口边界、唯一变化整对象断言以及全部关闭矩阵。合成/静态/重复检查，不是3446市场样本或cached/private/API/live/forward/盈利验证。

| 双侧条件 | 平仓 |
| --- | --- |
| 普通ROI+5/-5/+16/+28/-12，无独立市场确认 | false |
| ±5外+闭合4h ADX<20+forming小时价格反向破上根Low/High | true，原RG4正常分支 |
| 新结构等值、ADX>=20或ROI内区间 | 新分支false，原确认逻辑保留 |
| 原>=16趋势失败或联合动量/反向冲击、<=-12趋势/动量失败、>=28动量/反向冲击 | true |
| ROI<=-20 | true，唯一ROI-only灾难例外 |

外部profit/loss5/5为调用资格，非±5必平；AutoStopOrder=false，内区间不调用close。退出全模板统一，当前弱分类可不同于入场状态，不依赖hash/新字段；未做API/live/forward验证。

## 冻结撮合与资格

AF0/RG6/RG7×BTC/ETH/SOL/XRP，共12完整run，各初始1000。UTC2022-09-01含至2026-10-01不含，1491天/213周/每币至少192笔。每币>=0.9次/周、净正、四年稳定、跨币泛化全部联合门槛不放宽；报告四完整Sep–Aug年、额外2026-09、日历2023/24/25/2026 Jan–Sep；45月exit cohort仅归因非独立初始化。开发49月已观察非时间holdout；AAVE/ATOM/ETC/LINK收益未观察，历史资格待核，开发失败不读验证收益。

原backtest_engine_v7/standard_1m，已观察分钟close/下分钟Open；当前可用现金10%保证金/8倍、双边各fee0.0005/slip5bps、真实资金费率时间及原缺mark已观察结算分钟Close回退、外部5/5和-20例外全不变。完整真实率不等于精确mark覆盖，RG6全4578应用/1187回退，RG7单列自身应用/回退；不补零/造mark/修改engine/虚称真实成本资格完成。分钟活动非订单簿容量证明。

来源public-canonical-repaired-v2-funding-tail-v1，全public-archive/public-archive及verified-archive20261003-v2。四币data SHA、原replay/data helper、官方suffix54/overlap7/manifest SHA与RG5/6首次协议相同，cache每次完整重算身份。新daily字段原9配置已存在，不改递推种子/默认warmup/容差。

## 预声明审计、保护和恢复

主输出results/20261004-rg7-development4-canonical-repaired-v2-funding-tail-v1.json。主后完整复核8个AF0/RG6控制逐笔/metrics/年度/source及全账目/年度/方向/族/集中度、现金复利qty/fills/fees/真实funding。全部补充入口按canonical原闭合价量和原9配置fresh receiver核对，新增闭合日线ADX/PlusDI/MinusDI用原函数在真实receiver input−1 canonical闭合日线seed重算，观测daily forming不替代closed1。200输入/199闭合由实际接收器读出，不称独立数学/全顺序cache/private/API/live/forward。正常close复用原Position/gross mark-price-denominator ROI/外部gate/原完整RG4结构审计，forced-end单列；分支归因非去分支反事实PnL。

只读ARM截止仍01:39:58.217/01:40:03.394/01:40:08.651，17/17/17模板、221/218/7结果、三库v29相同；非446行完整新forward分析。禁止App/DB写入/分配/启用、生产/前端/config/仓库测试文件。正式技能1.0.8/trusted:false保护SHA不变，receiver-seed/funding-mark候选行为pending/aggregate null未晋级。完整候选/失败留temp_strategy，大输出/Go/诊断overlay在仓库外缓存，virtual诊断源实际不得创建。实际启动后保存handle；先poll同handle，不因观察超时重启；下次先阶段总结核goal/真实进程，paused停止且不自行恢复，不因阶段结束/困难标complete/blocked。
