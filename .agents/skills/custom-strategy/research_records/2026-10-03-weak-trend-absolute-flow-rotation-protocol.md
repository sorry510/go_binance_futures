# RG1：弱趋势二次试探主动成交额轮换（收益读取前协议）

2026-10-03北京时间21:42之后恢复目标，实际get_goal=active；第一条先总结上一阶段，保护hash复核和pgrep无遗留研究进程。上一goal turn是progress：完成RG0会计/原成本/真实信号审计及两完整RG1 JSON与真实Expr检查；没有重复RG0主回测。21:07 paused为已被本次恢复取代的历史状态。

## 单一机制、完整候选与冻结身份

RG1于21:01左右保存、21:03:04.914冻结文件身份，6954项真实Go/Expr检查0失败后停止，没有读取RG1收益。现补齐主研究协议，仍在首次完整回测与收益之前；不可称协议早于候选生成/合成检查。

| 文件（temp_strategy/20261003-weak-trend-absolute-flow-rotation/） | SHA-256 | snapshot version |
| --- | --- | --- |
| 00-absolute-flow-rotation-family.json | e2bce2504f2c819623b3695bf9d9db1ae9d44e444e11d85bb32d3268c1c922aa | 0d0d84884cf9c2e754910c1a27199575908b97f734d50313a53a4ef77125c578 |
| 01-v29c-absolute-flow-rotation.json | eb662fe25e0b8c6aa60ff0f0b217dfe32d1b601ba0917c30d7ac2231a30f2773 | 16efd5f1559eb2c2ed3ad53cfe2ac356428a691c738b74481dcac0cc617addb9 |

族4/组合6规则，均包含两侧开仓与统一确认退出。组合顺序base LONG / supplement LONG / base SHORT / supplement SHORT。原9指标、三周期1h/4h/1d、基础两入口及两个完整关闭对象与AF0/RG0逐字段相同；不新增指标或系统变量，不修改有意的forming[0]，不读取forming taker[0]，不依赖OpenStrategyHash。

保留RG0结构：4h闭合ADX<20；倒数第二闭合1h首先突破之前8小时极值，随后闭合1h再次试探同一或更远极值、收盘及蜡烛实体向目标方向恢复；当前已观察价格收复第二根反向极值0.10ATR以上但不超过0.35ATR。

唯一替换的逻辑维度是闭合成交量关系：LONG要求`Buy1 > Buy2`且`Sell1 < Sell2`，SHORT要求`Sell1 > Sell2`且`Buy1 < Buy2`；Buy为真实闭合TakerBuyAmount，Sell=Amount−Buy。两小时总成交额须正，Buy/Sell须非负。删除原缩总量与占比改善条件，而非叠加新过滤器：第二根总量可以增加或减少，比例改善不再替代绝对成交额增加。保留数据有效性保护不是参数网格。

目标假设：二次试探后供需方向有实际主动成交额轮换，可能比仅缩总量/占比改善更可迁移。它不是深度、订单簿吸收、真实资金流入或盈利证明。ADX弱不保证稳定震荡区间，不要求总量缩小或目标占比跨0.5。

388旧portable、830同方向全程序去空白比较无精确重复；相关v88的净主动成交额绝对纪录突破采用单根方向纪录/前8根绝对纪录和live突破，不是二次极值中两侧绝对额相反轮换。不是完整语义等价去重或alpha新颖性证明。

## 表达式与统一关闭矩阵

仓库外`verification/rg1_expr_checks.go`使用实际Go structs/expr执行全部规则：6954项0失败，包含原指标/入口/退出完整身份、合法和不合法Buy/Sell与总量方向/等值、闭合弱ADX、价格形态/live缓冲/不追价、当前taker独立、禁用/局部有序匹配以及两侧ROI/hash矩阵。含静态断言和重复合成案例，不是6954个独立市场/盈利样本，不是引擎私有selector或API/live/forward/顺序cache验证。结果`results/20261003-rg1-expr-checks.json`。

| 两侧情况 | 关闭结果 |
| --- | --- |
| ROI=+5/-5，或+16/+28/-12，仅ROI到达、市场信号未失败 | false |
| ROI>=16且趋势失败，或动量失败且相反方向累计成交量/实体冲击确认 | true |
| ROI<=-12且趋势或动量失败 | true |
| ROI>=28且动量失败或相反方向冲击确认 | true |
| ROI<=-20 | true，唯一ROI-only灾难止损例外 |

LONG/SHORT均独立覆盖，空/错误/实际开仓hash不改变统一关闭。外部profit/loss5/5只提供调用资格，内区间不运行关闭表达式；原AutoStopOrder=false。因此外部5%不是必定在5%成交，也无ROI-only固定盈利退出。

## 主对照与全部硬门槛

- AF0、RG0、RG1三个组合 × BTC/ETH/SOL/XRP，各从1000起完整重新撮合49月，共12次。族portable本轮不单独回测，不能借组合账本宣称族独立利润。
- UTC2022-09-01T00:00（含）至2026-10-01T00:00（不含）：1491天/213周，每币最低192笔，保留四完整Sep–Aug年度、2026-09增量与日历2023/24/25/2026 Jan–Sep。45月退出cohort仅归因，不是独立初始资金收益。
- 当前可用现金10%保证金、8倍、双边各0.0005手续费和5bps滑点，真实历史资金费率/时间，原引擎缺mark时按已观察结算分钟Close回退；外部5与5。next-minute-open标准v7/1m。费用、资金费、普通/灾难关闭参数不随币/窗口偷偷改变。
- 每币>=0.9次/周、净正、四完整年稳定、跨币泛化必须同时通过；不得选择优势币/删除失败年度/静态删成交/降低真实成本或频率门槛。开发窗历经研究已观察，不能称untouched time holdout。本候选参数只是在本轮收益之前冻结。
- 冻结未观察验证币AAVE/ATOM/ETC/LINK，其完整历史资格待核对；开发失败不读取这些币收益，也不把曾观察扩展币当新holdout。目标尚无合格发布策略，不complete/blocked/自行paused。

## 来源、成本限制与保护

复用`/Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1`。source=public-archive/public-archive、minute-repair=verified-archive、repair_version=20261003-v2；canonical跨周期量差和独立价格纠错证据保留，不把量差强行修到minute-sum。现有suffix manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547；每币54真实suffix、7 ARM overlap，源前缀已经在上一PV5完整DeepEqual核对。

原replay复制SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8；隔离data helper SHAa340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63，仅研究数据真实后缀，不改变engine/生产/DB/config。完整cache反序列化重新计算DatasetDataHash并校验日期/周期/source/repair/funding-tail身份，失败停止解释。四币hash BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 /ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 /SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 /XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。

RG0已发现历史mark缺口：5058资金费应用中1299回退；单个官方旧BTC时间也返回空mark。实际费率覆盖不等于全部结算mark覆盖，原引擎算术相符不等于精确交易所成本。RG1沿用同一真实率及原价格回退以保证对照，必须独立报告其实际结算/回退次数，不补零费用/资金费、不制造精确mark、不默改runtime；完整成本资格仍待证据，不以这项局部局限为理由停止其他安全研究。

配置SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa；engine ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad /environment b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5 /indicator_cache 1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0，与恢复核对一致。保留原current taker缓存缺口但本策略不读其[0]；无生产修复授权。

## 审计、持久化与恢复

完成主回测后核对全部成交会计、8个AF0/RG0完整共享控制、年度/日历/方向/族/集中度、真实填单/仓位/费用/真实funding率与mark回退。所有RG1实际补充入口以entry_time−1复核canonical闭合OHLC/quote/taker、之前8小时极值及买卖绝对额变化、live已观察价/quote、原9指标fresh receiver与真实200输入/199闭合种子；fresh不代表全顺序cache/私有selector/live/forward/数学独立实现。

主输出`results/20261003-rg1-development4-canonical-repaired-v2-funding-tail-v1.json`；每run完成落盘，观察超时不是terminal，不另起重复进程。新ARM程序只读元数据/v29另存`20261003-rg1-arm-metadata.json`与`...-db-v29.json`，不是新444行forward分析。禁止App操作、DB写入/分配/交易启用、app.conf/生产/前端修改、新仓库测试文件。辅助程序、大数据和所有失败诊断保留在仓库外缓存，全部完整候选在temp_strategy。

quota或中断恢复前，先阶段总结再核对get_goal/实际handle/进程/冻结身份；active才继续，paused立即停止目标工作并保留阶段检查点。
