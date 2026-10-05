# RG16：收缩区间反向扫边后闭合收回 — 收益前协议

## 冻结声明

2026-10-04北京时间13:43本协议建立时，RG16主回测尚未启动，也未读取其收益。RG13/RG14/RG15已完成并invalidated，所有旧主和核验句柄均已终止，不重复启动。当前实际goal active，本续接先给阶段总结。

本轮只改变RG15补充入口的价格响应几何：同向越界突破改为反向扫边再收回，首次信号同样按自身几何判断，位移改为从反向极值收回而不是强制同向实体。之前RG10的持续反向8小时压力极值拒绝完整配置已读；本轮使用四小时对前四小时收缩、首次自身扫边/收回以及weak OR aligned strong，而非RG10持续压力门槛。不能据配置不同断言alpha新颖性。

## 明确的两侧条件

- LONG：Low[1] < min(Low[2:6]) − 0.10ATR[1]，且Close[1] > 同一边界 + 0.10ATR[1]。SHORT：High[1] > max(High[2:6]) + 0.10ATR[1]，且Close[1] < 同一边界 − 0.10ATR[1]；相等不通过。
- 前一闭合[2]按[3:7]与ATR[2]定义同侧扫边并收回，两条件都成立则拒绝本次，避免连续相同信号。
- LONG buyQuote[1] < quote[1]/2，SHORT > quote[1]/2；quote[1]超过前8小时非负成交額均值，buy在[0,quote]合法区间。多空玩具例LONG low99.8/close100.3/open100.55/边界100/ATR1/quote150/buy60可达；SHORT为200价格镜像、high100.2/close99.7/open99.45/边界100/quote150/buy90。LONG可阴线、SHORT可阳线；这不证明被动吸收或交易者身份。
- recovery为LONG Close[1]−Low[1]、SHORT High[1]−Close[1]，保留0.15..2ATR界限。下限0.15已由严格0.10扫边+0.10收回蕴含，并非额外有效过滤，不调阈值网格。
- recent[2:6]宽度严格正且小于older[6:10]正宽；4h闭合weak ADX<20 OR相同方向EMA20/50与DI；闭合日线强反向排除；实时价−0.15..+0.35ATR保留且当前quote正。这些均保持RG15。
- 原9配置/1h4h1d/AF0基础整对象和顺序/完整uniform RG4关闭整对象不变。不引入MarketCondition、新指标/周期/entry-hash路由。保留用户有意价格Data[0]，不使用forming taker[0]或存储ratio作为入口。

## 已完成的收益前验证与身份

- 两完整portable均含long/short/close_long/close_short，独立族仅4规则、组合6规则。
- 族temp_strategy/20261004-contracted-range-opposed-quote-reclaim/00-contracted-range-opposed-quote-reclaim-family.json：SHA a647f88bb45736e45c869791fa66f29a7b0da48fd98db62730f345d4d916b547；version 9d2a2ffbe3955a29001e71bd1154c06493735d2376691d52e4f81751bfbe4d26。
- 组合01-v29c-contracted-range-opposed-quote-reclaim.json：SHA 2e574fb5580d80d3efdce2a2416cc9212c7972ad2c91fdfa7b0cbd25bbe12312；version 8478b5940bed703f70698cfd74ef39316770dfc623c341855c4645fed02572ce。
- 自测34460 actualterminal exit0，4158passed/0failed。明确独立数值oracle不解析/变换Expr；多空quote合法/严格半数/price几何/首次抑制/收缩/ADX方向/日线排除/recovery实时边界、原关闭完整矩阵/正常ROI-only拒绝/灾难−20例外/原基础可达/整对象身份核验。合成与本地规则顺序模型，不称private selector/历史sequential/live/API/forward/盈利通过。
- 自测结果20261004-rg16-expr-checks.json SHA 440255faf23e48e5e47eb720458cdb2466eb32efd8123c00614093443004a1ba。
- 前端实际TypeScript validator隔离VM两issue=null/9enabled指标/四类型/shape有效，SHA e4aa232d3a9ff5624a5d2d3b33a7f8718701501efdc92d8b356de71dfa23f3e8；无App/UI/production build/API。限定305旧portable/691enabled完整同侧入口0精确重复，身份扫描SHA f0cc031ed41e46980f12e0f2562a59f319235e0d9b9a86fe76b984c930fb21c4；排除自身、research/diagnostic/audit/>128KiB，不称全语义或alpha新颖。
- 新opening审计75557 actualterminal build exit0，源码SHA275af28123e918e88d19984e609c14057952a76c6a9ed09c607ddbec203459ce、binary SHA2d8359a6297f927a3b64c423d4567ba3cbf00019c49b1abc59ddeb4ec3cccb11。原诊断桥接仅导出新鲜环境；首次桥接文件名误读已纠正，无生产源码变化。它按rg16_和真实版本匹配，独立canonical扫边/收回/前次/recovery与所有原字段重算，不沿用旧突破证明。

## 完整撮合与不变门槛

- AF0/RG15/RG16组合 × BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT，同UTC2022-09-01inclusive..2026-10-01exclusive，12完整49月run，1491天/213周；每币≥0.9/周即192笔。完整四Sep–Aug年、额外Sep2026与日历2023/24/25、Jan–Sep2026均报告。
- 原backtest_engine_v7/standard_1m、observed minute close→next minute open、现金10%保证金×8、fee双边0.0005、slippage各侧不利5bps、真实funding及原缺mark分钟Close回退，外部5/5门控、AutoStop=false完全不变。确认退出和ROI−20灾难例外不变。
- public-canonical-repaired-v2-funding-tail-v1四币data SHA保持c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457/7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371/64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818/8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。54实际funding后缀/7overlap/manifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547保持；实际原200input/199closed种子不替换为150。
- 所有收益前参数冻结。四开发币净额/频率/完整年稳定同时通过后才考虑未阅验证资格；AAVE/ATOM/ETC/LINK收益未读取。精确交易所结算mark与真实订单簿成本仍pending，原回退算术正确不等于完整真实成本过关。无删组/挑币/侧/年界/减少费用或倒推inverse edge。
- 主输出results/20261004-rg16-development4-canonical-repaired-v2-funding-tail-v1.json是逐run恢复checkpoint；必须观察实际terminal，不能因timeout或文件存在重起已完成运行。

## 主后检查

全部12条会计/8 AF0-RG15完整共享控制对上一RG15study核对，路径只按固定workspace/realpath和SHA证明等价，其余snapshot严格。原fills/当前cash数量/费用/真实funding包含和分钟价fallback/活动全量与自身分别核验；新增开仓全字段/实际种子/4h+daily+ATR+范围+几何+fullExpr，所有normal退出原Position/ROI/whole uniform RG4与outer gate审计，forced end单独标识。0失败后才完整裁决，账本静态剔除不当反事实。

北京时间13:45后、尚未观察任何RG16收益时，进一步预先指定仅以入口自身已有ADX20 OR边界分weak/strong，全部补充逐笔配对实际opening/closing，描述侧/年/退出归因；不尝试其它阈值、剔除组或作为新撮合收益。候选、完整发布门槛与主回测均不改变。

## 权限与技能

没有App操作、DB写/分配/启用/下单/production/前端/config修改、材料删除或仓库测试文件。conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa、正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd未变。SkillMax新增评审副本结构有效/行为7例pending，未晋级；保留两旧草案与用户metrics CSV规则。阶段不等于完整目标完成，下次先阶段总结、再核actualgoal/真实进程后继续。
