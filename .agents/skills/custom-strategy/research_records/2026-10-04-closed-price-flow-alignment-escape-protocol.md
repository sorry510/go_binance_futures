# RG6：闭合价格与累计主动成交同向（首次收益前冻结）

2026-10-04北京时间02:01:44，3266 Go/Expr检查0失败之后、任何RG6完整收益之前冻结本协议。goal active。RG5主/三项审计均实际terminal exit0：12完整run/3144执行账目，所有四币频率达标但SOL/XRP净亏、年度稳定不满足；报告2026-10-04-persistent-closed-aggressor-escape-summary.md。不重启RG5原主。

## 完整候选和单维度假设

temp_strategy/20261004-closed-price-flow-alignment-escape/：
- 族00-closed-price-flow-alignment-escape-family.json SHAebfe6b5d7383629bf3ccdb45352e26c2da00abcb890dea1e9b49fa6e78604132 /version3b93536792671e214a1146ed4174001fe7b31f779855e194583c23156ce4f404。
- 组合01-v29c-closed-price-flow-alignment-escape.json SHA36bb384e90ddb0a6ae617b668a4a72d55eb98a37110c4e995a08442d13a3d01b /version22eced20b9d1cb331ee8fa52c92cb0884fa8d7ae9df5c8ee168ec863c6f029a8。

族4/组合6启用规则，包含双向开平仓。完整9个技术配置、AF0基础入口、完整RG4关闭对象与RG5相同。只给RG5补充开仓加此前闭合8小时[2..9]价格进展：LONG要求Close[2]>Open[9]，SHORT要求Close[2]<Open[9]，等值拒绝。窗口严格对齐此前sum(quote[2:10])/sum(buyQuote[2:10])，不是把累计单笔主动方向、相干性与价格进展误称独立多份证据。

RG5成交参与过滤改善四币总净但跨币/年度仍失败；成交方向持续而价格未推进可能是对手盘吸收，这是待检验机制，不是已证明的损失原因。RG6只检验新增价格确认，保留原[2:10]每小时有效性/加权净方向、本次闭合突破小时同向主动额、闭合4h EMA/弱ADX、首次区间放量脱离、body与live不追价。无新指标/周期/变量，forming价格[0]故意保留，不读forming taker0/hash。不做币别参数/删空/阈值网格或静态筛账目推算收益。

375旧portable程序/1626完整同侧程序比较无精确去空白重复（族/组合分别比较计数），不是语义/alpha新颖或盈利证明；相关v88/coherence程序已在RG5前读取，未读其收益。AAVE/ATOM/ETC/LINK未观察收益不读。

## 已验证及关闭矩阵

20207实际terminal exit0，verification/rg6_expr_checks.go及results/20261004-rg6-expr-checks.json，3266/0。覆盖唯一新增价格确认整对象身份、LONG/SHORT Open[9]-Close[2]负/零/正严格边界、RG5累计quote多数/加权/合法性/排除forming字段、全部原入口/基础可达和原关闭矩阵。合成/静态/重复案例，非3266真实市场样本或cached/private/API/live/forward/盈利证明。

| 双侧条件 | 关闭 |
| --- | --- |
| 只有普通ROI+5/-5/+16/+28/-12，没有市场确认 | false |
| ±5外且闭合4h ADX<20且当前小时价格向相反方向突破上根Low/High | true，RG4正常分支完整保留 |
| 等值、ADX>=20或ROI内区间 | 新分支false，旧确认分支仍按原逻辑 |
| 原>=16趋势或联合动量/反向冲击、<=-12趋势/动量、>=28动量/反向冲击 | true |
| ROI<=-20 | true，唯一ROI-only灾难例外 |

外部profit/loss5/5只是调用资格，不是±5必平；AutoStopOrder=false，内区间不运行关闭。全部退出统一，不依赖开仓hash或entry时的ADX状态。API/live/forward验证尚未执行。

## 冻结风险、来源、完整硬门槛和审计

AF0/RG5/RG6×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT，共12完整run。UTC2022-09-01含至2026-10-01不含，1491天/213周，每币>=0.9次/周即至少192笔、净正、四年稳定、跨币泛化联合门槛全保留。四完整Sep–Aug年、2026-09、日历2023/24/25/2026 Jan–Sep均报告；45月exit cohort仅归因非独立初始化。开发49月已观察不是时间holdout，验证币资格仍待核，开发失败不看验证收益。

原engine_v7/standard_1m，观察分钟close/下一分钟Open；各初始1000，当前可用现金10%保证金/8倍，双边各fee0.0005/slip5bps，真实funding时间/率和原缺mark观察结算分钟Close回退，外部5/5/-20例外都不变。资金费率时间完整不等于精确结算mark；RG5自身1783结算/475回退，真实成本资格未补齐，RG6另报实际回退；不伪造mark、补零或改engine。分钟有量不是订单簿容量保证。

复用public-canonical-repaired-v2-funding-tail-v1/public-archive两来源/verified-archive20261003-v2，四币原hash及replay/data helper/funding suffix manifest身份与RG5协议一致，cache hit仍完整重算身份。全部生产/config/正式SKILL保护SHA保持，不修改来源/闭合指标种子或精度容差。

主输出results/20261004-rg6-development4-canonical-repaired-v2-funding-tail-v1.json。预声明主完成后核8个AF0/RG5旧控制完整逐笔/metrics/年度/source、全账目/年度/方向/族/集中度和逐笔cash复利quantity/fills/fees/funding；全部RG6补充入口按canonical闭合价格Open[9]/Close[2]与原[2:10]累计quote/方向及原量价/EMA/ADX/ATR核对，实际200input/199closed原函数另算不称独立数学。正常退出复用原Position/gross mark-price-denominator ROI/外部gate与RG4结构分支审计，forced-end单列；不把fresh称全顺序缓存/private/API/live/forward或关闭分支反事实收益。

最新三库只读元数据截止01:39:58.217/01:40:03.394/01:40:08.651，17/17/17模板、221/218/7结果，三库v29语义相同；不是446行全内容前向新分析。无App/DB写入/分配/启用、生产/前端/config/新仓库测试文件。正式技能1.0.8/trusted:false，新技能行为pending/未晋级。候选失败全部留temp_strategy、大输出/Go/overlay在缓存外仓库；实际virtual overlay源不得创建。下次先阶段总结核goal/实际进程，不重复已终止主，不因观察超时重启；paused停止研究，不自行恢复或标complete/blocked。
