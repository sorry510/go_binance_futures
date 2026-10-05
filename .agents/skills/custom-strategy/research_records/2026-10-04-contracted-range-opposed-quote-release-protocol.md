# RG15：收缩后闭合价格突破与主动 quote 背离 — 收益前协议

## 当前冻结状态

- 2026-10-04北京时间13:06，在本协议保存时RG15主尚未启动或读取其历史收益。先保存两个完整JSON、actual Go/Expr矩阵、前端和完整入口身份扫描；收益前参数不随逐币结果变化。
- RG13/RG14均已完整失败，不能把亏损方向反转推定为盈利。这里交易方向不变：LONG仍价格向上突破、SHORT仍向下突破；只比较同向主动成交额支持与相反主动成交额/实际价格背离这一微观假设。所谓被动侧吸收是待检验解释，不是订单簿、钱包或因果证据。
- 原RG13的收缩[2:6]对[6:10]、closed[1]放量>prior8mean、合法buy区间、首次严格0.10ATR释放/previous[2]未释放、closed4h weak OR strong EMA/DI同向、closed日线反向排除、实体0.15..2ATR、实时保留−0.15..+0.35ATR全部原样。
- 唯一新入场维度为LONG buy[1]*2<quote[1]，SHORT buy[1]*2>quote[1]，两边严格半数等值拒绝；相同closed quote合法性与体价方向保持。forming Taker[0]及存储ratio不是此条件，价格/Data[0]用户有意保留。
- 原9指标/周期/AF0两基础整对象及顺序/原完整uniform RG4两关闭整对象不变，不叠加失败RG14退出确认、不增加indicator/variable/周期/ROI-only正常退出或entry-hash路由。保留所有long/short/close_long/close_short。

## 完整文件与实际合成检查

- 族：/Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-contracted-range-opposed-quote-release/00-contracted-range-opposed-quote-release-family.json，SHA2efa79d95b65673a5c388af8f4b6bcb353bf2f44bcec358cfc4bbe25448a72db，version7d27ad2b691ded8db941081c6050822584f87fc96af382cb2a298c32e8046f99。
- 组合：/Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-contracted-range-opposed-quote-release/01-v29c-contracted-range-opposed-quote-release.json，SHA8dba1d67bca704c9a093828e416b11d168aff8e2981b05dc05f255dedf506fc4，version12fd54f7b9f5bf3db8529edc9d811556c8be22e1f054d982df76235e9b6b1254。
- 初次checker2057 actualterminal exit1：4670/288，独立预期矩阵顺序字符串替换相互抵消，仍为旧同向多数；完整候选/字节SHA未变。首次错误结果SHAf5db7854f40e41dfc25cc08c45f9b8336c1513a07e70744a1948bc7c8c3ddcd7保留。
- 只另存精确两方向块原子修复checker_v2；5067 actualterminal exit0：4958passed/0failed，原strategy SHA不变。实际Go/Expr检查全程序替换边界、volume/quote合法/半数等值、weak/strong方向90矩阵、日线反向、收缩/新鲜释放/ATR1/2/实体/实时范围、forming/存储ratio/QPS/过去未用字段独立、全部旧close矩阵和基础整对象。
- 当前真实前端validator隔离VM两issue=null/9指标/四类型/shape通过；304限定portable/689enabled完整同侧entry去空白0精确重复（排除research目录/诊断名/>128KiB），不是全语义或alpha新颖性。
- 20261004-rg15-expr-checks-v2.json SHAfc0af20badc4743c095e01978d14e2f7e3ae43150748d5da7715408b27539a6f。
- 20261004-rg15-frontend-contract-checks.json SHAb1cd9d0de2ce7b1110d45cbbc6c3524fc2fd6f3b80308e3c6ee2739a3e5fd880。
- 20261004-rg15-portable-entry-identity-scan.json SHAbc3679f7f1b2a4df89bd990b8aaf4a4fdff8049c26940c57c0b9b60f01610689。

## 完整研究与硬门槛

- 完整组合AF0/RG13/RG15×BTCUSDT、ETHUSDT、SOLUSDT、XRPUSDT，共12run；独立族没有收益研究，不拿组合收益代替族证明。
- UTC2022-09-01inclusive到2026-10-01exclusive，1491日/213周，每币≥0.9次/周即至少192笔；四Sep–Aug完整年、额外Sep2026、日历2023/24/25完整年及Jan–Sep2026全部报告，不改年度定义。
- 原backtest_engine_v7/standard_1m/public-archive execution与indicator/verified-archive minute repair、canonical-repaired-v2-funding-tail-v1缓存和已验证54fundingtail/7overlap/manifest78758a90保持；原实际200input/199closed，不假定150数学等价。
- 原现金10%保证金/8倍/双边fee0.0005/各侧不利5bps/真实资金费与原缺mark分钟价回退/外部5与5门控/AutoStop=false完全保持。结算mark精确计价仍pending，不降低成本、伪造mark或只称费用算术正确即真实成本通过。
- 四开发币净/频率/四年稳定同时通过才考虑未阅验证资格，AAVE/ATOM/ETC/LINK收益未读取。未观察验证泛化仍pending，任何门槛失败完整JSON和错误全部保留，不选币/侧/删组/阈值网格。

## 主后实际核验

- 全12会计与8AF0/RG13共享完整控制对上一RG13study逐笔/metrics/年度/source复现，仅cache_hit是获得观察，候选path对固定workspace resolve+realpath身份，不忽略其它字段。
- 新opening helper32143 actualterminal build exit0，仍取original RG13InspectSignal参数化原receiver，但补充prefix正确为rg15_、独立canonical的LONG/SHORT主动quote方向已经反向，输出FlowDiverges；不拿旧同向FlowConfirms报告当新证明。完整闭合80字段/实际种子/4h/日线/ATR/收缩/新鲜/quote/实体/实时/fullExpr全核，0条不会伪称通过。
- closing用参数化原rg13_close_signal_audit binary及本轮真实input/SHA/version：完整uniform RG4未变，所有normalclose原Position/ROI/现金/outergate/fullExpr/独立weak结构，forcedend单独删失。无需也不得用RG14确认close checker。
- 全12actual fills/当前现金复利/qty/gross/双fee/funding包含/下一分钟活动及缺mark回退分别全量和自身统计，zero活动和算术必须0failed；exactvenue结算mark仍需证据。
- 主输出results/20261004-rg15-development4-canonical-repaired-v2-funding-tail-v1.json是逐run恢复checkpoint。实际terminal才说结束，不因为timeout/文件存在重起；主后report全部核完再裁决。

## 授权和恢复

- 仅研究候选/协议/隔离cache helper，无App操作、DB写/策略分配/启用/下单/production/前端/conf.app配置修改或仓库测试文件。其它dirty研究、正式skill外部metrics行和两pending草案不动，不晋级无strict-win技能。
- 以actualget_goal为准，active继续安全研究，paused停owned进程且保留已完成run，不自行恢复/complete/paused/blocked。上一turn为实际progress，本续接已先阶段总结，下次也先总结真实完成组数/门槛/失败/文件/livehandle和下一步。
