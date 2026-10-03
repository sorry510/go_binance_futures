# PV3：闭合主动成交改善的冻结对照

## 冻结假设（收益观察之前）

2026-10-03北京时间14:30–14:32生成完整JSON并冻结本协议。上一轮PV2读取形成小时taker[0]的收益因缓存缺漏已隔离，生产修复仍待用户授权。本轮不修复/替换引擎，不把PV2偷换成闭合版本或称其已修复，而是继续独立验证PV1的闭合价量假设。NowPrice、Open[0]、Amount[0]及原实时退出仍保留[0]语义。

PV3只在PV1缩量回调恢复入口追加一个方向条件：LONG `TakerBuyRatio[1] > TakerBuyRatio[2]`，SHORT `<`。前两个完整小时同期限Qps>0已有quote>0，不另加重复guard。比例是主动买入报价金额/总报价金额，不是持仓多空比、资金流净流入、真实被动吸收的证明。总报价量衰减与比例改善代数独立，验证的是回调期间对侧主动成交占比减少是否能减少PV1坏入口；不新增绝对比例阈值、回调深度、时间窗口或指标。

类似原子已有历史v80的三小时price/taker divergence；PV3不是全新微观结构优势。v80要求Close[1]/[2]/[3]及ratio[1]/[2]/[3]连续背离，无PV1完整的两逆势实体/缩量/4h方向与日线拒绝/ATR no-chase。独立历史扫描需要排除本轮新文件，报告完整条件的重复性，而非忽略旧试验。

## 不变项与候选

- 基础AF0两个入口、全部technology及两平仓对象完整保留；PV1→PV3只有顶层名、pv1→pv3入口名和上述两个追加条件变更。
- 独立族4类规则全部启用；组合入口顺序base LONG/supplement LONG/base SHORT/supplement SHORT，两个uniform AF0 closes。helper明确`-exit-source base`，无entry-bound或conditional guard，不依赖OpenStrategyHash。
- 既有原参数：4h EMA20/50方向与斜率/ADX20及同向DI，两根逆势1h与继续回调；日线ADX25强反向拒绝；ATR/price0.0015..0.035；NowPrice越过前小时极值且最多追0.35ATR；非零观察量；资金费率long≤0.0003/short≥-0.0003；Qps[1]<Qps[2]。
- 平仓沿用AF0：普通ROI≥16且趋势失败或动量失败+反向冲量、ROI≤-12且趋势/动量失败、ROI≥28且动量/反向冲量；仅ROI≤-20为灾难例外。外部5/5只是资格，不自动退出。无配置/数据库/启用/风险改动。

| 新完整JSON | SHA-256 |
| --- | --- |
| temp_strategy/20261003-pullback-closed-flow/00-closed-flow-improvement-family.json | 4a3d299c8da28be1c506bdaf169f26586696beed1cf49edc4bf321eb27c190a6 |
| temp_strategy/20261003-pullback-closed-flow/01-v29c-closed-flow-pullback.json | 9402fc7704b626b05e4d44c8d9920df6636e22546f644615e86c6186d26ebf10 |

## 回测与判定

同条件完整重撮合AF0/PV1/PV3×BTC/ETH/SOL/XRP，共12次四年运行；UTC2022-09-01..2026-08-31，1461日/208.714周，至少188笔/币才满足每币≥0.9次/周。初始1000USDT/币、当前现金10%保证金复利、8倍、双边0.0005费率及5bps滑点、历史资金费、外部5/5。实际standard engine v7/standard_1m、immutable canonical repaired v2、四币data hash与上一轮相同。缓存/结果都在/Users/zhz/Library/Caches/go-binance-strategy-research，不覆盖旧PV2失败证据。

联合门槛仍逐币频率达标/净收益正/四个Sep–Aug完整年度稳定/跨币泛化，另报告PF/最大回撤/方向与族集中度/去最佳交易敏感性。禁止改全日历分母、选盈利币、调杠杆或删旧交易推断反事实。开发失败则不查询冻结未观察AAVE/ATOM/ETC/LINK收益，完整日期资格仍待核对。所有四币窗口已经开发观察，不能当未见留出。

运行解释前验证闭合[1]/[2]实际接收者：用仓库外临时只读Go overlay桥接真实BuildMinuteClose/cachedKlinePriceSeries与uncached引用比较连续分钟/跨小时/零量/数组排序；它只用于诊断，无盈利运行引擎替换。已知当前taker[0]缓存差异必须保留，不修复。合成Go/Expr检查与局部有序模型不能冒称真实接收或端到端。完整回测后独立按signal=entry_time-1重建其闭合canonical小时比例并检查所有补充入口，canonical闭合量不强制等于分钟聚合，保留既有跨间隔source差异说明。

## 授权与保护

禁止App操作；只用app.conf注释arm程序只读查询白名单三个库，未修改配置、生产、前端或数据库，没有新增仓库测试文件。未修复的indicator_cache.go SHA1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0；environment.go SHAb98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5；app.conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa。前一goal turn完成12次研究、信号根因与技能晋级，是progress而非verified wait；本轮有新的完整候选与定向真实路径验证可推进，不把单独生产修复待批误认为整个研究必须停滞。

官方USD-M字段定义：[Binance public data](https://github.com/binance/binance-public-data#klines-1)。字段计算及真实接收行为以项目源码/诊断证据为准，机制是假设不是盈利承诺。

## 完成结果（不覆盖冻结内容）

12次四年运行全部结束，8个AF0/PV1控制完整复现。PV3四币频率均达标，但BTC/ETH净亏、四币均有亏损年度，失败保留，不进行未观察验证币收益、不发布/写库/启用。原主句柄91663已exit0，不重启。用量限制打断两项验证后root接手：1040项Expr通过（初版nil fixture失败保留）、真实receiver3833分钟闭合索引7666检查0失败但直接复现current taker缺口3689次、824笔补充信号独立审计0失败、1135笔交易分钟活动0零量填单、2988笔会计一致。详细范围及失败归因见2026-10-03-pullback-closed-flow-summary.md，不能把这些验证变成盈利/端到端通过。
