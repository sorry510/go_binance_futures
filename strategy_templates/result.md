# 策略研究结果

> 固定回测约束：4x 杠杆、TP 8、SL 6、fee 0.0005、slippage 5bps、单仓位、同参数跨币、lookback <= 100 天。
> 正式基线：strategy_templates ID 121，`v33 LONG + relaxed daily-ADX SHORT 正式候选`。
> 记录原则：每条路线记录假设、样本、结果、结论；失败路线冻结，避免后续重复排列组合。

## 2026-09-23 — v49：48h Semivariance 方向占比过滤 LONG

- 假设：过去 48h 上行半方差占总半方差 >= 70% 时，v33 LONG 的趋势扩张质量更高。
- 改动：仅给 v33 LONG 增加 semivariance gate；SHORT 与退出保持正式基线不变。
- 训练 10 币：BASE 573 笔，PF 1.398；CAND 524 笔，PF 1.355。
- symbol holdout（ADA/NEAR）：BASE 144 笔，PF 1.019；CAND 131 笔，PF 0.999。
- 2023 OOT：BASE 141 笔，PF 1.002；CAND 120 笔，PF 0.958。
- 年度：2024 PF 1.357 -> 1.394；2025 1.469 -> 1.244；2026 1.303 -> 1.415。
- 结论：局部年份改善，但训练总体、holdout、2023 OOT 都退化；不具备跨 regime 稳定性。
- 状态：**冻结 / 不入库**。不要继续调 48h 窗口或 0.70 阈值。

## 2026-09-23 — 48h Realized Skewness 入场过滤诊断

- 假设：v33 LONG 入场前 48 个 1h 收益的偏度可区分真实趋势扩张与尾部噪声。
- 样本：template 118 与正式候选相同的 LONG，10 币，共 302 笔；按固定偏度区间诊断，不调阈值。
- 总体：skew <-0.5 PF 2.774；-0.5~0 PF 2.449；0~0.5 PF 1.065；>=0.5 PF 1.245。
- 但年度方向明显冲突：2024 的 >=0.5 最强（PF 2.156），2025 则负/低偏度最强，2023 无稳定排序，2026 的 0~0.5 仅 PF 0.495。
- 结论：总体分组差异主要由年份组成驱动，不是跨 regime 稳定的 entry quality 因子。
- 状态：**冻结 / 不生成正式候选 / 不入库**。不继续调 24h/48h/72h 或 skew 阈值。

## 2026-09-23 — 1h Breakout Candle Acceptance / Body Quality

- 假设：v33 LONG 的突破 K 线若更接近区间高位收盘、实体占整根 K 线比例更高，则是假突破更少的“价格接受”信号。
- CLV 结果不稳定：2024 的 CLV >=0.85 PF 仅 0.941，而 2025 同组 PF 2.262；2026 的 0.70~0.85 又只有 PF 0.612。
- 实体占比总体有梯度：<0.35 PF 0.947，0.35~0.65 PF 1.304，>=0.65 PF 1.540。
- 但年度拆分不稳定：2023 <0.35 PF 5.604（仅 3 笔）、2024 <0.35 PF 3.498，2025 <0.35 PF 0.494；无法形成跨 regime 单调关系。
- 结论：价格接受度是描述性特征，但不能作为稳定 gate。
- 状态：**冻结 / 不生成正式候选 / 不入库**。不继续调 CLV 或 body/range 阈值。

## 2026-09-23 — v50：v33 LONG 0.50–1.00 ATR Controlled Breakout

- 假设：突破收盘超过前 12h 高点 0.50–1.00 ATR 的 LONG，是“足够强但不过度追涨”的高质量区间。
- 预诊断（302 个 v33 LONG）：该区间总体 PF 2.180；年度 PF 2023/2024/2025/2026 = 1.124 / 1.966 / 2.205 / 2.860，因此进入正式 Engine 验证。
- 实现：仅给 v33 LONG 增加固定 0.50–1.00 ATR gate；SHORT / CLOSE 规则不改。候选文件：`temp_strategy/v50/01-controlled-breakout-long.json`。
- 正式 Engine，训练 10 币：BASE 573 笔 PF 1.398 net +27769.8；CAND 394 笔 PF 1.433 net +16712.6。
- symbol holdout（ADA/NEAR/1000PEPE/SUI，2024+）：BASE 221 笔 PF 1.010 net +200.1；CAND 166 笔 PF 1.032 net +603.7。ADA/NEAR 改善，但 1000PEPE 与 SUI 退化。
- 14 币 2024+：BASE 690 笔 PF 1.342 net +28411.6，8/14 正；CAND 498 笔 PF 1.330 net +17673.5，11/14 正。
- 年度（12 老币）：2023 1.002 -> 1.153；2024 1.357 -> 1.394；2025 1.469 -> 1.195；2026 1.303 -> 1.640。
- 结论：能改善 2023/2026 和正币覆盖，但牺牲约 28–31% 交易、约 38% 总净收益，并显著伤害 2025；不是稳定升级。
- 状态：**冻结 / 不入库**。不继续调 0.4/0.6/0.8/1.2 ATR 等邻近阈值；后续优先研究新增独立 Setup，而不是继续过滤 v33。

## 2026-09-23 — v51：标准化 4h Shock Continuation 双向

- 假设：加密货币存在 intraday momentum；大幅 4h 冲击后若最后 1h 同向确认，价格可能继续延伸。该思路与项目已有 shock reversal 相反，属于新增 Setup，而非 v33 过滤。
- 规则：沿用旧 shock-reversal 的固定标准化定义与阈值，不搜索参数；LONG = shock >= 7 且 confirmation >= 1，SHORT = shock <= -7 且 confirmation <= -1。
- 候选文件：`temp_strategy/v51/01-normalized-shock-continuation.json`。
- standalone 正式 Engine 预筛（2023-01~2026-08）：BTC 319 笔 PF 0.929；ETH 369 笔 PF 1.032；BNB 295 笔 PF 0.671；XRP 444 笔 PF 0.751。
- 4 币合计：1427 笔，PF 0.957，net -1945.3，仅 1/4 正。
- 年度：2023 PF 0.994；2024 PF 0.975；2025 PF 0.865；2026 PF 1.031。
- 结论：交易频率很高，但没有正期望；不值得与 v33 合并，避免用高频负 alpha 稀释基线。
- 状态：**冻结 / 不做全币验证 / 不入库**。不继续搜索 shock=5/6/8 或 confirmation 阈值。

## 2026-09-23 — Binance Futures Metrics：Top Trader Positioning Divergence

- 数据源验证：Binance Vision USD-M `daily/metrics` 为 5m 粒度，抽查 BTC/ETH/BNB/XRP 的 2023/2024/2025/2026 文件均为 288 行/日 + header，字段稳定。
- 使用字段：`sum_toptrader_long_short_ratio`（Top Position）、`count_toptrader_long_short_ratio`（Top Accounts）、`count_long_short_ratio`（Global Accounts）。所有因子只用入场前最后一条已完成 5m 数据，避免未来函数。
- 静态 level 对 302 个正式 LONG 不稳定：2023/2024 高 divergence 更好，但 2026 明显反转，因此不做 level gate。
- 4h 动态变化初筛：LONG 的 Top Position/Top Accounts 上升总体较好（PF 1.541 vs 1.027），但 2023 反向；Top Position/Global 同样跨年反转。
- SHORT 四核心币初筛曾显示 Top Position/Global 4h 下降明显更好，因此扩大到正式 14 币验证。
- 14 币正式 SHORT 共 416 笔，metrics 有效 415 笔：D1_DOWN 246 笔 PF 1.212 net +5354.8；D1_UP 169 笔 PF 1.480 net +9386.0。
- 年度 D1：2023 DOWN 0.910 vs UP 1.174；2024 1.616 vs 1.350；2025 0.984 vs 1.166；2026 1.110 vs 1.897。
- 训练 10 币：DOWN PF 1.318 vs UP 1.502；holdout 4 币（ADA/NEAR/1000PEPE/SUI）：DOWN PF 0.934 vs UP 1.413。
- 逐币差异也明显，方向不是由统一机制驱动；四核心币结论属于样本选择偏差。
- 结论：简单的 Top Trader / Global positioning level 或 4h direction 不能稳定提升 v33，也不具备跨币可迁移性。
- 状态：**冻结简单 positioning gate / 不入库**。不继续调 1h/8h/12h 窗口或 divergence 阈值。

### Metrics standalone alpha 补充验证

- 为避免“过滤 v33 失败 ≠ 数据源本身无 alpha”，额外测试独立 Setup：四核心币 fresh 12h breakout + positioning 4h change 同向/反向。
- 2023-01~2026-08：BTC/ETH/BNB/XRP 共 10,645 个 breakout 事件，metrics 有效 10,637 个；静态归档 5,345 个交易日文件全部成功获取，0 download failure。
- D1（Top Position / Global）同向：4h mean +0.003%、win 44.6%；12h mean +0.049%、win 46.3%。反向：4h -0.006%、12h -0.015%。
- D2（Top Position / Top Accounts）同向与反向总体 12h mean 都约 +0.033%，没有区分度。
- 年度不稳定：例如 2024 D2 反向 12h +0.225%，同向 -0.024%；2026 则同向 +0.111%、反向 -0.057%。
- 结论：简单 positioning divergence/方向变化不具备足够强、稳定的独立预测力，不能支持 standalone breakout 策略。
- 路线状态：**整体冻结简单 Binance metrics positioning alpha**。除非以后引入本质不同的数据定义，否则不再做 level、1h/4h/8h 窗口和阈值排列组合。

## 2026-09-23 — Vortex(14) Crossover 独立趋势 Alpha

- 假设：Vortex 的 VI+/VI- 标准 14 周期交叉可捕获与 ADX/Donchian 不同的趋势形成速度。
- 规则：只使用标准 Vortex(14) crossover，不增加阈值；第一阶段仅检查交叉后 4h/12h 方向收益。
- 10 个老币、2023-01~2026-08：共 34,398 次交叉。
- 总体：4h mean -0.004%，win 48.7%；12h mean +0.005%，win 49.5%。
- 年度 12h mean：2023 +0.018%，2024 -0.010%，2025 0.000%，2026 +0.015%；没有可交易幅度。
- 逐币同样接近 0，ZEC 12h 甚至 -0.069%；不存在由少数币掩盖的稳定 edge。
- 结论：标准 Vortex crossover 在当前 USDT 永续样本没有方向预测力，进入手续费/滑点后只会更差。
- 状态：**冻结 / 不生成策略 / 不入库**。不继续搜索 Vortex period 或额外阈值。

## 2026-09-23 — Aroon(25) Oscillator Zero-Cross 独立趋势 Alpha

- 假设：最高/最低点的“新鲜度”差异可能捕获 Donchian 之外的趋势形成阶段。
- 规则：标准 Aroon(25)，只取 AroonUp-AroonDown 穿越 0 的方向，不增加 70/30 等二次过滤。
- 10 个老币、2023-01~2026-08：14,345 次交叉。
- 总体：4h mean -0.001%，win 47.1%；12h mean -0.007%，win 48.2%。
- 年度 12h：2023 -0.026%，2024 +0.031%，2025 -0.057%，2026 +0.043%，跨年方向反复。
- 结论：标准 Aroon crossover 没有足够的方向 alpha。
- 状态：**冻结 / 不生成策略 / 不入库**。不继续搜索 Aroon period 或 70/30 阈值。

## 2026-09-23 — Choppiness Index(14) 趋势启动双向 v52

- 假设：CHOP(14) 从标准趋势阈值 38.2 上方跌破 38.2，配合过去 14h 净动量方向，可捕获“震荡 -> 趋势”启动。
- 原始 alpha 诊断（10 老币）：7,366 个 transition；12h mean +0.110%，且 2023/2024/2025/2026 分别 +0.038% / +0.112% / +0.173% / +0.125%，因此进入正式 Engine。
- 将 CHOP<38.2 等价实现为 `sum(TR14)/range14 < 14^0.382 = 2.7404438598`，没有新增引擎指标；候选：`temp_strategy/v52/01-choppiness14-transition.json`。
- 正式 Engine，固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps：
  - BTC 672 笔 PF 0.913
  - ETH 732 笔 PF 0.818
  - BNB 708 笔 PF 0.791
  - XRP 872 笔 PF 0.548
- 四币合计 2,984 笔，PF 0.833，net -3976.3，0/4 正。
- 年度 PF：2023 0.841；2024 0.783；2025 0.886；2026 0.834。
- 结论：原始 forward-return 的小幅正值无法覆盖真实交易成本与固定 TP/SL 结构，不是可交易 alpha。
- 状态：**冻结 / 不扩大到 14 币 / 不入库**。不调 CHOP period 或 38.2/61.8 阈值。

## 2026-09-23 — v53：1h RSI(2) 极端回归 + EMA200 趋势过滤

- 假设：沿长期趋势方向，1h RSI(2) 极端回撤可能产生短周期均值回归；规则采用标准 RSI2 10/90 极值与 EMA200 趋势过滤，不搜索阈值。
- LONG：Close > EMA200 且 RSI2 从 >=10 下穿至 <10；SHORT 镜像为 Close < EMA200 且 RSI2 从 <=90 上穿至 >90。
- 候选：`temp_strategy/v53/01-rsi2-ema200-meanreversion.json`；固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps。
- BTC（2023~2026）：834 笔，PF 0.824，net -999.6。
- ETH（2023~2026）：1091 笔，PF 0.811，net -1000.0。
- BNB/XRP 从 2023 起存在本地 1h 历史缺口 `1671699600000~1671782399999`，未为本策略专门补数据。
- 为排除数据缺口干扰，XRP 改用 2024~2026：967 笔，PF 0.738；年度 PF 2024/2025/2026 = 0.737 / 0.821 / 0.792。
- 结论：高频但稳定负期望，不具备独立 mean-reversion alpha；失败与历史缺口无关。
- 状态：**冻结 / 不扩大 14 币 / 不入库**。不继续调 RSI2 的 5/10/15 或 85/90/95 阈值。

## 2026-09-23 — v33 24h Realized-Volatility Expansion Regime

- 假设：v33 的趋势 alpha 可能只在单币短期波动扩张时有效；定义 entry 前 24h realized volatility 与之前 7 个非重叠 24h realized volatility 均值之比，固定以 ratio > 1 区分 HIGH/LOW。
- 样本：v33 LONG 同源 302 笔，完全因果，最大回看约 8 天，不搜索阈值。
- 总体：LOW 61 笔 PF 1.773；HIGH 241 笔 PF 1.271。
- 年度：2023 LOW 2.899 vs HIGH 0.765；2024 LOW 0.755 vs HIGH 1.358；2025 LOW 2.549 vs HIGH 1.463；2026 LOW 1.003 vs HIGH 1.330。
- 结论：HIGH/LOW 优势逐年翻转，无法解释 2023 与后续年份的稳定差异。
- 状态：**冻结 / 不生成策略 / 不入库**。不继续调 3d/14d/30d 基准或 volatility ratio 阈值。

## 2026-09-23 — Binance Futures Positioning Ratios（Top Position / Top Account / Global）

- 数据源验证：Binance Vision `daily/metrics` 可取得 5m 历史数据；BTC/ETH/BNB/XRP 在 2023/2024/2025/2026 抽查日均为 288 行 + header，字段稳定，下载完整。
- 研究字段：`count_toptrader_long_short_ratio`、`sum_toptrader_long_short_ratio`、`count_long_short_ratio`；严格使用入场前最后一条已完成 5m 数据，无未来函数。
- 静态 level：`log(TopPosition / Global)` 与 `log(TopPosition / TopAccount)` 在 v33 LONG 条件样本里有分组差异，但 2023/2024 与 2026 出现明显 regime flip，不能作为稳定 gate。
- 4h delta：`TopPosition / TopAccount` 上升（D2_UP）在 v33 LONG 条件样本中 2024/2025/2026 PF = 1.349 / 2.320 / 1.312，2023 PF 0.956，看起来有潜力。
- 关键独立验证：BTC/ETH/BNB/XRP，2023-01~2026-08，每 3 天抽样，5:00~19:00 每小时观察，共 26,716 个非 v33 条件样本；D2 4h delta 按方向预测未来 4h。
- 独立结果：signed mean = **-0.0034%**，median -0.0121%，胜率 49.2%。年度 signed mean：2023 +0.0079%，2024 -0.0195%，2025 +0.0073%，2026 -0.0122%，无跨年一致性。
- 结论：positioning ratio 只在 v33 条件样本中出现条件共振，不具备独立 alpha；直接加入策略有较高条件过拟合风险。
- 状态：**整条 positioning-ratio 路线冻结 / 不入库**。不继续调 2h/4h/6h/12h 窗口或 ratio 阈值。

## 2026-09-23 — Exit 机制审计

- 当前 v33 CLOSE_LONG/CLOSE_SHORT 虽包含 trend/momentum reversal，但条件本身要求 ROI >= 8、ROI <= -6、ROI >= 14 或 ROI <= -10。
- Engine 固定硬 TP=8 / SL=6 会优先到达；template 118 的 420 笔历史交易退出原因为：stop_loss 258、take_profit 162，strategy close = 0。
- 结论：当前主动 close 规则在固定 TP/SL 下基本不可达；此前又已测试过多种提前退出方案，因此不继续做 exit 阈值微调。
- 状态：**退出微调路线冻结**。

## 2026-09-23 — v52：v33 LONG 突破前 12h 区间 2–3 ATR

- 假设：突破前 12 根 1h K 线的总 High-Low 区间位于 ATR14 的 2–3 倍，是“既非过度压缩、也非过度扩张”的稳定突破环境。
- 条件诊断（302 个 v33 LONG）曾显示该组总体 PF 2.098，年度 2023/2024/2025/2026 PF = 1.190 / 1.840 / 3.214 / 1.862，因此进入正式 Engine。
- 候选：`temp_strategy/v52/01-pre-range-2-3atr-long.json`；只过滤 LONG，SHORT/退出不变。
- 训练 10 币：BASE 573 笔 PF 1.398 net +27769.8；CAND 421 笔 PF 1.430 net +21988.6。
- holdout（ADA/NEAR/1000PEPE/SUI，2024+）：BASE 221 笔 PF 1.010 net +200.1；CAND 158 笔 PF 0.967 net -691.3。
- 14 币 2024+：BASE 690 笔 PF 1.342 net +28411.6，8/14 正；CAND 510 笔 PF 1.307 net +20905.9，8/14 正。
- 年度（12 老币）：2023 1.002 -> 1.461；2024 1.357 -> 1.395；2025 1.469 -> 1.429；2026 1.303 -> 1.218。
- 典型反例：BTC PF 2.265 -> 2.098、XRP 2.635 -> 1.514；ZEC 则改善，跨币一致性不足。
- 结论：能显著改善 2023，但以交易数、净收益、holdout 和 2026 为代价；再次证明条件分组不能直接转成 gate。
- 状态：**冻结 / 不入库 / 不调邻近 ATR 区间**。

## 2026-09-23 — v54：v33 + 2–3 ATR Precompression Funding-Bypass LONG

- 目标：不再过滤 v33，而是新增一个 LONG Setup 增频；保留原 v33 LONG，当原 funding 条件不支持时，如果突破前 12h 区间为 2–3 ATR，则允许额外 LONG。
- 候选：`temp_strategy/v54/01-precompression-funding-bypass-long.json`；SHORT / CLOSE 保持正式基线。
- 训练 10 币：BASE 573 笔 PF 1.398 net +27769.8；CAND 629 笔 PF 1.379 net +26322.9。交易 +9.8%，但 PF/净收益略退。
- holdout（ADA/NEAR/1000PEPE/SUI，2024+）：BASE 221 笔 PF 1.010 net +200.1；CAND 239 笔 PF 1.074 net +1602.1，holdout 有改善。
- 14 币 2024+：BASE 690 笔 PF 1.342 net +28411.6，8/14 正；CAND 749 笔 PF 1.339 net +28488.5，10/14 正。交易 +8.6%，PF 与净收益基本持平。
- 年度（12 老币）：2023 1.002 -> 0.978；2024 1.357 -> 1.266；2025 1.469 -> 1.351；2026 1.303 -> 1.458。
- 逐币明显分化：ETH/BNB/ADA/SUI 改善；DOGE/LTC/SOL/UNI 明显退化；BTC/XRP/ZEC近似中性或轻退。
- 结论：这是目前少数能在几乎不损失总体 PF 的情况下增频的方案，但新增交易的边际收益接近零，而且年度/币种不稳定；不能视为新的稳定 alpha。
- 状态：**保留研究结果，但不替换 ID121、不入库为正式策略**。不围绕 funding/precompression 再做邻近阈值搜索；后续若出现独立的新确认因子，可把 v54 的新增 Setup 作为待筛选样本，而不是直接实盘。

## 2026-09-23 — Ichimoku 9/26/52 Cloud Breakout 独立趋势 Alpha

- 假设：标准 Ichimoku 云层突破可能提供与 ADX/Donchian 不同的趋势形成信号。
- 定义：标准 Tenkan(9)、Kijun(26)、Senkou A/B(26 位移，B=52)；价格首次穿越当前因果云层，且 Tenkan/Kijun 同向；不加二次阈值。
- 10 老币，2023-01~2026-08：9,970 次事件。
- 总体：4h mean +0.020%，median -0.058%，win 47.4%；12h mean **+0.012%**，median -0.070%，win 48.2%。
- 年度 12h mean：2023 +0.030%，2024 -0.022%，2025 +0.020%，2026 +0.022%；逐币也明显分化（BNB/SOL/ZEC 正，AVAX/DOGE/ETH/LTC 负）。
- 结论：标准 Ichimoku crossover/breakout 的原始 edge 太小且不稳定，不足以覆盖手续费/滑点，更不值得进入固定 TP8/SL6 Engine。
- 状态：**冻结 / 不生成正式策略 / 不入库**。不调 9/26/52 或云层宽度阈值。

## 2026-09-23 — v55：v54 Funding-Bypass + 完整 Daily Bull Regime

- 目标：修正 v54 在 DOGE/LTC/SOL 等币上的低质量新增 LONG；不调阈值，而是给 bypass Setup 加入与现有 SHORT daily regime 完全镜像的日线多头结构。
- Daily bull regime：EMA20 > EMA50、EMA20/EMA50 均非下降、日线 Close > EMA20、阳线且 Close >= 前日 Close、ADX14 >=20、+DI > -DI。
- 候选：`temp_strategy/v55/01-daily-regime-funding-bypass-long.json`。
- 仅做 BTC/ETH/BNB/XRP 预筛：BASE 227 笔 PF 1.860 net +20181.5；CAND 233 笔 PF 1.842 net +20153.5，仅增加 6 笔。
- BTC：55 -> 57 笔，PF 2.265 -> 2.302；ETH：63 -> 65，1.402 -> 1.382；BNB：58 -> 60，1.200 -> 1.140；XRP 无新增交易。
- 年度：2023 PF 1.488 -> 1.304；2024 1.369 -> 1.457；2025 1.448 -> 1.420；2026 2.926 -> 2.914。
- 结论：完整 daily regime 几乎消除了 v54 的增频，同时没有产生稳定质量提升；尤其 2023 与 BNB 退化。
- 状态：**冻结 / 不扩大到 14 币 / 不入库**。不继续在 v54 上叠加 EMA/ADX/日线条件。

## 2026-09-23 — Spot vs Perpetual 4h Lead-Lag Confirmation

- 假设：若 Binance Spot 在 v33 LONG 入场前 4h 的涨幅高于 USD-M Perpetual，说明现货需求先行，趋势可能比杠杆驱动更可靠。
- 数据：Binance Vision spot monthly 1h K 线，临时读取、不落库；BTC/ETH/BNB/XRP 的正式 v33 LONG，共 115 笔。2025+ spot archive 时间戳为更高精度，统一转换到毫秒后有效 114/115。
- 定义：`spot_4h_return - perp_4h_return > 0` 为 SPOT_LEADS，否则 PERP_LEADS；只使用 entry 前已完成 K 线。
- 总体：SPOT_LEADS 45 笔 PF 1.621 net +2715.6；PERP_LEADS 69 笔 PF 2.079 net +6283.9。
- 年度：2023 spot 1.409 vs perp 2.908；2024 0.831 vs 2.477；2025 1.793 vs 1.505；2026 2.725 vs 2.209。
- 结论：lead-lag 确实分组，但优势方向在 2025 后翻转；“spot-led 更可靠”并不成立为跨 regime 统一机制。
- 状态：**冻结 / 不生成策略 / 不入库**。不搜索 1h/2h/8h lead 窗口或价差阈值。

## 2026-09-23 — v56：标准化 4h Shock Reversal 双向复核

- 背景：文献显示加密货币存在 intraday momentum 与 reversal，且大幅价格 jump 会改变二者关系；项目已有 shock reversal 文件，但未在本轮正式结果中验证。
- 规则：直接使用现有 `strategy_templates/volatility-shock-reversal/normalized-4h-shock-reversal-bidirectional-v1.json`，不调整 shock=7 或 confirmation=1。
- 正式 Engine，BTC/ETH/BNB/XRP，2023-01~2026-08，固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps。
- BTC：429 笔 PF 0.742；ETH：429 笔 PF 0.670；BNB：363 笔 PF 0.674；XRP：485 笔 PF 0.927。
- 四币合计：1,706 笔，PF **0.829**，net -3917.6，0/4 正。
- 年度 PF：2023 0.813；2024 0.910；2025 0.617；2026 0.881，全部 <1。
- 结论：大冲击后反转在当前固定 TP/SL 与成本下稳定负期望；此前 v51 shock continuation 也只有 PF 0.957。
- 状态：**整个 normalized shock continuation/reversal 家族冻结 / 不入库**。不再搜索 shock=5/6/8、confirmation 或中间混合阈值。

## 2026-09-23 — v54 Incremental Trades Meta-Labeling（价格/成交特征）

- 目标：完全不动 ID121 原 v33，只对 v54 相比 v33 真正新增的交易做质量识别。
- 精确重放后，v54 真正新增 **88 笔，全部为 LONG**；自身 PF **0.937**，net -534.3，27/88 胜。年度样本：2023=18、2024=30、2025=25、2026=15。
- 特征严格使用入场前已完成数据：4h/12h/24h return、quote-volume ratio、taker-buy delta、body/ATR、pre-range/ATR、12h path efficiency；不使用 symbol one-hot，不使用未来数据。
- 固定验证流程：10 老币 2023–2024 拟合 L2 Logistic；2025 做时间验证；2026 与 ADA/NEAR/1000PEPE/SUI 为未参与拟合的 OOS。概率阈值固定 0.44（依据 TP8/SL6 的经济 break-even 邻域），不搜索阈值。
- 训练：40 笔，AUC 0.726；P>=0.44 的 8 笔 PF 2.049，看似有效。
- 2025 验证：15 笔，AUC **0.361**；筛出的 3 笔 **0 胜，PF 0**。
- 2026 老币：10 笔，AUC 0.476；仅 1 笔过阈值。
- symbol holdout：23 笔，AUC **0.496**；筛出的 2 笔 PF 0.881。
- 结论：训练内可分，但时间外和 symbol holdout 均失效，属于典型 meta-model overfit；当前价格/成交特征无法稳定识别 v54 的好增量交易。
- 状态：**冻结该 meta-labeling 特征集与模型路线**。不调 Logistic 正则、概率阈值，也不换树模型/更复杂 ML 继续挖这 88 笔小样本。

## 2026-09-23 — v54 Incremental：Spot/Futures 成交量参与度

- 假设：v54 funding-bypass LONG 若同时出现现货成交量相对永续成交量的参与度上升，说明需求更偏现货驱动，新增交易质量可能更高。
- 定义：每个入场前最后已完成 1h，计算 `spot_quote_volume / futures_quote_volume`，再除以此前 24h 该比值均值；固定以 1.0 分为 HIGH / LOW，不调窗口、不调阈值。
- 数据：Binance Vision Spot monthly 1h，临时读取、不落库；1000PEPE 无同名 Spot，故 84/88 笔增量 LONG 有效。
- 总体：LOW 51 笔 PF 0.787 net -1216.5；HIGH 33 笔 PF 1.235 net +618.5。
- 年度 HIGH PF：2023 0.860；2024 **0.273**；2025 0.961；2026 4.911（仅 4 笔），严重不稳定。
- 训练币：LOW PF 0.560 vs HIGH 1.322；symbol holdout 却反转为 LOW 2.156 vs HIGH 1.029。
- 结论：训练内有分层，但年度与 symbol holdout 方向冲突，不具备跨币/跨 regime 稳定性。
- 状态：**冻结 Spot/Futures volume-share gate / 不入库**。不搜索 6h/12h/48h 归一化窗口或比例阈值。

## 2026-09-23 — v54 Incremental：Spot vs Futures Taker-Buy Imbalance

- 假设：v54 新增 LONG 若入场前最后完成 1h 的 Spot taker-buy ratio 高于 Futures taker-buy ratio，说明主动买盘更偏现货，可能比杠杆买盘更可靠。
- 定义：`spot_taker_buy_quote / spot_quote_volume - futures_taker_buy_quote / futures_quote_volume`，固定以 0 分界，不调阈值。
- 数据：Binance Vision Spot monthly 1h；1000PEPE 无同名 Spot，83 笔有效。
- 总体：SPOT_WEAKER 26 笔 PF 0.608 net -1586.7；SPOT_STRONGER 57 笔 PF 1.243 net +1033.4。
- 年度 SPOT_STRONGER PF：2023 1.377；2024 **0.607**；2025 1.677；2026 1.789，2024 明显失效。
- 训练币：SPOT_WEAKER 0.324 vs SPOT_STRONGER 1.321；但 symbol holdout **完全反转**：SPOT_WEAKER 3.443 vs SPOT_STRONGER 1.051。
- 结论：训练内关系强，但跨 symbol 不可迁移；Spot/Futures aggressor imbalance 不是稳定确认因子。
- 状态：**冻结整个 Spot/Futures 微观结构确认路线 / 不入库**。不继续调 1h/4h/24h taker/volume 组合。

## 2026-09-23 — 新增 Symbol Holdout：ONDOUSDT

- 目的：不再继续调策略，直接扩大真正未参与任何参数选择的 symbol holdout。
- 本地 ONDO 1h/1m 历史自 2024-01 上市后存在；由于回测构建器会为 1d 指标预取约 200 天，为避免把上市前不存在的数据误判为缺口，正式评估从 2024-08-15 开始，到 2026-09-01，仍超过 2 年。
- **ID121 v33 正式候选**：46 笔，PF **0.900**，net -283.9，Return -28.39%，MaxDD 56.30%。
- 年度：2024（8月中后）7 笔 PF 0.291；2025 25 笔 PF 1.052；2026 14 笔 PF 1.073。
- **v54 增频候选**：48 笔，PF **0.887**，net -314.4，Return -31.44%，MaxDD 61.01%。
- v54 年度：2024 PF 0.263；2025 1.016；2026 1.146。
- 结论：ONDO 是目前最干净的新 symbol holdout，结果显示 ID121 在该币上没有稳定正 edge；v54 也没有改善。2025/2026 虽略高于 1，但幅度不足以证明泛化。
- 状态：**ONDO 作为正式负向 holdout 保留**。不根据 ONDO 反向调参数，也不为它做 symbol-specific 适配。

## 2026-09-23 — ID121 同起点新币泛化 / Listing Maturity 诊断

- 为公平比较，把所有评估统一到 2024-08-15~2026-09-01；ID121 的 12 个老币已有正式保存回测，可直接截取。
- 老 12 币：459 笔，PF **1.368**，net +24163.7，8/12 正。
- 同期较新 3 币：1000PEPE 35 笔 PF 0.664；SUI 48 笔 PF 1.159；ONDO 46 笔 PF 0.900。合计 129 笔，PF **0.990**，仅 1/3 正。
- 年度：1000PEPE 2024/2025/2026 PF = 0.211 / 0.792 / 1.597；SUI = 2.617 / 0.989 / 1.372；ONDO = 0.291 / 1.052 / 1.073。
- 观察：三个新币到 2026 都改善，但并不存在随 listing age 单调上升的共同路径；SUI 2024 已强、2025 又回落，ONDO 2025/2026 接近 1。
- 结论：**新币 cohort 明显弱于老币 cohort，但“上市年龄”本身尚不能解释差异**；不能据此把选币门槛从 ≥2 年机械改成 ≥3 年。
- 状态：保留为 universe-generalization 证据；后续新候选必须同时在老币与新币 cohort 上验证，不做 symbol-specific 适配。

## 2026-09-24 — ID121 同起点泛化与收益集中度审计

- 为公平比较 ONDO 等新币，将旧 12 币统一截到 2024-08-15 ~ 2026-09-01，不改任何策略条件。
- 旧 12 币合计：459 笔，PF **1.368**，net +24163.7，8/12 正。
- 同期新 3 币：1000PEPE 35 笔 PF 0.664；SUI 48 笔 PF 1.159；ONDO 46 笔 PF 0.900；合计 129 笔 PF **0.990**，仅 1/3 正。
- 因此 15 币同起点总体仍约 PF **1.32**，但新增 symbol holdout 明显弱于旧币样本。
- 旧 12 币收益高度集中：XRP net +8380.9（34.7%）、BTC +7410.6（30.7%），两者合计约 **65.4%** 的净收益。
- 去掉 XRP + BTC 后，其余 10 币 386 笔仅 PF **1.148**，net +8372.2。
- 逐币 PF 中位数约 1.08（加入 1000PEPE/SUI/ONDO 后约 1.05），说明组合 PF 1.3+ 很大程度受少数强币拉动。
- 结论：ID121 仍有组合级 edge，但“同参数广泛跨币”的证据明显弱于总 PF 所示；新币外推与收益集中度都提示泛化风险。
- 状态：**保留 ID121 作为正式 benchmark / production candidate，但下调跨币泛化置信度**；不据此做 symbol-specific 参数适配。

## 2026-09-24 — ID121 动态 Symbol Eligibility：过去 90 天策略已实现净收益

- 动机：2025→2026 只有 4/12 老币连续两年为正，5/12 盈亏方向翻转，说明 edge 在币之间迁移；测试是否可用同一跨币规则动态决定“当前是否允许该币交易”。
- 规则：对每笔 ID121 交易，只使用该币此前 **90 天内、且已经平仓** 的同策略交易；至少 3 笔后，按过去 90 天已实现净收益 >0 / <=0 分组。无未来函数，lookback 90 天，不做 symbol-specific 参数。
- 总体：ROLL_POS 275 笔 PF 1.350 net +15757.9；ROLL_NONPOS 253 笔 PF 1.095 net +2600.3。
- 年度：
  - 2023：POS 0.995 vs NONPOS 1.380（反向）
  - 2024：POS 1.293 vs NONPOS 0.969
  - 2025：POS 1.561 vs NONPOS 0.848
  - 2026：POS 1.244 vs NONPOS 1.271（无区分）
- 结论：2024/2025 有明显择币信息，但 2023 反向、2026 消失；属于 regime-dependent performance chasing，不能作为统一跨年 eligibility。
- 状态：**冻结 rolling-performance gate / 不入库**。不继续搜索 30/60/100 天窗口、最低交易数或 PnL 阈值。


## 2026-09-24 — 新币 Holdout 评估规则修正：必须先满足历史 >= 2 年

- 重要修正：此前对 1000PEPE/SUI/ONDO 的新 symbol holdout，从它们历史不足 2 年时就开始评估，不符合系统真实选币规则“USDT 永续且至少 2 年历史”。
- 按本地最早历史时间重新定义 eligibility：1000PEPE 仅评估 2025-05-05 以后；SUI 仅评估 2025-05-03 以后；ONDO 仅评估 2026-01-20 以后。
- ID121 重新回测：1000PEPE 18 笔 PF 1.156（LONG 2.069 / SHORT 0.243）；SUI 35 笔 PF 0.914（LONG 1.402 / SHORT 0.784）；ONDO 14 笔 PF 1.073（LONG 0.585 / SHORT 仅 4 笔且全胜，样本过小）。
- 三币合计：67 笔 PF 1.039，net +171.5；LONG 29 笔 PF 1.073；SHORT 38 笔 PF 0.994。
- 结论：此前“新币 holdout 明确负向”的结论被修正。严格按真实 eligibility 后，新币总体接近盈亏平衡略正，但 edge 仍明显弱于老币；SHORT 泛化尤其弱。
- 状态：此段结论优先于此前 ONDO/NEW3 从 2024-08-15 起算的负向解读。以后所有新增合约 holdout 必须从其历史满 2 年后开始。


## 2026-09-24 — v54 Eligibility-Corrected New-Symbol Holdout

- 由于系统真实选币规则要求合约历史 >=2 年，重新在 eligibility 后比较 ID121 与 v54。
- 1000PEPE（2025-05-05 起）：BASE 18 笔 PF 1.156 net +199.2；v54 19 笔 PF **1.333** net +428.2。
- SUI（2025-05-03 起）：BASE 35 笔 PF 0.914 net -138.9；v54 39 笔 PF **1.018** net +37.3。
- ONDO（2026-01-20 起）：BASE 14 笔 PF 1.073 net +111.3；v54 15 笔 PF **1.146** net +283.7。
- 三币合计：BASE 67 笔 PF 1.039 net +171.5；v54 73 笔 PF **1.142** net +749.1。
- 结论：严格 eligibility 后，v54 在 3 个新 symbol holdout 上全部改善，并同时增加交易数；这与此前包含未满 2 年历史时期的负向结论不同。
- 状态：**重新开放 v54 作为增频候选**。下一步必须按动态 eligibility 在完整 universe 上重算，不能继续使用旧的 2024+ 固定 14 币结果作为最终判断。


## 2026-09-24 — 动态 Eligibility Universe：ID121 vs v54

- 按真实选币规则构建动态 universe：12 个老币从 2023-01 起；1000PEPE 从 2025-05-05；SUI 从 2025-05-03；ONDO 从 2026-01-20，统一结束于 2026-09-01。
- 总 eligible exposure = 2466 symbol-weeks。
- ID121：784 笔，PF **1.319899**，net +28211.99，10/15 正，频率 **0.317924 次/币/周**。
- v54：857 笔，PF **1.312781**，net +27765.33，10/15 正，频率 **0.347526 次/币/周**。
- v54 相对 ID121：交易数 +73（+9.3%）；PF 仅 -0.0071；净收益约 -1.6%；正币数不变。
- 结论：严格 eligibility 后，v54 是目前最接近“增频且基本不伤 PF”的候选，但频率仍只有约 0.35/币/周，距离 1/币/周目标很远。
- 状态：**v54 保留为增频候选，不替换 ID121**。后续优先寻找可叠加的独立 Setup，而非继续过滤 v33。

## 2026-09-24 — 新币 Holdout 校正：严格执行“历史 >= 2 年” Eligibility

- 重要校正：此前把 1000PEPE/SUI/ONDO 从 2024-08-15 同起点纳入评估，但这三币当时并未全部满足真实选币规则“合约历史 >=2 年”，因此该评估对 production 逻辑过于苛刻。
- 按各币自身满 2 年后才允许交易重新评估：
  - 1000PEPE：2025-05-05 起，18 笔，PF **1.156**；LONG 2.069，SHORT 0.243。
  - SUI：2025-05-03 起，35 笔，PF **0.914**；LONG 1.402，SHORT 0.784。
  - ONDO：2026-01-20 起，14 笔，PF **1.073**；LONG 0.585，SHORT 仅 4 笔且全赢，样本很小。
- 三币合计：67 笔，PF **1.039**，net +171.5，2/3 正。
- 年度：1000PEPE 2025 PF 0.680、2026 1.597；SUI 2025 0.596、2026 1.372；ONDO 仅 2026 PF 1.073。
- 结论：此前“新 3 币合计 PF 0.990”的负向 holdout 结论需要下调权重；真实 eligibility 下新币总体接近盈亏平衡略正，但仍明显弱于老币。
- 额外观察：1000PEPE/SUI 的主要拖累来自 SHORT；ONDO 则相反，说明不能用统一方向开关简单修复。
- 状态：**以后所有 symbol holdout 必须按实际 >=2 年 eligibility 截断**；旧同起点评估仅保留为诊断，不作为 production 泛化结论。

## 2026-09-24 — ID121 Production Eligibility 正式基线校正

- 从此处开始，所有跨币汇总严格执行真实选币规则：USDT 永续、历史 >=2 年；不能把新上市币在未满 2 年阶段的交易纳入 production 结果。
- 老 12 币（2024-01~2026-09）：583 笔，PF **1.368067541**，GP 104961.19，GL 76722.23，net +28238.96。
- 新 3 币仅在各自满 2 年后加入：1000PEPE/SUI/ONDO 合计 67 笔，PF **1.038749**，net +171.50。
- 合并后的 production-eligible universe：**650 笔，PF 1.3501，net +28410.5**。
- 这将替代“14/15 币统一固定起点”的数字作为后续正式 benchmark 口径。
- 结论：严格 eligibility 后，ID121 的组合级 edge 仍存在；但 2023 OOT PF≈1.00、收益集中于 BTC/XRP、新币 edge 较弱的问题仍未解决。

## 2026-09-24 — ID121 止损后 24h Re-entry 结构诊断

- 假设：若止损后短时间重新入场主要是 whipsaw，可用统一 cooldown 改善策略，不依赖额外技术指标。
- 定义：同币前一笔交易因 stop_loss 平仓后 24h 内发生的下一笔交易，和其它交易分组；只做诊断，不改规则。
- 总体：AFTER_SL_24H 50 笔 PF **1.356** net +2681.5；OTHER 674 笔 PF 1.326。
- 年度 AFTER_SL_24H：2023 PF 5.686（7笔）；2024 0.607；2025 1.276；2026 1.397。
- 结论：止损后重入并非稳定负 alpha；2024 虽差，但其它年份正向，加入 24h cooldown 会误删有效交易。
- 状态：**冻结 re-entry cooldown 路线**。不搜索 6h/12h/48h 等邻近窗口。

## 2026-09-24 — ID121 更早时间外验证：BTC/ETH 2021H2–2022

- 本地 BTC/ETH 1m chunk 从 2021-01-01 开始。为满足日线指标约 200 天 warmup，不下载新数据，最早干净评估期设为 2021-07-20 ~ 2022-12-31。
- BTC：35 笔，PF **0.737**，net -487.9；LONG 0.644，SHORT 0.791。2021H2 PF 0.503，2022 PF 0.837。
- ETH：30 笔，PF **0.507**，net -686.9；LONG 0.517，SHORT 0.499。2021H2 PF 0.375，2022 PF 0.607。
- 对照当前 ID121 已保存结果：
  - BTC 2023/2024/2025/2026 PF = 1.256 / 1.644 / 2.315 / 2.701
  - ETH 2023/2024/2025/2026 PF = 1.996 / 0.616 / 1.368 / 2.049
- 结论：ID121 不是全周期稳定策略；2021H2–2022 在 majors 上也明显负期望。2023 majors 已转正，但广泛 altcoin edge 当年仍弱，之后才逐步扩散。
- 状态：**保留 ID121 为当前 benchmark，但明确标记为 regime-dependent trend alpha**。不能把 2024–2026 PF 直接外推到更早市场周期。

## 2026-09-24 — 1h Compression Failed-Breakout Rejection v2 早期 OOT 复核

- 目的：寻找能覆盖 ID121 在 2021H2–2022 失效期的互补反转 Setup；直接使用现有 `volatility-expansion/compression-failed-breakout-rejection-bidirectional-v2.json`，不调参数。
- BTC 2021-07~2022-12：11 笔，PF 1.286，net +103.4；LONG 0.530，SHORT 4.435。2021H2 PF 0.489，2022 PF 1.801。
- ETH：21 笔，PF **0.525**，net -355.6；LONG 0.241，SHORT 0.958。2021H2 0.187，2022 0.989。
- 合计：32 笔，PF **0.773**，net -252.2，仅 1/2 正。
- 结论：假突破拒绝并不能稳定补足 v33 的旧周期缺口；BTC 2022 的局部效果不足以抵消 ETH 与 2021 的失败。
- 状态：**冻结 / 不扩展到全币 / 不入库**。不调 compression/extension/wick 阈值。

## 2026-09-24 — Signed Path-Efficiency Transition v1 早期 OOT 复核

- 目的：测试“价格路径从低效噪声转为高效单向运动”是否能作为 v33 在 2021H2–2022 的互补趋势 Setup。
- 直接使用现有 `path-efficiency/signed-efficiency-transition-bidirectional-v1.json`，不调 12h efficiency=0.5 或 48h trend 条件。
- BTC：389 笔，PF **0.782**，net -980.5；LONG 0.970，SHORT 0.627。2021H2 0.772，2022 0.813。
- ETH：508 笔，PF **0.856**，net -991.1；LONG 0.964，SHORT 0.765。2021H2 0.869，2022 0.827。
- 结论：高频但稳定负期望，尤其 SHORT 明显拖累；不能补足 v33 的旧周期失效。
- 状态：**冻结 path-efficiency transition 家族 / 不扩展 / 不入库**。不调 efficiency 阈值或 24h/72h trend 窗口。

## 2026-09-24 — ID121 2024–2026 Block Bootstrap 稳健性

- 目的：判断 PF 1.368 是否只是少数月份偶然贡献；以“同一自然月/季度内所有币交易”为一个 block，保留跨币同期相关性，不做 trade-level IID bootstrap。
- 样本：老 12 币 2024-01 起，583 笔，PF 1.368，33 个自然月、11 个季度 block。
- 100,000 次固定随机种子重采样：
  - 月 block：PF p05 **1.041**，p25 1.228，median 1.364，p75 1.505，p95 1.721；P(PF>1)=**96.92%**。
  - 季度 block：PF p05 **1.155**，median 1.369，p95 1.621；P(PF>1)=99.98%。
- 结论：在 2024–2026 这个 regime 内，组合 edge 并非只来自一两个幸运月份；月度 block 的保守下界仍略高于 1。
- 限制：这不是全周期证明。2021H2–2022 BTC/ETH 明显负、2023 全币接近盈亏平衡，因此 regime risk 仍是当前最大风险。

## 2026-09-24 — ID121 6 个月滚动 PF 诊断

- 目的：评估实盘中 edge 是否会出现持续数月失效；仅用于监控，不作为交易 gate。
- 老 12 币，按自然月滚动 6 个月聚合。
- 最差窗口：
  - 2024-05 ~ 2024-10：113 笔，PF **0.731**，net -2973.2。
  - 2023-07 ~ 2023-12：89 笔，PF 0.857。
  - 2023-04 ~ 2023-09：PF 0.866。
- 最近窗口：
  - 2025-12 ~ 2026-05：100 笔，PF **0.941**，net -1314.7。
  - 2026-01 ~ 2026-06：PF 1.329。
  - 2026-02 ~ 2026-07：1.245。
  - 2026-03 ~ 2026-08：1.440。
  - 2026-04 ~ 2026-09：**1.609**。
- 结论：ID121 即使在总体有效的 2024–2026，也存在约半年级别的负期望窗口；当前近期窗口已恢复较强。
- 状态：**rolling PF 仅作为上线后的健康监控指标，不作为自动停开策略条件**，避免重复 performance-gating 的过拟合。

## 2026-09-24 — Compression-Rejection SHORT-only 补充路线复核

- 早期 OOT 中 compression-rejection 的主要问题来自 LONG，SHORT 在 BTC 2021H2–2022 局部较好，因此单独检查 SHORT 是否可作为 v33 的补充 Setup。
- 既有 ID106（BNB/BTC/ETH/XRP，2023–2026）SHORT-only：64 笔，PF **0.622**，net -779.2。
- 逐币：BNB 0.450；BTC 0.977；ETH 0.616；XRP 0.550，0/4 稳定正。
- 年度：2023 0.588；2024 0.450；2025 0.672；2026 0.831，全部 <1。
- 结论：BTC 2022 的 SHORT 局部表现不可迁移，不能作为 v33 反周期补充。
- 状态：**compression-rejection 整个 family 完全冻结**，包括 SHORT-only。

## 2026-09-24 — 旧策略 ID92–112 单方向 Alpha 全量复查

- 目的：检查是否存在“整套策略失败，但 LONG 或 SHORT 单侧其实稳定”的遗漏 alpha，可作为 ID121 的补充 Setup。
- 对 ID92–112 所有已保存回测按 LONG/SHORT 独立聚合，并统计逐币正收益数量与年度最差 PF。
- 最好的几个：
  - ID103 Squeeze LONG：632 笔，PF 1.064，但仅 1/4 币正，最差年度 PF 0.559。
  - ID109 Funding crowd-failure LONG：408 笔，PF 1.053，仅 2/4 正，最差年度 0.649。
  - ID95 TSMOM+Donchian LONG：766 笔，PF 1.039，仅 1/4 正，最差年度 0.598。
- 其余方向全部 PF <=1；多数 family 两侧都明显负。
- 结论：旧策略库不存在可直接抽出并与 v33 组合的稳定单侧 alpha；此前“整套失败”的结论不是被另一侧拖累造成的。
- 状态：**旧 ID92–112 单方向组合路线冻结**。不再从已失败模板中挑 LONG/SHORT 拼装新策略。

## 2026-09-24 — ID121 固定名义本金标准化 PF 校正

- 问题：此前直接相加 `net_pnl`，每个币独立复利后，早期赚钱的币后续美元仓位更大，会放大 BTC/XRP 的 PnL 占比。
- 标准化方法：每笔使用 `net_pnl / (entry_price * quantity)` 作为固定名义本金收益；4x 杠杆只是统一常数，不影响 PF。
- 老 12 币 2024-01 起：583 笔，标准化 PF **1.386**，高于原始美元 PF 1.368。
- 标准化逐币：BTC 2.528、XRP 2.397、LTC 2.045、ZEC 1.717、ETH 1.512、SOL 1.437、BNB 1.372、ADA 1.173、NEAR 1.164；UNI 0.975、DOGE 0.933、AVAX 0.672。**9/12 为正**。
- BTC+XRP 的标准化净收益约占总标准化收益 **37.7%**，明显低于美元 PnL 口径的约 65.4%。
- 去掉 BTC+XRP 后，其余 10 币标准化 PF 仍约 **1.27**。
- 结论：此前“收益高度集中于 BTC/XRP”的担忧被独立复利放大；ID121 的跨币 edge 比美元 PnL 汇总显示得更均匀。
- 状态：后续跨币泛化优先同时报告 **标准化 PF + 原始美元 PF**，避免复利规模造成误判。

## 2026-09-24 — ID121 标准化年度 PF 与早期 OOT 再确认

- 固定名义本金收益定义：`net_pnl / (entry_price * quantity)`，用于消除独立复利导致的美元规模差异。
- BTC 2021H2–2022：35 笔，标准化 PF **0.817**；2021H2 0.607，2022 0.898。
- ETH 2021H2–2022：30 笔，标准化 PF **0.576**；2021H2 0.417，2022 0.649。
- 老 12 币标准化年度 PF：
  - 2023：141 笔，**1.028**
  - 2024：192 笔，**1.419**
  - 2025：228 笔，**1.305**
  - 2026：163 笔，**1.462**
- 结论：复利标准化不会消除 regime 差异。2021H2–2022 明显负，2023 近无 edge，2024–2026 稳定转正。

## 2026-09-24 — 新 3 币 Eligibility 后的标准化 PF 校正

- 继续采用固定名义本金收益 `net_pnl / (entry_price * quantity)`，消除独立复利造成的美元规模差异。
- 严格按各币“合约历史 >=2 年”后才纳入：
  - 1000PEPE：18 笔，raw PF 1.156，**NORM_PF 1.358**。
  - SUI：35 笔，raw PF 0.914，**NORM_PF 1.145**。
  - ONDO：14 笔，raw PF 1.073，**NORM_PF 1.307**。
- 新 3 币合计：67 笔，raw PF 1.038749，**NORM_PF 1.234236**。
- 结论：新币 raw-dollar 汇总接近盈亏平衡，主要受到独立复利路径影响；固定名义本金口径下三币均显示正 edge，虽然强度仍弱于老 12 币。
- 状态：后续 production 泛化统一优先看标准化 PF，同时保留 raw PF 作为实际独立账户复利路径参考。

## 2026-09-24 — ID121 标准化 6 个月滚动 PF 校正

- raw-dollar rolling PF 会受到各币独立复利后仓位美元规模变化影响，因此重新用固定名义本金收益计算。
- 标准化最差 5 个 6 个月窗口全部集中在 2023：
  - 2023-04~09：PF **0.816**
  - 2023-06~11：0.931
  - 2023-05~10：0.944
  - 2023-09~2024-02：0.956
  - 2023-07~12：0.962
- 最近窗口全部 >1：2025-12~2026-05 1.193；2026-01~06 1.699；2026-02~07 1.517；2026-03~08 1.326；2026-04~09 1.434。
- 结论：此前 raw-dollar 口径下 2024-05~10 PF 0.731 的“半年失效”主要受复利规模路径放大；标准化后真正持续弱期集中在 2023。

## 2026-09-24 — ID121 Production-Eligible 15 币最终标准化基线

- 正式 eligibility：老 12 币从 2024-01 起；1000PEPE/SUI/ONDO 仅在各自合约历史满 2 年后加入。
- 老 12 币：583 笔，标准化 GP 12.065810253364，GL 8.707978085807，NORM_PF 1.385604113。
- 新 3 币：67 笔，标准化 GP 1.356309317789，GL 1.098905682960，NORM_PF 1.234236331。
- 合计：**650 笔**，标准化 GP **13.422119571153**，GL **9.806883768767**，净标准化收益 **+3.615235802386**。
- 正式 production 标准化 PF：**1.368642669**。
- 对照 raw-dollar production PF ≈ **1.3501**，说明固定名义本金口径下 edge 略强，不依赖个别币复利放大。
- 后续所有候选必须同时与这两个 benchmark 比较：raw PF≈1.350、NORM_PF≈1.369，并严格执行 >=2 年 eligibility。

## 2026-09-24 — 生产 Universe 口径修正：仅在合约历史满 2 年后评估

- 发现：此前对 1000PEPE/SUI/ONDO 的“新币 holdout”包含了它们上市不足 2 年的阶段，但生产选币规则本身要求 **USDT 永续且历史 >=2 年**，因此旧评估口径过于苛刻且与真实运行不一致。
- 修正：不改策略，只从各币上市满 2 年后开始计入：
  - 1000PEPE：2025-05-05 起
  - SUI：2025-05-03 起
  - ONDO：2026-01-20 起
- 1000PEPE：18 笔 PF 1.156；LONG 11 笔 PF 2.069，SHORT 7 笔 PF 0.243。
- SUI：35 笔 PF 0.914；LONG 8 笔 PF 1.402，SHORT 27 笔 PF 0.784。
- ONDO：14 笔 PF 1.073；LONG 10 笔 PF 0.585，SHORT 4 笔全部盈利（样本极小）。
- 成熟新 3 币合计：67 笔 PF **1.039**，net +171.5，2/3 正；LONG 29 笔 PF 1.073，SHORT 38 笔 PF 0.994。
- 对比旧口径（三币 2024-08 起）PF 0.990，说明“至少 2 年历史”这个既定 universe 规则本身确实过滤掉了一部分不成熟阶段。
- 结论：ID121 在真实生产 universe 下的新币泛化从“略负”修正为“近盈亏平衡略正”，但仍明显弱于旧 12 币；不能据此宣称强泛化。
- 状态：**以后所有正式泛化评估按动态 >=2 年历史资格计入**，不再把未满 2 年阶段算进生产表现。

## 2026-09-24 — ID121 生产 Benchmark 稳健性 / Bootstrap

- 正式生产口径：仅在合约历史满 2 年后计入；2024-08-15 ~ 2026-09-01，共 15 个动态 eligible symbols。
- Benchmark：526 笔，PF **1.347**，net +24335.2；4.929 trades/week；0.362 trades/symbol/week。
- 年度 PF：2024 1.331；2025 1.423；2026 1.288。
- 逐币：10/15 净收益为正；median symbol PF **1.073**。最弱 AVAX 0.614、UNI 0.654；最强 XRP 3.073、BTC 2.410。
- Leave-one-symbol-out：PF 最低 1.241（移除 XRP），中位 1.352，说明单独去掉任何一个币后组合仍 >1。
- 固定随机种子 20260924，10,000 次 symbol bootstrap：
  - PF 5%/50%/95% = **1.131 / 1.344 / 1.784**
  - Pr(PF>1)=**99.98%**；Pr(PF>1.1)=98.35%；Pr(PF>1.2)=84.04%。
- 10,000 次 symbol-month block bootstrap（保留同币同月局部相关性）：
  - PF 5%/50%/95% = **1.047 / 1.337 / 1.717**
  - Pr(PF>1)=**97.4%**；Pr(PF>1.1)=90.42%；Pr(PF>1.2)=76.48%。
- 结论：ID121 的组合级正 edge 具有统计稳健性，不完全由 BTC/XRP 单点偶然驱动；但 cross-symbol 中位强度只有约 PF 1.07，真正问题是 **edge 分布不均、频率仍低**，而不是“整个策略完全没有 edge”。
- 后续目标：新策略/Setup 不仅比较总 PF，还必须提高 median symbol PF / positive-symbol coverage，并避免通过少数强币拉高总收益。

## 2026-09-24 — v54 按正式生产 Universe 重新评估：升级为第二候选

- 背景：此前 v54 使用静态 14 币 / 2024+ 口径时，被判定为“增频但年度 edge 不稳定”；随后正式 benchmark 修正为 **仅在合约历史满 2 年后才计入**，因此重新按相同生产口径评估 v54。
- 生产 universe：旧 12 币自 2024-08-15；1000PEPE 自 2025-05-05；SUI 自 2025-05-03；ONDO 自 2026-01-20；截止 2026-09-01。
- **ID121 benchmark**：526 笔，PF 1.347，net +24335.2，10/15 正，median symbol PF 1.073，4.929 trades/week，0.362 trades/symbol/week。
- **v54**：563 笔，PF **1.396**，11/15 正，median symbol PF **1.146**，5.276 trades/week，0.388 trades/symbol/week。
- 相对 ID121：
  - 交易数 +7.0%
  - 总 PF 1.347 -> **1.396**
  - median symbol PF 1.073 -> **1.146**
  - positive-symbol coverage 10/15 -> **11/15**
  - trades/symbol/week 0.362 -> **0.388**
- 新币成熟阶段：1000PEPE PF 1.333、SUI 1.018、ONDO 1.146，均不再是明显负向 holdout。
- 年度仍明显不均衡：2024 PF 1.444；2025 **1.063**；2026 **1.792**。此前 2023 old-symbol OOT 约 PF 0.978，也未优于 ID121。
- 结论：在真实生产 eligibility 下，v54 不再只是“零边际增频”；它同时改善频率、组合 PF、median symbol PF 和正币覆盖，值得升级为 **第二正式候选**。
- 但 2025/2026 regime 差异过大，因此 **暂不替换 ID121**。后续验证重点不再调 v54 参数，而是验证其 2025 弱势是否来自特定市场机制，以及是否能在不损害 2026 的前提下稳定化。

## 2026-09-24 — 生产 Benchmark 严格同起点校正（Supersedes earlier mixed-start aggregate）

- 发现并修正评估口径问题：此前 ID121 的“生产 benchmark”是从更早启动的历史 run 截取，而 v54 是从生产起点重新启动。由于 Engine 的权益/仓位大小存在路径依赖，两者不能严格直接比较。
- 本次对 **ID121 与 v54 都使用完全相同的动态生产起点重新启动 Engine**：
  - 旧 12 币：2024-08-15
  - 1000PEPE：2025-05-05（满 2 年）
  - SUI：2025-05-03（满 2 年）
  - ONDO：2026-01-20（满 2 年）
  - 截止 2026-09-01
- **严格 ID121**：519 笔，PF **1.395**，net +17015.9，10/15 正，median symbol PF **1.084**，4.863 trades/week，0.358 trades/symbol/week。
- ID121 年度：2024 PF 1.494；2025 PF 1.157；2026 PF 1.647。
- **严格 v54**：563 笔，PF **1.396**，net +18131.7，11/15 正，median symbol PF **1.146**，5.276 trades/week，0.388 trades/symbol/week。
- v54 年度：2024 PF 1.444；2025 PF 1.063；2026 PF 1.792。
- 严格差异：
  - 交易数：+44（+8.5%）
  - 总 PF：1.395 -> **1.396**（几乎完全持平）
  - net：+17015.9 -> **+18131.7**
  - positive symbols：10/15 -> **11/15**
  - median symbol PF：1.084 -> **1.146**
  - trades/symbol/week：0.358 -> **0.388**
- 但年度不是单调升级：v54 在 2024/2025 都弱于 ID121，仅 2026 明显更强。
- 结论：**v54 升级为与 ID121 并列的第二正式候选（增频候选），但不替换 ID121。** ID121 更稳定，v54 提供更高频率与更好的 cross-symbol 中位表现。
- 注意：此前基于“截取旧 run + 新币重跑”的 526 笔 PF1.347 及其 bootstrap，只保留为探索性结果，**不再作为正式 benchmark**。正式比较以后统一使用严格同起点 Engine。

## 2026-09-24 — v57：v54 + 对称 2–3 ATR Funding-Bypass SHORT

- 假设：既然 v54 的 precompression funding-bypass LONG 能在生产 universe 下增频且不损失总 PF，同一结构可能对 SHORT 也有效。
- 实现：保留 v54 全部逻辑；额外新增 SHORT Setup，仅在 `FundingRate < 0` 时允许，要求同样的 12h pre-range 2–3 ATR，并保留原 SHORT 的 daily regime / 4h trend / ADX strength / fresh breakdown / impulse / no-chase。
- 先做 7 币预筛（BTC/ETH/BNB/XRP + 1000PEPE/SUI/ONDO），使用与正式 benchmark 相同的动态 >=2 年生产起点。
- 相比 v54：
  - BTC：PF 2.333 -> 2.162
  - ETH：1.419 -> 1.336
  - BNB：1.163 -> 1.163（基本无变化）
  - XRP：3.122 -> 2.776
  - 1000PEPE：1.333 -> **0.858**
  - SUI：1.018 -> **0.809**
  - ONDO：1.146 -> 1.238（改善，但样本仅 18 笔）
- v57 7 币合计 261 笔 PF 1.674、5/7 正，但这个聚合值主要受 BTC/XRP 强币拉动；逐币 paired comparison 明确显示多数币相对 v54 退化。
- 结论：precompression funding-bypass **不具有 LONG/SHORT 对称性**。LONG bypass 是当前可保留机制，SHORT 镜像会明显稀释质量，尤其伤害 1000PEPE/SUI。
- 状态：**冻结 v57 / 不扩大到 15 币 / 不入库**。不继续调 SHORT funding 阈值或 precompression 区间。

## 2026-09-24 — Supertrend(10,3) Flip 独立趋势 Alpha

- 假设：标准 Supertrend(ATR10, multiplier=3) 的趋势翻转可能提供独立于 ADX/Donchian 的高频趋势 Setup。
- 参数采用最常见标准值 10/3，不做搜索；1h 因果计算，仅在 Supertrend direction flip 后观察未来方向收益。
- 10 老币，2023-01~2026-08：7,243 次 flip。
- 总体：4h mean -0.039%；12h mean **-0.029%**，median -0.093%，win 47.6%，仅 5/10 币 12h mean >0。
- 年度 12h mean：2023 +0.009%；2024 -0.069%；2025 -0.053%；2026 +0.016%，没有稳定正向。
- 逐币：SOL/UNI/AVAX略正，BNB/BTC/DOGE/ETH/ZEC 等为负，跨币一致性不足。
- 结论：标准 Supertrend flip 在成本前就没有足够 alpha，不值得进入固定 TP8/SL6 Engine。
- 状态：**冻结 / 不生成正式策略 / 不入库**。不搜索 ATR period 或 multiplier。

## 2026-09-24 — Lowest-Price Behavioral Anchor Reversal（单币适配）

- 灵感：近期研究显示，横截面 crypto reversal 若用形成期最低价格作为行为锚点，可优于传统 reversal；但原论文是 cross-sectional portfolio，不能直接用于当前单币 DSL。
- 为避免硬搬论文，只做一个无参数搜索的单币机制测试：30 天 formation（论文最短 horizon）；前一根 1h 收盘创 30 天新低后，当前 1h 收盘反包前一根 High 做 LONG；新高则镜像做 SHORT。
- 10 老币，2023-01~2026-08：785 次事件。
- 总体：4h mean -0.183%；12h mean **-0.435%**，median +0.116%，win 52.5%；24h mean -0.446%。均值显著为负，说明少数大亏损吞噬较高胜率。
- 逐币仅 BTC（+0.081%）和 SOL（+0.185%）12h mean >0，其余 8/10 为负；DOGE/ZEC/XRP 尤其差。
- 年度 12h mean：2023 -0.430%；2024 -0.535%；2025 -0.549%；2026 -0.091%，四年均负。
- 结论：最低价格锚点的论文优势不能简单迁移为单币极值反转；该单币机制稳定负 alpha。
- 状态：**冻结 / 不进 Engine / 不入库**。不调 formation=20/60/90 天或反包阈值。

## 2026-09-24 — Directional Change / Intrinsic-Time 1×ATR14

- 假设：按价格自身转折事件而不是固定时间 K 线定义趋势，可能提供独立于 ADX/Donchian 的 intrinsic-time alpha。
- 定义：1h close 维护运行 peak/trough；当价格从 peak 回撤 >=1×ATR14 触发 DOWN directional change，从 trough 反弹 >=1×ATR14 触发 UP；顺新方向观察 forward return。ATR multiplier 固定 1.0，不做搜索。
- 10 老币，2023-01~2026-08：37,197 次事件。
- 总体：4h mean -0.018%；12h mean **-0.045%**，median -0.040%，win 48.9%；24h mean -0.008%。
- 逐币仅 ETH 12h mean +0.006%，其余 9/10 为负；ZEC/XRP/DOGE 更差。
- 年度 12h mean：2023 -0.048%；2024 -0.085%；2025 -0.010%；2026 -0.033%，四年均 <=0。
- 结论：标准 intrinsic-time directional-change follow 在当前市场样本中没有正 alpha，且高频会被成本进一步恶化。
- 状态：**冻结 / 不进 Engine / 不入库**。不搜索 0.5/1.5/2.0 ATR threshold 或 overshoot 长度。

## 2026-09-24 — Hurst / Fractal Persistence Transition 原始 Alpha

- 假设：市场从随机/反持久状态进入持久状态时，顺同一窗口价格方向可能形成独立趋势 alpha。
- 定义：滚动 64h；对 lag=1/2/4/8/16 的 log-price 增量方差做 log-log slope，得到 Hurst H；当 H 从 <=0.5 跨到 >0.5 时，按过去 64h 价格方向做多/做空。0.5 为随机游走自然分界，不搜索阈值。
- 10 老币，2023-01~2026-08：2,982 次 transition。
- 总体：4h mean +0.049%；12h mean **+0.149%**，median +0.005%，win 50.0%；24h mean +0.158%。
- **10/10 币 12h mean 全部为正**：从 ETH +0.019% 到 SOL +0.350%，这是当前新机制里罕见的跨币一致性。
- 年度 12h mean：2023 +0.268%；2024 +0.137%；2025 +0.163%；2026 +0.007%。2026 基本衰减到零，是主要风险。
- 结论：原始 forward alpha 值得进入更真实的 TP/SL + 成本验证，但现有 DSL 无法直接表达完整 Hurst，暂不修改引擎。
- 状态：**进入二阶段验证**。下一步先做 1m 路径、单仓位、4x/TP8/SL6/fee/slippage 模拟；只有通过才考虑新增 Hurst indicator。

## 2026-09-24 — Hurst Persistence Transition 二阶段：Standard TP/SL + 成本

- 目的：验证 Hurst raw forward alpha 是否能被当前固定交易结构捕获；不修改引擎，按正式 `Engine.Run` standard 语义做临时模拟。
- 执行语义与正式 Engine 对齐：Hurst 信号在 1h completed bar 后产生；下一根 1m open 成交；4x；TP8/SL6 在每根 1m close 按 leveraged ROI 检查，触发后下一根 1m open 平仓；双边 fee=0.0005、slippage=5bps，并计入 funding；单仓位。
- 10 老币，2023-01~2026-08：2,769 笔，**PF 0.893**，net -7120.2，仅 BTC 1/10 正；1.448 trades/symbol/week。
- 逐币 PF：AVAX 0.807、BNB 0.671、BTC 1.036、DOGE 0.868、ETH 0.883、LTC 0.938、SOL 0.728、UNI 0.961、XRP 0.708、ZEC 0.839。
- 年度 PF：2023 0.949；2024 0.797；2025 0.904；2026 0.930，四年全部 <1。
- 结论：虽然 Hurst transition 的固定 12h forward mean 跨 10 币均为正，但该漂移无法被固定 TP8/SL6 + 成本结构捕获；属于“平均漂移存在，但路径/尾部不适合当前执行”的机制。
- 状态：**冻结 Hurst / fractal persistence 路线，不新增引擎指标、不入库**。不调 H threshold、window、TP/SL。

## 2026-09-24 — 24h Log-Price Regression Slope t-stat Transition

- 假设：与简单 momentum 不同，用线性回归 slope 的统计显著性定义趋势启动；过去 24 根 1h log(price) OLS，当 slope t-stat 首次跨过 +2/-2 时顺方向。
- 参数固定：24h 自然日窗口、|t|=2 统计显著性阈值；不做搜索。
- 10 老币，2023-01~2026-08：13,589 次事件。
- 总体：4h mean +0.067%；12h mean **+0.041%**，median -0.051%，win 48.6%；24h mean +0.001%。
- 逐币 8/10 的 12h mean 为正，但 BTC -0.030%、UNI -0.096%；多数币 median 仍为负。
- 年度 12h mean：2023 -0.005%；2024 +0.082%；2025 +0.030%；2026 +0.070%。
- 结论：统计显著趋势启动存在轻微 drift，但幅度太小、median 为负、2023 不正，成本前就不足以支持正式 Engine。
- 状态：**冻结 / 不进 Engine / 不入库**。不调 12/48h window 或 t-stat 1.5/2.5/3。

## 2026-09-24 — 24h Standardized Return CUSUM Shift

- 假设：用序贯累计偏移检测趋势漂移，比固定动量阈值更早识别状态变化。
- 定义：每个 1h return 用此前 24h return 的 mean/std 标准化；双边 CUSUM 累积 z-score，首次达到 +3/-3 时顺方向并重置。24h 与 3σ 均为固定统计定义，不搜索参数。
- 10 老币，2023-01~2026-08：38,272 次事件。
- 总体：4h mean -0.014%；12h mean **-0.015%**，median -0.053%，win 48.6%；24h mean -0.070%。
- 逐币仅 AVAX/ETH/UNI 3/10 的 12h mean 略正；其余为负。
- 年度 12h mean：2023 -0.049%；2024 -0.063%；2025 +0.056%；2026 -0.002%。
- 结论：CUSUM shift 主要捕捉短期噪声/过冲，而非可持续 drift；成本前已经没有 edge。
- 状态：**冻结 / 不进 Engine / 不入库**。不调 baseline window 或 CUSUM threshold。

## 2026-09-24 — 24h Return Sign Imbalance

- 假设：趋势若真实存在，不仅累计收益应同向，过去 24 个小时的涨跌方向也应出现显著多数；该机制完全忽略收益幅度，避免大单根 K 线支配 momentum。
- 定义：过去 24 个 1h return 中，上涨小时数首次达到 >=18 做 LONG，<=6 做 SHORT；在独立 Bernoulli(0.5) 下双侧尾概率约 2.27%，窗口/阈值固定，不搜索参数。
- 10 老币，2023-01~2026-08：1,052 次事件。
- 总体：4h mean -0.003%；12h mean **+0.102%**，median -0.065%，win 48.7%；24h mean +0.174%。
- 逐币仅 6/10 的 12h mean 为正；ETH -0.743%、DOGE -0.211%、LTC -0.166%、AVAX -0.021%。
- 年度 12h mean：2023 +0.028%；2024 **-0.158%**；2025 +0.033%；2026 +0.718%，明显依赖 2026。
- 结论：方向一致性存在局部 drift，但跨币与跨年不稳定、median 为负，无法作为第三个通用 Setup。
- 状态：**冻结 / 不进 Engine / 不入库**。不搜索 17/19 个上涨小时或 12/48h 窗口。

## 2026-09-24 — 24h Mann–Kendall Trend-Significance Transition

- 假设：非参数 Mann–Kendall 趋势检验对极端单根 K 线更稳健；过去 24 个 1h close 首次达到 5% 显著上升/下降趋势时顺方向。
- 定义：24h window，Mann–Kendall Z 首次跨过 ±1.96；均为统计自然阈值，不搜索参数。
- 10 老币，2023-01~2026-08：13,446 次事件。
- 总体：4h mean +0.071%；12h mean **+0.056%**，median -0.035%，win 48.9%；24h mean +0.034%。
- 逐币仅 6/10 的 12h mean 为正；BNB/ETH/LTC/UNI 为负。
- 年度 12h mean：2023 +0.042%；2024 +0.102%；2025 **-0.010%**；2026 +0.111%。
- 结论：非参数显著趋势仍只有很薄的 drift，median 为负且 2025 失效；与 OLS t-stat 同属高频弱 edge。
- 状态：**冻结整个 24h statistical-significance trend family / 不进 Engine / 不入库**。不继续 runs test / Theil–Sen / 邻近显著性阈值。

## 2026-09-24 — Dynamic Momentum Cycle：causal κ=5 SMA Turning-Point 近似

- 文献背景：Borgards (2021) 将 time-series momentum 定义为连续 turning-point momentum cycles；positive cycle 要求连续两个 peak 与两个 trough 同时抬高，negative cycle 镜像。formation period 完成后入场，cycle 条件失效后退出；论文使用 moving-average smoothing filter、κ=5。
- 由于公开论文未给出 smoothing filter 的完整算法，本轮**不冒充原论文复现**，只做一个明确的 causal approximation：trailing SMA5 slope sign flip 确认 turning point；最近四个确认点构成 HH+HL / LH+LL 时，仅第一次 formation transition 发信号。
- 10 老币，2023-01~2026-08：20,490 个 approximation events。
- 总体：4h mean +0.005%；12h mean **+0.034%**，median -0.011%，win 49.6%；24h mean +0.022%。
- 年度 12h mean：2023 +0.056%；2024 +0.021%；2025 **-0.016%**；2026 +0.095%。
- 逐币仅 6/10 mean >0；BNB/DOGE/LTC/UNI 为负。
- 关键问题：事件密度远高于论文 1h momentum-cycle trading results，证明该 causal SMA slope-flip 近似过于敏感，不能代表论文原 turning-point filter。
- 结论：**冻结该近似实现，不冻结论文机制本身**。只有找到原 smoothing filter 精确定义后才允许重新验证；不围绕这个近似调 SMA 长度/确认 bar。

## 2026-09-24 — Dynamic Momentum Cycle causal κ=5 approximation：4h 复核

- 动机：原论文报告 dynamic momentum 在较低频率（1D/1h）成本后仍有效，而高频 5m 被交易成本吞噬；因此保持同一 causal SMA5 turning-cycle 近似，仅降低到 4h，避免继续改算法参数。
- 10 老币，2023-01~2026-08：4,982 个 formation-like events。
- 总体：4h mean -0.045%；12h mean **+0.003%**，median +0.003%，win 50.0%；24h mean +0.109%。
- 逐币 6/10 mean >0，但 DOGE/SOL/UNI/XRP 为负；XRP -0.226%、SOL -0.204%。
- 年度 12h mean：2023 -0.093%；2024 -0.091%；2025 +0.244%；2026 -0.070%，明显 regime-specific。
- 结论：降低到 4h 后仍无法得到稳定 edge，进一步证明 trailing-SMA slope turning-point 并非原论文 smoothing filter 的有效替代。
- 状态：**冻结 1h/4h causal SMA approximation**。仅因论文明确显示 1D 更强，再做一次 daily approximation；若仍失败，则完全停止该近似路线。

## 2026-09-24 — Dynamic Momentum Cycle causal κ=5 approximation：1D 最终复核

- 目的：原论文报告 1D dynamic momentum 成本后表现最强之一，因此在不改任何 turning-point approximation 参数的前提下，将同一 trailing-SMA5 cycle 结构降到日线做最终复核。
- 10 老币，2023-01~2026-08：806 个 formation-like events。
- 总体：1d mean -0.145%；3d mean **-0.368%**，median -0.485%，win 45.5%；5d mean -0.492%。
- 逐币仅 AVAX/BNB/SOL/UNI 4/10 的 3d mean >0；BTC -0.647%、ETH -0.626%、XRP -1.532%、ZEC -1.310%。
- 年度 3d mean：2023 -0.972%；2024 -0.604%；2025 +0.467%；2026 -0.373%。
- 结论：同一 causal SMA turning-point approximation 在 1h、4h、1D 三个频率都没有稳定 edge；该近似无法代表 Borgards 原 smoothing-filter momentum cycle。
- 状态：**完全关闭 causal SMA approximation 路线**。原论文机制仅保留“待原算法/代码”研究项，不再做任何近似参数调整。

## 2026-09-24 — Huang et al. Volume-Weighted TSMOM：不适用于当前单币约束

- 论文：Huang, Sangiorgi, Urquhart, *Cryptocurrency Volume-Weighted Time Series Momentum* (2024)。
- 公开方法描述的核心是 **volume-weighted market return**，并基于 TSMOM 构造 volume-weighted winner-minus-loser portfolios；第三方策略资料也将其描述为多币组合、周期性 rebalancing。
- 这不是“每个 symbol 自己用历史 volume 加权自己的 return 就产生独立 LONG/SHORT”的单资产规则；若自行这样改写，会成为未经原论文支持的新策略。
- 当前固定约束禁止 Benchmark / 横截面市场组合依赖，并要求同一套单币规则可跨合约独立运行，因此该论文框架与项目约束不兼容。
- 结论：**不实现、不回测，不把论文 portfolio alpha 强行改造成单币 signal。**
- 状态：**方法不适用 / 冻结**。若未来允许 cross-sectional universe/portfolio signal，再单独重开。

## 2026-09-24 — Hurst Persistence Transition 二阶段：1m Engine-like TP/SL 验证

- 二阶段严格模拟标准 Engine 语义：Hurst 1h 信号在小时收盘确认，下一根 1m open 入场；单仓位；4x；TP8 / SL6 按 gross leveraged ROI 在 minute-close 触发，下一根 1m open 平仓；双边 fee=0.0005、slippage=5bps，并计入本地 funding。
- 10 老币，2023-01~2026-08：2,769 笔实际可执行交易。
- 总体：PF **0.893**，net -7120.2，只有 BTC 1/10 币净收益为正。
- 逐币 PF：AVAX 0.807、BNB 0.671、BTC 1.036、DOGE 0.868、ETH 0.883、LTC 0.938、SOL 0.728、UNI 0.961、XRP 0.708、ZEC 0.839。
- 年度 PF：2023 0.949；2024 0.797；2025 0.904；2026 0.930，四年全部 <1。
- 结论：Hurst transition 的 +0.149% 12h forward mean 是小幅漂移，不具备当前固定 TP8/SL6 所需的幅度/路径；成本和止盈止损后稳定负期望。
- 状态：**Hurst / fractal persistence family 冻结 / 不修改 DSL 或引擎 / 不入库**。不搜索 Hurst window、threshold 或 ATR/exit 适配。

## 2026-09-24 — Return-Trajectory 100d Nearest-Analog（1-NN）原始 Alpha

- 灵感：近期研究指出 BTC return predictability 依赖历史 return trajectory；为避免 TDA/随机森林复杂化，做一个最小、可解释、lookback<=100d 的单币 analog 测试。
- 定义：使用最近 24h 的 6 个 4h log returns，按 RMS 归一化保留方向/形状；在过去 100 天且不与当前轨迹重叠的历史中找欧氏距离最近的一段，以该历史轨迹之后的下一根 4h return 符号预测当前方向。无概率阈值、无 symbol-specific 参数。
- 10 老币，2023-01~2026-08：76,923 个信号。
- 总体：4h mean +0.006%；12h mean **+0.015%**，median +0.002%，win 50.0%；24h mean +0.015%，幅度远低于交易成本。
- 逐币 6/10 mean>0，但幅度均很小；年度 12h mean：2023 +0.032%、2024 +0.050%、2025 **-0.029%**、2026 +0.005%。
- 结论：简单 rolling nearest-trajectory analog 没有经济上可交易的 edge；论文中的 trajectory predictability 不能靠 1-NN 直接迁移到当前单币高频框架。
- 状态：**冻结 / 不进 Engine / 不入库**。不搜索 k-NN、trajectory length、distance threshold 或复杂 ML/TDA。

## 2026-09-24 — v54 更早 OOT：BTC/ETH 2021H2–2022

- 目的：检验 v54 是否是独立于 ID121 的第二 regime，而不仅是 2024–2026 趋势环境中的增频扩展。
- 使用与此前 ID121 early-OOT 完全相同区间：2021-07-20 ~ 2022-12-31；本地 BTC/ETH 历史足够，固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps。
- BTC：ID121 35 笔 PF 0.737；v54 37 笔 PF **0.690**。2022：0.837 -> **0.770**。
- ETH：ID121 30 笔 PF 0.507；v54 34 笔 PF **0.411**。2022：0.607 -> **0.583**。
- v54 新增 LONG 在 early OOT 反而更弱：BTC LONG PF 0.577，ETH LONG PF 0.353；并没有补足 ID121 的熊市/早期 regime 缺陷。
- 结论：v54 不是独立的跨周期 alpha，而是当前 v33 trend family 的增频扩展。它在严格 2024–2026 production universe 下值得作为并列增频候选，但不能提高 2021H2–2022 的周期鲁棒性。
- 状态：**保留 v54 为第二候选，但明确归类为同一 trend-alpha family**；后续第三 Setup 必须寻找能在 ID121/v54 共同失效期提供互补性的不同机制。

## 2026-09-24 — Bollinger(20,2) Band Re-entry 标准均值回归

- 目的：寻找能补 ID121/v54 在 2021H2–2022 失效期的独立 mean-reversion Setup；不用 RSI(2)，直接测试价格越过 Bollinger(20,2) 极端后重新回到带内。
- 定义：前一根 1h Close < LowerBand 且当前 Close >= 当前 LowerBand 做 LONG；前一根 Close > UpperBand 且当前 Close <= 当前 UpperBand 做 SHORT。标准 20/2 参数，不搜索。
- 2023–2026 10老币：19,826 次事件，12h mean **-0.033%**，median +0.098%，win 52.4%，仅 3/10 币 mean>0。
- 年度 12h mean：2023 +0.019%；2024 -0.038%；2025 -0.025%；2026 -0.119%。
- Early OOT：BTC 2021H2–2022 mean +0.051%，ETH +0.119%，但正向几乎全部来自 2021H2；2022 BTC -0.017%、ETH -0.016%，没有补足真正的 2022 弱势。
- 结论：较高胜率来自小幅回归，但少数大趋势亏损吞噬收益；在固定 TP8/SL6 下更不具备优势。
- 状态：**冻结标准 Bollinger re-entry / 不进 Engine / 不入库**。不搜索 band period、std multiplier 或额外 RSI 过滤。

## 2026-09-24 — v57 Early-OOT：Negative-Funding SHORT Bypass 熊市复核

- 目的：v57 在 2024+ 明显伤害 v54，但可能理论上补足 2021–2022 熊市中 funding 经常为负、原 SHORT 被阻挡的问题；因此做最后一次 early-OOT 复核。
- BTC 2021H2–2022：ID121 PF 0.737；v54 0.690；v57 **0.648**。2022 单年：0.837 -> 0.770 -> **0.709**，放宽 SHORT 越多越差。
- ETH：ID121 0.507；v54 0.411；v57 0.542。2022 单年 v57 0.792，高于 ID121 0.607，但仍明显 <1；且与 BTC 方向冲突。
- v57 BTC SHORT PF 0.660；ETH SHORT 0.786，均未形成正期望。
- 结论：负 funding 并不是“应该继续追空”的稳定确认；它更可能代表拥挤/反身性风险。该 SHORT bypass 既不能改善当前生产期，也不能稳定修复 early bear OOT。
- 状态：**v57 / negative-funding SHORT bypass 全面冻结**。不再从 funding 符号放宽 SHORT。

## 2026-09-24 — 90d Time-Trend t-stat（±2）Follow vs Fade

- 文献规则：对价格线性趋势斜率计算 t-stat；经典 TREND 在 t>+2 做多、t<-2 做空；另有研究指出当趋势强度接近 2 时可能进入反转区。±2 为文献原始显著性阈值，不调。
- 为满足 lookback<=100d，使用 90 日 log-price OLS trend；每个满足 |t|>=2 的日线点，比较顺趋势 FOLLOW 与反向 FADE 的未来 3d return。
- 2023–2026 10老币：11,678 个事件。总体 FOLLOW mean +0.022%，FADE -0.022%，两者都太小且逐币仅 5/10 为正。
- 年度出现明显 regime flip：
  - 2023：FOLLOW **-0.438%** / FADE +0.438%
  - 2024：FOLLOW +0.185% / FADE -0.185%
  - 2025：FOLLOW +0.235% / FADE -0.235%
  - 2026：FOLLOW +0.124% / FADE -0.124%
- Early OOT 也不统一：BTC 2022 FOLLOW +0.247%，但 ETH 2022 FOLLOW -0.673%（FADE +0.673%）。
- 结论：trend-strength t-stat 能很好描述 regime 差异，但无法提供一个统一跨币静态交易方向；用它做 FOLLOW/FADE gate 会直接变成 regime fitting。
- 状态：**作为诊断保留，交易规则冻结 / 不进 Engine / 不入库**。不搜索 t-stat threshold 或 horizon。

## 2026-09-24 — ID121 vs v54 严格同起点 Fixed-Notional 标准化比较

- 目的：消除各币独立复利/仓位规模路径，确认 v54 的 raw PF 优势是否真实；ID121 与 v54 使用完全相同的动态 eligibility 起点、相同 dataset、相同 Engine。
- ID121：519 笔，raw PF 1.395080；**normalized PF 1.385786**，normalized net +2.943552；13/15 币标准化净收益为正；median normalized symbol PF **1.307005**。
- v54：563 笔，raw PF 1.395847；**normalized PF 1.383515**，normalized net +3.193649；13/15 正；median normalized symbol PF **1.347779**。
- v54 相比 ID121：
  - 交易数 +44（+8.5%）
  - overall normalized PF 基本完全持平，且微降 1.3858 -> 1.3835
  - normalized net 因交易增多提升 +2.9436 -> +3.1936
  - median symbol PF 提高 1.3070 -> 1.3478
  - 标准化正币覆盖均为 13/15
- 年度 normalized PF：
  - 2024：ID121 1.539 / v54 1.485
  - 2025：ID121 1.240 / v54 1.159
  - 2026：ID121 1.536 / v54 1.675
- 逐币并非单调升级：LTC 1.920 -> 1.514、DOGE 1.123 -> 1.027、XRP 2.802 -> 2.727；但 ADA/BNB/NEAR/1000PEPE/SUI/ONDO 等改善。
- 结论：**v54 的价值是“近零 PF 成本的增频 + 更好的跨币中位数”，不是更高的整体 edge。** 它保留为并列增频候选；ID121 仍是更稳的主 benchmark。

## 2026-09-24 — Statistically Meaningful Trend（SMT：|t|>2 且 R²>65%）

- 文献定义：time-trend 斜率 t-stat 超过 ±2，并要求线性回归 R²>65%，只保留“显著且路径确实由趋势解释”的 trend；阈值均来自原方法，不调参。
- 为满足 lookback<=100d，使用 90 日 log-price OLS trend，顺 slope 方向观察未来 3d。
- 2023–2026 10老币：3,962 个事件，3d mean **-0.178%**，median -0.202%，win 48.5%，仅 4/10 币 mean>0。
- 年度：2023 -0.224%；2024 -0.329%；2025 +0.142%；2026 -0.464%。
- Early OOT：BTC 2021H2–2022 mean -0.475%，ETH -1.395%；ETH 2022 仍 -0.923%。
- 结论：高 t-stat + 高 R² 的“干净趋势”在 crypto 并不等于后续延续，反而经常对应过度延伸；不能补 ID121/v54 的弱 regime。
- 状态：**SMT / regression-trend family 冻结 / 不进 Engine / 不入库**。不搜索 R²、t-stat 或 horizon。

## 2026-09-24 — Hurst Persistence Transition 二阶段：1m TP8/SL6 Engine-like 验证

- 原始 Hurst alpha 曾表现为 10/10 币 12h mean >0、总体 +0.149%，因此进入更真实的交易验证。
- 执行语义严格贴近现有 Engine：Hurst 信号在 1h close 完成后，下一根 1m open 开仓；单仓位；4x；TP8/SL6 按 leveraged gross ROI 触发；触发后下一根 1m open 平仓；开/平各 5bps slippage；双边 fee=0.0005；包含 funding cashflow。
- 10 老币 2023-01~2026-08：2,769 笔实际交易。
- 总体：PF **0.893**，net -7120.2，仅 **1/10** 币为正。
- 代表逐币：BTC PF 1.036；AVAX 0.807；BNB 0.671；DOGE 0.868；ETH 0.883；LTC 0.938；SOL 0.728；XRP 0.708；ZEC 0.839。
- 年度 PF：2023 0.949；2024 0.797；2025 0.904；2026 0.930，四年全部 <1。
- 结论：Hurst transition 的 forward mean 虽跨币为正，但收益形态属于缓慢漂移，无法匹配固定 4x/TP8/SL6 的路径要求；进入真实成本和退出结构后稳定负期望。
- 状态：**Hurst / fractal persistence 路线冻结 / 不新增 indicator / 不入库**。不调 H 窗口、0.5 阈值、lag 集合或 TP/SL。

## 2026-09-24 — Kalman Local-Trend Innovation Breakout

- 假设：local-linear-trend state-space 模型的标准化 innovation 若达到异常水平且与滤波 trend 同向，可能代表新的结构性位移。
- 定义：1h log-price，Kalman local level + trend；measurement/process 噪声尺度来自过去 24h return variance；当 |innovation z|>=2 且 sign(innovation)=sign(filtered trend) 时顺方向；z 回到 |1| 内重新 armed。固定 2σ，不搜索参数。
- 10 老币，2023-01~2026-08：4,520 次事件。
- 总体：4h mean +0.104%；12h mean +0.116%，median -0.064%，win 48.7%；24h mean -0.018%。
- 逐币 7/10 的 12h mean >0；AVAX/BTC/ETH/SOL/UNI较好，但 BNB -0.018%、XRP -0.195%、ZEC -0.079%。
- 年度 12h mean：2023 **-0.086%**；2024 +0.182%；2025 +0.272%；2026 +0.141%。
- 结论：2024–2026 有一定冲击延续，但 2023 OOT 反向且跨币不一致，24h 又衰减为负；不足以作为跨 regime 第三 Setup。
- 状态：**冻结 / 不进 Engine / 不入库**。不调 state noise、2σ threshold 或 re-arm 阈值。

## 2026-09-24 — 4h Price Curvature Zero-Cross Reacceleration

- 假设：价格自身的离散二阶曲率从负转正/正转负，并与 8h 主方向一致时，可能代表趋势重新加速；该结构独立于 ADX acceleration。
- 定义：1h log-price，`curvature = log(C_t) - 2*log(C_{t-4}) + log(C_{t-8})`；curvature 穿越 0 且 8h return 同向时顺方向。零阈值为数学自然边界，不搜索参数。
- 10 老币，2023-01~2026-08：38,315 次事件。
- 总体：4h mean **+0.001%**；12h mean +0.069%，median -0.051%，win 48.6%；24h mean +0.054%。
- 逐币 8/10 的 12h mean >0，但 LTC -0.010%、UNI -0.026%；绝大多数 median 仍为负。
- 年度 12h mean：2023 +0.030%；2024 +0.015%；2025 +0.150%；2026 +0.088%。
- 结论：存在轻微长期 drift，但 4h 位移几乎为零，无法覆盖当前成本，更不匹配 4x/TP8/SL6 的目标路径。
- 状态：**冻结 / 不进 Engine / 不入库**。不调 2h/6h/8h curvature 间隔或二阶阈值。

## 2026-09-24 — 3-Bar Directional Streak Breakout

- 假设：连续 3 根同向 1h close 后，若下一根 close 再突破这 3 根的最高/最低价，可能代表短时动量延续，适合固定 TP8/SL6。
- 定义：前三根 1h close 严格单调上升/下降；当前 close 突破此前三根 High/Low 后顺方向。固定 3-bar，不搜索 2/4/5。
- 10 老币，2023-01~2026-08：36,460 次事件。
- 总体：4h mean -0.024%；12h mean **+0.014%**，median -0.065%，win 48.4%；24h mean +0.035%。
- 仅 4/10 币 12h mean >0：AVAX/BNB/ETH/ZEC；BTC/DOGE/LTC/SOL/UNI/XRP 为负或近零。
- 年度 12h mean：2023 -0.020%；2024 +0.010%；2025 +0.030%；2026 +0.041%。
- 结论：连续上涨/下跌后的突破没有足够 continuation edge，4h 甚至轻微反转；远不足以覆盖成本。
- 状态：**冻结 / 不进 Engine / 不入库**。不搜索 streak 长度或 breakout bar 数量。

## 2026-09-24 — v54 vs ID121 严格 Paired Attribution

- 目的：解释 v54 为什么在严格生产口径下能增频且总 PF 不降，区分“新增 Setup 真有 alpha”与“single-position 改变交易时序”的作用。
- 对 15 个动态 eligible symbols 使用完全相同起点、相同 Dataset，分别重放 ID121 与 v54；按 `side + entry_time` 将交易分为 COMMON / BASE_ONLY / V54_ONLY。
- COMMON：509 笔，PF **1.410**，net +17275.5。
- ID121 独有：10 笔，PF **0.724**，net -259.5。
- v54 独有：54 笔，PF **0.972**，net -142.1。
- 因此 v54 的 54 笔“新增/替代”交易本身只接近盈亏平衡，并不是独立强 alpha；v54 能维持组合 PF 的一部分原因，是 single-position 时序变化同时挤掉了 10 笔更差的 ID121-only 交易。
- 年度 V54_ONLY：2024 14 笔 PF 0.862；2025 23 笔 PF **0.638**；2026 17 笔 PF **1.591**。
- 年度 BASE_ONLY：2024 3 笔全亏；2025 3 笔 PF **6.519**；2026 4 笔全亏。
- 2025 的核心问题很清楚：v54 新增交易明显负期望，同时挤掉少量非常好的 ID121 交易；2026 则恰好反过来。
- 所有 BASE_ONLY / V54_ONLY 均为 LONG，SHORT 不是差异来源。
- 结论：**v54 保持“并列增频候选”定位，但不能称其 funding-bypass Setup 为独立正 alpha。** 它的优势高度 regime-dependent；不应继续放宽 bypass 来追频率。

## 2026-09-24 — 第三独立 Setup 搜索阶段结论

- 当前严格生产 benchmark 必须使用“动态历史 >=2 年 eligibility + 完全相同起点重新启动 Engine”的口径，禁止再混用截断旧 run。
- **ID121**：主 benchmark，严格生产口径 519 笔，PF 1.395，median symbol PF 1.084，10/15 正，0.358 trades/symbol/week。
- **v54**：并列增频候选，563 笔，PF 1.396，median symbol PF 1.146，11/15 正，0.388 trades/symbol/week；fixed-notional normalized overall PF 与 ID121 基本持平。
- paired attribution 显示 v54-only 54 笔 PF 0.972；其组合优势来自“近零边际新增交易 + single-position 挤掉更差 base-only 交易”，不是新的独立强 alpha。2025 v54-only PF 0.638、2026 1.591，regime dependence 明显。
- 本阶段新增并已冻结：Hurst/fractal persistence、Kalman innovation、price curvature、3-bar streak breakout、Directional Change/intrinsic-time、Lowest-Price behavioral anchor；此前 CUSUM、SMT regression trend、Return Sign Imbalance、CHOP、Aroon、Vortex、Ichimoku、Supertrend 等也已冻结。
- 共同失败模式：很多机制在 forward mean 上略正，但进入 4x + TP8/SL6 + fee/slippage 后失效；这说明当前策略要求的不是“轻微 drift”，而是足够快且足够大的路径扩张。
- 当前结论：**尚未找到第三个跨币、跨 regime、能独立覆盖成本并匹配 TP8/SL6 的 Setup。**
- 后续研究原则：不再堆传统技术指标或邻近阈值；只有真正新的数据机制或能产生强短时位移的结构才进入测试。ID121/v54 参数保持冻结。

## 2026-09-24 — Binance Mark Price vs Last Price Dislocation

- 动机：mark price 直接参与强平；极端行情研究显示 mark、spot、futures 在 liquidation cascade 中可发生明显不同步，因此测试 mark-last dislocation 是否能提前指示强位移。
- 数据：Binance Vision USD-M `markPriceKlines` 公共历史归档；确认 BTC/ETH/SOL/XRP 在 2021/2023/2026 均有长期 1m/5m 数据。诊断数据流式读取，不落库。
- 固定定义：5m mark close 与同刻 futures last close 的 log-dislocation；用此前 24h（288 根）均值/std 标准化；首次 `|z|>=3` 触发，`|z|<1` re-arm；按 mark 相对 last 的方向做 continuation。不搜索 sigma 阈值。
- 4 核心币 2023-01~2026-08，共 **25,055** 个事件。
- 逐币 4h signed mean：BTC -0.0099%；ETH +0.0193%；SOL +0.0315%；XRP +0.0534%。
- 总体：1h mean +0.0116%；4h mean **+0.0206%**，win 50.9%；12h mean **-0.0025%**。
- 年度 4h mean：2023 +0.0470%；2024 -0.0002%；2025 +0.0298%；2026 +0.0037%。
- 结论：mark-last 极端偏离虽与 liquidation mechanics 有关，但其后续方向 edge 只有几个 bp，12h 已完全消失；无论 continuation 或反向都不足以覆盖 fee/slippage。
- 状态：**mark-last dislocation family 冻结 / 不进 Engine / 不入库**。不调 2σ/4σ 或 1m/15m 邻近窗口。

## 2026-09-24 — Taker-Flow Price-Impact Residual / Kyle-like Liquidity Depletion

- 假设：当价格位移远大于相同主动订单流通常能解释的幅度时，说明被动流动性被耗尽，价格可能继续向 residual 方向扩张。
- 定义：过去 24h 用 `1h log return ~ taker signed-flow fraction` 做 OLS；当前 residual / 历史回归残差 sigma 首次达到 `|z|>=3` 且实际 return 与 residual 同向时触发；`|z|<1` re-arm。24h/3σ 固定，不搜索参数。
- 10 老币，2023-01~2026-08：7,981 次事件。
- 总体：4h mean +0.050%，median **-0.074%**，win 47.2%；12h mean +0.042%；24h mean -0.070%。
- 逐币 8/10 的 4h mean 略正，但 LTC -0.043%、XRP -0.007%；多数币 median 为负。
- 年度 4h mean：2023 **-0.063%**；2024 +0.061%；2025 **+0.180%**；2026 +0.017%，明显 regime-dependent。
- 结论：异常 price impact 在部分年份有 continuation，但 2023 反向、24h 衰减为负；反向交易又会伤害 2025，因此不是稳定方向 alpha。
- 状态：**Kyle/price-impact residual family 冻结 / 不进 Engine / 不入库**。不调 baseline window、sigma threshold 或改成简单 impact ratio。

## 2026-09-24 — Binance BookDepth ±1% Notional Imbalance

- 新数据源：Binance Vision USD-M `daily/bookDepth`，周期性盘口深度快照，字段包含 timestamp、±1%~±5% depth/notional；BTC 单日压缩文件约 0.46MB。相比 `bookTicker`（BTC 单月约 1.98GB），bookDepth 可用于受控抽样研究。
- 固定初筛：BTC/ETH/SOL/XRP；每季度固定连续 3 天，首 24h warmup；±1% bid/ask notional imbalance，过去24h z-score 首次 `|z|>=2` 且 raw imbalance 同向时触发；下一分钟 open 后观察 15m/1h/4h。参数不搜索。
- 2023–2025 初筛：1,383 事件；1h mean +0.0496%，win 55.2%，4/4 币 mean>0；年度 2023/2024/2025 = +0.0389% / +0.0518% / +0.0547%。
- 数据格式注意：2026 起 percentage 从 `-1/1` 变为 `-1.00/1.00`；修正为数值解析后单独重测 2026。
- 2026：291 事件；15m mean -0.0021%，1h mean **+0.0036%**，4h mean **-0.0124%**；仅 BTC +0.1354%，ETH -0.0083%、SOL -0.0126%、XRP -0.0426%。
- 结论：简单 near-book level imbalance 在 2023–2025 有弱预测力，但到 2026 基本消失且跨币反转，不是稳定第三 Setup。
- 状态：**简单 ±1% depth imbalance 冻结 / 不进 Engine**。保留 bookDepth 数据源，仅允许测试本质不同的盘口形状机制，不调 z-score、深度百分比或时间窗口。

## 2026-09-24 — Binance BookDepth Near-vs-Far Concentration Asymmetry

- 假设：不是比较 bid/ask 总量，而比较哪一侧流动性更贴近现价：`bid1%/bid5% - ask1%/ask5%`。正值表示买方深度更集中在近端，负值表示卖方更集中。
- 固定定义：过去24h shape z-score；首次 `|z|>=2` 且 raw shape 与 z 同向时按 shape 方向交易；下一分钟 open 后观察 15m/1h/4h。不调阈值。
- 先只做最困难的 2026 季度抽样（Jan/Apr/Jul，每季度连续3天，首24h warmup）。
- BTC：51 事件，1h mean +0.0455%，win 62.7%；ETH：86，+0.0367%，58.1%；SOL：60，+0.0723%，53.3%；XRP：85，**-0.1014%**，43.5%。
- 四币合计：282 事件；15m mean +0.0098%；1h mean **+0.0043%**；4h mean +0.0828%；仅 3/4 币 1h mean>0。
- 结论：盘口形状比简单 imbalance 在 BTC/ETH/SOL 上更健康，但 XRP 明显反向，整体 1h edge 近零；不足以扩大到旧年份或正式 Engine。
- 状态：**depth concentration asymmetry 冻结 / 不入库**。不调 ±2%/±3% depth level 或 z threshold。

## 2026-09-25 — Binance BookDepth Global Near-Liquidity Vacuum

- 假设：若总盘口近端深度相对远端深度突然大幅收缩，说明价格附近出现 liquidity vacuum；随后价格可能沿此前 1h momentum 继续扩张。
- 定义：`(bid1% + ask1%) / (bid5% + ask5%)`；过去24h z-score，首次 z<=-2 触发，z>-1 re-arm；方向取触发前 60m futures momentum。不调阈值。
- 仅先做最困难的 2026 季度抽样（Jan/Apr/Jul，每季度连续3天）。
- BTC：9 事件，1h mean +0.0880%，win 66.7%；ETH：20，+0.0926%，45.0%；SOL：28，**-0.2729%**，28.6%；XRP：34，**-0.1774%**，29.4%。
- 四币合计：91 事件；15m mean -0.0089%；1h mean **-0.1212%**，win 36.3%；4h mean **-0.2040%**；仅 2/4 币 1h mean>0。
- 结论：总 near-book vacuum 并不会稳定沿既有 momentum 扩张，SOL/XRP 反而明显反转；不是第三 Setup。
- 状态：**global liquidity-vacuum family 冻结 / 不入库**。不调 z threshold 或 near/far depth level。

## 2026-09-25 — Binance BookDepth Forward-Side Liquidity Vacuum

- 假设：上涨前若 ask 侧近端深度相对 5% 远端深度异常变薄，价格上方阻力消失，应利于 continuation；下跌时镜像使用 bid 侧。该机制区别于总盘口 vacuum 和 bid/ask imbalance。
- 定义：`ask1%/ask5%`、`bid1%/bid5%` 各自做过去24h z-score；前60m momentum>0 且 ask z<=-2 做 LONG，momentum<0 且 bid z<=-2 做 SHORT；z>-1 re-arm。不调阈值。
- 2026 季度困难样本（Jan/Apr/Jul，每季度连续3天）：
  - BTC：9 事件，1h mean +0.2009%，win 77.8%
  - ETH：21，+0.0862%，38.1%
  - SOL：23，-0.0011%，60.9%
  - XRP：36，-0.0611%，41.7%
- 四币合计：89 事件；15m mean +0.0131%；1h mean **+0.0157%**，win 49.4%；4h mean +0.0870%；仅 2/4 币 1h mean>0。
- 结论：前方单侧流动性真空在 BTC 有局部效果，但总体只有几个 bp 且跨币不一致，不能作为第三 Setup。
- 状态：**forward-side bookDepth vacuum 冻结 / 不扩展旧年份 / 不入库**。bookDepth family 暂停，不继续搜索深度层级、z-score 或 momentum horizon。

## 2026-09-25 — Binance BookDepth Forward-Side Withdrawal Velocity

- 假设：静态深度也许不重要，但若上涨前 ask 1% notional 突然撤单、下跌前 bid 1% notional 突然撤单，前方阻力快速消失可能触发强 continuation。
- 定义：逐分钟 `log(depth_t/depth_{t-1})`，分别对 bid1% / ask1% 做过去24h z-score；前60m momentum>0 且 ask withdrawal z<=-3 做 LONG，momentum<0 且 bid withdrawal z<=-3 做 SHORT；z>-1 re-arm。固定 3σ，不调参数。
- 2026 季度困难样本：
  - BTC：51 事件，1h mean -0.0748%，win 39.2%
  - ETH：53，-0.0487%，37.7%
  - SOL：35，-0.0533%，45.7%
  - XRP：43，+0.1482%，62.8%
- 四币合计：182 事件；15m mean -0.0176%；1h mean **-0.0104%**，win 45.6%；4h mean +0.0662%；仅 1/4 币 1h mean>0。
- 结论：近端撤单并不是稳定的“路径打开”信号，BTC/ETH/SOL 反而偏短时过冲/反转；XRP 单币例外不可迁移。
- 状态：**dynamic bookDepth withdrawal 冻结；整个 bookDepth 研究线暂停**。不尝试反向交易、不调 2σ/4σ、不改深度层级。

## 2026-09-25 — Taker Order-Flow Persistence / Lag-1 Autocorrelation

- 假设：机构拆单会让主动订单流出现持续性；当 taker imbalance 从无自相关跨到正自相关，且近期主动流方向一致，可能形成独立 continuation alpha。
- 初筛使用 1h 数据避免直接加载多年 1m：过去24根 1h 的 signed taker fraction `2*taker_buy_quote/quote_volume-1` 做 lag-1 autocorrelation；当 autocorr 从 <=0 穿到 >0 时触发，方向取最近4h累计 taker flow。零点为自然阈值，不搜索参数。
- 10 老币 2023-01~2026-08：14,365 次事件。
- 总体：4h mean **+0.0037%**，median -0.0393%，win 48.3%；12h mean +0.0419%；24h mean +0.0241%；仅 6/10 币 4h mean>0。
- 年度 4h mean：2023 +0.0147%；2024 -0.0365%；2025 -0.0106%；2026 +0.0672%。
- 结论：order-flow persistence 的短时 edge 近乎为零，且年度方向翻转；没有理由继续下钻到 1m trade-sign autocorrelation。
- 状态：**order-flow persistence/autocorrelation family 冻结 / 不进 Engine / 不入库**。不调 autocorr window、flow horizon 或正相关阈值。

## 2026-09-25 — Bipower-Variation Statistical Jump Detection

- 目的：用 bipower variation 区分连续波动与真正 jump，避免重复旧的简单 4h shock ratio；只测试被统计识别出的 jump 后 continuation / fade。
- 定义：过去24根 1h return 的 bipower variance `(pi/2)*mean(|r_t||r_{t-1}|)` 估计连续波动；当前 1h return 首次达到 |z|>=3 触发，|z|<1 re-arm。3σ 为固定统计阈值，不搜索。
- 10 老币 2023-01~2026-08：8,178 次 jump。
- CONTINUATION：4h mean +0.0712%，median **-0.0895%**，win 46.7%，12h mean +0.0914%，8/10 币 4h mean>0。
- FADE：4h mean -0.0712%，仅 2/10 币为正。
- 年度 continuation 4h mean：2023 **-0.0139%**；2024 +0.0269%；2025 +0.1915%；2026 +0.1012%。
- 结论：统计 jump continuation 在 2024–2026 有一定尾部收益，但 2023 反向、median/胜率均负，仍属于少数大事件驱动；不能形成统一跨 regime 第三 Setup。
- 状态：**bipower/statistical-jump family 冻结 / 不进 Engine / 不入库**。不调 sigma threshold、BV window 或改成 fade。
## 2026-09-25 — COIN-M vs USD-M Perpetual Lead-Lag

- 假设：COIN-M inverse perpetual 与 USD-M linear perpetual 的保证金结构/参与者不同，若 COIN-M 先发生短时位移，USD-M 可能随后跟随。
- 数据：Binance Vision monthly 5m klines；BTC/ETH/BNB/XRP 的 COIN-M *USD_PERP 与对应 USD-M USDT 永续，2023-01~2026-08。全部流式读取，不落库；四币 88×2 月文件均完整。
- 定义：COIN-M 5m return - USD-M 5m return 用过去24h标准化；首次 |z|>=3 且 COIN-M 确实朝 z 方向移动得更远时，按 COIN-M 方向观察 USD-M 后续 15m/1h/4h；|z|<1 re-arm。不调阈值。
- BTC：1,027 事件，1h mean -0.0507%，win 44.4%；ETH：975，+0.0204%；BNB：1,420，+0.0530%；XRP：1,068，-0.0565%。
- 合计：4,490 事件；15m mean +0.0069%；1h mean **-0.0038%**，win 48.5%；4h mean **-0.0606%**，仅 2/4 币 1h mean>0。
- 年度 1h mean：2023 -0.0150%；2024 -0.0231%；2025 +0.0252%；2026 -0.0082%。
- 结论：COIN-M 的相对领先位移不会稳定转化为 USD-M 后续方向收益，市场间价差被快速套利；不是第三 Setup。
- 状态：**COIN-M/USD-M price-discovery lead-lag 冻结 / 不进 Engine / 不入库**。不调 1m/15m、sigma threshold 或改做简单 relative-return threshold。

## 2026-09-25 — Self-Exciting Same-Sign 1m Jump Clustering

- 文献动机：高频研究确认 crypto jump 存在 self-excitation / temporal clustering，负 jump aftershock 尤其持久；因此区别于“单个 shock/jump”，单独测试 jump cluster 是否可交易。
- 定义：1m return 用过去60m bipower variance 标准化；|z|>=4 为 jump；只有第二个**同方向 jump 在 5 分钟内再次出现**才触发 cluster event。5分钟取自 jump-clustering 文献的 5× grid-size 思路，不搜索窗口。
- BTC/ETH/SOL/XRP，2023-01~2026-08：4,422 个 cluster。
- continuation：15m mean -0.0348%；1h mean **-0.0290%**，win 41.5%；4h mean -0.0343%；仅 XRP 1h mean +0.0121%，BTC -0.0248%、ETH -0.0098%、SOL -0.1199%。
- 年度 continuation 1h mean：2023 -0.0360%；2024 -0.0673%；2025 +0.0068%；2026 -0.0179%。
- 反向 fade 数学上约 +0.029%/1h，但只有约 2.9bp，显著低于当前双边 fee+slippage 所需位移，不能视为可交易 alpha。
- 结论：jump self-excitation 在统计上存在，但方向 aftershock 不等于可盈利 continuation；反转幅度又太小。
- 状态：**jump-cluster/Hawkes-like event family 冻结 / 不进 Engine / 不入库**。不调 jump sigma、cluster window 或改做 fade。
## 2026-09-25 — Full-Market Supervised LONG Setup Discovery：1h 标签通过、1m Engine 失败

- 方法区别于此前 88 笔 v54 meta-label：对全部小时候选建立方向标签，老10币 2023–2024训练、2025验证、2026时间OOS、ADA/NEAR/1000PEPE/SUI/ONDO symbol holdout；不使用 symbol/time 特征。
- 11 个原始特征：1/4/12/24h return、range/ATR、body/ATR、CLV、quote-volume ratio、taker fraction、taker delta、12h path efficiency；LONG/SHORT 最初共用对称线性模型。
- 对称模型 OOS 失败，但 LONG-only 固定 L2 logistic 在粗略 1h-close TP/SL 标签上表现稳定：2025 胜率 52.05%、2026 54.74%、symbol holdout 50.59%、全 OOS 51.88%；13/15 币超过近似成本后 break-even。
- 关键复核：将收敛后的 LONG score 固定，不调 p=0.5 阈值；改用真实 1m replay、下一分钟 open、单仓位、4x、TP8/SL6、双边 fee0.0005、5bps slippage、funding。
- 精确 OOS：1,155 笔，PF **0.930**，net -2450.7，仅 5/15 币正。
- 年度：2024 40 笔 PF **0.362**；2025 814 笔 PF **0.903**；2026 301 笔 PF 1.225。
- 主要 holdout：ADA 0.574、NEAR 0.801、1000PEPE 0.894、SUI 0.593、ONDO 0.821；全部 <1。
- 结论：1h close 首触标签与正式 minute-close TP/SL 路径差异足以制造明显假 alpha；大样本和 OOS 不能弥补错误标签定义。
- 状态：**当前 LONG linear score 冻结 / 不入库**。监督式 discovery 若继续，必须从训练阶段就使用 exact 1m TP/SL first-hit 标签；不调模型阈值或加特征挽救本模型。
## 2026-09-25 — Full-Market Supervised Setup Discovery：Exact 1m TP/SL Labels

- 为排除上一轮 1h-close 粗标签制造假 alpha，本轮从训练阶段开始就使用正式路径语义生成标签：信号为已完成1h特征；下一小时第一根1m open +5bps 入场；之后逐分钟 close 按4x ROI 首次触达 TP8/SL6 判定；无人工持仓期限。
- 使用 block min/max 加速 exact first-hit；训练标签若 exit 跨入2025则剔除，2025验证若 exit 跨入2026则剔除，避免 label 穿越时间切分。
- 动态 eligibility 与生产口径一致：老10币从2023；ADA/NEAR 2024-08-15；1000PEPE 2025-05-05；SUI 2025-05-03；ONDO 2026-01-20。
- Exact 数据集共 771,296 个可明确判定方向样本；仍只用既定11个原始价格/成交特征，不使用 symbol/time 特征。
- 对称 L2 Logistic：OOS_ALL AUC 0.5124；p>=0.5 仅195个，胜率 **41.54%**，近似成本后 ROI 明显负。
- LONG-only exact-label：训练选中 19 个，胜率 68.42%，但严格验证立即失效：2025 48个胜率 **35.42%**；2026 21个 **42.86%**；symbol holdout 24个 50.00%；全OOS 45个 **46.67%**。
- LONG-only OOS 仅6/13个有信号的 symbols 达到近似成本后正期望；ZEC 41个仅36.59%，LTC 4个全亏，ETH 1个亏损。
- 与上一轮粗标签形成鲜明对照：粗标签看似 2025/2026/holdout 均 >50%，exact 1m 标签后完全消失，证明标签路径误差是主要假 alpha 来源。
- 结论：当前常规价格/成交特征对固定 4x + TP8/SL6 的 exact path outcome 没有稳定可迁移预测力；不是靠换模型或概率阈值可以合理修复的问题。
- 状态：**整个 supervised setup-discovery 路线冻结**。不换树模型/神经网络，不调 p threshold，不扩展相邻技术特征；除非未来引入本质不同的新数据源，否则不再做监督学习挖掘。
## 2026-09-25 — ID121 / v54 真 Forward OOS：2026-09-01 ~ 09-12

- 当前本地 15 个生产币共同完整数据截至 2026-09-12 15:59 UTC；这段 9 月数据从未参与任何策略/阈值选择，因此作为真正 forward OOS 单独记录。
- 使用完全相同起点与正式 Engine 重放 ID121 / v54，固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps。
- 两套策略结果完全一致：**8 笔，PF 0.405，net -392.3，仅 1/15 币正；组合约 4.8 trades/week**。v54 没有产生任何 bypass 增量交易。
- 8 笔全部为 LONG，没有 SHORT：AVAX 1 SL；BNB 1 SL；UNI 2 SL；ZEC 2 SL + 1 TP；ONDO 1 SL。
- ZEC 3 笔 PF 1.247 / net +52.8；其余发生交易的币均为负。
- paired attribution：COMMON 8 笔 PF0.405；BASE_ONLY=0；V54_ONLY=0。
- 结论：这段新样本的弱势来自 **v33 LONG 主 Setup 本身**，不是 v54 funding-bypass；但只有8笔，统计量太小，不能据此否定 ID121/v54，也不能据此调参。
- 状态：**作为真正 forward warning 保留**。后续持续累积 9 月及之后未见样本；禁止回头针对这8笔优化规则。
## 2026-09-25 — Price–Taker-Flow Absorption Divergence

- 假设：若主动卖盘占优但 1h K 线仍收涨，说明被动买盘吸收卖压，后续可能上涨；主动买盘占优但价格收跌则镜像做空。区别于 v41：这里主动流与价格方向保持相反，不依赖 sweep/ADX/MFI。
- 定义：signed taker fraction = 2*taker_buy_quote/quote_volume-1；flow<0 且 Close>Open 做 LONG，flow>0 且 Close<Open 做 SHORT；只取 fresh divergence，不设幅度阈值。
- 10 老币 2023-01~2026-08：72,980 个事件。
- 总体：1h mean -0.0013%；4h mean **-0.0028%**，median -0.0213%，win 48.9%；12h mean +0.0017%；仅 5/10 币 4h mean>0。
- 年度 4h mean：2023 -0.0246%；2024 +0.0052%；2025 +0.0009%；2026 +0.0116%。
- 结论：价格与主动流背离本身没有稳定 forward edge；“被动吸收”必须依赖更复杂的盘口/结构信息，而这些相邻 bookDepth/v41 路线也已验证失败。
- 状态：**price-flow absorption divergence 冻结 / 不进 Engine / 不入库**。不加 taker threshold、volume threshold 或 ADX/MFI 过滤继续挖。
## 2026-09-25 — ID121 Forward 1/8 坏窗口历史基准

- 目的：判断 2026-09 真 forward 的 8 笔仅1胜是否已经明显偏离 ID121 历史分布，避免因小样本回撤过度反应。
- 使用 ID121 已有成功回测的每币最新 run，自 2024-08-15 后按 entry_time 合并成 459 笔历史交易；只做统计诊断，不改策略。
- 连续8笔滚动窗口共452个：**85个（18.81%）出现 <=1 笔盈利**；其中25个 0胜、60个 1胜。
- 因此当前 forward OOS 的 1胜/8笔，在 ID121 历史中大约每5个八笔窗口就会出现一次，并非罕见尾部异常。
- 90天滚动窗口按7天步进、至少10笔：94个窗口中 **14个（14.9%）PF<1**；PF q10=0.944，median=1.343，q90=2.031，min=0.793，max=2.464。
- 近期历史也曾出现连续弱期，例如 2026-03 多个90d窗口 PF约0.94、2026-05 中旬最低约0.805，随后6月又恢复到 PF>2。
- 结论：2026-09 PF0.405 是必须记录的真 forward warning，但只有8笔，仍落在策略历史常见坏窗口范围内；**当前证据不足以停用 ID121/v54，更不足以据此调参**。
- 状态：继续累积 untouched forward OOS；只有当坏窗口在更大样本（例如几十笔）持续超出历史滚动分布时，才重新评估候选。
## 2026-09-25 — Trade-Size Roundness / Algorithmic Participation（无时钟版本 Early Gate）

- 灵感：2026 `Quarter-Hour Effect` 论文用 trade-size trailing-zero share 识别算法参与，并发现固定 quarter-hour opening 的 order imbalance 可预测4–12h收益；但固定时钟相位违反本项目禁止 `NowTime %` 的约束，因此只测试更底层的 roundness 是否能**脱离时钟独立工作**。
- 严格按论文机械资格思想：qty 按 Binance LOT_SIZE step 缩放；仅 scaled_qty>=10 的交易进入 TZshare(1)，scaled integer 末位为0算至少1个 trailing zero。BTC/ETH step=0.001、SOL=0.01、XRP=0.1。
- 无时钟定义：逐分钟 TZshare(1) 相对过去24h causal baseline 首次 <=-2σ，按同一分钟 taker order imbalance 的符号预测未来 15m/1h/4h；不使用 minute-of-hour / quarter-hour。
- 数据成本评估：2026 单日 individual trades 压缩文件约 BTC 26MB、ETH 46MB、SOL 11MB、XRP 8MB；初筛取 Jan/Apr/Jul 各1事件日 + 前1日 baseline，全部流式、不落盘。
- BTC：42 事件，1h mean **+0.0285%**，win 42.9%，经济幅度远低于成本要求。
- ETH：13 事件，1h mean **-0.1083%**，与 BTC 方向不一致。
- 由于论文没有证明 roundness 脱离时钟相位具有预测力，且两个最核心币初筛已经缺乏一致性/经济幅度，主动停止后续 SOL/XRP 下载，避免为低先验机制消耗数百MB逐笔数据。
- 结论：**无时钟 trade-size-roundness 版本 early rejection**；不扩大年份/币种，也不违反约束去测试 quarter-hour 固定窗口。

## 2026-09-25 — Forward 8-Trade PF Percentile 补充

- 2026-09 真 forward 的 8 笔 PF=0.405；ID121 历史自 2024-08-15 后共有452个连续8笔窗口，其中 **114个（25.22%）PF<=0.405**。
- 因此当前 forward PF 大约落在历史最差四分之一，而非 5%/1% 异常尾部；与此前“<=1胜窗口占18.81%”的结论一致。
- 状态：继续观察，不据此停用或调参。

## 2026-09-25 — Binance BookDepth Forward-Side Liquidity Vacuum

- 假设：上涨前如果 ask 一侧近端/远端深度异常变薄，或下跌前 bid 一侧异常变薄，说明前进方向阻力消失，随后可能继续扩张。
- 定义：分别计算 `bid1%/bid5%` 与 `ask1%/ask5%` 的过去24h z-score；前60m momentum>0 且 ask ratio z<=-2 做多，momentum<0 且 bid ratio z<=-2 做空；z>-1 re-arm。不调阈值。
- 先做最困难的 2026 季度抽样（Jan/Apr/Jul，每季度连续3天）。
- BTC：9 事件，1h mean +0.2009%，win 77.8%；ETH：21，+0.0862%，38.1%；SOL：23，-0.0011%，60.9%；XRP：36，**-0.0611%**，41.7%。
- 四币合计：89 事件；15m mean +0.0131%；1h mean **+0.0157%**，win 49.4%；4h mean +0.0870%；仅 2/4 币 1h mean>0。
- 结论：前方单侧流动性真空不能稳定预测方向扩张，跨币一致性不足，1h edge 过小。
- 状态：**forward-side vacuum 冻结 / 不扩展旧年份 / 不入库**。不调 depth level、momentum horizon 或 z threshold。

## 2026-09-25 — Binance BookDepth 5m One-Sided Withdrawal Shock

- 假设：如果 ask 近端深度在 5 分钟内异常快速撤走，而 bid 没有同步撤走，则做多；bid 异常撤走则镜像做空。该机制关注动态撤单，而非静态 imbalance/concentration。
- 定义：±1% bid/ask notional 的 5m log-change；各自用过去24h change 分布标准化；ask change z<=-3 且 bid z>-1 做多，bid z<=-3 且 ask z>-1 做空；z>-1 re-arm。不调阈值。
- 2026 季度抽样（Jan/Apr/Jul，每季度连续3天）。
- BTC：62 事件，1h mean -0.0163%，win 45.2%；ETH：73，+0.0076%，56.2%；SOL：27，**-0.2060%**，40.7%；XRP：14，+0.2433%，64.3%。
- 四币合计：176 事件；15m mean -0.0143%；1h mean **-0.0148%**，win 50.6%；4h mean +0.0443%；仅 2/4 币 1h mean>0。
- 结论：单侧撤单 shock 跨币方向严重冲突，整体近零/略负；不能作为强位移第三 Setup。
- 状态：**dynamic withdrawal family 冻结 / 不扩展旧年份 / 不入库**。不继续做补单、5m/10m 窗口或 2σ/4σ 邻近变体。

## 2026-09-25 — 1m Trade-Arrival Intensity Shock

- 假设：单位时间成交笔数突然爆发、但价格尚未同步扩张时，可能代表信息到达/交易活跃先行，随后价格沿近期方向加速。
- 数据直接使用本地 Binance 1m replay 的 `trade_count`，无需逐笔下载。
- 固定定义：过去60m trade-count 均值/std；当前 count z>=3，同时当前1m |return| <= 过去60m return 1σ；方向取过去5m price momentum；count z<1 re-arm。不搜索阈值。
- BTC：12,636 事件，1h mean +0.0001%；ETH：12,761，+0.0137%；SOL：13,600，-0.0097%；XRP：12,986，+0.0007%。
- 合计 51,983 事件：15m mean -0.0010%；1h mean **+0.0010%**，win 46.2%；4h mean **-0.0107%**。
- 年度 1h mean：2023 +0.0006%；2024 -0.0001%；2025 +0.0081%；2026 -0.0084%。
- 结论：trade-count burst 是活跃度状态，不提供可交易方向 edge；经济幅度接近零。
- 状态：**trade-arrival/intensity family 冻结 / 不入库**。不调 sigma、60m baseline 或 momentum horizon。

## 2026-09-25 — Spot vs Futures Trade-Count Intensity Divergence

- 假设：Spot 成交到达率相对 Futures 异常领先时，可能代表现货信息流先行，随后 Futures 沿 Spot 短时方向跟随。
- 定义：Spot/Futures 各自用过去60m平均 trade_count 归一化；`log(spot_intensity/futures_intensity)` 用过去24h做 z-score；首次 z>=3 触发，z<1 re-arm；方向取 Spot 过去5m price momentum；信号分钟完成后下一分钟 Futures open 进入。不测试 Futures-leading 反向版本。
- 为节约数据与避免低先验过拟合，只做最困难的 2026 early gate。
- BTC：621 事件，1h mean +0.0168%，win 51.5%；ETH：418，+0.0308%，53.6%；SOL：1,275，-0.0052%，48.6%；XRP：403，-0.0030%，50.9%。
- 四币合计：2,717 事件；15m mean -0.0010%；1h mean **+0.0057%**，win 50.4%；4h mean **+0.0001%**；仅2/4币为正。
- 结论：Spot 活跃度异常领先并不能稳定转化为 Futures 方向收益，经济幅度近零。
- 状态：**spot/futures trade-intensity divergence 冻结 / 不扩展 2023–2025 / 不入库**。不调 sigma、60m baseline 或 spot momentum horizon。
## 2026-09-25 — Token Unlock / Supply-Shock 外生事件可行性审计

- 动机：连续市场/微观结构特征多数只有 bp 级 edge，难以匹配 4x + TP8/SL6；token unlock 属于预先已知的外生供给冲击，公开研究报告 52 个 Binance unlock 中 46/52 在 72h 为负，原始位移显著更大。
- 复现源：HoKwang Kim (2026) 公开 GitHub vibe-investing，File01 为 52 行 author-curated 事件主表；File06/08/10 被作者明确标注为 reconstructed/structural templates，不是原始逐时市场数据。
- 作者 Appendix-E “回测”并非真实路径回测：直接用 72h return 取负并扣 20bps，假设 T-24h short / T+72h cover；zero slippage、无 funding、无 TP/SL first-hit，且代码读取 structural-template File10。因此作者报告的高胜率/Sharpe 不能直接迁移到本项目。
- 生产 eligibility 审计：按本项目既定规则仅保留 days_from_listing >= 730、实际 unlock_pct>0、非上市日事件后，52 个样本仅剩 **9 次 / 8 个独立币**（2024=5，2025=4）。
- 9 次 eligible 事件：7/9 72h 下跌（77.8%），平均 **-9.22%**，median -10.19%；相对 BTC 7/9 跑输，平均相对收益 -11.95%。
- team/investor 子集：8 次，7/8 下跌，平均 -12.14%；unlock>=5% 子集仅 **3 次**，3/3 下跌，平均 -19.25%；5%+ cliff 更只剩2次。
- eligible symbols：IMX、AXS、RON、APT(2次)、FTM、SEI、ID、FET。当前严格 15-symbol production universe 在这些历史事件发生时 **0 个交集**；SUI/ONDO 的强 unlock 发生时尚未满足 >=2 年历史资格。
- 数据精度问题：公开主 CSV 实际只有 unlock_date，没有文档声称的真实 unlock_timestamp_utc；全仓库搜索也未发现精确事件时刻。任意假设 00:00 UTC 会人为决定 T-24h entry 与 minute-level TP/SL path。
- 结论：**token unlock 是目前少数具有足够位移强度、值得未来继续观察的外生事件机制，但现有公开数据不足以在本项目规则下做可信 exact Engine 回测。**
- 状态：**不入库、不改引擎、不用作者回测结果做策略依据；保留为 future event-watch research。** 只有获得可验证的精确 unlock UTC 时间，并积累足够 >=2年 eligible 事件后，再按本项目 1m Engine / TP8-SL6 / fee / slippage / funding 做预注册式 OOS 验证。
## 2026-09-25 — Binance USDⓈ-M Futures Delist Announcement Direct-Short

- 目的：测试一个与连续指标完全不同的外生事件 alpha：Binance 官方宣布某 USDⓈ-M perpetual 即将 delist 后立即做空。
- 事件仅使用 Binance 官方公告，发布时间与 settlement 时间均精确；普通 Spot trading-pair removal 不计入。
- 2024 discovery set 共17个官方 Futures-delist 合约；严格执行既定 production 规则：公告前730天仍能查询到该合约 K 线，最终仅 6 个 eligible：ANT、DGB、AUDIO、WAVES、XEM、OMG。
- 执行：公告后下一根 1m open 开 SHORT；4x；TP8/SL6 按 minute close 首次触发、下一分钟 open 平；双边 fee=0.0005、双边5bps slippage；若不触发则计划在公告所称 settlement 前平仓。
- ANT：SL，-6.71%；DGB：SL，-7.36%；AUDIO：TP，+7.20%；WAVES：1分钟后 SL，-8.57%；XEM：2分钟后 TP，+16.08%；OMG：1分钟后 SL，-6.93%。
- 合计：6 事件，2胜4负；normalized PF 约 **0.79**，normalized net 约 **-6.29%**。
- 关键风险：delist 公告会制造极端双向反身性，不是单向卖压；WAVES/OMG 在公告后立即 short squeeze，XEM 则极速下跌。同一事件类别方向不稳定。
- 结论：**Futures delist announcement 具有大位移，但 direct-short 不是稳定 alpha。** 2024 discovery 已为负，无需再用2025做条件筛选或事后寻找“哪些 delist 才该空”。
- 状态：**direct-delisting-short family 冻结 / 不入库**。不按币种、公告提前天数、流动性或 delist reason 做后验过滤。

## 2026-09-25 — BookDepth Directional Forward-Side Liquidity Vacuum

- 假设：若价格上涨前 ask 侧近端/远端深度异常变薄，或下跌前 bid 侧近端/远端深度异常变薄，说明前进方向阻力消失，可能形成短时扩张。
- 定义：ask 使用 `ask1%/ask5%`，bid 使用 `bid1%/bid5%`；过去24h z-score，z<=-2 触发，z>-1 re-arm；上涨 momentum 配 ask vacuum 做 LONG，下跌 momentum 配 bid vacuum 做 SHORT。固定 2σ，不调参数。
- 仅做 2026 困难样本（Jan/Apr/Jul，每季度连续3天）。
- BTC：9 事件，1h mean +0.2009%，win 77.8%；ETH：21，+0.0862%；SOL：23，-0.0011%；XRP：36，-0.0611%。
- 四币合计：89 事件；15m mean +0.0131%；1h mean **+0.0157%**，win 49.4%；4h mean +0.0870%；仅 2/4 币 1h mean>0。
- 结论：前向一侧 vacuum 对 BTC/ETH 有局部信号，但 SOL/XRP 不迁移，整体 1h edge 过小，不值得扩大历史范围。
- 状态：**directional forward-vacuum family 冻结 / 不进 Engine / 不入库**。不调 depth level 或 z threshold。

## 2026-09-25 — BookDepth Liquidity Resiliency / Persistent Vacuum

- 假设：瞬时深度冲击本身不够，只有近端流动性在冲击后数分钟仍不补回，才代表真实 liquidity withdrawal，可能产生更强位移。
- 定义：`bid1%+ask1%` 相对过去24h首次跌到 z<=-2；5 分钟后若仍 <=-1σ，则判定“未恢复”；方向取冲击前60m momentum，随后观察15m/1h/4h。阈值固定，不搜索。
- 2026 BTC/ETH/SOL/XRP 困难抽样：共 55 个可用事件；部分 SOL/XRP 日期归档 404，未人工补数据。
- 总体：15m mean +0.0563%；1h mean **-0.0007%**，win 50.9%；4h mean **-0.0439%**；仅 2/4 币为正。
- 结论：持续撤单也没有形成稳定方向 edge，而且事件非常稀疏；不足以作为第三 Setup。
- 状态：**liquidity-resiliency / persistent-vacuum family 冻结**。不放宽到 1/3/10 分钟或调整 sigma 阈值。

## 2026-09-25 — BookDepth Curve Convexity：Continuation 初筛

- 新结构：不看总深度，而看 1%→2% 新增 notional 占 1%→5% 总新增 notional 的比例；ask/bid 分开标准化。
- 假设初版：若上涨前 ask 侧近端墙异常薄，或下跌前 bid 侧近端墙异常薄，则顺 momentum continuation。
- 固定定义：`near_share=(depth2%-depth1%)/(depth5%-depth1%)`；过去24h z-score，z<=-2 触发，z>-1 re-arm；方向取前60m momentum。不调参数。
- 2026 困难样本（BTC/ETH/SOL/XRP，Jan/Apr/Jul 季度抽样）共 122 个事件。
- BTC 1h mean -0.1278%；ETH -0.1186%；SOL -0.2949%；XRP -0.1943%；**0/4 币为正**。
- 总体：15m mean -0.0667%；1h mean **-0.2058%**，win 34.4%；4h mean **-0.3692%**。
- 结论：原 continuation 假设被强烈否定；但负向一致性异常强，提示可能存在“近端墙突然变薄后价格反而回撤”的 mean-reversion 机制。
- 状态：**continuation 冻结；仅允许用完全相同事件定义做反向 OOS 验证，不调阈值。**

## 2026-09-25 — BookDepth Curve Convexity：Reversal OOS 复核

- 2026 continuation 初筛呈现 0/4 币为正、1h mean -0.2058%、4h -0.3692%，因此只允许一次固定方向反转的 OOS 检验，不调事件定义。
- OOS 使用 2025，完全相同的 depth-convexity 事件：`(depth2%-depth1%)/(depth5%-depth1%)` 的 24h z<=-2，z>-1 re-arm；唯一变化是交易方向与此前 momentum 相反。
- BTC：3 事件，1h mean +0.1351%；ETH：38，+0.0319%；SOL：24，+0.2378%；XRP：46，-0.0190%。
- 四币合计：111 事件；15m mean +0.0248%；1h mean **+0.0581%**，win 48.6%；4h mean **-0.0709%**；3/4 币 1h 为正。
- 结论：存在短时 mean-reversion 倾向，但 OOS 强度远弱于 2026，且 4h 已反转为负；不匹配 4x/TP8/SL6 所需的持续位移。
- 状态：**bookDepth convexity family 全面冻结 / 不进 Engine / 不入库**。不扩 2024、不调 depth 档位或 sigma 阈值。

## 2026-09-25 — Flow-to-Liquidity Pressure（Taker Flow × BookDepth）

- 动机：单独 taker-flow、OFI、Kyle residual、bookDepth 均已冻结；最后测试它们的结构性交互——同样的主动单，在浅盘口中的冲击应远大于深盘口。
- 定义：每分钟 `signed_taker_quote = 2*taker_buy_quote - quote_volume`；买压除以 ask1% notional，卖压除以 bid1% notional；形成 signed flow-to-depth pressure。过去24h z-score 首次 `|z|>=3` 触发，`|z|<1` re-arm，按 pressure 符号顺势。不调阈值。
- 仅先做 2026 困难样本（BTC/ETH/SOL/XRP，Jan/Apr/Jul 季度抽样）。
- BTC：93 事件，1h mean -0.0158%；ETH：104，-0.1006%；SOL：108，+0.0362%；XRP：97，-0.0644%。
- 四币合计：402 事件；15m mean -0.0044%；1h mean **-0.0355%**，win 45.3%；4h mean **-0.0966%**；仅 1/4 币为正。
- 结论：异常 aggressive-flow / available-depth 比例并不会稳定形成 continuation；负向幅度也不足以支持反手构造新的 mean-reversion Setup。
- 状态：**flow×depth pressure family 冻结 / 不扩旧年份 / 不进 Engine / 不入库**。

## 2026-09-25 — Trade Arrival Intensity Burst

- 动机：项目里的 Qps 实际是 quote volume / 秒，并不是成交笔数；因此单独验证“成交到达强度”是否提供独立微观结构信息。
- 定义：1m `trade_count` 相对过去24h滚动分布，首次 z>=3 触发，z<1 re-arm；按该分钟价格方向做 continuation；不加入 volume/taker 条件，不调阈值。
- 四核心币 2023-01~2026-08：64,070 个事件。
- BTC：15,731，1h mean -0.0078%；ETH：15,793，-0.0015%；BNB：16,640，-0.0010%；XRP：15,906，-0.0141%；**0/4 为正**。
- 总体：15m mean -0.0008%；1h mean **-0.0060%**，win 45.2%；4h mean -0.0092%。
- 年度 1h mean：2023 +0.0064%；2024 -0.0172%；2025 -0.0018%；2026 -0.0113%。
- 结论：成交笔数爆发不提供稳定 continuation edge，且幅度远低于成本；与 quote-volume Qps 不同，但同样不能形成第三 Setup。
- 状态：**trade-arrival burst family 冻结 / 不扩 10 币 / 不进 Engine / 不入库**。

## 2026-09-25 — Buy/Sell VWAP Adverse-Selection Deviation

- 灵感：2026 微观结构研究显示 buy/sell VWAP-to-mid deviation 在多币种上具有稳定 SHAP 形状，并与短时压力/回归有关。
- 本地数据无需逐笔下载：1m chunk 完整保存 total base/quote volume 与 taker-buy base/quote volume，因此可精确恢复：
  - buyVWAP = taker-buy quote / taker-buy base
  - sellVWAP = (total quote - taker-buy quote) / (total base - taker-buy base)
- 定义：`d=log(buyVWAP/sellVWAP)`；过去24h z-score，首次 `|z|>=3` 触发，`|z|<1` re-arm；按短时 microstructure reversion 假设做反向交易。固定参数，不搜索。
- 四核心币 2023-01~2026-08：118,842 个事件。
- BTC：31,796，1h mean -0.0008%；ETH：29,872，-0.0020%；BNB：29,973，-0.0045%；XRP：27,201，-0.0232%；**0/4 为正**。
- 总体：15m mean -0.0026%；1h mean **-0.0072%**，win 49.5%；4h mean -0.0232%。
- 年度 1h mean：2023 -0.0036%；2024 -0.0121%；2025 -0.0037%；2026 -0.0102%。
- 反向 continuation 的数学对应幅度也仅约 +0.0072%/1h，远低于成本，因此无需另跑一次。
- 结论：buy/sell VWAP adverse-selection deviation 在分钟级存在的统计结构不足以形成当前交易框架所需的经济 edge。
- 状态：**VWAP-adverse-selection family 冻结 / 不扩 10 币 / 不进 Engine / 不入库**。

## 2026-09-25 — Universal Fast-Move Linear Discovery（严格 OOS）

- 目的：手工单特征接近饱和后，检查可解释 1h 聚合特征之间是否存在简单线性交互；模型仅用于 alpha discovery，不直接作为策略。
- 标签：每个已完成 1h bar 后，下一分钟 open 作为潜在入场；分别构造 LONG/SHORT 样本，观察接下来24h内是否先触达与标准 Engine 一致的 4x TP8 或 SL6；未在24h解析的样本不参与分类训练。
- 特征不含 symbol：1/4/12/24h directional return、body/ATR、taker imbalance、buy/sell VWAP deviation、CLV、ATR%、QPS ratio、trade-count ratio、avg trade-size ratio、12h path efficiency、12h breakout distance。
- 固定切分：BTC/ETH/BNB/XRP；2023–2024 训练，2025 验证，2026 完全 OOS。L2 Logistic 固定参数；只用训练预测概率第90百分位作为参与阈值，不在验证集调阈值。
- 训练 resolved 103,611 样本，TP-first 基准胜率 41.85%，train p90 threshold=0.4366。
- 2025：AUC **0.5112**；top-decile 7,642 个候选，6,756 resolved，TP-first 胜率 **44.69%**。
- 2026：AUC **0.5143**；3,857 个候选，3,093 resolved，TP-first 胜率 **43.13%**。
- 以 +8/-6 gross ROI 粗算，2025 resolved edge 仅约 +0.26%/笔，2026 约 +0.04%/笔；尚未计入双边 fee 与 5bps slippage，经济上不足。
- 结论：当前 1h price/volume/taker/trade-count/VWAP 聚合特征的简单统一线性组合没有隐藏出足够强的快速 TP8/SL6 alpha。
- 状态：**该线性 discovery 路线冻结 / 不转 DSL / 不入库**。不调 Logistic 正则或概率阈值。

## 2026-09-25 — Trapped Aggressor / VWAP-vs-Close Adverse Selection

- 定义：利用 1m Kline 精确恢复 buyVWAP 与 sellVWAP；若 buyVWAP 显著高于分钟 close，视为 aggressive buyers underwater，做 SHORT；若 sellVWAP 显著低于 close，视为 aggressive sellers underwater，做 LONG。
- buy-gap / sell-gap 各自使用过去24h z-score，首次 z>=3 触发，z<1 re-arm；不调参数。
- 四核心币 2023-01~2026-08：131,452 个事件。
- BTC：36,603，1h mean -0.0075%；ETH：33,674，-0.0109%；BNB：32,503，-0.0002%；XRP：28,672，+0.0055%；仅 1/4 为正。
- 总体：15m mean +0.0003%；1h mean **-0.0037%**，win 49.5%；4h mean -0.0020%。
- 年度 1h mean：2023 -0.0117%；2024 -0.0010%；2025 +0.0042%；2026 -0.0078%。
- 结论：aggressor-underwater 状态在 1m 聚合层面没有稳定后续方向性，经济幅度近零。
- 状态：**trapped-aggressor / VWAP-vs-close family 冻结 / 不扩 10 币 / 不进 Engine / 不入库**。

## 2026-09-25 — Orthogonalized 24h Order Flow（剔除同期 Return）

- 灵感：2026 JFM 研究将 order flow 拆为与同期 return 相关的 transitory component 与正交后的 permanent component；后者在其跨市场/横截面数据中具有更持久预测力。
- 单币实现：过去24h `OF=log(taker_buy_quote/taker_sell_quote)` 与同期24h log return；使用此前90天小时样本滚动回归 `OF = alpha + beta*ret`，当前 residual 的符号预测未来方向。90天 <=100天，不用 Benchmark/MarketCondition，不设幅度阈值。
- 为避免 2022 本地缺口，2023Q1 仅用于90天 warmup，正式从约2023-04开始；10老币全部使用同样规则。
- 10币共 299,280 个小时信号；总体未来24h signed mean **+0.1121%**，win 50.7%，8/10 币全样本 mean>0。
- 逐币：AVAX +0.0522%；BNB -0.0480%；BTC +0.1044%；DOGE +0.1111%；ETH +0.1257%；LTC +0.1055%；SOL +0.0954%；UNI -0.0119%；XRP +0.0371%；ZEC +0.5494%。
- 年度：2023 **-0.0493%**；2024 +0.1113%；2025 +0.3252%；2026 **-0.0253%**。
- 结论：正交化确实比 raw taker-flow 更有结构，但 Binance 单市场版本仍呈明显 regime dependence，2023/2026 反向；不能直接作为统一 Setup。
- 状态：**24h orthogonal-order-flow 冻结**。只允许按原论文明确提出的 weekly horizon 做一次预先指定复核，不搜索其它窗口。

## 2026-09-25 — Orthogonalized Weekly Order Flow（论文指定 horizon 复核）

- 背景：24h Binance Futures residualized order-flow 在 2024/2025 有结构，但 2023/2026 反向；原论文明确报告 weekly horizon 的 permanent order-flow effect 更强，因此仅复核这一预先指定尺度，不做窗口搜索。
- 定义：过去7日 `OF=log(taker_buy_quote/taker_sell_quote)` 与同期7日 return；用此前90天小时样本滚动回归剔除 return component；residual 符号预测未来7日方向。其它参数不变。
- 10老币，每币约29,640小时有效信号。
- 逐币未来7日 signed mean：AVAX -0.6975%；BNB +0.0491%；BTC -0.2787%；DOGE -0.5179%；ETH -0.8108%；LTC -0.2373%；SOL -0.5152%；UNI -0.4379%；XRP -1.1205%；ZEC +1.0364%；仅 **2/10** 为正。
- 总体未来7日 signed mean **-0.3530%**。
- 年度：2023 -0.1602%；2024 -0.0765%；2025 +0.0628%；2026 +0.0798%。
- 结论：论文的 world-order-flow weekly permanent component 无法直接迁移成 Binance 单一永续 order flow；跨币一致性严重失败。
- 状态：**weekly orthogonal-order-flow 冻结**。不再搜索 2/3/5/10 日 horizon。

## 2026-09-25 — Combined Spot + Perpetual Orthogonalized Order Flow

- 动机：单 Binance Futures 的 residualized order flow 在 2024/2025 有 edge，但 2023/2026 反向；尝试用同币 Spot + USD-M Perpetual 联合主动买卖流，更接近“跨市场真实需求”而非单纯杠杆流。
- 定义：每小时合并 Spot 与 Perp 的 taker-buy quote / taker-sell quote；过去24h形成联合 `OF=log(buy/sell)`，再用此前90天滚动回归剔除同期24h futures return；residual 符号预测未来方向。没有额外阈值。
- 四核心币 2023-01~2026-08：
  - BTC：11,720 信号，24h mean +0.0139%
  - ETH：19,177，+0.0728%
  - BNB：19,322，+0.0683%
  - XRP：7,604，+0.4220%
  - 全样本 **4/4 为正**。
- 总体：4h mean +0.0167%；12h +0.0438%；24h **+0.1053%**，win 50.0%。
- 年度：2023 **-0.0383%**；2024 +0.2327%；2025 +0.1455%；2026 **-0.1928%**。
- 结论：Spot+Perp 联合 flow 改善了 cross-symbol 一致性，但仍然只在 2024/2025 有效，2026 明显反向；不能作为跨 regime 第三 Setup。
- 状态：**Binance-internal combined-order-flow 冻结**。不调 24h/90d 窗口；order-flow 路线若继续，只考虑本质不同的跨交易所“world flow”数据源。

## 2026-09-26 — Cross-Exchange World Order Flow（Binance + Bybit）稀疏可行性验证

- 动机：单 Binance Futures、Spot+Perp 联合 order flow 都呈现明显 regime dependence；原论文强调的是真正跨市场 world-order-flow，因此用 Bybit 作为第二交易所做最后一次机制复核。
- 数据可用性：Tardis 公共历史可直接读取 Bybit / Binance Futures 单日 trades；为控制数据量，本次脚本每月仅抽取 1 日 Bybit trades，与本地 Binance Futures 同日 24h flow 合并，再用此前3个月稀疏样本对同期 return 做 residualization。
- 计划样本：BTC/ETH，2023-01~2026-08，共88个“每月1日”观测；Bybit 下载受 60s timeout 影响，仅 **39/88** 成功。
- residualization 后最终仅 **9 个有效预测样本**：
  - Binance-only：mean -0.2571%，win 33.3%
  - World(Binance+Bybit)：mean **-0.2287%**，win 44.4%
  - 2023 World mean **-1.2329%**
  - 2024 仅2个样本，World mean +3.2861%
  - BTC World mean -0.3416%；ETH -0.1722%
- 结论：当前公开下载链路下样本稀疏且缺失严重，world-flow 没有形成可用正 edge；结果不足以支持继续下载全量逐日 Bybit trades。
- 状态：**cross-exchange world-order-flow 暂冻结**。除非未来已有稳定、低成本、连续的第二交易所历史 flow 数据源，否则不继续该路线。

## 2026-09-26 — Binance Spot Whole-Token Delist Announcement Direct-Short

- 目的：测试一个与 Futures delist 不同的外生事件：Binance 宣布整币从 Spot 下架后，只要 USD-M perpetual 仍在交易，则公告后立即做空。
- 事件源严格只取 “Binance Will Delist ...” 整币公告，排除仅移除某个 Spot trading pair 的普通通知。
- 2024 discovery 共33个公告 token；用 Binance Vision HEAD 做 production eligibility 审计：公告月仍有 USD-M 1m 历史，且公告前24个月同月已有期货历史。最终 9 个 eligible：ANT、XMR、OMG、WAVES、XEM、REEF、UNFI、REN、BLZ。
- 执行固定：公告后下一根1m open开 SHORT；4x；TP8/SL6 按 minute-close 触发、下一分钟 open 平；双边 fee=0.0005、双边5bps slippage；最长72h，未触发则退出。Funding 月度归档同时检查；实际9笔均在首次 funding cashflow 前完成，funding=0。
- 逐笔：ANT SL -9.95%；XMR TP +7.51%；OMG TP +13.36%；WAVES SL -8.58%；XEM TP +8.02%；REEF SL -8.22%；UNFI SL -8.54%；REN SL **-18.54%**；BLZ SL -7.06%。
- 合计：9 事件，3胜6负；PF **0.474**；normalized net **-32.00%**，avg -3.56%。
- 结论：Spot 整币 delist 与 Futures delist 一样具有大位移，但方向高度反身；直接 short 会频繁遭遇 announcement squeeze，不能作为稳定第三 Setup。
- 状态：**spot-whole-token-delist direct-short family 冻结 / 不用2025做条件筛选 / 不入库**。不按公告提前天数、币种、流动性或 delist reason 做后验过滤。

## 2026-09-26 — Binance Monitoring Tag Addition Direct-Short：2024 Discovery

- 动机：Spot/Futures 最终 delist direct-short 都失败，可能因为最终退市公告产生强烈双向 squeeze；Monitoring Tag 属于更早期的“风险评级恶化”，理论上卖压更渐进、反身性更低。
- 事件源：仅取 Binance 官方“Extend the Monitoring Tag”新增 token，排除移除 Monitoring/Seed Tag 的币。
- 2024 共4批官方公告（Jan/Apr/Jul/Oct）；用 Binance Vision 审计 production eligibility：公告月仍有 USD-M 1m 历史，且公告前24个月同月已有期货历史。最终 9 个 eligible：ANT、REEF、XMR、ZEC、ZEN、UNFI、WAVES、BAL、BLZ。
- 固定执行：公告后下一根1m open做 SHORT；4x；TP8/SL6按 minute-close 首次触发、下一分钟open平；双边fee=0.0005、双边5bps slippage；最长72h。Funding归档同时检查，9笔均在首次实际 funding cashflow 前结束。
- 逐笔：ANT SL -7.20%；REEF TP +8.22%；XMR SL -6.88%；ZEC TP +8.04%；ZEN TP +7.64%；UNFI SL -7.29%；WAVES TP +8.30%；BAL TP +7.58%；BLZ SL -7.57%。
- 合计：9事件，5胜4负；PF **1.374**；normalized net **+10.84%**；avg **+1.20%/事件**。
- 所有9笔均在72h内先触 TP/SL，无 TIME/EOF 退出。
- 结论：这是当前外生事件路线中第一条在 discovery set 同时满足“大位移 + 固定 TP8/SL6 后 PF>1”的机制，值得做严格年份 OOS。
- 状态：**进入 2025 OOS；参数与方向全部冻结。** 2025 不允许根据结果修改 eligibility、72h、TP/SL 或 short 方向。

## 2026-09-26 — Binance Monitoring Tag Addition Direct-Short：2025 OOS

- 2024 discovery 后参数与方向全部冻结；2025 OOS 不允许修改 eligibility、72h、TP8/SL6、short 方向或成本参数。
- 2025 已确认 Monitoring Tag 新增批次经同一 Binance Vision production eligibility 审计后，最终 9 个 eligible：FLM、NKN、PERP、LEVER、MDT、BAKE、IDEX、DENT、SXP。
- 固定执行与 2024 完全相同：公告后下一根1m open SHORT；4x；TP8/SL6 minute-close trigger + next-minute-open exit；双边 fee=0.0005、双边5bps slippage；最长72h。
- 逐笔：FLM TP +36.86%；NKN TP +10.24%；PERP TP +16.69%；LEVER SL **-31.26%**；MDT TIME -0.80%；BAKE SL **-40.08%**；IDEX TIME -0.80%；DENT TP +8.16%；SXP TP +8.75%。
- 2025 OOS：9事件，5胜4负；PF **1.107**；normalized net **+7.77%**；avg +0.86%/事件。
- 2024+2025 合并：18事件，10胜8负；PF约 **1.183**；normalized net约 **+18.61%**；avg约 +1.03%/事件。
- 风险：事件型 gap/squeeze 会让“SL6”在下一分钟 open 实际成交时严重越过目标，LEVER/BAKE 分别出现约 -31%/-40% 单笔损失；尾部风险远高于普通连续策略。
- 结论：2025 OOS 未否定 Monitoring Tag short 的正 edge，但 PF 已明显低于 discovery，且极端 gap risk 很大；暂不足以替换/并列 ID121/v54。
- 状态：**进入 2026 第二层 OOS，参数继续冻结。** 只有 2026 仍保持正 edge，才考虑升级为第三候选；否则保留为事件研究路线。

## 2026-09-26 — Binance Monitoring Tag Addition Direct-Short：2026 Second OOS & Final

- 2024 discovery PF1.374、2025 OOS PF1.107 后，规则继续完全冻结，并对 2026-01~2026-09 官方 Monitoring Tag 新增事件做第二层 OOS。
- 2026 共8批已确认公告，经相同 Binance Vision eligibility 审计后得到17个 eligible：FLOW、ATA、GTC、NTRN、PHB、RDNT、NFP、HFT、STORJ、TLM、VANRY、LSK、STX、GLMR、ICX、MOVR、RARE。
- 固定执行不变：公告后下一根1m open SHORT；4x；TP8/SL6 minute-close trigger + next-minute-open exit；双边 fee0.0005、双边5bps slippage；最长72h。
- 2026 OOS：17事件，8胜9负；PF **0.910**；normalized net **-8.13%**；avg -0.48%/事件。
- 主要尾部：VANRY -23.66%、NFP -13.21%、HFT -12.89%；正向大单包括 PHB +14.95%、ATA +13.38%、STX +10.45%。
- 三年合并 2024+2025+2026：35事件，18胜17负；PF约 **1.054**；normalized net约 +10.49%；avg约 +0.30%/事件。
- 结论：2024/2025 的正 edge 没有在更大 2026 OOS 延续；长期组合接近盈亏平衡，且 announcement gap tail 极大。不能作为第三正式候选。
- 状态：**Monitoring Tag direct-short family 冻结 / 不入库**。不按事件原因、年份、币种或公告后首分钟走势做事后过滤；ID121/v54 仍是仅有正式候选。
## 2026-09-26 — Binance Margin Trading-Pair Removal Direct-Short：2024 Discovery

- 动机：测试比 Spot/Futures 最终 delist 更早的外生风险事件——Binance 单独移除 Margin trading pair 时，只要同 token 的 USD-M perpetual 仍可交易且历史 >=2 年，则公告后直接 SHORT。
- 事件源：Binance 官方 CMS `firstCatalogId=161` 的 2024 “Notice of Removal of Margin Trading Pairs” 独立公告；共 11 批、57 个 token-level 事件。排除整币 Spot delist 中附带的 Margin 移除，避免与已冻结 Spot-delist family 重叠。
- Production eligibility 固定为：公告月 USD-M 1m Vision 月文件存在、24 个月前同月文件存在、公告日仍有 1m 可交易数据。最终 18 个 eligible：SAND、ZEN、ALICE、BAL、SXP、LINA、DGB、TLM、APE、BNB、ETH、DAR、CHZ、QTUM、C98、REN、BAND、GTC。
- 固定执行：官方 CMS publishDate 后下一根 1m open SHORT；4x；TP8/SL6 minute-close trigger + next-minute-open exit；双边 fee=0.0005、双边 5bps slippage；最长 72h；single-position；funding 按正式 backtest engine 语义计入。
- Funding 校正：Binance Vision fundingRate CSV 只有 `calc_time/funding_interval_hours/last_funding_rate` 三列；按 `service/backtest/engine.go` 在缺少 MarkPrice 时使用对应 1m bar.Close 作为 fallback mark。修正后 DGB TIME 从 -0.80% 变为 -0.44%，整体结论不变。
- 结果：18 事件，6 胜 12 负；6 TP、11 SL、1 TIME；PF **0.603**；normalized net **-31.32%**；avg **-1.74%/事件**。
- 主要结果：SXP +7.37%、LINA +7.63%、APE +7.65%、DAR +8.81%、C98 +7.74%、REN +8.35%；TLM -8.48%、BAND -8.34%、QTUM -7.80%，其余多数 SL 约 -6.5%~-6.8%。
- 结论：Margin pair removal 并没有形成稳定的 announcement-short edge；即使发生在最终退市之前，公告后的反向/无效反应仍占主导，discovery set 明显失败。
- 状态：**Margin trading-pair removal direct-short family 冻结 / 不进入 2025 OOS / 不入库**。不按币种、pair quote、公告月份、首分钟反应或后续是否整币退市做事后过滤；ID121/v54 仍是正式候选。

## 2026-09-26 — Binance Monitoring Tag Removal Direct-Long：2024 Feasibility

- 动机：与已冻结的 Monitoring Tag Addition → SHORT 相反，测试 Binance 解除 Monitoring Tag 是否代表风险评级改善并产生公告后 LONG edge。
- 2024 官方季度调整中，只有 2024-07-01 公告明确移除 Monitoring Tag，涉及 MLN、ZEN；1 月/10 月是 Seed Tag removal，4 月无 Monitoring removal，因此不混入本 family。
- 按既定 production eligibility（公告月 USD-M 1m 存在、24 个月前同月已存在、公告后仍可交易）审计：MLN 不满足 USD-M 历史门槛，只有 ZEN eligible。
- 结论：2024 discovery 最终仅 **1 个 eligible 事件**，样本量不足以评估 PF、期望或跨币泛化；不根据 ZEN 单笔结果决定是否扩展年份。
- 状态：**Monitoring Tag Removal direct-long family 因 discovery 样本不足冻结 / 不进入 2025 / 不入库**。Seed Tag removal 保持为不同事件类型，不为增加样本而事后合并。

## 2026-09-26 — Event Replay Funding Parser Audit

- 审计发现旧 `/tmp/spot_delist_short_2024.py` / Monitoring Tag 回放 helper 假定 Vision monthly fundingRate CSV 含第 4 列 MarkPrice；实际归档仅有 `calc_time,funding_interval_hours,last_funding_rate` 三列，导致旧 helper 静默跳过 funding rows。
- 正式 backtest engine 的语义已核对：Funding 记录缺少 MarkPrice 时使用当前 1m `bar.Close` 作为 fallback mark；LONG 支付正 funding，SHORT 收取正 funding。
- 本轮 Margin Removal discovery 已按该正式语义修正并重跑，PF 从 0.6004 变为 0.6029，net 从 -31.70% 变为 -31.32%，结论不变。
- Spot Whole-Token Delist 已用修正后的 funding parser 重跑：9 笔 funding 仍全部为 0，3 胜 6 负，PF **0.474476**，normalized net **-31.9982%**，与原结果一致，因此该 family 的冻结结论不受 parser bug 影响。Monitoring Tag 历史结果本轮仍未重跑，其旧记录中的 fee/slippage 有效，但涉及“funding 已计入”的精确 PF/net 数字继续视为待校正历史值。

## 2026-09-26 — Binance Seed Tag Removal Direct-Long：2024 Feasibility

- 动机：Seed Tag removal 表示 Binance 认为原先“新项目/高风险高波动”标签已不再需要，预先定义为潜在正面评级事件，测试公告后直接 LONG；与 Monitoring Tag removal 保持独立 family。
- 2024 官方事件只有两批：2024-01-04 移除 GMX、SUSHI 的 Seed Tag；2024-10-03 移除 PENDLE、SEI 的 Seed Tag。4 月与 7 月没有 Seed Tag removal，不为扩大样本混入其他 Tag 事件。
- 按既定 production eligibility 用 Binance Vision monthly USD-M 1m 做 HEAD 审计：事件月文件四币均存在；24 个月前同月仅 SUSHIUSDT 存在。GMX、PENDLE、SEI 均不满足事件时合约历史 >=2 年。
- 最终 discovery 只有 **1 个 eligible 事件：SUSHI**。样本不足以评估 PF、期望、跨币泛化，也不应根据单笔结果决定是否扩大年份。
- 状态：**Seed Tag Removal direct-long family 因 2024 discovery 样本不足冻结 / 不进入 2025 OOS / 不入库**。不放宽 >=2 年门槛，也不与 Monitoring Tag removal 合并。

## 2026-09-26 — Binance Spot Trading-Pair Removal Direct-Short：2024 Discovery

- 动机：测试 Binance 仅移除某个 Spot trading pair 时，base token 的 USD-M perpetual 是否存在公告后直接 SHORT 的事件型 edge；与整币 Spot delist、Margin pair removal 保持为独立 family。
- 事件源固定为 Binance 官方 2024 `Notice of Removal of Spot Trading Pairs` 公告，共 **42 批**；按每批公告提取被移除 pair 的 base asset，并仅在同一公告内去重同 base，得到 **189 个 token-level 事件 / 154 个 unique base**。不按 quote asset、币种、月份、公告后首分钟反应或流动性做筛选。
- Production eligibility 固定为：公告月 Binance Vision USD-M perpetual 1m 月文件存在，且 24 个月前同月文件也存在；公告日仍必须有 1m 可交易数据。182 个 unique base+event-month key 全部完成双 HEAD 审计，最终 **60 个 eligible 事件 / 51 个币种**，且公告日 1m 数据 **60/60 可用**。
- 固定执行：官方 publish time 后下一根 1m open 做 SHORT；4x；TP8/SL6 按 minute-close 首次触发、下一分钟 open 平；双边 fee=0.0005、双边 5bps slippage；最长 72h；single-position；funding 按正式 backtest engine 语义计入，Vision fundingRate 缺少 MarkPrice 时使用对应 1m bar.Close fallback。
- 2024 discovery：**60 事件，25 胜 35 负；25 TP、34 SL、1 TIME；PF 0.787；normalized net -52.14%；avg -0.87%/事件；median -6.78%/事件。** Funding 合计约 +1.77%，仍不足以抵消价格亏损与约 24.01% 的双边手续费成本。
- 月度表现同样不稳：12 个月中 **8 个月净值为负**；4 月、9 月、10 月、11 月为正，其中 10 月局部较强，但这是 discovery 内部结果，不据此筛月份或事件。
- 结论：Spot 单 trading-pair removal 并没有形成稳定的 announcement-short edge；样本量已足够且整体 PF<1、净期望为负，局部月份正收益不能支持事后条件化。
- 状态：**Spot trading-pair removal direct-short family 冻结 / 不进入 2025 OOS / 不入库**。不调 TP/SL、72h、eligibility，不按 pair quote、币种、月份、首分钟反应或后续是否整币退市做后验过滤；ID121/v54 仍为正式候选。

## 2026-09-26 — Binance Spot New Trading-Pair Announcement Direct-Long：2024 Discovery

- 假设：Binance 为已经存在的币新增 Spot trading pair，会扩大可交易入口与现货流动性，公告后的短时新增需求可能形成 LONG edge。该方向、事件定义与执行参数在查看收益前固定，不根据结果筛 quote asset、币种或月份。
- 事件源：Binance 官方 CMS `firstCatalogId=48`，标题以 “Notice on New Trading Pairs & Trading Bots Services on Binance Spot” 开头的 2024 公告。共 50 篇官方公告；只提取正文中 “Binance will open trading ...” 段落里的新 Spot pairs；同一公告同 base 去重后得到 177 个 token-level events / 124 个 unique bases。
- eligibility：base 必须存在对应 `BASEUSDT` USD-M perpetual；公告时仍有 1m 可交易数据；并要求公告时间向前精确 2 年时已经存在 1m 历史。结果 69 个 eligible events / 48 个币；106 个因没有 24 个月前月文件排除，2 个虽有该月文件但精确历史不足 2 年排除。eligibility 完全不使用收益结果。
- 执行：公告 `publishDate` 后下一根 1m open 做 LONG；4x；TP=8、SL=6；minute-close 首次达到 ROI gate，下一分钟 open 平仓；max hold=72h；fee=0.0005 双边；slippage=5bps 双边；funding 按正式 Engine 语义，Vision fundingRate 无 MarkPrice 时用当前 1m bar.Close；LONG 支付正 funding。ROI 公式已与 `utils.FuturesLeveragedROI` / backtest `grossROI` 核对一致。按 symbol 单仓位执行，本样本没有 overlap skip。
- 2024 discovery：69 trades，25 胜 / 44 负；25 TP / 41 SL / 3 TIME；PF **0.687649**；normalized net **-91.1872%**；avg **-1.3216%/event**；median **-6.5512%**；min **-8.3301%**；max **+9.1858%**。funding 合计 **-1.9974%**，fees 合计 **27.5692%**。
- 月度仅 4、7、8 月为正，其余有样本月份为负；8 月 4/4 胜只是 discovery 内描述性结果，禁止据此后验筛月份。部分 repeated symbols 为正也不得据此筛币。
- 结论：新增 Spot trading-pair 公告本身没有产生可用的公告后 LONG alpha，且 PF 明显低于 1。**Spot new-trading-pair direct-long family 冻结 / 不进入 2025 OOS / 不入库**。不反向做 SHORT，不按 USDC/FDUSD/TRY/EUR/BRL 等 quote、月份、币种、公告到开盘间隔、首分钟反应或流动性做后验优化；ID121/v54 仍为正式候选。

## 2026-09-26 — Binance Margin Additions Direct-Long：2024 Discovery

- 假设：Binance 独立 Margin 公告新增 borrowable asset 或 Cross/Isolated Margin trading pair，会扩大杠杆交易入口并带来短时新增需求，因此预注册公告后直接 LONG。方向、eligibility、TP/SL、持仓上限与成本参数均在查看收益前固定。
- 事件源：Binance 官方 CMS `catalogId=48` 中标题以 `Binance Margin Adds...` 开头的 2024 独立 Margin 新增公告；共 **21 篇**。排除 Spot/Futures/多产品新币上市公告，避免把新币 listing 效应混入 Margin 机制。正文解析保留表格 cell/row 边界，只提取公告明确列出的新增 Margin pair，并补充明确 new borrowable asset；同一公告同 base 去重后得到 **168 个 token-level events / 121 个 unique bases**。审计中修复了旧解析把相邻表格单元格拼成 `PENDLE/USDCALT` 一类伪 pair、同时漏掉边界币种的问题。
- Production eligibility：事件时必须存在对应 `BASEUSDT` USD-M perpetual 1m 数据，且公告时间向前精确两年已有 1m 历史，并且公告后存在可交易的下一根 1m bar。网络/TLS 异常只重试，不作为不合格理由。最终 **52 个 eligible events / 34 个币**；其余 **116** 个均因两年前对应月份没有 USD-M 1m 历史而排除。eligibility 不使用收益结果。
- 固定执行：官方 `publishDate` 后**下一根完整 1m bar 的 open**做 LONG；入场目标为 `floor(publish_ms / 60000) * 60000 + 60000`。抽查 `2024-01-03 05:15:03` 已确认进入 `05:16:00`，不会再错误跳到 `05:17:00`。4x；TP8/SL6 按 minute-close 首次达到 ROI gate、下一分钟 open 平仓；max hold=72h；双边 fee=0.0005；双边 slippage=5bps；funding 按正式 backtest Engine 语义计入，Vision fundingRate 缺 MarkPrice 时用当前 1m bar.Close；LONG 支付正 funding；single-position。52 笔没有 overlap skip。
- 2024 discovery：**52 trades，15 胜 / 37 负；15 TP / 36 SL / 1 TIME；PF 0.528482；normalized net -116.4420%；avg -2.2393%/event；median -6.6524%；min -7.6191%；max +18.1156%。** Funding 合计 **-2.1102%**，fees 合计 **20.7532%**。
- 月度也没有稳定性：有样本的 9 个月中仅 6 月、8 月净值为正；1、2、3、4、7、9、11 月均为负。月度与 repeated-symbol 结果只作描述，不据此后验筛月份、quote asset、币种或事件子类型。
- 结论：Margin 新增交易入口/可借资产公告没有形成可用的公告后 LONG alpha；样本量足够且 PF 明显低于 1、净期望为负。
- 状态：**Margin additions direct-long family 冻结 / 不进入 2025 OOS / 不入库**。不反向改做 SHORT，不按 USDC/FDUSD/USDT、币种、月份、borrowable-vs-pair、首分钟反应或流动性做后验优化；ID121/v54 仍为正式候选。
- 审计归档：`strategy_templates/research/margin-additions-direct-long/2024-discovery/`；包含完整事件集、eligibility、逐笔结果、固定参数、protocol、provenance、replay 脚本与 SHA-256 manifest。旧的 149 events / 47 eligible 运行日志仅作为 parser 修复历史保留在 `legacy/`，不属于最终结果证据。

## 2026-09-26 — Open Interest / Price State Transition：2026 Early Gate → 2025 OOS

- 新机制：不再使用 Top Trader positioning ratio，而直接研究 Binance USD-M metrics 的 sum_open_interest 与价格状态；价格用同期 sum_open_interest_value / sum_open_interest 作为 mark proxy。
- 预注册定义不调参：4h OI change 从 <=0 穿到 >0 时，按过去4h价格方向做 continuation；4h OI change 从 >=0 穿到 <0 时，逆过去4h价格方向做 liquidation-exhaustion reversal。只用自然零点，不设 OI/return 幅度阈值。
- 2026 early gate：Expansion-continuation 132事件，4h signed mean **-0.0653%**，仅2/4币为正，失败。Contraction-reversal 129事件：1h **+0.0359%**、4h **+0.1765%**、12h **+0.5865%**，3/4币4h为正，达到事前门槛，因此保持完全相同定义进入 2025 全年历史 OOS。
- 2025 OOS 共 **7,989** 事件：1h **-0.0064%**、4h **-0.0267%**、12h **-0.0094%**；BTC/ETH/BNB/XRP 的4h mean分别约 **-0.0009% / -0.0551% / -0.0265% / -0.0211%**，**0/4币为正**。
- 结论：2026 小样本的 OI contraction reversal 是明显 regime/sample 假象，独立 2025 大样本完全不复现；没有理由进入 1m TP8/SL6 Engine。
- 状态：**整个 OI-price-state transition family 冻结 / 不进 Engine / 不入库**。不搜索 2h/6h/12h lookback、不加 OI magnitude/price-return threshold、不改变 zero-cross 定义、不反向挽救，也不为此新增生产 OI indicator。
- 数据注意：Binance Vision metrics 可重建，但历史 observation 的 point-in-time publication latency 未独立验证；由于 OOS 已失败，无需继续为该路线做 availability 审计。
- 审计归档：strategy_templates/research/open-interest-price-state/2026-early-gate/。

## 2026-09-26 — Extreme Funding Settlement Reversal：2023–2024 Discovery → 2025–2026 OOS

- 假设：极端 funding settlement 代表永续合约一侧拥挤；结算后拥挤侧应出现再平衡。规则完全独立于 v33/v54：过去30次已完成 funding 计算均值/std，首次 `|z|>=2` 触发；正 funding 做 SHORT、负 funding 做 LONG；`|z|<1` 才 re-arm。
- 为避免 contemporaneous look-ahead，第一阶段采用保守执行：funding timestamp 后**下一完整 1h bar open** 入场，只观察1h/4h/12h signed forward return。固定10个老币，不使用 symbol-specific 条件。
- 2023–2024 discovery：**1,173**事件；1h mean **+0.0677%**，4h **+0.0553%**，12h **+0.2748%**；7/10币4h为正。但年度已经翻转：2023 4h **+0.1721%**，2024 **-0.0613%**。
- 2025–2026 OOS：**969**事件；1h mean **+0.0030%**，4h **-0.0081%**，12h **-0.1255%**；仅5/10币4h为正。2025 4h **+0.0894%**，2026 **-0.1593%**。
- 结论：discovery 幅度本身就远低于当前双边 fee+slippage 所需经济位移，而且跨年方向不稳定；OOS 后 4h/12h 直接转负，不值得进入精确 1m TP8/SL6 Engine。
- 状态：**整个 extreme-funding-settlement reversal family 冻结 / 不进 Engine / 不入库**。不调30次窗口、2σ、1σ re-arm、入场延迟，也不反向做 continuation 挽救。
- 审计归档：`strategy_templates/research/funding-settlement-shock-reversal/2023-2026/`。

## 2026-09-26 — Funding Interval Compression：数据可行性审计

- 目标：研究 Binance funding settlement frequency 收紧是否代表极端拥挤/去杠杆压力；原计划识别 8h/4h -> 1h，再按触发前 funding 符号反向交易。
- 本地 `market_funding_rates` 在 2025-05-02 后的间隔分布：8h 共 **26,936** 条 / 23币；4h 共 **10,997** 条 / 5币；**1h 为 0 条**。因此当前本地归档无法重建 Binance 2025-05 后自动 1h funding transition。
- 退一步只检查 8h -> 4h：严格要求切换前连续两个约8h间隔，整个本地历史仅找到 **1 个事件**（SOLUSDT，2022-11-09 20:00 UTC），且按本地历史事件时合约年龄不足2年，production-eligible = **0**。
- 结论：该路线不是“alpha 回测失败”，而是当前数据与生产资格下 **不可验证 / 样本不足**。
- 状态：**冻结可行性研究**。不放宽 >=2年门槛、不降低流动性条件、不把无关 funding-frequency 事件混入凑样本；只有未来获得完整 point-in-time 1h funding 历史后再重启。
- 审计归档：`strategy_templates/research/funding-interval-compression/2025-2026-feasibility/`。

## 2026-09-26 — Premium Index Extreme Reversal：2024 Discovery → 2025/2026 OOS

- 假设：Binance Premium Index 直接反映永续 impact bid/ask 相对现货指数的压力；过去24h（288根5m）z-score 首次 `|z|>=3` 时，正 premium 做 SHORT、负 premium 做 LONG，`|z|<1` re-arm。
- 执行诊断固定为 5m 信号完成后下一完整 1h open 入场；10老币，同参数；2024 discovery、2025 OOS1、2026 OOS2；不使用 symbol-specific 条件。
- 2024 discovery：**8,434**事件；1h mean **+0.0031%**，4h **+0.0057%**，12h **+0.0756%**；仅 **2/10** 币4h为正。
- 2025 OOS1：**9,603**事件；4h mean **+0.0063%**，12h **+0.0898%**；6/10币4h为正，经济幅度仍接近0。
- 2026 OOS2：**5,894**事件；1h **+0.0375%**，4h **+0.0806%**，12h **+0.0489%**；9/10币4h为正。虽后期变强，但这是后出现的 regime，且4h幅度仍低于当前双边 fee+slippage 所需位移，不能反向据此调早期规则。
- 结论：discovery 本身没有可交易 edge，2025 也没有确认；2026 局部改善不能把弱机制升级为策略。
- 状态：**Premium Index extreme-reversal family 冻结 / 不进 1m Engine / 不入库**。不搜索 2σ/4σ、12h/48h baseline、re-arm、symbol filter 或 continuation。
- 审计归档：`strategy_templates/research/premium-index-extreme-reversal/2024-2026/`。

## 2026-09-26 — Upbit New-Market Announcement Direct-LONG：2024 Discovery

- 假设：Upbit 新增交易支持/新增市场公告会带来韩国现货新增需求，因此公告后立即 LONG 已在 Binance USD-M 交易至少2年的同币合约。
- 官方事件宇宙：2024 共 **44 篇** Upbit Trade-category 新交易支持/新增市场公告；完整 ticker 解析得到 **66 token-events / 61 unique tokens**。
- Production eligibility 使用 Binance Vision：事件时 USD-M 历史 >=730天、公告前24h QuoteVolume >=500万 USDT、公告后仍有可交易数据。最终 **12个 eligible / 12币**：JASMY、ARPA、EGLD、FIL、NEAR、XLM、UNI、INJ、GAL、ENS、NEO、SOL。
- 固定执行：Upbit `first_listed_at` 后下一根1m open LONG；4x；TP8/SL6 minute-close trigger + next-minute-open exit；双边 fee=0.0005、双边5bps slippage；funding；最长72h。
- 2024 exact 1m discovery：**12笔，1 TP / 11 SL；PF 0.0961；normalized net -92.89%；avg -7.74%/事件**。仅 ENS +9.88%；INJ 约 -19.99%，公告型 gap/slippage 风险明显。
- 结论：Upbit listing direct-LONG 在 discovery 被强烈否定。
- 状态：**整条 Upbit new-market direct-LONG family 冻结 / 不进入2025 OOS / 不入库**。尤其不因 11/12 亏损而事后反向改做 SHORT，也不按 KRW/USDT市场、币种、公告时刻、首分钟走势或后续更新筛选。
- 审计归档：`strategy_templates/research/upbit-new-market-direct-long/2024-discovery/`。
