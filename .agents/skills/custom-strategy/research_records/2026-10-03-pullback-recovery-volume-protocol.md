# PV5：恢复突破的累计成交确认

2026-10-03北京时间19:04:55冻结完整JSON，任何PV5收益读取之前确定。PV4→PV5只在补充LONG/SHORT追加`kline_1h.Qps[0] >= mean(kline_1h.Qps[1:9]) * 0.90`并改名，阈值0.90沿用v29原退出冲量规则，不做阈值搜索。价格/总量[0]保留；不用已知缓存错误的forming taker[0]，不修生产。原基础入口、9指标、两个统一关闭对象逐字段原样。

## 接收语义与可检验假设

实际live GetLineFloatValues以Binance CloseTime−OpenTime为分母；历史klineQPSDurationSeconds和cachedKlineQPSDurationSeconds对forming overlay恢复完整名义周期。1h分母3599.999秒，因此此门槛等价于本小时已观察累计quote turnover达到前8个完整小时均值的90%，不是elapsed-minute成交速度、方向性买卖压力或OI。既有Qps[1]/[2]>0意味着均值正；保留固定8小时窗，不因较早/较晚信号改阈值。四年标准分钟信号只用到本分钟前一信号分钟结束，不引用当前小时未来量。

要检验的是价格恢复突破是否缺少足够已观察成交，而不是断言活跃度过滤必然创造利润。383个既有portable入口去空白全程序扫描无精确补充入口重复；成交确认原子v29已经用在退出，不算新发现，也不是任意等价表达式的形式化去重。

## 完整候选

| 文件（temp_strategy/20261003-pullback-recovery-volume/） | SHA-256 |
| --- | --- |
| 00-recovery-cumulative-volume-family.json | f42180704d5230c1baaa869db364c851feb289efe23fab0d4f4bad3e968e7f93 |
| 01-v29c-recovery-cumulative-volume-pullback.json | 3d75ea164b80ff90ddf95a2b8f91785650fa0ce05730238c4de9e3f286b6d724 |

基础AF0、PV4控制及PV5组合在一个原standard v7/1m引擎分别完整重撮合，各BTC/ETH/SOL/XRP，不静态删除PV4账本模拟PV5。base LONG/supp LONG/base SHORT/supp SHORT；统一AF0关闭，不读OpenStrategyHash。普通16/28盈利与-12亏损仍需信号，仅-20灾难例外ROI-only；外部profit/loss=5/5不等于自动退出。

## 日期与数据身份

主对照窗口仍UTC2022-09-01..2026-08-31，48个月/1461天/每币188笔门槛；必须重现8个AF0/PV4原控制逐笔、metrics、年度、来源hash。随后同候选和参数独立扩展至2026-09-30（49个月/1491天），覆盖技能更新后的2023/2024/2025/2026 Jan–Sep；不删除已有较差早期年度。新窗口数据hash/结果分开，频率按全部1491天，每币至少192笔。独立报告四个完整Sep–Aug年度、新增2026年9月，以及日历2023/24/25/2026 Jan–Sep交易cohort；45个月cohort不是从1000重新开跑的独立收益率。

冻结后官方16个2026年9月1m/1h/4h/1d CHECKSUM URL均200且格式有效；只是可用性，不等于完整内容/连续性/资金费通过，采集仍要原loader完整验证。官方月文件通常首个周一发布，实际本次已可读，不假定还未发布。新增月份错误须保留采集证据，不能算零交易/策略失败。复用checksum验证档案，独立request-key数据缓存；canonical volume差异保留，verified-archive v2纠错规则不变。

初始1000/币、当前现金10%保证金复利、8倍、双边各0.0005手续费与5bps滑点、真实历史资金费；每币>=0.9次/周、净正、四年稳定、跨币门槛不放宽。冻结未见AAVE/ATOM/ETC/LINK完整期资格仍待验证；开发失败不读取其收益，不只选盈利币。

### 新增月采集失败与限定真实资金费补充

原49月采集在BTC价格/连续性/既有v2纠错验证完成后报资金费覆盖不足，terminal exit1，零收益run，没有用零资金费或重试活进程。19:19三库只读核查：go_binance四币资金费只到UTC2026-09-12T16:00Z，每币Sep36条；另外两库无该期记录。既有loader只补前缀，不补尾部；不是月K线档案不存在。

19:21:56后既有官方BinanceSource.Funding GET /fapi/v1/fundingRate只读探测四币各61条，7条与ARM重叠的真实rate/mark逐条一致，54条尾部到Sep30覆盖；各0失败。冻结响应manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547，不重复REST请求或伪造结算价格。若来源/重叠/时序/覆盖不一致仍停止，不放宽原资金费校验。

仅仓库外verification/pv5_arm_data_funding_tail_copy.go基于原loader另存，追加hash固定官方真实尾部并保留source.funding_tail；原helper、engine、production/DB/配置都不改。新request严格限定49月固定日期，cache命中校验同manifest身份；独立cache public-canonical-repaired-v2-funding-tail-v1，新结果名带funding-tail-v1，和失败采集/48月结果区分。checksum档案可通过archive-cache-root共享已有验证文件。三组仍同原引擎同source完整重跑，后续必须验证旧年月source字段和逐笔前缀没有漂移；未达门槛不得发布。

## 验证与边界

真实Expr检查唯一raw差异、0/低于/等于/高于当前量门槛、闭合均值改变、时钟不应改变固定周期量定义、原闭合ratio/DI、入口可达/禁用/有序本地模型、关闭矩阵。独立逐补充入口聚合信号时间之前的当前1m quote，前8个canonical闭合1h均量，与真实原9指标BuildMinuteClose receiver Qps0/均值/完整程序比对；另做连续分钟/跨小时cached–uncached–live fixed CloseTime转换QPS对照。fresh入口重建不声称整个顺序cache/私有selector/live外部网络/forward/订单簿保证。

同条件会计/年度/方向/族及分钟活动审计，不把手续费/滑点/资金费遗漏当优势。三库仅程序只读元数据/v29，不操作App、修改app.conf、生产/前端、写库或启用；无仓库测试文件。全部结果与临时helper在研究缓存，失败完整策略保留。

## 实际完成结果（冻结之后补记）

48月原73829与49月真实资金费尾部59539均terminal exit0，两套各12次，共24次；首个49月缺资金费93276 exit1仍保留。49月PV5净BTC/ETH/SOL/XRP=+207.636/+320.725/+21.112/+728.925，周频0.667/0.765/0.836/0.653全失败，各有亏损完整年度。AF0/PV4也无联合通过，不获取冻结未见币收益或发布。

真实Expr1664/0；连续QPS字段/转换各34497检查0失败；49月219笔补充信号与fresh原9指标接收者0失败；三组2136笔活动0零量、会计逐笔恒等0/累计最大1.549e-12。四币全部原bar/资金费字段前缀exact一致，12条同版本原ledger前缀完全重现。完整结果/source身份/范围在`2026-10-03-pullback-recovery-volume-summary.md`和仓库外results/verification保留；以上诊断不是live/forward/订单簿或盈利保证。
