# RG14：闭合实体与主动 quote 确认弱结构退出 — 收益前协议

## 当前状态及单维假设

- 2026-10-04 北京时间 10:28，RG14 尚未启动历史收益撮合或读取其收益。先保存两个完整 JSON、10330 项实际 Go/Expr 检查、当前前端 validator VM、限定关闭身份扫描和本协议，再启动完整主。
- RG13 完整 12 run / 3953 笔已实际结束并 invalidated；所有入场维持不动。RG13 原自然强弱组并无一致盈利，不从亏损推反向 alpha、删侧或挑币。
- 新全配对 canonical / original 关闭诊断 79523 actual terminal exit0：2623 正常、0 forced、0 失败，原 added-only 1160 中未同时确认反向 closed body/合法 opposite active quote majority 为 BTC 117/248、ETH 159/318、SOL 142/288、XRP 178/306，合计 596/1160。诊断只取原退出 signal time 已观察 closed[1]，四个 original/canonical 值全部匹配，不使用后续小时或假想延迟收益。
- 唯一候选机制：原弱结构关闭仍必须 closed 4h ADX[1]<20、ROI>=5 或 <=-5、实时 Close[0]严格越过上一 closed Low/High；额外要求 closed[1] 实体反向且 closed quote>0 / buy legal [0,quote] / buy*2 严格反向多数。没有新增阈值网格、指标、周期、开仓过滤或 forming Taker[0]。
- 两个正常关闭程序统一用于所有基础和补充来源，没有 hash/entry-family 路由。AF0 已确认利润/亏损反转、扩展利润及原 ROI<=-20 灾难例外逐字保留；未增加普通 ROI-only。保留用户有意的 Data/价格[0]。候选允许 normal closing change，不擅改外部5与5门控或8倍。

## 完整策略身份及本地验证

- 族：/Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-closed-body-quote-confirmed-weak-exit/00-closed-body-quote-confirmed-weak-exit-family.json；SHA 38a922b73d8569daf6b88c702256f7e277f50a11c188af84045f0a373e34eb66；version 9d12e9d6b2b2fadca4b459346e60a688f9cf8edd265db08f4e90a77dc3d479d7。
- 组合：/Users/zhz/work/binance/go_binance_futures/temp_strategy/20261004-closed-body-quote-confirmed-weak-exit/01-v29c-closed-body-quote-confirmed-weak-exit.json；SHA b91741c9f1338f06cc9077c8eda2a32b20f305e8143d9cdc0698a7f2fbfa36bf；version d126788c8f43777fa25d5b1d75d37a6b50ceca00bbaf15a5fa09f8375817c13d。
- 两 JSON 都含完整 long/short/close_long/close_short，原 9 指标配置完全相同。组合所有 4 enabled entries 整对象与顺序和 RG13 相同，族 2 supplement entries 相同，保留 rg13_ 名称体现身份，不宣称新增入场 alpha。
- Expr 11213 actual exit0：10330 passed / 0 failed。复用未变入场矩阵并重新实际执行；关闭新矩阵覆盖 ROI/ADX/实时严格高低点等值、closed body 阴/阳/doji、正/零/非法 quote、半数等值及合法区间、forming taker/存储 ratio 无关、全部旧确认与灾难例外、无 entry-hash 路由。不是 cached sequential/private selector/API/live/forward、独立指标数学或盈利证明。
- 当前前端 utils/technology.ts 实际 installed TypeScript→isolated VM validator，两 JSON issue=null、9 enabled、四类型和 rule shape 通过；未操作 App、build/install 或修改前端。
- 限定302 portable files / 623 enabled closes，完整程序去空白比较0精确重复，排除 research 目录/诊断文件/>128KiB。是新关闭身份扫描，不是全语义/alpha 新颖性；本轮入场特意完全相同，绝不冒称新入口0重复。
- 20261004-rg14-expr-checks.json：SHA b1d0ab478b911a145eb0595bfb1d46649997aec8237c94a0d82b665312d8234b。
- 20261004-rg14-frontend-contract-checks.json：SHA d47db3cb72d78f882980ee71c6410117fa7d54f50705b0cebf6dcdf81c797ceb。
- 20261004-rg14-portable-close-identity-scan.json：SHA 625a8471e41e7cb81148d0bcd3691e19ce78f26561a750c2f9a27b288e822ed4。
- 20261004-rg13-close-hourly-body-active-quote-confirmation-diagnostic.json：SHA 4323e0431cf30e985e8a2e6a497cbbf19b389fe7d5326b39443efe5c7344ed94。

## 固定撮合和全部裁决门槛

- 只主撮合完整组合：AF0 / RG13 / RG14 × BTCUSDT、ETHUSDT、SOLUSDT、XRPUSDT，共12完整49月run。族不单独测收益，不将组合收益转为族证明。
- UTC2022-09-01inclusive 至2026-10-01exclusive，共1491日/213周；每币≥0.9次/周即至少192笔。四完整Sep–Aug年、额外Sep2026、完整日历2023/24/25与Jan–Sep2026均报告；不挑有利年度定义。
- backtest_engine_v7 / standard_1m；public-archive execution+indicator、verified-archive minute repair；固定 canonical-repaired-v2-funding-tail-v1 cache / 原manifest SHA 78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547。原实际200input/199closed种子，不假定150与200精确数学等价。
- 风险/成本固定：8倍、当前现金10%保证金、双边fee 0.0005、不利5bps各侧、真实funding rates/原缺mark观测分钟回退、外部profit/loss5/5、AutoStop=false。精确 settlement mark 仍 pending，不能凭算术一致宣布真实venue成本已通过，更不能减成本。
- 四开发币freq/net/四年稳定全部同时通过以后才考虑验证资格，不读取AAVE/ATOM/ETC/LINK收益做筛选；未观察验证币跨币泛化仍pending。任何门槛失败都保留失败完整JSON，不删组/删侧/选币/改年度/降低频率或成本。

## 主后必须实际验证

- 主实际 terminal 后，全12逐笔会计/价格数量/费用/净/年度/side/families；8共享AF0/RG13控制要与上一完整RG13 study全账目/metrics/source一致，cache_hit是唯一源获得观察差异。candidate path只resolve固定workspace+realpath身份，不忽略其他字段；所有错误保留。
- 所有新组合实际 rg13_ supplement 开仓仍核 original/canonical 8×10闭合字段、original200种子/4h方向/日线/ATR1/2/收缩/新鲜释放/quote合法多数；不能因入场code相同就冒称新的receiver核验完成。
- 所有新组合 normal close 取实际 exit_time−1、original Position/environment/ROI/cash/外部门控、完整新 Expr 和 AF0 full old Expr；独立canonical闭合body/quote、实时结构和closedADX再算新 added branch，forcedend单独删失。旧RG13 closing helper不能不改新增confirmation就冒充RG14通过。
- 全12实际现金复利/下一分钟实际活动/fill价格/双fee/funding mark应用与包含规则；缺mark回退分别全量和自身计数，不声称精确交易所成本。
- 结果保留 results/20261004-rg14-*.json；主逐run输出就是恢复checkpoint，不因timeout另起、不将存在文件当terminal。当前尚无新策略收益或发布。

## 安全边界

- 仅本地研究JSON/协议/isolated verification helpers；没有DB写/策略分配/启用/下单/生产代码或指标/前端/conf/app.conf修改，没有App/UI操作或仓库新增测试。
- 遵守实际get_goal；若active允许安全研究，若paused立即停止owned工作且保留完成run，不自行恢复。阶段完成不自行complete/paused/blocked。
- 用户要求下次恢复前先阶段总结：真实完成组数/全硬门槛/失败原因/文件/真实活进程/下一步。正式skill外部metrics行和其它dirty修改全部保留，两个pending SkillMax草案无行为strict-win不晋级。
