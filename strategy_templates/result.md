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


## 2026-09-26 — Upbit Risk-Warning Direct-SHORT：2023–2024 Discovery
- 2024 单年只有4个 production-eligible，因此在查看任何收益前把 discovery 扩为2023–2024；方向和资格规则不变。
- 官方风险警示宇宙：27篇公告、33 token-events、29 unique tokens；>=2年 USD-M 历史 + 前24h QuoteVolume>=500万 后剩 12事件/10币。
- exact 1m SHORT：12笔，4胜8负，PF 0.4524，normalized net -37.22%，avg -3.10%。
- 结论：冻结 / 不进2025-2026 OOS / 不入库。不按风险原因筛选、不只保留正式指定、不反向改 LONG。
- 归档：strategy_templates/research/upbit-risk-warning-direct-short/2023-2024-discovery/。

## 2026-09-26 — Permutation Entropy Continuation
- 固定定义：1h收益最近48根、ordinal order=3 normalized permutation entropy；H<=0.85首次触发，H>=0.90 re-arm；方向取过去12h收益符号。
- 2023–2024 discovery 仅3事件/3币；2025 8事件；2026 6事件。
- 虽部分 forward signed move 为正，但频率远低于项目要求，样本不足以判断跨币泛化。
- 结论：因样本/频率不足冻结。不放宽 entropy 阈值、不改窗口/order、不换相邻 entropy 指标挽救。
- 归档：strategy_templates/research/permutation-entropy-continuation/2023-2026/。

## 2026-09-26 — Average Trade Size Shock Continuation
- 固定定义：1h log(quote_volume/trade_count) 对过去168h做 z-score；首次 z>=3 触发、z<1 re-arm；方向取触发小时涨跌方向。
- 2023–2024 discovery：45事件，12h signed mean +0.2025%，但仅 3/6 有样本币为正。
- 2025：210事件，12h +0.1975%，6/10为正；2026：172事件，12h -0.0258%，仅4/10为正。
- 结论：成本余量不足且 2026 regime failure，冻结 / 不进 exact Engine / 不入库。不扫相邻 z/window，也不事后反向。
- 归档：strategy_templates/research/average-trade-size-shock-continuation/2023-2026/。


## 2026-09-26 — VPIN-style Toxicity Continuation
- 1h taker quote 构造24h toxicity，过去720h z-score，|z|>=3 首次触发，方向跟随触发小时。
- 2023–2024 discovery：42事件，12h signed mean -0.2886%，0/3 有样本币为正。
- 2025：+0.1122%；2026：-0.2496%，明显 regime flip。
- 结论：冻结，不反向、不调 z/window。归档：strategy_templates/research/vpin-style-toxicity-continuation/2023-2026/。

## 2026-09-26 — Roll Effective-Spread Shock Reversal
- 24h lag-1 return covariance 构造 Roll spread，720h z-score，z>=3，方向与触发小时相反。
- discovery 46事件，12h +0.3682%，但只有 BTC/ETH 有样本且仅1/2为正；2025 -0.2629%，2026 +0.1602%。
- 结论：跨 regime 不稳定，冻结。归档：strategy_templates/research/roll-spread-shock-reversal/2023-2026/。

## 2026-09-26 — Volume Concentration Shock Continuation
- 24h QuoteVolume HHI，720h z-score，z>=3，方向跟随过去24h return。
- 1h诊断曾较强：2025 12h +0.7873%（8/10正），2026 +0.1128%（6/10正）。
- frozen exact 1m discovery：99笔，41胜58负，TP23/SL25/TIME51，PF 0.8612，normalized net -34.84%，avg -0.352%。
- 结论：统计 edge 无法覆盖实际执行成本，冻结；不延长12h持仓、不调参数。归档：strategy_templates/research/volume-concentration-shock-continuation/2023-2026/。

## 2026-09-26 — Binance Leverage / Margin Tier Change
- 官方 2023–2024 共58篇档位调整公告，解析326个合约事件；首档最大杠杆机械分类：loosening127、tightening25、unchanged174。
- >=2年历史 + 前24h QuoteVolume>=500万 后剩68事件/61币：loosening53、tightening15。
- 预注册方向：loosening LONG、tightening SHORT。全体12h signed mean -0.3494%，仅30/61币为正；2024单年63事件为 -0.6217%。
- tightening 子样本虽 +2.2096%，但完整 family 失败后只保留 tightening 属于事后选赢家，因此不采用。
- 结论：整条 family 冻结 / 不进 exact Engine / 不做2025-2026 OOS。归档：strategy_templates/research/binance-leverage-margin-tier-change/2023-2024-discovery/。

## 2026-09-26 — Binance–Bybit Price Dislocation Catch-up
- 1h d = Bybit return - Binance return，720h causal z-score，|z|>=3，方向取 d 符号，在 Binance 做 catch-up。
- 2023–2024 discovery：671事件，12h -0.0503%，5/10正。
- 2025：+0.5290%，9/10正；2026：-0.0331%，7/10正。
- 结论：discovery 为负且 2026 edge 消失，不能用2025反推规则；冻结，不调2σ/4σ、不改 reversal。归档：strategy_templates/research/cross-exchange-price-dislocation-catchup/2023-2026/。


## 2026-09-27 — Binance Leverage-Tier First-Bracket Shock：2024 Discovery → 2025/2026 OOS
- 机制：只看 Binance USD-M 官方杠杆/保证金档位调整中「最小名义档最大杠杆」的真实变化。上调预注册 LONG，下调预注册 SHORT；首档最大杠杆不变的纯容量/maintenance tier 调整排除。
- 资格：事件时合约历史 >=2年，事件前24h QuoteVolume >=500万 USDT；不因样本少放宽。
- 执行固定：下一根1m open；4x；TP8/SL6；双边 fee=0.0005 + 5bps slippage；funding；最长24h。
- 2024 discovery：9笔，6TP/3SL，PF **2.0625**，normalized net **+24.42%**，avg +2.71%。
- 2025 OOS1：10笔，6TP/4SL，PF **1.4967**，normalized net **+15.94%**，avg +1.59%。
- 2026 OOS2：10笔，3TP/7SL，PF **0.4334**，normalized net **-33.99%**，avg -3.40%。
- 2026-09 当前月 monthly archive 未封存；9/25 资格用 Vision daily 1h，exact 1m/funding 用 Binance public USD-M REST 补齐，规则与前两期一致。
- 结论：虽然 discovery + OOS1 很强，但 OOS2 硬失败，**整条 family 冻结 / 不入库 / 不进 Engine**。禁止事后只保留 loosening LONG、把 tightening 反向、改24h horizon或降低2年/500万门槛。
- 归档：strategy_templates/research/binance-leverage-tier-first-bracket-shock/2024-2026/。

## 2026-09-27 — Binance Portfolio Margin Collateral-Ratio Repricing：2024 Feasibility
- 机制预注册：Portfolio Margin collateral ratio 上调→LONG，下调→SHORT；同名 USD-M；事件时历史>=2年、前24h QuoteVolume>=500万。
- 2024 官方事件宇宙：5批、36 asset-events；严格资格后仅 **7笔**：LINK/AAVE/DOT/MATIC/WOO SHORT，NEAR/APT LONG。
- 在查看任何收益前尝试检索2023扩充 discovery，但未找到可复现的同类调整序列；因此不使用2025/2026制造 discovery。
- 预注册最低 discovery 样本=8，故**未运行收益回放**。
- 结论：**样本不足冻结 / 不进 Engine / 不入库**。不降低2年/500万门槛。
- 归档：strategy_templates/research/binance-portfolio-margin-collateral-ratio/2024-feasibility/。

## 2026-09-27 — Aggregate Stablecoin Supply Zero-Cross
- 外部流动性机制：DefiLlama peggedUSD 总 circulating USD；过去7个完整日供应变化从<=0穿到>0做LONG，从>=0穿到<0做SHORT；记录日 d 最早在 d+1 00:00 UTC 使用。
- 资格审计修正：不能用本地1h缓存首条记录代表合约上市时间；改用 Binance Vision 月档确认 >=2年。修正后2023–2024覆盖10老币。
- 2023–2024 discovery：866币×信号样本；24h signed mean -0.375%，72h -0.834%；24h/72h均 0/10币为正。
- 2025：230样本；24h -0.650%，72h -1.848%，72h 0/10为正。2026：310样本；24h -0.755%，0/10为正。
- 结论：预注册“供应扩张LONG/收缩SHORT”方向被强烈否定，family冻结 / 不进exact Engine / 不入库。禁止事后反向。
- 归档：strategy_templates/research/stablecoin-supply-zero-cross/2023-2026/。

## 2026-09-27 — Volume Concentration Shock：Exact Replay
- 24h QuoteVolume HHI / 720h causal z-score，z>=3 首次触发，z<1 re-arm，方向跟随过去24h return。
- 统计 diagnostic 很强：2023-2024 12h +0.487%；2025 +0.787%（8/10正）；2026 +0.113%（6/10正）；全样本9/10币正。
- 但 frozen exact 1m discovery（4x、TP8/SL6、双边fee+5bps、funding、12h TIME）99笔：41胜58负，23TP/25SL/51TIME，PF **0.8612**，net **-34.84%**。
- 结论：统计终点 edge 无法覆盖路径与成本，**冻结 / 不做 exact OOS / 不入库**。不改 12h horizon、不扫 HHI/z 邻近值。
- 归档：strategy_templates/research/volume-concentration-shock-continuation/2023-2026/。


## 2026-09-27 — Spot–Perp Basis Convergence
- 固定定义：Binance Spot 与 USD-M 1h 同时刻 close，basis=log(perp/spot)；过去720h causal z-score，首次 |z|>=3 触发，|z|<1 re-arm；正basis做SHORT、负basis做LONG。
- 2023-2024 discovery：730事件，12h signed mean **+0.022%**，6/10币正；2025 +0.506%；2026 +0.195%。
- discovery 经济幅度近零，不能因为后续年份更强而晋级。**冻结 / 不做 exact Engine / 不入库**。
- 归档：strategy_templates/research/spot-perp-basis-convergence/2023-2026/。

## 2026-09-27 — Binance Futures Tick-Size Adjustment：Feasibility
- 预注册：tick变细=LONG、tick变粗=SHORT；>=2年历史、前24h QuoteVolume>=500万。
- 已检索的2023-2024官方批次在严格资格后仅剩 TRB、VET、SOL **3事件**；12h signed mean +4.52%，2/3正。
- n=3 低于最低可研究规模，**按样本不足冻结**。不降低2年门槛，不消耗2025/2026制造 discovery。
- 归档：strategy_templates/research/binance-futures-tick-size-adjustment/2023-2024-feasibility/。

## 2026-09-27 — Amihud Illiquidity Shock Reversal
- 固定定义：1h abs(log return)/QuoteVolume，过去24h求和；相对过去720h做 causal z-score；首次 z>=3，z<1 re-arm；方向反转过去24h return。
- 2023-2024 discovery：198事件，12h signed mean -0.0953%，5/10币正；2025 -0.3342%，3/10正；2026 -0.0154%，5/10正。
- 结论：预注册 reversal 方向失败且幅度远低于成本，冻结 / 不改 continuation / 不进 exact Engine / 不入库。
- 归档：strategy_templates/research/amihud-illiquidity-shock-reversal/2023-2026/。

## 2026-09-27 — Coinbase Listing Catalyst：Data Feasibility
- Coinbase 官方自2022起不再为每个新资产发布独立 blog，而改用官方社交账号发布 listing 公告。
- 当前可访问公开检索无法保证2023-2024官方 X 历史事件全集与精确时间戳完整，因此不能用零散搜索结果构造无偏 discovery universe。
- 状态：data-feasibility blocked / 不做挑样本回测。若未来获得可审计的 @CoinbaseAssets/@CoinbaseExch 历史完整归档，再预注册 direct-LONG 验证。

## 2026-09-27 — COIN-M Quarterly Term-Structure Reversion
- 固定定义：最近到期且 DTE>7天的 COIN-M quarterly 与 COIN-M perpetual 做 annualized basis；过去720h causal z-score，|z|>=3 触发、|z|<1 re-arm；contango极端 SHORT、backwardation极端 LONG。
- 2023-2024 discovery：66事件，12h signed mean -1.1828%，仅3/8币正；2025：965事件，-0.2283%，3/9正；2026 +0.0473%但仅4币有数据。
- 结论：预注册 mean-reversion 方向明确失败，冻结 / 不事后改 continuation / 不进 exact Engine。
- 归档：strategy_templates/research/coinm-quarterly-term-structure-reversion/2023-2026/。

## 2026-09-27 — Official COIN-M Liquidation Cascade Continuation
- Binance Vision COIN-M daily liquidationSnapshot 覆盖9个老币，共3587个日文件；精确去重后按小时聚合 BUY/SELL 强平量。
- 固定定义：log1p(hourly liquidation qty) 对过去720h z-score；z>=3首次触发，z<1 re-arm；BUY强平占优做LONG、SELL占优做SHORT，测试 cascade continuation。
- 2023H2 discovery：831事件，12h -0.3751%，仅2/9币正；2024 OOS1：1160事件，12h -0.1270%，仅2/7币正。
- 结论：continuation 冻结。反号 liquidation-exhaustion reversal 仅是看完结果后生成的新假设，未验证；只能留给未来真正未见 liquidation 数据。
- 归档：strategy_templates/research/coinm-liquidation-cascade-continuation/2023-2024/。

## 2026-09-27 — Binance Margin One-Hour Interest Waiver Direct-SHORT
- 机制预注册：借入指定 crypto 自动减免1小时利息，降低借币卖空融资成本，因此活动开始时固定 SHORT；>=2年 USD-M 历史、前24h QuoteVolume>=500万。
- 可审计事件全集使用2023三批、2024四批；严格资格后2023=22 token-events、2024=27。
- 2023 discovery：12h signed mean +0.5132%，8/10币正；2024 OOS1：12h -0.4427%，8/15币正，整体反向。
- 结论：跨活动批次不稳定，冻结 / 不做 exact TP8-SL6 / 不删除2024-11坏批次 / 不反向 LONG。
- 归档：strategy_templates/research/binance-margin-one-hour-interest-waiver-short/2023-2024/。

## 2026-09-27 — COIN-M Quarterly Term-Structure Convergence
- 固定定义：最近到期且 DTE>7d 的 COIN-M delivery vs COIN-M perpetual，annualized basis=log(quarterly/perp)*365/DTE；过去720h z-score，|z|>=3，|z|<1 re-arm；正 contango 做 SHORT USD-M、负 backwardation 做 LONG。
- 2023-2024 discovery：1,313事件，12h signed mean **+0.0006%**，仅3/9币正；2025 **-0.2176%**，3/9正；2026 +0.0473%，3/4正。
- 结论：统计层面即失败，**冻结 / 不进 exact Engine / 不调 DTE、z 或方向**。
- 归档：strategy_templates/research/coinm-quarterly-term-structure-convergence/2023-2026/。

## 2026-09-27 — COIN-M / USD-M OI-Share Reversal：Early Gate
- 核心4币2023-2024：52事件；1d +0.257%，3d -0.178%，7d **-0.036%**，仅2/4币7d为正。
- 结论：跨币结构不存在，**early gate 冻结 / 不扩9币与OOS**。
- 归档：strategy_templates/research/coinm-usdm-oi-share-reversal/2023-2024-early-gate/。

## 2026-09-27 — COIN-M / USD-M Taker Divergence：Early Gate
- 固定定义：daily mean signed taker imbalance 的 CM-UM 差，30d |z|>=2，按 divergence 符号交易。
- 核心4币2023-2024：141事件；1d +0.063%，3d +0.535%，7d **+0.036%**，仅2/4币正；BTC/BNB负、ETH/XRP正。
- 结论：**early gate 冻结 / 不扩币与OOS**。
- 归档：strategy_templates/research/coinm-usdm-taker-divergence/2023-2024-early-gate/。

## 2026-09-27 — Binance Unplanned Network/Security Suspension：Feasibility
- 完整枚举 Maintenance Updates catalog 157 的2023-2024约220篇公告，并全文扫描事故/攻击/异常语义，排除计划 network upgrade / hard fork / wallet maintenance。
- 仅机械识别出 TORN DAO incident 与 Multichain situation 两类清晰非计划事故，低于最低样本8。
- 结论：**样本不足，不做收益回放，不混入计划维护制造样本**。
- 归档：strategy_templates/research/binance-unplanned-network-suspension/2023-2024-feasibility/。

## 2026-09-27 — Coin Metrics Active-Address Growth
- 10老币 Community AdrActCnt；最近7日均值 / 前7日均值的 log growth 零穿越，正向LONG、负向SHORT；信号日完成后下一UTC日入场。
- 2023-2024 discovery：951事件，7d **-0.249%**，4/10正；2025 -0.122%，5/10；2026 -0.104%，3/10；全样本1749事件 -0.192%，仅2/10正。
- 结论：**冻结 / 不因负结果事后反向**。
- 归档：strategy_templates/research/coinmetrics-active-address-growth/2023-2026/。

## 2026-09-27 — Coin Metrics MVRV Extreme Reversal
- 10老币 CapMVRVCur；log(MVRV) 对过去90日 z-score，z>=2 SHORT、z<=-2 LONG，|z|<1 re-arm；下一UTC日入场。
- 2023-2024 discovery：107事件，7d **-0.684%**，5/10正；2025 +0.675%，7/10；2026 **-6.020%**，仅2/10正。
- 结论：discovery失败且 second OOS 明显崩溃，**MVRV reversal 冻结 / 不改 momentum**。
- 归档：strategy_templates/research/coinmetrics-mvrv-extreme-reversal/2023-2026/。

## 2026-09-27 — COIN-M Quarterly Term-Structure Convergence
- 固定定义：同币 COIN-M 最近到期且 DTE>7天的季度合约相对 COIN-M perpetual 的 annualized log basis；过去720h causal z-score，首次 |z|>=3，|z|<1 re-arm；正 contango 做 SHORT USD-M，负 backwardation 做 LONG。
- 2023-2024 discovery：1313事件、9币，12h signed mean **+0.00056%**，仅3/9币正；2025 **-0.2176%**，3/9正；2026 +0.0473%，仅剩4个长期季度合约族。
- 全样本2707事件，12h **-0.0692%**。discovery 经济幅度近零且 OOS1 反向，**冻结 / 不进 exact Engine / 不入库**。
- 归档：strategy_templates/research/coinm-quarterly-term-structure-convergence/2023-2026/。

## 2026-09-27 — Binance First-Tier Notional Capacity Change
- 基于已有完整 326-event leverage/margin-tier census，只取首档最大杠杆不变但首档 notional cap 变化的纯容量事件。
- 共57事件，全部为 capacity expansion；严格 >=2年 后剩15事件/15币。
- 预注册 expansion=LONG：1h mean -0.248%，4h -0.598%，12h **-1.456%**；12h 8/15正但负尾显著。
- 结论：**冻结 / 不反向改 SHORT / 不进 exact Engine / 不入库**。
- 归档：strategy_templates/research/binance-first-tier-notional-capacity-change/2023-2024-discovery/。

## 2026-09-27 — Binance First-Tier MMR Change：Feasibility
- 在完整326-event leverage/margin-tier census 中，排除首档最大杠杆和首档 notional cap 变化后，仅剩2个纯首档 MMR 变化事件，且均为 MMR 下调。
- 两个事件在发生时都不满足 USD-M 历史>=2年，最终 **0 eligible**。
- 结论：**feasibility 不足 / 冻结 / 不消耗后续年份 / 不降低资格门槛**。
- 归档：strategy_templates/research/binance-first-tier-mmr-change/2023-2024-feasibility/。

## 2026-09-27 — Binance vs Bybit Funding Divergence Reversal
- 同一 settlement timestamp 配对；两边 funding 都按实际历史结算间隔换算为单位小时费率；diff=Binance-Bybit，过去90天 causal z-score。
- 首次 |z|>=3，|z|<1 re-arm；Binance更高做SHORT、更低做LONG。
- 2023-2024 discovery：266事件，12h **-0.2993%**，仅2/10币正；2025 +0.0296%，4/10正；2026 +0.0185%，3/10正。
- 结论：预注册 crowding-reversal 方向失败，**冻结 / 不反向改 continuation / 不加 absolute funding/OI filter / 不进 exact Engine**。
- 归档：strategy_templates/research/binance-bybit-funding-divergence-reversal/2023-2026/。

## 2026-09-27 — Large Aggressor Order Flow：2026 Early Gate
- 用 Binance Vision aggTrades；前一完整 UTC 日的 aggTrade notional q99 定义“大额主动单”，逐小时 tail signed-flow 相对前一日24h baseline 首次 |z|>=3 触发，|z|<1 re-arm；按 flow 符号顺势。
- 固定 Jan/Apr/Jul 15 三个 signal day。SOL 仅1事件（12h +0.681%）；XRP 4事件（12h -0.054%）；合计5事件，1h +0.161%、4h +0.168%、12h +0.093%，12h 仅1/2币正。
- BTC/ETH 大文件下载未完成，但 SOL/XRP early gate 已显示频率过低、跨币12h不一致且经济幅度不足，因此主动停止扩展。
- 结论：**early freeze / 不降 q99→q95 / 不降3σ→2σ / 不为找正结果继续下载 BTC/ETH**。
- 归档：strategy_templates/research/large-aggressor-order-flow/2026-early-gate/。

## 2026-09-27 — Binance Network Upgrade / Hard-Fork Catalyst LONG
- Binance Maintenance Updates 官方目录：2023-2024 共175篇明确 Network Upgrade / Hard Fork 公告，机械解析191 token-events；严格 >=2年 + 24h QV>=500万 后剩 **82事件/32币**。
- 预注册 LONG：1h +0.090%，4h +0.108%，12h **+0.074%**；仅14/32币12h均值为正。
- 年度明显翻转：2023 +0.726%，2024 **-0.610%**。
- 结论：**regime 不稳定且经济幅度不足，冻结 / 不进 exact Engine / 不按升级类型后验筛选 / 不反向 SHORT**。
- 归档：strategy_templates/research/binance-network-upgrade-catalyst-long/2023-2024-discovery/。

## 2026-09-27 — CoinMetrics Daily Supply Shock Reversal
- CoinMetrics Community SplyCur 日变化，对前90个完整UTC日做 z-score；首次 |z|>=3，|z|<1 re-arm；供给正冲击=SHORT、负冲击=LONG；下一UTC日 open 入场。
- 2023-2024 discovery：77事件，7d +0.182%，仅3/8币正；2025 +3.113%，6/8正；2026 **-0.842%**，4/8正。
- 结论：跨币/跨 regime 不稳定，**冻结 / 不事后只挑1d horizon / 不进 exact Engine / 不入库**。
- 归档：strategy_templates/research/coinmetrics-supply-shock-reversal/2023-2026/。

## 2026-09-27 — CoinMetrics Exchange Netflow：Feasibility
- Community catalog 中 FlowInExUSD / FlowOutExUSD / SplyExUSD 仅 BTC/ETH 支持；XRP/ADA/LINK/BCH/LTC/DOGE/UNI/ZEC 不支持。
- 结论：不满足“同一策略跨很多合约”，**data-feasibility blocked / 不做 BTC/ETH 特例**。
- 归档：strategy_templates/research/coinmetrics-exchange-netflow/feasibility/。

## 2026-09-27 — Binance Loan Collateral-Asset Addition LONG
- 20个官方2023-2024 Loan批次，严格区分 collateral 与 loanable；collateral-only universe 201 token-events，production eligibility 后60事件。
- 2023 discovery：53事件，12h **-0.705%**，仅14/41币正；16个 eligible batch，batch-equal 12h **-0.552%**，仅7/16批次正。
- 2024 OOS1：7事件，**0/7正**，12h **-5.426%**。
- 结论：**新增抵押用途→LONG 明确失败，冻结 / 不反向 SHORT / 不进 exact Engine / 不入库**。
- 归档：strategy_templates/research/binance-loan-collateral-addition-long/2023-2024/。

## 2026-09-27 — CoinMetrics Transaction-Count Growth
- 10老币 Community TxCnt；最近7日均值 / 前7日均值 log-growth 零穿越，正向LONG、负向SHORT；信号日完成后下一UTC日入场。
- 2023-2024 discovery：867事件，7d **+0.266%**，7/10正；2025 OOS1 **-0.532%**，5/10正；2026 OOS2 **-0.729%**，3/10正。
- 全样本1609事件，7d -0.134%。两个时间外均反向，**冻结 / 不调窗口 / 不事后反向 / 不入库**。
- 归档：strategy_templates/research/coinmetrics-txcount-growth/2023-2026/。

## 2026-09-27 — CoinMetrics NVT / Adjusted Transfer Value：Feasibility
- Community API 对 NVTAdj 与 TxTfrValAdjUSD 返回 HTTP 403；当前无凭证条件下不可复现。
- 结论：**data-access blocked**；不使用推算 NVT 或其它口径替代。
- 归档：strategy_templates/research/coinmetrics-nvt-transfer-value/feasibility/。

## 2026-09-27 — CoinMetrics Transaction-Count Growth
- 复用 Active Address 的固定变换：最近7日 TxCnt 均值 / 前7日均值，log growth 零穿越；正向LONG、负向SHORT；下一UTC日open入场。
- 2023-2024 discovery：867事件，7d **+0.266%**，7/10币正；2025 OOS1 **-0.532%**，5/10正；2026 OOS2 **-0.729%**，仅3/10正。
- 全样本1609事件，7d -0.134%，4/10正。结论：**forward失效，冻结 / 不反向 / 不调7d窗口**。
- 归档：strategy_templates/research/coinmetrics-txcount-growth/2023-2026/。

## 2026-09-27 — CoinMetrics NVT / Adjusted Transfer Value：Feasibility
- 10币查询 NVTAdj 与 TxTfrValAdjUSD 时，CoinMetrics Community API 当前返回 HTTP 403；TxCnt 同期仍可公开访问。
- 结论：**data-access blocked / 不假设付费数据 / 不做BTC-ETH特例**。
- 归档：strategy_templates/research/coinmetrics-nvt-transfer-value/feasibility/。

## 2026-09-27 — CoinMetrics Holder-Base Growth
- AdrBalCnt 表示非零余额地址数量，区别于当天 Active Address；仍使用固定最近7日均值/前7日均值的 growth 零穿越，正向LONG、负向SHORT。
- 2023-2024 discovery：293事件，7d **-0.519%**，仅2/9有事件币正；2025 +0.680%；2026 +1.571%。
- discovery 方向明确失败，不能用后两年转正反推规则。结论：**冻结 / 不调窗口 / 不继续枚举相邻 activity 指标**。
- 归档：strategy_templates/research/coinmetrics-holder-base-growth/2023-2026/。

## 2026-09-27 — US Macro Release First-Hour Continuation
- 初始预注册事件集：CPI / Employment Situation(NFP) / PPI / FOMC；官方ET发布时间转UTC；每币自身事件后60分钟方向，下一根5m open顺势入场。
- 2023-2024 discovery：88事件时点、875币×事件实例；12h **+0.0037%**，仅4/10币正，2023为负、2024仅+0.0887%，整体 family 失败。
- FOMC 子集在 discovery 为 +1.397%、75.5%胜率，因此只作为新生成假设，规则冻结后测试完全未见的2025/2026 FOMC。
- FOMC OOS：129实例，12h **-0.3298%**，仅4/10币正；2025 **-0.5935%**。
- 结论：**宏观混合 family 与 FOMC-only 均冻结 / 不进 exact TP8-SL6 / 不入库**。
- 归档：strategy_templates/research/us-macro-release-first-hour-continuation/2023-2026/。

## 2026-09-27 — Wikipedia Attention Shock：Feasibility
- 计划用英文 Wikipedia daily pageviews 的90日异常作为注意力冲击；但10币 canonical 页面历史不可比。
- BNB token页面2023审计窗口仅个位数浏览量；XRP Ledger主相关页面无法回溯到2023；若为不同币改用网络/公司/代币不同类型页面，会引入 symbol-specific 语义偏差。
- 结论：**data-quality blocked / 未看收益 / 不按币挑页面**。
- 归档：strategy_templates/research/wikipedia-attention-shock/feasibility/。

## 2026-09-27 — OI / Turnover Extreme Reversal：Early Gate
- 固定定义：hour-end OI notional / 同小时 QuoteVolume 的 log ratio，720h causal z-score；首次 z>=3、z<1 re-arm；反转过去12h return。
- BTC/ETH/BNB/XRP 2023-2024：前三币 **0事件**，XRP仅1事件且12h -1.536%。
- 结论：**触发过稀且 gate 失败，冻结 / 不降3σ→2σ / 不扩10币与OOS**。
- 归档：strategy_templates/research/oi-turnover-extreme-reversal/2023-2024-early-gate/。

## 2026-09-27 — Kraken Spot Listing Catalyst：Feasibility
- Kraken官方 WordPress Asset Listings 2023-2024 全归档：机械排除地域扩展、margin、OTC、network/funding 后，得到56篇首次全球交易文章、85 asset-events。
- 严格 Binance USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅 **2 eligible**：LIT、RSR；2023为0。
- 结论：**样本不足，未看收益 / 冻结 / 不降低2年门槛**。
- 归档：strategy_templates/research/kraken-spot-listing-catalyst/2023-2024-feasibility/。

## 2026-09-27 — CoinMetrics Transaction-Count Growth
- 10老币 CoinMetrics Community TxCnt；最近7日均值 / 前7日均值的 log growth 零穿越，正向LONG、负向SHORT；指标日完成后下一UTC日入场。
- 2023-2024 discovery：867事件，7d **+0.266%**，7/10币正；2025 OOS1 **-0.532%**，5/10正；2026 OOS2 **-0.729%**，3/10正。
- 全样本1609事件，7d -0.134%，仅4/10币正。
- 结论：**连续两个OOS翻负，冻结 / 不缩短到1d/3d / 不事后反向 / 不入库**。
- 归档：strategy_templates/research/coinmetrics-transaction-count-growth/2023-2026/。

## 2026-09-27 — Binance Loanable-Asset Addition SHORT
- 复用同一20篇官方 Loan/VIP Loan 公告，重新机械解析 New Loanable Assets；213 token-events / 161 unique，严格 >=2年 + 24h QV>=500万 后剩56事件。
- 2023 diagnostic：46事件，12h SHORT +0.723%，26/40币正、10/16批次正；2024 diagnostic：10事件 +2.806%、9/10正，但仅1个批次。
- frozen exact 1m 2023 discovery：46笔，17胜29负，12TP/19SL/15TIME，PF **0.6939**，normalized net **-46.45%**，avg -1.01%；16/40币、7/16批次净正。
- 结论：**路径/成本关失败，冻结 / 不跑2024 exact / 不反向LONG / 不改12h horizon / 不入库**。
- 归档：strategy_templates/research/binance-loanable-asset-addition-short/2023-2024/。

## 2026-09-27 — Binance Simple Earn Asset-Addition LONG：Feasibility
- 2023-2024 官方目录机械筛出31篇新资产 Simple Earn Locked/Flexible Products 公告、39 token-events。
- 严格 USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅剩 **4事件：BTC/ANKR/DYDX/WOO**。
- 低于最低 discovery 样本8，**未查看收益，样本不足冻结 / 不降低2年门槛 / 不入库**。
- 归档：strategy_templates/research/binance-simple-earn-asset-addition-long/2023-2024-feasibility/。

## 2026-09-27 — Binance–Bybit Volume-Dominance Lead
- 同币1h log(Bybit linear turnover / Binance QuoteVolume)，过去720h causal z-score；首次 |z|>=3，|z|<1 re-arm。
- Bybit异常占优时跟随该小时Bybit return；Binance异常占优时跟随Binance return；下一根Binance 1h open入场。
- 2023-2024 discovery：1042事件，12h **-0.0428%**，4/10币正；2025 +0.0184%，5/10；2026 **-0.0640%**，4/10。
- 全样本2022事件，12h -0.0287%。结论：**冻结 / 不反向 / 不调z / 不进exact Engine / 不入库**。
- 归档：strategy_templates/research/binance-bybit-volume-dominance-lead/2023-2026/。

## 2026-09-27 — CoinMetrics Network-Fee Growth
- CoinMetrics Community FeeTotNtv 对8个老币完整可用；最近7日均值/前7日均值 log-growth 零穿越，正向LONG、负向SHORT；下一UTC日open入场。
- 2023-2024 discovery：723事件，7d **-0.347%**，仅3/8币正；2025 **-1.024%**，2/8正；2026 **-1.275%**，仅1/8正。
- 全样本1338事件，7d -0.672%，2/8正。
- 结论：预注册需求增长方向在 discovery 与两个OOS均失败，**冻结 / 不事后反向 / 不再枚举相邻 CoinMetrics activity 指标 / 不入库**。
- 归档：strategy_templates/research/coinmetrics-network-fee-growth/2023-2026/。

## 2026-09-27 — Binance Spot/Margin Minimum Order Size Reduction：Feasibility
- 2021-2024 官方同类历史仅4个独立批次；2023-08-31 虽影响大量交易对，但同一政策时点不能按多币伪装成独立事件。
- 结论：**未看收益即因独立 batch 数不足冻结**。
- 归档：strategy_templates/research/binance-spot-minimum-order-size-reduction/2021-2024-feasibility/。

## 2026-09-27 — Binance Convert Asset Addition LONG：Feasibility
- 2023-2024 官方标题全集得到40个明确新增 crypto token-event；21个事件月已有 USD-M，但 **0个**满足两年前同月已有 USD-M 历史。
- 结论：与 >=2年生产资格天然冲突，**未看收益冻结**。
- 归档：strategy_templates/research/binance-convert-asset-addition-long/2023-2024-feasibility/。

## 2026-09-27 — Binance Vision BookTicker Top-of-Book：Feasibility
- bookTicker 是独立 top-of-book event stream，但 BTC 单日压缩约90-185MB、月档4-8GB；当前公开历史观察到约2024-04即结束，缺2025/2026 OOS。
- 结论：**historical-OOS blocked**；不为无法做近期验证的 family 下载多GB历史。
- 归档：strategy_templates/research/binance-bookticker-top-of-book/feasibility/。

## 2026-09-27 — Binance Secondary USDC-Perpetual Launch SHORT
- 官方2024 USDC perpetual launch universe 25 token-events；用 USDC 合约第一根 Vision K线确定实际 launch 时间，避免延期公告歧义。
- 严格 >=2年 + 24h QV>=500万 后剩14事件、9独立 launch 批次。
- 预注册 SHORT 原 USDT perpetual：1h **-0.491%**、4h **-1.215%**、12h **-1.281%**，仅4/14正；batch-equal 12h **-0.681%**，4/9批次正。
- 结论：discovery 明确失败，**冻结 / 不事后改 LONG / 不消耗小样本2025 OOS / 不入库**。
- 归档：strategy_templates/research/binance-usdc-perpetual-secondary-launch-short/2024-discovery/。

## 2026-09-28 — Binance–Bybit OI Divergence Reversal：Early Gate
- 固定定义：log(Binance USD-M OI / Bybit linear OI)，过去720h causal z-score；首次 |z|>=3，|z|<1 re-arm；Binance OI异常占优做SHORT、Bybit异常占优做LONG。
- BTC/ETH/BNB/XRP 2023-2024：60事件，12h signed mean **+0.3582%**，win 58.3%，但仅 **2/4币** 12h均值为正；BTC/XRP为负。
- 每币两年仅13-17次触发，频率明显低于目标。
- 结论：**early gate 冻结 / 不扩10币与OOS / 不降3σ→2σ / 不改 continuation / 不入库**。
- 归档：strategy_templates/research/binance-bybit-oi-divergence-reversal/2023-2024-early-gate/。

## 2026-09-28 — COIN-M / USD-M Funding Divergence Reversal：Early Gate
- 同一 settlement timestamp 配对 COIN-M / USD-M funding，并按实际 funding_interval_hours 换算单位小时费率；diff=CM-UM，过去90天 causal z-score。
- 首次 |z|>=3，|z|<1 re-arm；COIN-M funding异常更高做SHORT USD-M、更低做LONG。
- BTC/ETH/BNB/XRP 2023-2024：59事件；1h -0.102%、4h -0.377%、12h **-0.095%**，仅 **2/4币** 12h为正；每币两年仅11-19次触发。
- 结论：**early gate 冻结 / 不扩9币与OOS / 不降3σ / 不改 continuation / 不入库**。
- 归档：strategy_templates/research/coinm-usdm-funding-divergence-reversal/2023-2024-early-gate/。

## 2026-09-28 — CoinMetrics Issuance-Rate Shock：Feasibility
- 计划指标：IssTotNtv/SplyCur；log issuance-rate 对前90日 causal z-score，+3σ SHORT、-3σ LONG、|z|<1 re-arm；下一UTC日open。
- Community 可变序列只有 BTC/ETH/ADA/BCH/LTC/DOGE/ZEC **7币**；LINK/UNI IssTotNtv 全程为0，BNB无可用序列。
- 预注册最低覆盖=8币，因此**未看收益即冻结 / 不降低覆盖要求 / 不做symbol-specific替代**。
- 归档：strategy_templates/research/coinmetrics-issuance-rate-shock/feasibility/。

## 2026-09-28 — Binance–Bybit Level-Premium Convergence
- 固定定义：同币 Binance USD-M / Bybit linear perpetual 1h close 的 log price-level premium；过去720h causal z-score，首次 |z|>=3，|z|<1 re-arm；Binance premium高做SHORT，低做LONG。
- 2023-2024 discovery：697事件，12h **-0.0971%**，仅3/10币正；2025 +0.3731%，5/10正；2026 **-0.0186%**，4/10正。
- 全样本1313事件，12h +0.0456%，但仅3/10币正。
- 结论：**discovery失败，冻结 / 不用2025反推 / 不改continuation / 不进exact Engine / 不入库**。
- 归档：strategy_templates/research/binance-bybit-level-premium-convergence/2023-2026/。

## 2026-09-28 — Bybit USDT Perpetual Delist → Binance SHORT：Feasibility
- Bybit 官方 Delistings 2023-2024 全集：46篇 derivatives-tagged 公告；正文审计 BIT/ZBC 后得到48个 unique USDT perpetual symbol-events。
- 预注册方向：Bybit 衍生品下架后 SHORT Binance USD-M；资格仍为事件时历史>=2年、前24h QuoteVolume>=500万。
- 严格资格后仅剩 **5事件：TOMO/OCEAN/UNFI/BLZ/FTM**，低于最低 discovery 样本8。
- 结论：**未看收益即样本不足冻结 / 不降低门槛 / 不混入Spot delist / 不入库**。
- 归档：strategy_templates/research/bybit-usdt-perpetual-delist-short/2023-2024-feasibility/。

## 2026-09-28 — Upbit Trading-Support Termination → Binance SHORT：Feasibility
- Upbit 官方 2023-2024 trade 公告完整枚举：12篇交易支持终止公告、15 token-events、15 unique token。
- 严格 Binance USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅剩 **2 eligible：LINA、OMG**；12事件月无 USD-M，1个流动性不足。
- 低于最低 discovery 样本8，**未查看收益即冻结 / 不降低门槛 / 不入库**。
- 归档：strategy_templates/research/upbit-trading-support-termination-short/2023-2024-feasibility/。

## 2026-09-28 — Bybit USDT Perpetual Launch → Binance LONG：Feasibility
- Bybit 官方 New Listings 服务端分页完整枚举：严格上线标题207 token-events / 205 unique symbols；排除 Pre-Market、转换和 adjustment 标题。
- 其中68个事件月 Binance 已有 USD-M，但 **68/68 均不满足事件时历史>=2年**；production-eligible = 0。
- 结论：与 >=2年生产资格天然冲突，**未看收益即冻结 / 不降低门槛 / 不入库**。
- 归档：strategy_templates/research/bybit-usdt-perpetual-launch-long/2023-2024-feasibility/。

## 2026-09-28 — Bybit Risk-Limit Adjustment → Binance USD-M
- 官方2023-2024三类公告完整枚举37篇；只解析结构化 before/after 表，不OCR。22篇可解析，156合约事件，152有明确 expansion/contraction。
- 严格 >=2年 + 24h QV>=500万 后剩 **25事件/24币/8批次**。1h diagnostic 12h signed mean +2.33%，batch-equal +4.24%，表面较强。
- frozen exact 1m（公告后下一1m open、4x、TP8/SL6、fee/slippage/funding、12h）：**25笔，7胜18负，4TP/18SL/3TIME，PF 0.3944，net -76.91%**。
- expansion PF 0.503；contraction 4/4全SL。结论：**路径/成本关明确失败，冻结 / 不跑2025-2026 OOS / 不反向 / 不改TP/SL/horizon / 不入库**。
- 归档：strategy_templates/research/bybit-risk-limit-adjustment/2023-2024-discovery/。

## 2026-09-28 — Upbit Trading-Support Termination → Binance SHORT：Feasibility
- 2023-2024 Upbit 官方完整终止支持全集：12篇 / 15 token-events / 15 unique token；预注册方向 SHORT。
- 严格 Binance USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅剩 **2事件：LINA、OMG**；12个事件月无USD-M，SRM流动性不足。
- 低于最低 discovery 样本8，因此**未看收益即冻结 / 不降低2年和流动性门槛 / 不入库**。
- 归档：strategy_templates/research/upbit-trading-support-termination-short/2023-2024-feasibility/。

## 2026-09-28 — Binance Spot vs USD-M Index Price Lead
- 固定定义：同币 Binance Spot close / USD-M Index Price close 的 log premium；过去720h causal z-score；首次 |z|>=3，|z|<1 re-arm；spot premium正做LONG USD-M、负做SHORT。
- 2023-2024 discovery：1752事件，12h **+0.246%**，9/10币正；2025 OOS1 **+0.0119%**，5/10正；2026 OOS2 **+0.0334%**，5/10正。1h 在各阶段均为负。
- 结论：discovery 广度没有转化为可持续 OOS 幅度，**冻结 / 不反向 / 不进 exact Engine / 不入库**。
- 归档：strategy_templates/research/binance-spot-vs-index-price-lead/2023-2026/。

## 2026-09-28 — Funding Settlement Pre-Move Reversal
- 只用真实 market_funding_rates.funding_time 触发，不使用 NowTime %；反转结算前完整1h return，结算小时open入场。
- 2023-2024 discovery：21,711事件，4h **+0.0160%**，7/10币正；2025 OOS1 **-0.0596%**，仅1/10正；2026 OOS2 **-0.0189%**，3/10正。
- 全样本40,195事件，4h -0.0111%，经济幅度远低于成本。
- 结论：**两个OOS连续失败，冻结 / 不改continuation / 不改窗口 / 不入库**。
- 归档：strategy_templates/research/funding-settlement-premove-reversal/2023-2026/。

## 2026-09-28 — Upbit Trading-Support Termination → Binance SHORT：Feasibility
- Upbit官方2023-2024交易公告全集：12篇交易支持终止公告、15 token-events / 15 unique。
- 预注册方向 SHORT；资格：事件时 Binance USD-M 存在、历史>=2年、前24h QuoteVolume>=500万。
- 严格资格仅剩 **2事件：LINAUSDT、OMGUSDT**；12个事件月无USD-M，1个未过流动性。
- 低于最低 discovery 样本8，**未看收益即冻结 / 不降低门槛 / 不进exact replay / 不入库**。
- 归档：strategy_templates/research/upbit-trading-support-termination-short/2023-2024-feasibility/。

## 2026-09-28 — Bybit Spot Listing → Binance LONG：Feasibility
- Bybit官方 New Listings 完整分页：2023-2024 明确 Spot listing **257 token-events / 253 unique**；排除 perpetual、Pre-Market、Convert、Margin-only。
- 43个事件月 Binance 已有 USD-M；其中41个历史<2年、1个24h QV<500万，最终仅 **1 eligible：KAVAUSDT**。
- 低于最低样本8，**未看收益即冻结 / 不降低门槛 / 不继续枚举相邻外部Spot listing交易所来凑样本 / 不入库**。
- 归档：strategy_templates/research/bybit-spot-listing-catalyst-long/2023-2024-feasibility/。

## 2026-09-28 — Binance Options IV / Skew：Feasibility
- Binance Vision Options 历史目录：BVOLIndex 仅 BTC/ETH；EOHSummary 仅 BTC/ETH/BNB/XRP/DOGE，共5个 underlying。
- 低于新数据源最低跨币覆盖8；用 BTC/ETH Options 当其它币 market factor 又会落入禁止的 Benchmark/MarketCondition 类。
- 结论：**未看收益即 coverage blocked / 不做5币特例 / 不入库**。
- 归档：strategy_templates/research/binance-options-iv-skew/feasibility/。

## 2026-09-28 — Monitoring Tag Addition SHORT：Funding Parser Corrected Replay
- 保留原始35事件、SHORT、4x、TP8/SL6、双边fee0.0005、双边5bps、72h、single-position；唯一修改是 fundingRate 按真实三列解析，并在无 MarkPrice 时使用 funding 所在1m bar.Close，和正式 Engine 一致。
- corrected 2024 discovery：PF **1.3775**，net **+10.91%**（旧1.374/+10.84）。
- corrected 2025 OOS：PF **1.1121**，net **+8.13%**（旧1.107/+7.77）；MDT TIME 因正 funding 从约-0.80%修正为-0.44%。
- corrected 2026 second OOS：PF **0.9069**，net **-8.44%**（旧0.910/-8.13）。
- 三年35事件 corrected：PF **1.0552**，normalized net **+10.61%**；18胜17负。
- 结论：funding parser bug 对数值影响很小，不改变最终冻结结论。2026 OOS仍失败，长期接近盈亏平衡且 gap tail 极大；不入库、不升级为第三候选。
- 原始有bug helper继续保存在 replay/；修正版在 replay_corrected/，用于审计。

## 2026-09-28 — Volume Profile / Value Area Breakout
- 固定定义：过去24个已完成1h，typical price=(H+L+C)/3、QuoteVolume加权；15%/85% weighted price quantile 作为中央70% Value Area；首次收盘突破VAH做LONG、跌破VAL做SHORT，回到价值区re-arm。
- Binance Vision真实上线月确认10老币在2023均已满2年；不使用本地1h第一根时间作为合约年龄。
- 2023-2024 discovery：21,303事件，12h **-0.0022%**，6/10币正；2025 +0.0576%；2026 +0.1208%。
- discovery 经济幅度为零且略负，后期改善不能反推。**冻结 / 不进exact 1m / 不改reversal / 不调value-area宽度或窗口 / 不入库**。
- 归档：strategy_templates/research/volume-profile-value-area-breakout/2023-2026/。

## 2026-09-28 — Return Autocorrelation Regime
- 最近24个1h log return 的 lag-1 autocorrelation；rho由<=0穿到>0时跟随过去4h方向，rho由>=0穿到<0时反转过去4h方向；零点为唯一自然阈值。
- 2023-2024 discovery：14,995事件，12h **+0.0235%**，8/10币正；2025 OOS1 **-0.0308%**，3/10正；2026 OOS2 **-0.0667%**，3/10正。
- 全样本28,200事件，12h -0.0086%。结论：**两个OOS连续翻负，冻结 / 不只保留一侧regime / 不调窗口或阈值 / 不进exact Engine / 不入库**。
- 归档：strategy_templates/research/return-autocorrelation-regime/2023-2026/。

## 2026-09-28 — Upbit Trading-Support Termination → Binance SHORT：Feasibility
- 2023-2024 Upbit 官方交易支持终止全集：12篇公告、15 token-events / 15 unique token。
- 严格 Binance USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅剩 **2事件：LINA、OMG**。
- 低于最低 discovery 样本8，**未查看收益即冻结 / 不降低2年门槛 / 不混入风险警示或交易对移除 / 不入库**。
- 归档：strategy_templates/research/upbit-trading-support-termination-short/2023-2024-feasibility/。

## 2026-09-28 — Binance DeFi Staking Removal → SHORT：Feasibility
- 2023-2024 官方全集只有2个独立 policy batch：2023-12-05 BTC/ETH；2024-01-09 BNB/USDT/XVS/DAI/CVX。
- 同批多个 token 不视为独立事件；独立 batch 数远低于最低 discovery 规模。
- 结论：**未看收益即冻结 / 不按 token 拆成伪独立样本 / 不入库**。
- 归档：strategy_templates/research/binance-defi-staking-removal-short/2023-2024-feasibility/。

## 2026-09-28 — Binance-Bybit Positioning Divergence Reversal：Early Gate
- 固定定义：1h log(Bybit long/short account ratio / Binance global long/short account ratio)，过去720h causal z-score；首次 |z|>=3，|z|<1 re-arm；Bybit相对更偏多→SHORT Binance，相对更偏空→LONG。
- BTC/ETH/BNB/XRP 2023-2024：60事件，12h +0.1264%，win 56.7%，但仅 2/4币 正。
- 2023 +0.6247%、4/4正；2024 -0.3097%、仅2/4正。BTC/XRP全 discovery 均值为负，且每币两年仅8-29事件。
- 结论：年度 regime flip + breadth/frequency 不足，early gate 冻结 / 不扩10币与OOS / 不降3σ / 不改 continuation / 不入库。
- 归档：strategy_templates/research/binance-bybit-positioning-divergence-reversal/2023-2024-early-gate/。

## 2026-09-28 — Binance-Bybit Premium Index Divergence Reversal：Early Gate
- 固定定义：Bybit linear premium-index close - Binance USD-M premium-index close；1h 对齐，过去720h causal z-score；首次 |z|>=3，|z|<1 re-arm；Bybit premium相对更高→SHORT Binance，相对更低→LONG。
- BTC/ETH/BNB/XRP 2023-2024：462事件，12h **-0.0908%**，win 47.4%，仅 **2/4币** 正。
- 2023 **-0.3841%**、仅1/4正；2024才转为 +0.1391%、3/4正。Discovery 本身方向错误且年度翻转。
- 结论：**early gate 冻结 / 不扩10币与OOS / 不降3σ / 不改 continuation / 不入库**。
- 归档：strategy_templates/research/binance-bybit-premium-index-divergence-reversal/2023-2024-early-gate/。

## 2026-09-28 — Spot-Perp Price-Impact Allocation：Early Gate
- 固定定义：Spot 与 USD-M Perp 分别用过去168h估计 return~signed taker fraction 的无截距 impact slope；log(|lambda_spot|/|lambda_perp|) 对过去720h做 causal z-score；首次 |z|>=3，|z|<1 re-arm。
- Spot impact异常占优则跟随已完成Spot taker flow；Perp异常占优则跟随Perp taker flow；下一根Perp 1h open入场。
- BTC/ETH/BNB/XRP 2023-2024：44事件，12h **-0.6592%**，win 40.9%，仅 **1/4币** 正；2023 -0.4330%，2024 -0.8658%。
- 结论：**机制方向本身失败，early gate 冻结 / 不反向 / 不降3σ / 不扩10币与OOS / 不入库**。
- 归档：strategy_templates/research/spot-perp-price-impact-allocation/2023-2024-early-gate/。

## 2026-09-28 — DeFiLlama Protocol Exploit → Binance SHORT
- 事件全集机械构建：DeFiLlama Hacks 中 DeFi Protocol/Token，排除 Rugpull；必须 defillamaId→protocol.id→protocol.symbol 唯一映射，不人工猜ticker；>=2年 USD-M + 事件日前完整UTC日 QV>=500万。
- 2023-2024 discovery 严格资格剩8事件/7币。下一UTC日open SHORT endpoint diagnostic：1d +1.793%、3d +3.659%、7d **+2.706%**，6/8正；2023 +1.849%、2024 +4.135%，通过预注册晋级gate。
- 真 OOS 2025-2026 严格资格剩6事件/6币：3d **+4.730%**，7d **+7.277%**、5/6正；2025 +8.587%、2026 +5.967%。
- 但冻结 exact 1m（下一UTC日00:00 open、4x、TP8/SL6、双边fee+5bps、funding、最长72h）失败：
  - discovery：8笔，3TP/5SL，PF **0.675**，net -11.61%；
  - OOS：6笔，2TP/4SL，PF **0.553**，net -13.08%；2025 PF0.532、2026 PF0.573；
  - 全14笔：PF **0.620**，net -24.69%。
- 结论：多日终点下跌真实存在，但固定执行路径经常先反抽触发SL，**路径/成本关失败，冻结 / 不扩大SL / 不延长72h / 不延迟入场 / 不按攻击类型后验筛选 / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-exploit-short/2023-2026/。

## 2026-09-28 — COIN-M Perpetual Secondary Launch → USD-M LONG：Feasibility
- Binance Vision 全量 COIN-M perpetual 中，2023-2024 只有 WIF/DOGS/SUI/WLD 四个首次月档。
- 四者事件月都有对应 USD-M，但两年前月档全部不存在，严格 >=2年 后 **0 eligible**。
- 结论：**天然与生产资格冲突 / 未看收益冻结 / 不降低2年门槛 / 不入库**。
- 归档：strategy_templates/research/coinm-perpetual-secondary-launch-long/2023-2024-feasibility/。

## 2026-09-28 — Funding Interval Compression：Binance Vision Full Audit
- 官方 Binance Vision fundingRate 月档重审正式15币，2024-01~2026-08。
- BTC/ETH/BNB/XRP/SOL/DOGE/LTC/AVAX/UNI/ZEC/ADA/NEAR/1000PEPE/SUI 全程只有8h；ONDO全程只有4h。
- **1h rows = 0，真实 interval-change transition = 0**。ONDO不是8h→4h压缩事件。
- 结论：此前 blocker 不是本地表缺数，官方归档本身也没有 production-eligible compression event。**永久冻结 / 不再重开该 family**。

## 2026-09-28 — Binance Multi-Assets Margin Support：Feasibility
- 2023-2024 Binance 官方标题全集共7篇相关公告；正文审计后真正的方向性 crypto margin-asset 事件仅 ADA/DOT/SOL/XRP 的支持撤销。
- TUSD/USDP 是稳定币新增；2024-02 是全局默认模式更新；auto-exchange threshold 是全局制度变化；SOL 旧公告有延期/重发，不能拆成伪独立样本。
- 独立方向事件远低于最低 discovery 规模8，**未看收益即冻结 / 不把重复阶段拆样本 / 不入库**。
- 归档：strategy_templates/research/binance-multi-assets-margin-support/2023-2024-feasibility/。

## 2026-09-28 — OI Creation Efficiency Continuation：Early Gate
- 固定定义：max(ΔOI notional,0)/QuoteVolume 做 log1p，过去720h causal z-score；首次 z>=3、z<1 re-arm；方向跟随触发小时已完成 return。
- BTC/ETH/BNB/XRP 2023-2024：1063事件；1h +0.0154%、4h +0.0108%、12h **-0.0100%**，win49.6%。
- 12h：BTC +0.1303%、ETH +0.0808%、BNB +0.0374%、XRP -0.3020%；虽3/4币正，但总体经济 edge 为零且未达到预注册 +0.10% gate。
- 结论：**early gate冻结 / 不删XRP / 不降3σ / 不改720h / 不扩10币与OOS / 不入库**。
- 归档：strategy_templates/research/oi-creation-efficiency-continuation/2023-2024-early-gate/。

## 2026-09-28 — DeFiLlama Chain TVL Price-Residual Flow
- 9条原生链；日频 TVL return 用前90日 beta 对本币 return 做 causal residual，最近7日 residual 求和；零上穿 LONG、零下穿 SHORT；下一UTC日open。
- 动态要求 Binance USD-M 已有>=2年历史。2023-2024 discovery：880事件/8币，7d **+0.0391%**，仅4/8币正。
- 2025 +0.1034%、5/9正；2026 +0.3116%、5/9正。后期增强但 discovery 广度/经济幅度不足。
- 结论：**冻结 / 不后验只留SUI-XRP-SOL / 不调90d beta或7d窗口 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-tvl-price-residual/2023-2026/。

## 2026-09-28 — DeFiLlama Chain Stablecoin Supply Growth：Feasibility
- 计划研究链级 stablecoin circulating USD 的7日增长零穿越，区别于已冻结的全市场 aggregate stablecoin supply。
- 2023-2024 同语义可用原生链仅 **7个：ETH/BNB/SOL/AVAX/ADA/NEAR/SUI**；Bitcoin 无对应 chain stablecoin endpoint，Ripple 历史直到 2025-04-02 才开始。
- 低于预注册最低覆盖8，**未查看收益即冻结 / 不加入无关链凑样本 / 不降低覆盖门槛 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-supply-growth/feasibility/。

## 2026-09-28 — KuCoin Futures Delist → Binance SHORT：Feasibility
- KuCoin 官方 Delistings CMS API 可完整分页，但当前历史最早仅到 2023-06-30。
- 2023-2024：28篇官方 Futures delist 公告，去重后43 contract-events；排除 Trading Bot/Earn/Margin 伴随公告与重复 Updated。
- 严格 Binance USD-M 历史>=2年 + 前24h QuoteVolume>=500万 后仅剩 **7事件：IOST/TOMO/OCEAN/GAL/MATIC/BLZ/FTM**。
- 低于最低 discovery 样本8，且官方API无法扩到2022；**未看收益即冻结 / 不降低门槛 / 不用搜索引擎补历史 / 不入库**。
- 归档：strategy_templates/research/kucoin-futures-delist-short/2023-2024-feasibility/。

## 2026-09-28 — KuCoin Spot Whole-Token Delist → Binance SHORT：Feasibility
- KuCoin 官方 Delistings CMS + 公告正文完整解析 2023-2024：34篇 whole-token/project 下架公告，240 token-events / 240 unique token。
- 排除 Earn/Trading Bot/Margin/ETF 伴随公告；token 从官方正文交易对段机械解析。
- 仅3个 token 在事件月仍有 Binance USD-M；严格历史>=2年 + 前24h QV>=500万 后仅 **ANT 1事件**。
- 结论：**未看收益即 feasibility 冻结 / 不降低门槛 / 不再扩同类外部Spot下架 / 不入库**。
- 归档：strategy_templates/research/kucoin-spot-whole-token-delist-short/2023-2024-feasibility/。

## 2026-09-28 — Kaufman Efficiency Ratio Continuation：Early Gate
- 标准1h ER(24)=净24h位移/24h总路径长度；ER从过去720h滚动中位数下方上穿时触发，方向跟随已完成24h return，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024 共5126事件；12h：BTC -0.0936%、ETH -0.0279%、BNB -0.0320%、XRP -0.0086%，**4/4均为负**；合计 -0.0406%。
- 结论：**early gate失败，冻结 / 不扩10币与OOS / 不事后改fade / 不入库**。
- 归档：strategy_templates/research/kaufman-efficiency-ratio-continuation/2023-2024-early-gate/。

## 2026-09-28 — Chaikin Money Flow Zero-Cross：Early Gate
- 标准1h CMF(20)；上穿0做LONG、下穿0做SHORT，下一1h open，不加趋势确认或其它阈值。
- BTC/ETH/BNB/XRP 2023-2024 共8290事件；12h **+0.0070%**，win49.1%；BTC +0.0111%、ETH +0.0148%、BNB -0.0044%、XRP +0.0054%。
- 远低于预注册 +0.10% gate，**冻结 / 不扩10币与OOS / 不调period或阈值 / 不入库**。
- 归档：strategy_templates/research/chaikin-money-flow-zero-cross/2023-2024-early-gate/。

## 2026-09-28 — Money Flow Index(14) Center Cross：Early Gate
- 标准1h MFI(14)，上穿50 LONG、下穿50 SHORT，下一1h open；不加20/80或趋势确认。
- BTC/ETH/BNB/XRP 2023-2024 共7814事件；12h **-0.0060%**，win48.9%；BTC/ETH略正，BNB/XRP为负。
- 结论：**early gate失败，冻结 / 不扩10币与OOS / 不再枚举OBV/CCI/RSI等相邻传统TA / 不入库**。
- 归档：strategy_templates/research/money-flow-index-center-cross/2023-2024-early-gate/。

## 2026-09-28 — Price–Volume Correlation Regime：Early Gate
- 1h log return 与 log QuoteVolume change 的24h Pearson correlation；rho由<=0上穿0时跟随已完成4h return，rho由>=0下穿0时反转4h return；下一1h open。
- BTC/ETH/BNB/XRP 2023-2024 共3938事件；1h -0.0097%、4h +0.0205%、12h **+0.0265%**，win49.4%；BTC/BNB/XRP略正，ETH近零。
- 远低于预注册 +0.10% gate，**冻结 / 不扩10币与OOS / 不只保留单侧regime / 不调correlation窗口 / 不入库**。
- 归档：strategy_templates/research/price-volume-correlation-regime/2023-2024-early-gate/。

## 2026-09-28 — Binance Seed Tag Addition：Feasibility
- Binance官方2023-2026完整标题审计：只有2023-07-26一次性“Introducing Seed Tags & Monitoring Tags”制度上线；之后Seed Tag相关公告均为 removal，没有独立新增批次。
- 同一制度上线时多个token不能拆成伪独立事件；独立batch远低于最低样本。
- 结论：**未看收益即 feasibility blocked / 不按token拆批次 / 不入库**。
- 归档：strategy_templates/research/binance-seed-tag-addition/2023-2026-feasibility/。

## 2026-09-28 — Binance Spot Tick-Size → USD-M Cross-Market Signal
- 预注册：BASE/USDT Spot tick变细→LONG USD-M，tick变粗→SHORT；官方实际生效时间；>=2年 + 24h QV>=500万。
- 2023-2024：107 symbol-events，严格 eligible **20事件/18币/9批次**；endpoint 12h signed mean +0.918%。
- frozen exact 1m：20笔、8TP/12SL，PF **0.8297**，net **-13.91%**，原双向 family 路径关失败。
- 分侧仅作 attribution：变细→LONG 13笔 2TP/11SL，PF **0.2711**、net -54.69%；变粗→SHORT 7笔 6TP/1SL，PF **7.1198**、net +40.77%。后者只能作为新生成假设，不能用 discovery 自证。
- 2025-2026 forward：14批、144个 BASE/USDT 调整，**144/144 全为tick下调，tick上调=0**，因此 SHORT 假设 forward 无事件可验证。
- 2021-2022 earlier-OOT：已锁定18篇官方文章，但 Binance CMS article-detail 当前统一HTTP错误；正文尚未重建，**未查看收益**。
- 结论：**原双向 family冻结；tick上调→SHORT仅保留待独立验证，不入库**。
- 归档：strategy_templates/research/binance-spot-tick-size-cross-market/2023-2026/。

## 2026-09-28 — Parkinson Range-vs-Close Variance Regime：Early Gate
- 最近24h Parkinson range variance / close-to-close realized variance；自然阈值1。上穿1反转已完成4h方向，下穿1跟随4h方向，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：3238事件；1h -0.0122%、4h约0、12h **+0.0056%**，win49.4%；BNB/XRP为负。
- 结论：**early gate失败，冻结 / 不换Garman-Klass/Yang-Zhang继续挖相同机制 / 不入库**。
- 归档：strategy_templates/research/parkinson-range-vs-close-variance-regime/2023-2024-early-gate/。

## 2026-09-28 — DeFiLlama Protocol TVL Shock
- unique protocol.symbol，排除 CEX/Chain/Canonical Bridge；日频 USD TVL log-return 相对前90个连续UTC日 causal z-score；首次 z<=-3，z>=-1 re-arm；>=2年 + signal-day QV>=500万。
- 预注册 SHORT discovery 2023-2024：严格 **96事件/16币**；1d -0.274%、3d -2.144%、7d **-4.283%**；2023/2024均为负，仅5/16币正，明确失败。
- discovery 生成的镜像 LONG 只允许去未见OOS：2025-2026严格 **46事件/16币**；1d +0.626%，3d -0.747%，7d **-5.813%**；2025 -5.882%、2026 -5.351%，仅6/16币正。
- 结论：**两个方向跨regime均不成立，整个family冻结 / 不按类别或z强度后验筛选 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-tvl-shock/2023-2026/。

## 2026-09-28 — DeFiLlama Protocol Fee Activity：Feasibility
- 40个老协议token中仅11个暴露 dimensions.fees；免费 fee-summary 当前只有 **5币：SPELL/GRT/ENS/AXS/LDO** 具备足够2023-2024历史。
- CVX/ANKR/RSR endpoint当前400；EOS/API3/LPT历史起点过晚。
- 低于新数据源最低跨币覆盖8，**未看收益即 coverage blocked / 不做5币特例 / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-fee-activity/feasibility/。

## 2026-09-29 — Wick Rejection Asymmetry：Early Gate
- 单根1h=(lower_wick-upper_wick)/(high-low)，最近24h取均值；上穿0 LONG、下穿0 SHORT，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024 共5095事件；1h -0.0083%、4h -0.0163%、12h **-0.0145%**，仅XRP略正。
- 结论：**early gate失败，冻结 / 不改窗口或反向 / 不入库**。
- 归档：strategy_templates/research/wick-rejection-asymmetry/2023-2024-early-gate/。

## 2026-09-29 — USDC-vs-USDT Perpetual Dislocation Catch-Up
- 固定信号：1h log(USDC-perp/USDT-perp) 相对 prior720h causal z-score；首次 |z|>=3，|z|<1 re-arm；USDC相对贵→LONG USDT，USDC相对便宜→SHORT USDT；下一完整1h open；目标USD-M历史>=2年。
- discovery 2024-2025：1154事件，1h +0.1561%、4h +0.1700%、12h +0.2158%，17/27币正；2024 +0.4418%，2025 +0.0525%。
- untouched 2026-01~08 OOS：578事件，1h +0.0927%、4h +0.1139%、12h +0.2849%，19/27币正，endpoint gate通过。
- frozen exact 1m（同一下一1h open、4x、TP8/SL6、fee0.0005/side、5bps/side、funding、12h、single-position/symbol）：1732 signals中32重叠跳过，1700 trades、0 data errors。
  - discovery exact：1140笔，471TP/529SL/140TIME，PF 0.9923，net -30.86%；
  - 2026 OOS exact：560笔，193TP/231SL/136TIME，PF 0.9177，net -151.77%；
  - 合计 PF 0.9688，net -182.64%；OOS LONG PF0.8634、SHORT PF0.9610，双侧均失败。
- 一致性审计：exact实际成交子集 endpoint 仍为正（discovery 12h +0.2156%、OOS +0.2864%）；OOS gross（已含双边5bps滑点、未扣fee）PF 1.0421、平均 +0.1294%，但4x双边手续费平均约0.3999%/笔，funding近零。结论是真实成本裕度不足，而非信号/时间对齐错误。
- 结论：冻结 / 不降成本假设 / 不调z/window/leverage / 不只留majors或单方向 / 不入库。
- 归档：strategy_templates/research/usdc-usdt-perpetual-dislocation-catchup/2024-2026/。

## 2026-09-29 — Realized Kurtosis Reversal：Early Gate
- 最近24个1h log return raw kurtosis 从<=3上穿>3时触发，反转已完成24h方向，下一1h open；3为高斯自然基准。
- BTC/ETH/BNB/XRP 2023-2024：2422事件；1h +0.0033%、4h -0.0131%、12h **-0.0315%**；BTC/ETH/BNB均负，仅XRP略正。
- 结论：**early gate失败，冻结 / 不后验改continuation / 不调window或阈值 / 不入库**。
- 归档：strategy_templates/research/realized-kurtosis-reversal/2023-2024-early-gate/。

## 2026-09-29 — KuCoin Earn Delist → Binance SHORT：Feasibility
- 2023-2024官方Earn下架全集：24篇，机械解析34 token-events；严格>=2年 + QV>=500万后仅7事件：APE/GRT/GAL/DAR/BAND/IOST/ALGO。
- 低于最低样本8，**未看收益即冻结 / 不降低门槛 / 不入库**。
- 归档：strategy_templates/research/kucoin-earn-delist-short/2023-2024-feasibility/。

## 2026-09-29 — KuCoin Earn Addition → Binance LONG
- 完整KuCoin Earn关键词档案998篇/50页；严格语义筛选35篇2023-2024新增Saving/Staking文章；官方产品表完整解析172 token-events / 166 unique，35/35数量校验通过。
- 严格>=2年 + QV>=500万后50事件/49币/14批。看收益前锁定2023 discovery、2024 untouched OOS。
- 2023 discovery 44事件：1h **-0.1723%**、4h **-0.1080%**、12h **-0.7185%**；batch-equal 12h **-0.1737%**，仅4/11批次正；43币仅18正。
- discovery明确失败，**不查看2024六个OOS事件 / 不反向SHORT / 冻结 / 不入库**。
- 归档：strategy_templates/research/kucoin-earn-addition-long/2023-2024/。

## 2026-09-29 — Snapshot Governance Value-Accrual：Feasibility
- 仅用DeFiLlama governanceID安全映射8个成熟Snapshot spaces；2023-2024完整626 proposals。
- 以proposal创建时刻为因果事件；严格fee-switch/token-buyback/token-burn/revenue-share/protocol-fee-distribution语义只剩6事件/3币（LDO/LRC/APE）；emissions/inflation严格事件=0。
- verified Snapshot ticker-only扩 universe 因语义映射不安全被拒绝。
- **低于最低样本8，未看收益即冻结 / 不入库**。
- 归档：strategy_templates/research/snapshot-governance-value-accrual/2023-2024-feasibility/。

## 2026-09-29 — Spot-vs-Perp Average Trade Notional Divergence：Early Gate
- 1h log((futures QuoteVolume/trade_count)/(spot QuoteVolume/trade_count)) 相对 prior720h causal z-score；首次 |z|>=3、|z|<1 re-arm；方向跟随触发小时 futures return，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：397事件；1h -0.0172%、4h -0.0652%、12h **-0.0183%**；BTC/XRP略正，ETH/BNB负。
- 结论：**early gate失败，冻结 / 不反向、不调z/window / 不入库**。
- 归档：strategy_templates/research/spot-perp-average-trade-notional-divergence/2023-2024-early-gate/。

## 2026-09-29 — Spot-vs-Perp Intrabar Range Excess：Feasibility
- 1h E=log(log(H/L)_futures / log(H/L)_spot)，prior720h causal z-score；预注册 z>=3 首触发、z<1 re-arm，反转触发小时 futures return。
- 时间戳对齐正常，BTC/ETH/BNB/XRP 各16113有效小时；2023-2024最大z仅 **1.4621 / 1.7653 / 0.6528 / 1.0393**，固定3σ条件 **0事件**。
- 结论：**no-event feasibility冻结 / 不降3σ制造样本 / 不继续相邻range变体 / 不入库**。
- 归档：strategy_templates/research/spot-perp-intrabar-range-excess/2023-2024-feasibility/。

## 2026-09-29 — COIN-M Liquidation Exhaustion Reversal：OOS Feasibility Recheck
- 2023H2/2024 continuation失败后曾明确生成“liquidation exhaustion reversal”假设，并规定只能用未来未见数据验证。
- 2026-09-29重新审计 Binance Vision：原9币 liquidationSnapshot 均停在 **2024-10-10~14**；BTC/ETH/DOT到10-14，XRP/ADA/LINK/LTC到10-13，BCH到10-10，BNB到10-14；**2025+ 文件=0**。
- 结论：**historical-OOS blocked / 不用旧样本反向自证 / 不入库**。
- 归档：strategy_templates/research/coinm-liquidation-exhaustion-reversal/2025-2026-feasibility/。

## 2026-09-29 — KuCoin Futures Tick-Size → Binance USD-M
- KuCoin官方2024-2026完整 tick-size 公告全集：28独立批次、282 contract-events、0解析失败；**282/282均为tick下调**，因此预注册family实际只有LONG。
- 严格 Binance >=2年 + 前24h QV>=500万：2024-2025 discovery **20事件/20币/11批**；2026 untouched OOS **15事件/15币/5批**。
- discovery：1h -0.0680%、4h +0.4111%、12h **-0.6555%**；2024 -0.2142%、2025 -0.7333%；仅 **3/11批次**、**6/20币** 12h为正。
- discovery gate明确失败，**不查看2026 OOS收益 / 不事后改SHORT / 冻结 / 不入库**。
- 归档：strategy_templates/research/kucoin-futures-tick-size-cross-market/2024-2026/。


## 2026-09-29 — Wick Rejection Asymmetry：Early Gate
- 1h `(lower wick - upper wick)/(high-low)` 零穿越；上穿0 LONG、下穿0 SHORT，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024 共33553事件；1h -0.0124%、4h -0.0128%、12h **-0.0039%**，win49.4%；BTC/ETH/BNB均负。
- 结论：**early gate失败，冻结 / 不加wick强度或body/volume过滤 / 不入库**。
- 归档：strategy_templates/research/wick-rejection-asymmetry/2023-2024-early-gate/。

## 2026-09-29 — Aggressor Buy/Sell Average Trade-Size Asymmetry：Data-Cost Feasibility
- 该机制区别于总Average Trade Size Shock与q99 Large Aggressor Flow；拟测 `log(avg aggressive-buy notional / avg aggressive-sell notional)` 的720h causal z-score。
- 本地 market_trades 对 BTC/ETH/BNB/XRP 均为0行；若做2023-2024 gate需大体量下载 aggTrades。
- **未看收益即 data-cost blocked**；不是alpha失败，暂不为单一未验证信号下载多年原始成交档。
- 归档：strategy_templates/research/aggressor-trade-size-asymmetry/data-cost-feasibility/。

## 2026-09-29 — Lo-MacKinlay Variance Ratio Regime：Early Gate
- 最近96个1h return 的 VR(4)=Var(overlapping 4h return)/(4×Var(1h return))；自然阈值1。上穿1跟随已完成4h方向，下穿1反转4h方向，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：2177事件；1h -0.0174%、4h +0.0012%、12h **-0.0228%**；BTC/XRP为负。
- 结论：**early gate失败，冻结 / 不扫q=2/8或窗口 / 不入库**。
- 归档：strategy_templates/research/variance-ratio-regime/2023-2024-early-gate/。

## 2026-09-29 — Semivariance Regime：Early Gate
- 最近48个1h return：SV+/SV- 自然阈值1；上穿1 LONG、下穿1 SHORT，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：3225事件；1h -0.0153%、4h -0.0069%、12h **-0.0482%**，win48.0%；仅BNB近零。
- 结论：**standalone early gate失败，冻结 / 不改窗口、不加70%阈值、不反向 / 不入库**。
- 归档：strategy_templates/research/semivariance-regime/2023-2024-early-gate/。

## 2026-09-29 — Volatility Clustering Regime：Early Gate
- 最近48个1h squared log return 的lag-1 autocorrelation；零为自然阈值。上穿0跟随已完成4h方向，下穿0反转4h方向，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：2936事件；1h -0.0278%、4h -0.0480%、12h **+0.0193%**，win49.3%；ETH/BNB/XRP略正但幅度不足。
- 结论：**低于+0.10% gate，冻结 / 不改成abs-return、不调窗口/阈值 / 不入库**。
- 归档：strategy_templates/research/volatility-clustering-regime/2023-2024-early-gate/。

## 2026-09-29 — Runup-vs-Drawdown Excursion Asymmetry：Early Gate
- 最近24个1h close：最大有序trough→peak log-runup / 最大有序peak→trough log-drawdown；自然阈值1，上穿1 LONG、下穿1 SHORT，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：3211事件；1h +0.0227%、4h +0.0252%、12h **+0.0213%**，win47.3%；ETH为负，BTC/BNB近零，主要由XRP贡献。
- 结论：**远低于+0.10% gate，冻结 / 不调窗口或阈值 / 不入库**。
- 归档：strategy_templates/research/runup-drawdown-excursion-asymmetry/2023-2024-early-gate/。

## 2026-09-29 — Extreme Return Tail Asymmetry：Early Gate
- 最近24个1h return：max positive / abs(max negative)，自然阈值1；上穿1 LONG、下穿1 SHORT，下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：3106事件；1h +0.0027%、4h +0.0123%、12h **-0.0031%**，win46.7%；BTC/ETH/BNB均负，仅XRP略正。
- 结论：**early gate失败，冻结 / 不加极端幅度阈值、不调窗口 / 不入库**。
- 归档：strategy_templates/research/extreme-return-tail-asymmetry/2023-2024-early-gate/。


## 2026-09-29 — KuCoin Margin Addition → Binance LONG：Feasibility
- KuCoin官方 margin trading 完整CMS档案：2023-2024共97篇新增文章，机械解析 **105 token-events / 105 unique**，97/97文章成功解析。
- 62事件在当月存在 Binance USD-M；严格历史>=2年 + 前24h QV>=500万 后仅 **TRB、MTL 2事件**。
- 结论：**低于最低样本8，未看收益即冻结 / 不降低生产门槛 / 不入库**。
- 归档：strategy_templates/research/kucoin-margin-addition-long/2023-2024-feasibility/。

## 2026-09-29 — KuCoin Futures Leverage/Risk-Limit → Binance USD-M：Discovery
- 官方before/after表机械解析；非下降且至少一项上升→LONG，非上升且至少一项下降→SHORT，mixed排除；公告后下一完整1h open。
- 严格>=2年 + QV>=500万后 **112事件/86币/12批**，LONG97、SHORT15。
- discovery 2023-2025：1h **-0.0617%**、4h **-0.6097%**、12h **-1.0978%**；batch-equal 12h **-0.8755%**，仅5/12批正。
- 年度：2023 +0.6359%，2024 -2.0438%，2025 -3.7504%；仅28/86币平均为正；LONG/SHORT两侧均为负。
- 结论：**discovery gate明确失败，冻结 / 不解析2026 OOS / 不拆方向救活 / 不入库**。
- 归档：strategy_templates/research/kucoin-futures-risk-limit-cross-market/2023-2025-discovery/。

## 2026-09-29 — Binance Spot Step-Size Adjustment：Feasibility
- 已审计2021-2024 Spot tick-size公告集中，仅 **2021-08-12** 一篇标题包含 Step Size；独立批次上限=1。
- **未看收益即样本不足冻结 / 不拆同批token伪造独立样本 / 不入库**。
- 归档：strategy_templates/research/binance-spot-step-size-adjustment/2021-feasibility/。

## 2026-09-29 — KuCoin Corporate-Action Support → Binance LONG：Feasibility
- KuCoin官方完整关键词CMS档案机械筛选 swap/migration/rebranding 首次支持公告：2023-2024共81篇、82 token-events / 80 unique。
- 严格 Binance USD-M 历史>=2年 + 前24h QV>=500万后仅 **5事件：SXP/EGLD/ALPHA/NEO/MATIC**。
- 低于最低样本8，**未看收益即冻结 / 不降低门槛 / 不入库**。
- 归档：strategy_templates/research/kucoin-corporate-action-support-long/2023-2024-feasibility/。

## 2026-09-29 — KuCoin Deposit+Withdrawal Closure → Binance SHORT
- KuCoin官方2023-2024同时关闭 deposit+withdrawal 全集：49事件/43 unique token；严格>=2年 + QV>=500万后 **12事件/11币**，全部位于2023。
- 预注册SHORT、下一完整1h open；discovery：1h -0.0187%、4h **-0.5548%**、12h **-0.3821%**；event win58.3%，7/11币均值正。
- 虽breadth勉强，但经济均值为负且未过+0.50% gate，**冻结 / 不查看后续OOS / 不对同一关闭事件事后反向LONG / 不入库**。
- 归档：strategy_templates/research/kucoin-deposit-withdrawal-closure-short/2023-2024/。

## 2026-09-29 — DeFiLlama Chain DEX-Volume Residual Flow
- daily DEX-volume log return 用前90日 causal beta 对 native-token return 做 residual，最近7日 residual sum 零穿越；正穿LONG、负穿SHORT，下一UTC日open；动态>=2年。
- 2023-2024 discovery：**2334事件/12币，7d +0.3447%，9/12币正**，通过预注册 +0.25% / 60% breadth gate。
- 2025 OOS：1521事件/14币，7d仅 **+0.0551%**，8/14正（57.1%）；2026 OOS：1027事件/14币，7d **-0.0657%**，6/14正。
- 结论：**OOS明显衰减并在2026翻负，冻结 / 不按链后验筛选 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-dex-volume-residual/2023-2026/。

## 2026-09-29 — DeFiLlama Chain Fee Residual Flow
- 与DEX-volume residual同一因果框架：daily chain-fee log return 用prior90d beta对native-token return残差化；最近7日 residual sum 零穿越，正穿LONG/负穿SHORT，下一UTC日open，动态>=2年。
- 2023-2024 discovery：**1892事件/12币，7d +0.1788%，8/12币正**；breadth通过，但低于预注册 +0.25% economic gate。
- 复用脚本机械计算了2025/2026字段，但这是 discovery gate 检查前的实现失误；后续值仅保留审计，**不允许用于救活/调参/反向**。
- 结论：**discovery冻结 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-fee-residual/2023-2024-discovery/。


## 2026-09-29 — COIN-M Quarterly Expiry Basis Signal
- 事件：COIN-M季度合约08:00 UTC到期；信号取到期前最后完整1h的 log(quarterly/perp)；严格>=2年 + 24h QV>=500万。
- 预注册 unwind：正basis→SHORT USD-M、负basis→LONG，08:00 open入场。2023-2024 **72事件/9币/8批**：1h -0.418%、4h -0.975%、12h **-0.583%**；两年均负，仅3/9币、2/8批正。
- discovery 生成镜像 continuation 只去未见2025 OOS：**31事件/9币**，12h **-0.564%**，仅1/9币正。
- 结论：**两方向都失败，整个family冻结 / 不调basis幅度或延迟入场 / 不进exact / 不入库**。
- 归档：strategy_templates/research/coinm-quarterly-expiry-basis-signal/2023-2025/。

## 2026-09-29 — DeFiLlama Chain Bridge Netflow：Data-Access Feasibility
- 机制：链级跨链桥净流入应支持原生币、净流出形成压力；预注册拟用最近7个UTC日 `depositUSD-withdrawUSD` 合计零穿越，正穿LONG、负穿SHORT，下一UTC日open；要求至少8个production-eligible原生链。
- 在**未查看任何收益前**检查历史源：`bridges.llama.fi/bridgevolume/Ethereum` 与 bridge catalog 当前均返回 **HTTP 402 Payment Required**；alternate `api.llama.fi` bridgevolume 路径也无法得到可用历史响应。
- 结论：**data-access blocked / 不抓取可视化页面拼残缺历史 / 不降低完整可审计数据要求 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-bridge-netflow/feasibility/。

## 2026-09-29 — KuCoin Deposit+Withdrawal Resumption → Binance LONG：Feasibility
- 预注册机制：外部交易所同时恢复 crypto deposit+withdrawal 后，跨所套利/流动性通道重新开放，固定 LONG Binance USD-M，下一完整1h open；先要求>=2年 + 24h QV>=500万。
- 完整 KuCoin CMS “Deposit and Withdrawal Services” 语料 **1215篇**；2023-2024 resumption/reopen 标题候选 **36篇**。
- 机械分类后 **35/36** 是 token swap/migration/rebranding 完成公告，属于已冻结 corporate-action family，必须排除；唯一非 corporate-action 是 **PIX法币**充提维护恢复，不对应crypto USD-M。
- 因此独立 crypto resumption 事件 **0个**，**未看收益即 feasibility失败 / 不把corporate-action completion重复包装成新family / 不入库**。
- 归档：strategy_templates/research/kucoin-deposit-withdrawal-resumption-long/2023-2024-feasibility/。

## 2026-09-29 — DeFiLlama Protocol DEX Market-Share Flow
- 完整 universe 由 DefiLlama DEX protocol symbol 与 Binance Vision USD-M 历史目录机械求交：59 ticker；只保留到2026-08可能满足>=2年的合约，并在看收益前排除 CHR/FLOW/LIT/OMNI 4个明确 ticker 身份碰撞，最终研究 universe 22币。
- 固定信号：协议全部 DEX module 的7日成交额 / 全市场DEX 7日成交额，与前7日市场份额比较；份额增速零上穿 LONG、零下穿 SHORT，下一UTC日open；>=2年且 signal-day QuoteVolume>=500万。
- discovery 2023-2024：**607事件/10个实际触发币**；1d **-0.1161%**、3d **-0.5286%**、7d **-0.7074%**，仅 **2/10币** 7d均值为正。
- 分年：2023 7d **-0.4592%**，2024 **-0.9934%**，连续两年为负；预注册 gate（mean7>=+0.25%、breadth>=60%、两年均正）明确失败。
- 结论：**discovery冻结 / 不查看2025-2026 OOS / 不事后反向 / 不调7d窗口或阈值 / 不进exact TP8/SL6 / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-dex-market-share/2023-2026/。

## 2026-09-29 — DeFiLlama Protocol DEX Share Momentum
- 机制：按 DeFiLlama protocol ID→symbol 聚合同一协议代币的全部 DEX adapter；计算 trailing-7d protocol DEX volume / trailing-7d global DEX volume，再取该 share 的 7d log-change 零穿越；正穿 LONG、负穿 SHORT，下一 UTC 日 open。
- 严格动态资格：USD-M 历史>=2年、signal-day QuoteVolume>=500万；2023-2024 discovery 最终 **519事件/8币**。
- canonical discovery：1d **-0.1460%**、3d **-0.2756%**、7d **-0.5292%**，仅 **1/8币** 7d 均值为正；远低于预注册 +0.25% / 60% breadth gate。
- 身份审计发现 LIT ticker collision：DeFiLlama LIT=Lighter，而 Binance 历史 LITUSDT=Litentry；含 LIT 的首次 run 已移入 legacy，剔除后失败更明确。
- 结论：**discovery gate 明确失败，冻结 / 不读取2025-2026 OOS / 不反向 / 不扫窗口 / 不筛 UNI 等子集 / 不进 exact / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-dex-share-momentum/2023-2026/。

## 2026-09-29 — Snapshot Governance Rejection → Binance SHORT：Feasibility
- 复用 governanceID 安全映射的 Snapshot 完整 626 proposals；只接受 exactly-2 choices，且能机械识别一正一负（For/Against、Yes/No、Approve/Reject 等）的干净二元提案。
- 2023-2024 共 **43** 个 clean binary proposals，但真正“负面选项胜出”仅 **2 个，且全部是 LDO/2023**。
- 低于最低 discovery 样本8且无跨币 breadth，**未查看任何收益即冻结 / 不放宽为反对率阈值 / 不纳入模糊多选语义 / 不入库**。
- 归档：strategy_templates/research/snapshot-governance-rejection-short/2023-2024-feasibility/。

## 2026-09-29 — DeFiLlama Protocol DEX Market-Share Flow
- 机制：按 DeFiLlama protocol token 聚合同一代币的全部 DEX 版本成交量，再除以全市场 DEX 日成交量得到协议 market share；最近7日平均份额 / 前7日平均份额的 log change 零穿越，正穿 LONG、负穿 SHORT，下一 UTC 日 open；动态 USD-M 历史>=2年 + signal-day QV>=500万。
- 在看收益前冻结 discovery=2023-2024，gate=7d signed mean >= +0.25% 且 >=60% 有事件币种均值为正；不扫阈值、不按协议后验筛选、不反向。
- discovery：**519事件 / 8个有事件币**；1d **-0.1242%**、3d **-0.3934%**、7d **-0.4786%**，仅 **2/8币** 7d均值为正。RAY/DODO 保留在机械 universe，但严格连续数据/零穿越规则下 discovery 事件=0。
- 结论：**discovery gate 明确失败，冻结 / 2025-2026 OOS 未计算 / 不事后反向 SHORT / 不进 exact TP8/SL6 / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-dex-market-share/2023-2026/。

## 2026-09-29 — DeFiLlama Protocol Derivatives Market-Share：Data-Access Feasibility
- 机制设想：去中心化永续/衍生品协议日成交量占全市场 derivatives volume 的份额变化，作为协议使用量与手续费来源竞争力变化，映射协议代币；先要求完整 aggregate denominator + protocol-level 历史，再冻结信号。
- 在**未查看任何收益前**检查数据源：DeFiLlama 公共 protocols 目录仍能看到 dYdX/GMX/Gains/Synthetix 等 derivatives adapter，但 /overview/derivatives 与实测的 /summary/derivatives/{adapter} 均统一返回 **paid API required**。
- 因无法从当前免费公开 API 机械重建完整且可审计的历史 market share，**data-access blocked / 不抓可视化页面 / 不拼零散第三方历史 / 不看 discovery/OOS / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-derivatives-market-share/feasibility/。

## 2026-09-29 — DeFiLlama Chain Stablecoin Supply Growth：Coverage Audit Correction → Discovery
- 修正旧 feasibility：原 2026-09-28 target-chain audit 漏了 TRX/Tron、Polygon/MATIC、Fantom/FTM 等同语义原生链；旧阶段**未查看收益**，因此可做 outcome-independent universe 修正。修正后同一15-chain映射中12链具备2023-2024 stablecoin历史；事件层继续严格动态 USD-M 历史>=2年 + signal-day QV>=500万。
- 冻结信号：g_t=log(chain totalCirculatingUSD.peggedUSD_t / supply_t-7d)；上穿0 LONG、下穿0 SHORT，下一UTC日open；discovery gate 在看收益前固定为 7d signed mean >= +0.25% 且 >=60% 有事件币种均值为正。
- 2023-2024 discovery：**1086事件 / 11个有事件币**；1d **-0.1924%**、3d **-0.0759%**、7d **+0.1777%**；**7/11币（63.6%）** 7d均值为正。
- breadth gate通过，但 economic gate +0.25% 未通过。结论：**冻结 / 2025-2026 OOS 未计算 / 不调增长窗口或幅度阈值 / 不按链筛选 / 不反向 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-supply-growth/2023-2026/；旧 feasibility README 已标记 superseded audit。

## 2026-09-29 — DeFiLlama Chain Stablecoin Velocity
- 机制：链级 weekly velocity = 7日 DEX volume sum / 同7日 stablecoin supply mean；flow=log(最近7日velocity / 前7日velocity)，零上穿LONG、零下穿SHORT，下一UTC日open；动态>=2年 + signal-day QV>=500万。不是单独DEX流量或stablecoin增长。
- discovery gate 在看收益前冻结为 7d signed mean >= +0.25% 且 >=60% 有事件币种均值为正；不扫窗口/阈值、不做symbol-specific过滤。
- 2023-2024 discovery：**806事件/11币，7d +1.0474%，10/11币正**（1d +0.1142%、3d +0.5436%），强通过 discovery。
- untouched 2025 OOS：**482事件/12币，7d -0.0218%**，8/12币正，经济edge基本消失并略负。
- 2026 OOS：**360事件/12币，7d +0.2701%**，但仅 **6/12币正**；幅度回升但breadth失败。
- 结论：**跨regime不稳定，冻结 / 不进exact TP8/SL6 / 不删弱币救活 / 不调窗口或velocity阈值 / 不反向 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-velocity/2023-2026/。

## 2026-09-29 — DeFiLlama Chain Bridge Netflow：Open-Source Recheck
- 对既有 402 blocker 做源码级复核：DeFiLlama `bridges-server`（rev dd37313f...）确认历史资金流来自 Postgres `bridges.daily_volume/hourly_volume`，且源码存在按 timestamp 查询历史日统计的 `bridgedaystats` 路由以及 day/week/month netflows。
- 但部署端 `netflows`、`bridgedaystats`、`bridgevolume` 实测均统一 **HTTP 402 paid API required**；公开 `llama-bridges-data` S3 bucket 禁止 list（403），已知可读 `lastRecordedBlocks.json/recordedBlocks.json` 只有区块进度、不含历史USD资金流；数据库日备份使用另一个可配置bucket且未声明public-read。
- 若自行重建需回放大量 bridge adapters/链上日志，成本相当于重新构建历史数据库。结论：**仍 data-access/data-cost blocked / 未看收益 / 不抓前端残缺数据 / 不入库**。
- 归档更新：strategy_templates/research/defillama-chain-bridge-netflow/feasibility/。

## 2026-09-29 — Spot-vs-Perp Trade-Count Share Divergence：Early Gate
- 机制：1h `log(perp trade_count / spot trade_count)` 对 prior720h 做 causal z-score；首次 |z|>=3、|z|<1 re-arm。Perp交易到达异常占优视为杠杆拥挤→SHORT；Spot异常占优→LONG；下一1h open。
- BTC/ETH/BNB/XRP 2023-2024：**341事件**；1h +0.0116%、4h +0.0362%、12h **-0.0888%**，win12 47.2%；仅 **1/4币** 12h均值为正（ETH +0.2422%，BTC/BNB/XRP均负）。
- 结论：**early gate明确失败，冻结 / 不反向 / 不降3σ / 不改720h窗口 / 不扩10币与OOS / 不进exact / 不入库**。
- 归档：strategy_templates/research/spot-perp-trade-count-share-divergence/2023-2024-early-gate/。

## 2026-09-29 — Gate USDT Futures Insurance-Fund Stress：Historical Feasibility
- 机制：外部交易所保险基金异常净流出代表清算/穿仓压力，拟作为 Binance USD-M 的系统性去杠杆信号；先只审计官方历史数据，不看收益。
- Gate 官方 `GET /api/v4/futures/usdt/insurance` 实测：`limit=100` 与 `limit=1000` 都只返回 **31个日点**，范围 **2026-08-30~2026-09-29**；接口没有 from/to 历史参数。
- Binance 官方 `/fapi/v1/insuranceBalance` 也只是当前 snapshot，无历史时间参数。
- 结论：**historical-data blocked / 未看收益 / 可保留未来forward采集价值 / 不用非官方scrape回填 / 不入库**。
- 归档：strategy_templates/research/gate-futures-insurance-fund-stress/feasibility/。

## 2026-09-29 — Gate Futures Liquidation/OI Contract Stats：Historical Feasibility
- Gate 官方 `contract_stats` 明确暴露 `long_liq_usd`、`short_liq_usd`、`open_interest`，机制上可用于跨交易所清算压力信号。
- 在**未看任何收益前**直接请求 BTC_USDT/ETH_USDT 的 2023-01-01、BTC_USDT 的 2024-01-01 历史，全部返回 **`from time exceeds 180-day limit`**。
- 结论：**仅最近180天可用，无法满足2023-2024 discovery + later OOS，historical-OOS blocked / 可用于未来forward采集 / 不降低研究协议 / 不入库**。
- 归档：strategy_templates/research/gate-futures-liquidation-stats/feasibility/。

## 2026-09-29 — GitHub Core Release Catalyst：Discovery
- 外生事件源：为每个 token 预先固定 canonical core-node/client GitHub repository；只接受 `draft=false`、`prerelease=false` 且有 `published_at` 的 GitHub Release，同币同UTC日多 release 折叠到最早发布时间；不以tag/commit补事件。
- 固定方向 LONG，release 后下一完整 Binance USD-M 1h open 入场；事件时历史>=2年、前24h QuoteVolume>=500万。discovery=2023-2024；冻结 gate：>=80事件、>=8币、12h mean>=+0.25%、>=60%币正、且2023/2024都为正。
- discovery：**257事件/10币**；1h -0.0207%、4h +0.1653%、12h **+0.1364%**，**7/10币** 12h均值为正；symbol-equal 12h +0.1232%。
- 年度：2023 12h **+0.3607%**；2024 **-0.0018%**，跨年 gate 失败且整体经济幅度低于 +0.25%。
- 结论：**discovery冻结 / 2025-2026 OOS保持未读 / 不事后反向SHORT / 不换repo、不做semver/title筛选 / 不进exact / 不入库**。
- 归档：strategy_templates/research/github-core-release-catalyst/2023-2026/。

## 2026-09-29 — Aggressor Buy/Sell Average Trade-Size Asymmetry：Blocker Relief → 2024 Early Gate
- 旧状态是 data-cost blocked；本轮用**看收益前冻结**的 TRX/ATOM/LTC/UNI 4个成熟且2024 aggTrades档案体量可控的币解除 blocker，没有先下载 BTC/ETH 多年大档。
- 每小时按 aggTrade 的 `last_trade_id-first_trade_id+1` 还原raw trade count；buyer_is_maker=false/true分别为主动买/卖，计算两侧平均notional，`A=log(avg_buy_notional/avg_sell_notional)`；prior720h causal z，首个 z>=3 LONG、z<=-3 SHORT，|z|<1 re-arm，下一1h open。
- 冻结 gate：>=20事件、12h signed mean>=+0.10%、>=3/4币正。实际 **288事件**；1h +0.0517%、4h +0.0625%、12h **-0.0326%**，win12 50.69%，仅 **2/4币** 12h均值为正（TRX/UNI正，ATOM/LTC负）。
- 结论：**blocker已解除但alpha early gate失败，整个family冻结 / 不再扩BTC/ETH或多年下载 / 不改z/window/direction / 不看OOS / 不进exact / 不入库**。
- 归档：strategy_templates/research/aggressor-trade-size-asymmetry/2024-early-gate/。

## 2026-09-29 — DeFiLlama Stablecoin Borrow-Cost / Leverage-Demand：Feasibility
- 机制：链级 stablecoin 借款利率/利用率作为杠杆美元需求代理；先只检查可审计历史源，不看收益。
- 免费 DeFiLlama yields pool catalog 与普通 `chart/{pool}` 可访问，但历史仅有 deposit-side APY/TVL；真正的 borrow APY/borrow history `chartLendBorrow/{pool}` 实测 **HTTP 402 paid API required**。
- 用 deposit APY 替代会改变机制；按当前仍存活 pool 手工重建历史还会引入 survivorship/pool-selection bias。
- 结论：**data-access/data-quality blocked / 未看收益 / 不用存款收益替代 / 不手选当前pool / 不进 discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/defillama-stablecoin-borrow-cost/feasibility/。

## 2026-09-29 — DeFiLlama Token-Holder Revenue Flow：Boundary Audit Correction
- 新机制只使用直接回流 token holder/staker 的经济价值：fee share、buyback、burn、staking/locked-governance distribution；pre-return source map 与 identity/methodology 排除规则保持原样。
- 审计发现初始 discovery 允许 2024-12-25~31 信号的 7d endpoint 落入 2025-01-01~07，破坏 discovery→untouched OOS 边界；旧 discovery/OOS 输出已保留为 `legacy_boundary_leak_*`，不再用于晋级判断。
- canonical 修正：discovery 事件必须完整 7d endpoint <= 2024-12-31；冻结信号、方向、7v7窗口、>=2年与 QV>=500万门槛均不变。
- 修正后 2023-2024 discovery：**547事件/10币**；1d +0.2449%、3d +0.2436%、7d **+1.2513%**；2023 **+0.9338%**、2024 **+1.5326%**，但仅 **5/10币** 7d均值为正，breadth **50% < 60%**。
- 因 canonical discovery breadth gate 已失败，协议上**不允许进入 OOS**；此前生成的 2025~2026 OOS 仅保留 legacy 审计，不得作为候选证据。
- 结论：**discovery冻结 / 不用较窄后验 universe 救活 / 不删 KNC/OGN 等弱币 / 不调7v7或方向 / 不进exact TP8/SL6 / 不入库**。
- 归档：strategy_templates/research/defillama-token-holder-revenue-flow/2023-2026/。

## 2026-09-29 — DeFiLlama CEX Token Reserve / Inflow：Feasibility
- 机制：CEX token balance / net inflow 代表可立即卖出的交易所供给变化，拟作为同币 Binance USD-M 的外生供给压力信号；本阶段只审计历史源，**未看收益**。
- 免费 `api.llama.fi/cexs` 可访问，但 Binance 条目只有当前 `currentTvl/cleanAssetsTvl` 与滚动 `inflows_24h/1w/1m` 等交易所级聚合快照，没有历史序列或 per-token balance history。
- 官方 API docs 审计：free OpenAPI 共31个 path，**没有 CEX history/reserve endpoint**；Pro OpenAPI 才有 `GET /api/inflows/{protocol}/{timestamp}`，匿名调用返回 API-key error。
- 结论：**data-access blocked / 不抓前端私有缓存、不拼不完整历史 / 不进 discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/defillama-cex-token-reserve-flow/feasibility/。

## 2026-09-29 — Binance Funding Cap/Floor Adjustment：Feasibility
- 完整分页 Binance 官方 CMS catalog 49：**4436篇**，其中 funding 相关标题 **90篇**；本阶段只审计事件覆盖，未看收益。
- 为避免和已研究的 leverage/margin-tier、funding-interval family 混淆，只接受**独立 cap/floor 调整**；同时修改 leverage、margin tier 或 funding settlement frequency 的公告全部排除，市场级统一规则也只算一个batch。
- 2023-2024 真正干净的 standalone capped-funding 事件仅 **2023-10-01 OGNUSDT 1批**；2023-09-28 全市场规则同时涉及 settlement-frequency 且不是独立token事件。
- 结论：**样本<8，未看收益即冻结 / 不把混杂 leverage-tier 公告重复包装成 funding alpha / 不拆全市场规则伪造样本 / 不入库**。
- 归档：strategy_templates/research/binance-funding-cap-floor-adjustment/feasibility/。

## 2026-09-29 — DeFiLlama Protocol Value-Accrual Revenue：Feasibility
- 机制：用协议 `ProtocolRevenue` / `HoldersRevenue` 直接价值回流，而不是gross fees，作为协议代币结构性现金流信号；先做coverage，不看收益。
- 复用此前安全映射的成熟协议token；免费 fees-summary 明确支持 `dailyProtocolRevenue` / `dailyHoldersRevenue`。
- 2023-2024 有完整非零 ProtocolRevenue 历史的仅 **LDO、ENS、AXS 3币**；LPT HoldersRevenue 也只从 **2024-08-14** 起，discovery期仅139个非零日；其它映射币为0、无序列或2025+才开始。
- 结论：**低于最低8币，未看收益即 coverage blocked / 不用gross fees或supply-side revenue替代 / 不进 discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-value-accrual-revenue/feasibility/。


## 2026-09-29 — DeFiLlama Chain Stablecoin Depeg Stress：Feasibility
- 机制：链级 USD stablecoin 的 supply-weighted 历史价格压力；DeFiLlama `stablecoincharts/{chain}` 同时提供 `totalCirculating.peggedUSD` 与 `totalCirculatingUSD.peggedUSD`，两者比值可还原历史加权peg。语义验证中 2023-03-12 Ethereum 比值约 **0.9764**，真实重现 USDC 脱锚。
- 看收益前冻结事件：首次 ratio<=0.995 触发、恢复到>=0.999才 re-arm，固定 SHORT 原生币；为防小池噪声，事件日链上 peggedUSD 名义供给>=**5000万美元**，并要求 Binance USD-M 历史>=2年；coverage 最低8币且至少8个独立UTC日期。
- 原始2023-2024覆盖：57 chain-events / 12币 / 46个UTC日期；加入固定 5000万美元 + >=2年质量门后仅 **34事件 / 7币 / 27日期**，剩 ETH/BNB/SOL/AVAX/NEAR/MATIC/FTM。
- 结论：**7<8，未看任何收益即 coverage blocked / 不撤回5000万美元质量门、不加入未成熟OP/APT/ARB/SUI、不把一次跨链USDC脱锚拆成伪独立alpha / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-depeg-stress/feasibility/。

## 2026-09-29 — DeFiLlama Chain Stablecoin-to-TVL Capital Rotation
- 机制：链级 `USD stablecoin nominal supply / chain TVL` 作为资金在“稳定币现金”与风险型DeFi仓位之间的相对配置；最近7日ratio log-change零穿越，上穿固定SHORT原生币、下穿LONG，下一UTC日open。
- 动态资格：USD-M历史>=2年、signal-day QuoteVolume>=500万；discovery严格2023-2024且7d endpoint不得跨入2025。预注册 gate：7d signed mean>=+0.25%、>=60%币正、2023/2024两年都正。
- discovery：**1124事件/11币**；1d +0.1096%、3d -0.1083%、7d **-0.0818%**，仅 **4/11币** 7d均值为正。
- 年度：2023 7d **+0.2393%**，2024 **-0.3818%**；跨年稳定性直接失败。
- 结论：**discovery gate失败 / 2025-2026 OOS未读 / 不反向、不改7d窗口、不删弱币 / 不进exact TP8/SL6 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-tvl-rotation/2023-2026/。

## 2026-09-29 — DeFiLlama Chain Fee Yield / Capital Productivity
- 机制：最近7个完整UTC日链上 user fees 总额 / 同7日平均 chain TVL，作为单位锁仓资本的手续费生产率；与前7日同口径比较，log growth 零上穿 LONG、零下穿 SHORT，下一UTC日open。
- 动态资格：USD-M历史>=2年、signal-day QuoteVolume>=500万；discovery严格2023-2024且7d endpoint不得跨界。预注册 gate：7d signed mean>=+0.25%、>=60%币正、2023/2024两年都正。
- discovery **780事件/12币**：1d +0.1462%、3d +0.2427%、7d **+0.8336%**，**10/12币正**；2023 **+1.3448%**，2024 **+0.3841%**，强通过 discovery。
- untouched 2025：**588事件/13币，7d +0.1465%，8/13币正**；方向仍正但经济幅度明显降到 +0.25% gate 以下。
- 2026：**415事件/13币，7d +0.1468%，仅7/13币正（53.8%）**；经济幅度仍弱且breadth失败。
- 结论：**OOS衰减，冻结 / 不调7v7窗口或阈值、不删弱币、不反向 / 不进exact TP8/SL6 / 不入库**。
- 归档：strategy_templates/research/defillama-chain-fee-tvl-yield/2023-2026/。

## 2026-09-29 — DeFiLlama Chain DEX Turnover / TVL
- 机制：最近7个完整UTC日 DEX volume 总额 / 同7日平均 chain TVL，衡量单位DeFi资本的交易周转强度；与前7日同口径比较，log growth 零上穿 LONG、零下穿 SHORT，下一UTC日open。
- 动态资格：USD-M历史>=2年、signal-day QuoteVolume>=500万；discovery严格2023-2024且7d endpoint不得跨界。预注册 discovery gate：7d signed mean>=+0.25%、>=60%币正、2023/2024两年都正。
- discovery **922事件/12币**：1d +0.2506%、3d +0.5813%、7d **+0.8263%**，**10/12币正**；2023 **+0.9719%**，2024 **+0.6839%**，强通过 discovery。
- untouched 2025：**542事件/13币，7d +0.2203%，8/13币正**；方向和breadth保持，但经济幅度略低于既定 +0.25% 强门槛。
- 2026：**388事件/13币，7d +0.3817%，9/13币正**，重新高于强门槛。
- 结论：**near-pass但不严格晋级 exact**。OOS已看过后不能把 +0.25% 事后放宽成 +0.20%；保留为重点forward观察 family，但不调7v7、不删弱币、不进TP8/SL6、不入库。
- 归档：strategy_templates/research/defillama-chain-dex-tvl-turnover/2023-2026/。

## 2026-09-29 — DeFiLlama Chain Stablecoin Depeg Stress
- 机制：链级 USD stablecoin basket 的隐含价格 `implicit_peg=totalCirculatingUSD.peggedUSD/totalCirculating.peggedUSD`；每日 stress=`abs(log(implicit_peg))`，再计算最近7日平均stress / 前7日平均stress 的 log flow。压力零上穿→SHORT native token，零下穿→LONG，下一UTC日open。
- chain→native-token 与动态>=2年资格直接复用此前已审计的 Chain Stablecoin Supply Growth universe；signal-day QV>=500万；discovery严格2023-2024且7d endpoint不得越界到2025。
- 预注册 gate：7d signed mean>=+0.25% 且>=60%有事件币为正。实际 **996事件/11币**；1d **+0.1224%**、3d **+0.0634%**、7d **-0.4077%**，win7 48.09%，仅 **4/11币（36.4%）** 7d均值为正。
- 结论：**discovery明确失败 / 2025-2026 OOS未计算 / 不反向 / 不调7v7 / 不改成单一USDT/USDC挑选 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-depeg-stress/2023-2026/。

## 2026-09-29 — Snapshot Governance Turnout Shock：Feasibility
- 机制：proposal结束时的最终治理参与度 `scores_total` 相对同space此前已结束proposal的自身历史异常放大；严格规定最终票数只能在 `end` 后使用，不能在created时回看，避免look-ahead。
- 复用既有 DeFiLlama governanceID 安全映射 proposal corpus；实际仅 **7个 USD-M token space：APE/ENS/GTC/LDO/LINA/LRC/REN**，且 LINA/LRC/REN 历史极稀疏。
- 低于新数据源最低8币，**未看收益即 coverage blocked / 不扩大模糊Snapshot身份映射 / 不进入discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/snapshot-governance-turnout-shock/feasibility/。

## 2026-09-29 — Bitget USDT Futures Delist → Binance SHORT：Discovery
- 官方数据完整性：Bitget Help Center `Trading pair delisting` section 可连续分页；page16~24 从2025-01连续覆盖到2022-05，完整包住2023-2024，不依赖搜索引擎拼事件。2023-2024共有 **39篇** futures/perpetual delist候选公告。
- 先做 outcome-independent 清洗：排除 FET/OCEAN/AGIX、RNDR、GAL、FRONT、KLAY、TOMO、COCOS 等 merge/rebrand/swap 类 corporate action、10000AIDOGE面额生命周期，以及非USDT coin-margined合约；得到52个独立token-events。
- 严格 Binance eligibility（公告时USD-M存在、实际历史>=2年、此前完整24h QV>=500万）后 **20事件/20币/12独立批次**。冻结方向 SHORT，官方 `datePublished` 后下一完整1h open入场；看收益前 gate 固定为 event 12h mean>=+0.50%、batch-equal>=+0.50%、>=60%币正、>=60%批次正。
- discovery：1h **+2.3269%**、4h **+2.6439%**、12h **+4.9583%**；batch-equal 12h **+9.0632%**，但 win12仅30.0%，只有 **6/20币**、**4/12批次** 为正。2023仅2事件，12h -0.9629%；2024 18事件 +5.6162%。
- 高均值主要集中在 OMG/WAVES/UNFI 等极少数大跌事件，绝大多数事件方向相反；因此 breadth gate 明确失败。
- 结论：**discovery冻结 / 不查看2025-2026 OOS / 不后验只留暴跌币、不加severity过滤 / 不反向或延迟入场 / 不进exact / 不入库**。
- 归档：strategy_templates/research/bitget-usdt-futures-delist-short/2023-2024-discovery/。

## 2026-09-29 — DeFiLlama Chain Stablecoin Source Composition
- 机制：比较链上 USD stablecoin 的桥入来源与本地铸造来源，`composition=log(totalBridgedToUSD/totalMintedUSD)`，取7日变化零穿越；桥入来源相对增强→LONG native token，减弱→SHORT，下一UTC日open。与总stablecoin supply、depeg stress、velocity及全资产bridge netflow不同。
- chain→native-token 与>=2年 eligibility 复用已审计 stablecoin universe；signal-day QV>=500万；discovery=2023-2024且7d endpoint不越界。预注册 gate：7d mean>=+0.25%、>=60%币正。
- discovery：**810事件/9币**；1d **+0.2175%**、3d **-0.1447%**、7d **-0.0632%**，win7 49.63%，仅 **3/9币** 7d均值为正。
- 结论：**discovery失败 / 2025-2026 OOS未计算 / 不反向、不调7d、不加composition阈值 / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-source-composition/2023-2026/。

## 2026-09-29 — GitHub Core Security Advisory：Feasibility
- 机制：固定 canonical core repo 的公开 vulnerability/security advisory 作为负面外生安全事件；repo identity 完全复用 GitHub Core Release Catalyst，避免收益后挑repo；只做coverage，未看收益。
- GitHub 官方 Repository Security Advisories API 审计10个固定repo：2023-2024 **只有 ethereum/go-ethereum 2条**（均high severity）；其余9个repo公开advisory=0。
- 这不能解释为其它项目“没有漏洞”，而是项目披露渠道不一致，因此无法形成同语义跨币事件全集。
- 结论：**coverage/data-definition blocked / 不改用事后项目特定CVE渠道拼样本 / 不进discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/github-core-security-advisory/feasibility/。

## 2026-09-29 — Bitget Cross-Product Asset Exit → Binance SHORT：Post-Hoc Feasibility
- 该机制由 Bitget futures-only 2023-2024 breadth失败后生成，因此**不回旧 discovery 自证**；新协议固定 2025 discovery、2026 untouched OOS。
- 定义：同一 token 必须在 Bitget 官方 spot delist 与 futures delist 公告中同时出现，两个发布时间相差<=7天；信号时间取较晚公告（届时 cross-product exit 才完全公开）；最低覆盖8事件/8币，未过覆盖门前不看收益。
- 2025官方完整archive：**86篇 futures delist候选 + 54篇 spot delist公告**；spot正文机械解析 TOKEN/USDT、TOKEN/USDC 后，满足7天配对的只有 **HIFI 1个**（间隔约146.3h）。
- 结论：**未看2025收益即样本不足冻结 / 2026保持未读 / 不扩大7天窗口、不混入futures-only凑样本 / 不入库**。
- 归档：strategy_templates/research/bitget-cross-product-asset-exit-short/2025-feasibility/。

## 2026-09-30 — Spot-vs-Perp Order-Book Imbalance Divergence：Feasibility
- 机制：比较 Spot 与 USD-M 近端 bid/ask depth 的供需差，寻找真实现货需求与杠杆市场盘口分歧；未看收益。
- Binance Vision USD-M daily 有 `bookDepth`；但官方 Spot daily 目录只有 `aggTrades/klines/trades`，**没有 Spot bookDepth**。
- 结论：**同源历史数据 blocked / 不用不同供应商盘口拼接造成采样定义偏差 / 不进 discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/spot-perp-bookdepth-divergence/feasibility/。

## 2026-09-30 — GitHub Core Merged-PR Activity：2023-2024 Early Gate
- 固定 repo 映射复用 GitHub Core Release family，不按收益换仓库；只用 GitHub Search 服务器侧 `merged:` 时间，避免 commit author timestamp look-ahead。
- 月频信号：`log1p(当月merged PR)-mean(log1p(前3个月))`；>0 LONG、<0 SHORT；次月1日00:00 UTC open。为保持 untouched OOS，最后 discovery signal month=2024-11，完整7d endpoint 不跨入2025。
- BTC/ETH/BNB/XRP 2023-2024：**92事件**；1d -0.0420%、3d +0.1788%、7d **+0.3612%**；但仅 **2/4币** 为正。年度：2023 **-0.3072%**、2024 **+1.0903%**。
- 结论：经济幅度过门但 breadth + cross-year gate 失败，**冻结 / 不扩其余6个repo / 不读取2025+ / 不调窗口或方向 / 不进exact / 不入库**。
- 归档：strategy_templates/research/github-core-merged-pr-activity/2023-2024-early-gate/。

## 2026-09-30 — DeFiLlama Chain Stablecoin Bridged-Share：Discovery
- 机制：`totalBridgedToUSD / totalCirculatingUSD` 表示链上稳定币中外部桥接资本占比；7日 share change 零上穿 LONG、零下穿 SHORT，下一UTC日open；复用15-chain映射、>=2年、QV>=500万。
- discovery 2023-2024：**809事件/9币**；1d +0.2756%、3d -0.1169%、7d **-0.1316%**，仅 **4/9币** 7d均值为正。
- 年度：2023 **-0.5191%**，2024 **+0.1714%**，明显 regime flip。
- 结论：**discovery gate失败，整个 bridged-share family 冻结 / 不反向解释为桥接依赖风险 / 不调7d窗口 / 不看OOS / 不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-bridged-share/2023-2024-discovery/。

## 2026-09-30 — DeFiLlama Protocol External-Treasury Runway：Feasibility
- 机制：协议 treasury 的**非 OwnTokens 外部资产**增减代表运营 runway/抗风险能力，区别于TVL、fees与holder revenue；先做coverage，未看收益。
- outcome-independent universe：从 DeFiLlama `/protocols` 中 `treasury != null` 且 ticker 唯一的条目机械筛选，再要求 Binance USD-M；得到 API3/CVX/ENS/GNS/HFT/LDO/PERP/RSR 8个候选，8个 treasury endpoint 都有非自有资产历史。
- 但生产历史>=2年后，2023-2024 实际可用仅 **API3、CVX、ENS、LDO、RSR 5币**；GNS 无 USD-M，HFT/PERP 到2025才满2年。
- 结论：**低于最低8币，未看收益即 coverage blocked / 不手工加入UNI/AAVE等知名parent protocol补样本 / 不进discovery/OOS/exact / 不入库**。
- 归档：strategy_templates/research/defillama-protocol-external-treasury-runway/feasibility/。

## 2026-09-30 — GitHub Core Commit Activity：Feasibility
- 机制：core repo commit 活跃度作为开发强度代理；repo 映射复用 GitHub Release family，不按收益换仓库。
- 关键审计：Git author/committer timestamp 不等于 commit 进入默认分支、对市场可见的时间；按 git log 历史重放可能产生 look-ahead。
- 结论：**timestamp-semantics blocked / 未看收益 / 需要 GH Archive PushEvent 或等价可审计 visibility-time 源后才能重开 / 不入库**。
- 归档：strategy_templates/research/github-core-commit-activity/feasibility/。

## 2026-09-30 — Binance Futures Index Constituent / Weight Adjustment：Feasibility
- 完整扫描 Binance `Latest Binance News` 2023-2024 官方分页，固定标题关键词 `index price|constituent|weight|price index`，机械候选 **0篇**。
- 结论：**coverage blocked / 未看收益 / 不从其它 futures update 手工猜 index 事件 / 不入库**。
- 归档：strategy_templates/research/binance-futures-index-constituent-adjustment/feasibility/。

## 2026-09-30 — Binance USD-M Minimum Notional Adjustment：Feasibility
- 2023-2024 官方完整标题扫描只找到 **2篇** USD-M minimum-notional 公告，共 **6合约：BTC/ETH/LINK/BCH/ETC/LTC**；Spot/Margin minimum-order-size 明确排除。
- 低于预设最低8 event-symbol coverage，因此**未看收益即冻结 / 不拼其它order-size规则凑样本 / 不入库**。
- 归档：strategy_templates/research/binance-futures-minimum-notional-adjustment/feasibility/。

## 2026-09-30 — Binance Margin Dynamic Interest Rate：Feasibility
- 2023-2024 官方标题完整扫描中，真正 Margin 借币利率相关只有 2023-02-20 `Binance Margin Introduces Dynamic Interest Rate Updates` 一篇规则公告；其它命中均为已覆盖的 Futures funding-rate/leverage 事件。
- 没有可审计的按资产历史 rate-change event series。结论：**event-history/data-definition blocked / 未看收益 / 不入库**。
- 归档：strategy_templates/research/binance-margin-dynamic-interest-rate/feasibility/。

## 2026-09-30 — DeFiLlama Chain USDT+USDC Share
- 新机制：`(USDT + USDC nominal circulating) / all USD-stablecoin nominal circulating`；7日 share change 零上穿 LONG、零下穿 SHORT，下一UTC日open。数据全部同源 DeFiLlama，复用动态 USD-M 历史>=2年 + signal-day QV>=500万。
- 看收益前 coverage 过门并冻结 discovery=2023-2024、7d endpoint 不越过2024-12-31；gate=7d signed mean >=+0.25% 且 >=60% 有事件币种均值为正。
- canonical discovery：**754事件 / 10币**；1d **-0.1620%**、3d **-0.1229%**、7d **+0.2645%**，win7 51.33%；仅 **5/10币** 7d均值为正。
- 年度：2023 7d **-0.2338%**，2024 **+0.7371%**；symbol-equal 7d +0.4906%。经济幅度门槛刚过，但 breadth=50% 未过，且跨年符号不稳定。
- 结论：**discovery breadth gate失败 / 2025-2026 OOS保持未读 / 不删弱币、不调7d、不反向、不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-bluechip-stablecoin-share/2023-2026/。

## 2026-09-30 — DeFiLlama Chain Stablecoin Concentration (HHI)
- 新机制：按链重建全部 DeFiLlama `peggedUSD` 稳定币日度份额，`HHI=sum(share_i^2)`；7日 HHI change 零上穿代表集中度上升→SHORT，零下穿代表多样化上升→LONG。信号日前至少3个活跃USD稳定币，逐币合计/官方aggregate必须在[0.98,1.02]。
- 数据审计：339个USD稳定币历史；事件 coverage ratio **0.99999986~1.00000009**，无重复，LONG/SHORT=412/411。
- discovery 2023-2024：**823事件 / 11币**；1d **-0.0518%**、3d **+0.0305%**、7d **-0.1168%**，win7 51.28%；**7/11币正**。
- 年度：2023 7d **+0.3663%**，2024 **-0.5692%**；symbol-equal 7d -0.0292%。breadth 过门但经济幅度失败且跨年翻转。
- 结论：**discovery失败 / 2025-2026 OOS未读 / 不反向、不做entropy/effective-N变体、不调7d、不进exact / 不入库**。
- 归档：strategy_templates/research/defillama-chain-stablecoin-concentration-hhi/2023-2026/。

## 2026-09-30 — Binance USD-M Liquidation History：Feasibility
- 官方 Binance Vision `data/futures/um/daily/` 目录只含 aggTrades/bookDepth/bookTicker/indexPriceKlines/klines/markPriceKlines/metrics/premiumIndexKlines/trades，**没有 USD-M liquidationSnapshot/forceOrders 历史档案**。
- COIN-M liquidationSnapshot 已另做 family；近期 REST forced-order 保留窗口不能替代2023-2024历史。
- 结论：**historical-data blocked / 未看收益 / 不拼第三方清算历史 / 不入库**。
- 归档：strategy_templates/research/binance-usdm-liquidation-history/feasibility/。

## 2026-09-30 — OKX Listing / Delisting Archive：Feasibility
- 官方全球 Help Center 能看到 2023-2024 深历史，但分页快照不稳定：相邻/同页缓存出现不同总文章数和时间位置，无法证明 outcome-independent 事件全集完整。
- 本研究主机对 `us.okx.com/app.okx.com/eea.okx.com/www.okx.com` 官方API域名均 TLS EOF，当前无法用API直接校验完整分页；这是当前环境访问限制，不代表OKX API普遍不可用。
- 结论：**archive-consistency/data-access blocked / 未看收益 / 不用搜索引擎结果补事件 / 不入库**。只有拿到稳定官方API或可重放完整Help Center快照后再重开。
- 归档：strategy_templates/research/okx-announcement-event-archive/feasibility/。

## 2026-09-30 — v62：1h Inside-Bar + 4h Trend Breakout
- 目标：只用现有 DSL/行情数据构造可直接回测与实盘运行的新 Setup；无外部数据、无 MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v62/01-inside-bar-trend-breakout.json`。
- 固定结构：已完成1h inside bar，4h EMA20/50 + ADX14/DMI 同向，当前15m首次穿越 mother-bar 高/低触发；LONG/SHORT/CLOSE_LONG/CLOSE_SHORT 完整。Engine 固定 4x / TP8 / SL6 / fee0.0005 / slippage5bps。
- BTC：100笔 PF **1.113**；ETH：100笔 **1.021**；BNB：99笔 **0.973**；XRP：14笔 **0.757**。
- 四核心币：**313笔，标准化 PF 1.017，2/4正，0.409次/币/周**；LONG 148 / SHORT 165。
- 年度 PF：2023 **0.854**、2024 **0.994**、2025 **1.036**、2026 **1.520**。220 SL / 90 TP / 3 end_of_data。
- 结论：频率达到阶段要求，但 expectancy 与跨币/跨年一致性明显不足；**冻结 v62 / 不调 inside-bar、EMA、ADX 或确认周期 / 不入库**。

## 2026-09-30 — v63：1h Engulfing + 4h Trend Confirmation
- 目标：继续寻找完全可由当前项目 DSL/Engine 直接运行的独立 Setup；无外部数据、无 MarketCondition/Benchmark/NowTime%。最终可运行版本：`temp_strategy/v63/03-engulfing-trend-native-trigger.json`。
- 结构冻结：4h EMA20/50 + ADX14/DMI 定方向；上一根完成1h K 对前一根形成 body engulfing；当前1h从内部实际突破 engulfing high/low 后按 `NowPrice` 触发。固定 4x / TP8 / SL6 / fee0.0005 / 5bps，四类规则完整。
- 早期15m确认版本四核心币：372笔，PF 1.220，3/4正，0.486次/币/周；但 SOL 本地15m历史缺口，按“不擅自补库”原则停止扩展。
- 最终 native-trigger 版本在原四币：372笔，PF **1.218**，3/4正；结果与15m版近似。BTC 1.171、ETH 1.041、BNB 0.975、XRP 1.836。
- 参数冻结后在此前未见的 SOL/DOGE/LTC/AVAX/UNI/ZEC：**587笔，PF 1.464，4/6正，0.511次/币/周**；年度 2023/2024/2025/2026 = 1.703/1.261/1.686/0.998。
- 合并老10币：**959笔，标准化 PF 约1.383，7/10正，约0.501次/币/周**；年度约 1.421/1.352/1.491/1.139，达到进入symbol holdout的预设门槛。
- 最终动态>=2年 symbol holdout：ADA/NEAR/1000PEPE/SUI/ONDO 共 **204笔，PF 0.977，仅1/5正，0.369次/币/周**。ADA 1.516；NEAR 0.925；1000PEPE 1笔全亏；SUI 0.868；ONDO 0.398。年度 2025 PF 0.306、2026 0.713。
- 结论：老币样本表现很强，但新 symbol 泛化明确失败；**冻结整个 v63 family / 不按holdout删币、不删方向、不改engulfing或趋势参数 / 不入库**。

## 2026-09-30 — v64：1h KDJ(9,3,3) Cross + 4h Trend
- 目标：项目原生独立 Setup；只用现有 KDJ/EMA/ADX/Kline DSL，无外部数据、无 MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v64/01-kdj-cross-4h-trend.json`。
- 固定结构：4h EMA20/50 + ADX14/DMI 定方向；1h 标准 KDJ(9,3,3) K/D 实时交叉并要求当前1h candle同向；不加 J 阈值、MFI、volume 或参数搜索。固定 4x / TP8 / SL6 / fee0.0005 / 5bps，四类规则完整。
- 四核心币：BTC 111笔 PF 1.145；ETH 113笔 1.006；BNB 118笔 0.874；XRP 115笔 1.476。
- 合计：**457笔，标准化 PF 1.135，3/4正，0.597次/币/周**；LONG 226 / SHORT 231。
- 年度 PF：2023 1.068、2024 1.417、2025 **0.815**、2026 1.314；322 SL / 131 TP / 4 end_of_data。
- 结论：频率很好但整体PF低于预设 early gate≈1.15，且2025出现明显负期望；**冻结 v64 / 不调KDJ周期、J阈值、EMA/ADX或增加过滤器 / 不扩币 / 不入库**。

## 2026-09-30 — v65：1h Doji/窄实体突破 + 4h Trend
- 项目原生 Setup；只用 Kline/EMA/ADX/NowPrice，无外部数据、无 MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v65/01-doji-breakout-4h-trend.json`。
- 固定定义：上一根完整1h实体 <= 全振幅20%；当前1h从该K内部向4h EMA20/50 + ADX/DMI趋势方向突破高/低触发。固定4x / TP8 / SL6 / fee0.0005 / 5bps，四类规则完整。
- BTC 112笔 PF 1.164；ETH 108笔 1.075；BNB 111笔 0.881；XRP 114笔 1.380。
- 四核心币：**445笔，标准化 PF 1.136，3/4正，0.582次/币/周**；年度 PF 2023 1.044、2024 1.401、2025 **0.892**、2026 1.151。
- 结论：低于预设 early gate≈1.15，且2025负期望；**冻结 v65 / 不调20%阈值、不加volume/ATR过滤、不扩币 / 不入库**。

## 2026-09-30 — v66：1h Outside-Bar Continuation + 4h Trend
- 项目原生 Setup；只用 Kline/EMA/ADX/NowPrice，无外部数据、无 MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v66/01-outside-bar-continuation.json`。
- 固定定义：上一根完整1h high>前一根high 且 low<前一根low，实体方向与4h EMA20/50+ADX/DMI趋势一致；当前1h从 outside-bar 内部继续突破其高/低触发。固定4x / TP8 / SL6 / fee0.0005 / 5bps。
- BTC 88笔 PF 1.288；ETH 94笔 0.959；BNB 85笔 1.075；XRP 97笔 1.136。
- 合计：**364笔，标准化 PF 1.102，3/4正，0.476次/币/周**；年度 PF 2023 1.003、2024 1.587、2025 **0.719**、2026 1.041。
- 结论：整体与2025均不过门；**冻结 v66 / 不反向、不加body/volume阈值、不扩币 / 不入库**。

## 2026-09-30 — v67：1h TakerBuyRatio 0.5 Cross + QPS + 4h Trend
- 项目原生 Setup；只用 Futures Kline `TakerBuyRatio/Qps`、EMA、ADX、NowPrice，无外部数据、无 MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v67/01-taker-flow-cross-4h-trend.json`。
- 固定规则：上一根完整1h TakerBuyRatio 穿越0.5，QPS>=此前8h均值；当前1h再突破该信号K高/低，且4h EMA20/50+ADX/DMI同向。固定4x / TP8 / SL6 / fee0.0005 / 5bps。
- 四核心币：304笔，PF **1.265**，3/4正，0.397次/币/周；年度 2023/2024/2025/2026 = 0.996/1.623/0.921/1.488。
- 参数冻结后未见6币 SOL/DOGE/LTC/AVAX/UNI/ZEC：539笔，PF **1.560**，5/6正，0.470次/币/周；年度 1.768/1.404/1.830/0.856。
- 合并老10币：**843笔，标准化PF约1.465，8/10正**；年度约1.482/1.481/1.602/1.029，达到holdout门槛。
- 动态>=2年 holdout ADA/NEAR/1000PEPE/SUI/ONDO：**256笔，PF 0.960，仅1/5正，0.463次/币/周**。ADA 1.543；NEAR 0.951；1000PEPE 0.777；SUI 0.812；ONDO 0.311。年度2025 PF 0.474、2026 0.655。
- 结论：老币样本强，但新symbol泛化明确失败；**冻结整个v67 family / 不调0.5、QPS、趋势条件、不删弱币 / 不入库**。

## 2026-09-30 — v68：Two-Bar Pullback Resume + 4h Trend
- 项目原生 Setup；两根连续1h逆趋势实体作为回踩，当前1h突破最近一根回踩K高/低恢复4h EMA20/50+ADX/DMI主趋势；无外部数据、无MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v68/01-two-bar-pullback-resume.json`。
- 四核心币：BTC 102笔 PF 1.326；ETH 109笔 1.014；BNB 111笔 0.955；XRP 15笔 0.809。
- 合计：**337笔，标准化PF 1.065，2/4正，0.440次/币/周**；年度PF 2023 0.951、2024 0.993、2025 1.123、2026 1.391。
- 结论：整体/breadth均不过early gate；**冻结v68 / 不改成1根或3根回踩、不加EMA-touch/volume过滤、不扩币 / 不入库**。

## 2026-09-30 — v69：1h QPS Climax Failure Reversal
- 项目原生 Setup；上一根1h为此前8h最高QPS方向K，下一小时若从不利一侧重新穿越其振幅中点则反向；无外部数据、无MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v69/01-qps-climax-failure-reversal.json`。
- 四核心币：BTC 724笔 PF 0.748；ETH 825笔 0.881；BNB 756笔 0.858；XRP 1014笔 0.843。
- 合计：**3319笔，标准化PF 0.835，0/4正，4.338次/币/周**；年度PF 2023 0.931、2024 0.708、2025 0.916、2026 0.798。
- 结论：高频但稳定负期望；**冻结v69 / 不事后反向成continuation、不调8h窗口或中点定义 / 不入库**。

## 2026-09-30 — v70：1h QPS Dry-Up Breakout + 4h Trend
- 项目原生 Setup；最近3个完整1h QPS均低于更早12h均值，当前小时向4h EMA20/50+ADX/DMI主趋势方向突破这3小时高/低区间。无外部数据、无MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v70/01-qps-dryup-breakout.json`。
- 四核心币：BTC 108笔 PF 1.213；ETH 113笔 1.030；BNB 111笔 0.935；XRP 16笔 0.741。
- 合计：**348笔，标准化PF 1.032，2/4正，0.455次/币/周**；年度PF 2023 **0.854**、2024 1.059、2025 1.013、2026 1.437。
- 结论：整体/breadth/2023均不过门；**冻结v70 / 不调3h或12h窗口、不加当前QPS阈值、不扩币 / 不入库**。

## 2026-09-30 — v71：1h NR4 Breakout + 4h Trend
- 项目原生 Setup；上一根完整1h振幅为最近4根最窄，当前小时从NR4 K内部向4h EMA20/50+ADX/DMI趋势方向突破高/低触发。无外部数据、无MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v71/01-nr4-breakout-4h-trend.json`。
- 四核心币：BTC 110笔 PF 1.166；ETH 111笔 1.105；BNB 118笔 0.886；XRP 16笔 0.835。
- 合计：**355笔，标准化PF 1.036，2/4正，0.464次/币/周**；年度PF 2023 0.929、2024 1.050、2025 1.029、2026 1.241。
- 结论：整体/breadth不过门；**冻结v71 / 不改NR7、不加ATR/volume过滤、不扩币 / 不入库**。

## 2026-09-30 — v72：1h 12h Liquidity Sweep Reversal
- 项目原生纯价格行为 Setup；上一根1h刺破此前12h极值但收回区间内，当前小时重新穿越 sweep K 中点确认反转；无趋势/量/外部数据/MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v72/01-liquidity-sweep-reversal.json`。
- 四核心币：BTC 523笔 PF 0.879；ETH 624笔 0.836；BNB 581笔 0.849；XRP 656笔 0.729。
- 合计：**2384笔，标准化PF 0.817，0/4正，3.116次/币/周**；年度PF 2023 0.763、2024 0.780、2025 0.887、2026 0.832。
- 结论：跨币/跨年稳定负期望；**冻结v72 / 不事后反向成continuation、不调12h或中点确认 / 不入库**。

## 2026-09-30 — v73：1h Three-Bar Market-Structure Continuation
- 项目原生纯价格结构 Setup；最近3根完整1h形成连续 higher-high+higher-low 或 lower-high+lower-low，当前小时继续突破最近高/低触发；无趋势/量/外部数据/MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v73/01-three-bar-market-structure.json`。
- 四核心币：BTC 965笔 PF 0.841；ETH 1242笔 0.830；BNB 1157笔 0.867；XRP 1378笔 0.906。
- 合计：**4742笔，标准化PF 0.864，0/4正，6.198次/币/周**；年度PF 2023 1.012、2024 0.809、2025 0.776、2026 0.972。
- 结论：高频但跨币稳定负期望；**冻结v73 / 不事后反向、不改变3-bar长度、不加趋势过滤 / 不入库**。

## 2026-09-30 — v74：1h Confirmed Pivot Breakout + 4h Trend
- 项目原生 Setup；5-bar confirmed pivot，`[3]` 高/低点分别高于/低于两侧各2根后确认，当前小时向4h EMA20/50+ADX/DMI趋势方向突破 pivot。无外部数据、MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v74/01-confirmed-pivot-breakout.json`。
- 四核心币：BTC 84笔 PF 1.276；ETH 80笔 0.993；BNB 73笔 1.189；XRP 50笔 1.392。
- 合计：**287笔，标准化PF 1.210，3/4正，0.375次/币/周**；年度PF 2023 **0.888**、2024 1.552、2025 **0.853**、2026 1.942。
- 结论：总PF/breadth过线但两个完整年份明显负期望，跨regime不稳定；**冻结v74 / 不调pivot宽度、不加volume/ATR过滤、不扩币 / 不入库**。

## 2026-09-30 — v75：1h Effort-vs-Result Breakout
- 项目原生 Wyckoff 风格 Setup；上一根1h QPS高于此前8h均值而振幅低于此前8h平均振幅，当前小时从其内部突破高/低，方向即交易方向。无外部数据、MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v75/01-effort-result-breakout.json`。
- 四核心币：BTC 716笔 PF 0.828；ETH 879笔 0.859；BNB 862笔 0.747；XRP 1030笔 0.855。
- 合计：**3487笔，标准化PF 0.824，0/4正，4.557次/币/周**；年度PF 2023 0.933、2024 0.718、2025 0.838、2026 0.857。
- 结论：跨币/跨年稳定负期望；**冻结v75 / 不事后反向、不调8h基线或QPS/range倍率 / 不入库**。

## 2026-09-30 — v76：Funding Crowding × Taker-Flow Reversal
- 项目原生 Setup；负funding下1h TakerBuyRatio由<=0.5上穿做LONG，正funding下由>=0.5下穿做SHORT，并要求当前小时突破信号K高/低确认。无外部数据、MarketCondition/Benchmark/NowTime%。JSON：`temp_strategy/v76/01-funding-taker-reversal.json`。
- BTC 3笔 PF 0.190；ETH 1笔 PF 0；BNB 51笔 PF 1.290；XRP 176笔 PF 0.538。
- 合计：**231笔，标准化PF 0.688，仅1/4正，0.302次/币/周**；年度PF 2023 1.171、2024 0.438、2025 0.709，2026无有效交易。
- 结论：总体负期望且跨币事件密度极不均衡；**冻结v76 / 不调funding或0.5阈值、不扩币 / 不入库**。

## 2026-09-30 — v77：Previous-Day Range Acceptance Continuation
- 项目原生 Setup；前一完整日收盘突破前前一日高/低并同向收盘，当前日继续突破前一日高/低，且4h EMA20/50+ADX/DMI同向。无外部数据、MarketCondition/Benchmark/NowTime%。
- 四核心币：BTC 91笔 PF 1.240；ETH 83笔 1.111；BNB 89笔 0.953；XRP 10笔 0.638。
- 合计：**273笔，标准化PF 1.071，2/4正，0.357次/币/周**；年度PF 2023 0.984、2024 0.929、2025 1.144、2026 1.487。
- 结论：early gate失败；**冻结v77 / 不调接受定义、EMA/ADX或增加过滤器 / 不扩币 / 不入库**。
- 审计归档：`strategy_templates/research/previous-day-range-acceptance/2026-09-30-early-gate/`。

## 2026-09-30 — v78：Legacy-Derived 3m Exhaustion Wick Reversal
- 项目原生 JSON，可由现代 DSL 表达；核心为 3m 连续方向 run + 20-bar 新极值 + 长反向影线 + 当前3m确认突破。
- 在任何收益产生前，DatasetBuilder 加载 BTCUSDT 3m 即报历史缺口：`[[1672495200000 1756619999999]]`。
- **未评估收益 / 不是 alpha 失败**；按“不擅自补历史、不写DB”原则，标记 data-path blocked。
- 结论：冻结当前3m family；不自动下载/回填、不改成别的周期后冒充同一测试。
- 审计归档：`strategy_templates/research/legacy-3m-exhaustion-wick-reversal/2026-09-30-early-gate/`。

## 2026-09-30 — v79：1h Exhaustion Wick Reversal
- 项目原生 Setup；20-bar 新极值 + 前9根至少7根同向 + 反向影线/body>0.66 + 当前1h突破反转K确认。
- 四核心币：BTC 120笔 PF 0.607；ETH 100笔 0.734；BNB 139笔 0.701；XRP 136笔 0.589。
- 合计：**495笔，标准化PF 0.652，0/4正，0.647次/币/周**；年度PF 2023 0.881、2024 0.516、2025 0.652、2026 0.676。
- 结论：跨币/跨年稳定负期望；**冻结v79 / 不调20-bar、7/9、0.66或时间周期 / 不入库**。
- 审计归档：`strategy_templates/research/1h-exhaustion-wick-reversal/2026-09-30-early-gate/`。

## 2026-09-30 — v80：3h Price-Taker Flow Divergence Reversal
- 项目原生 Setup；连续3个完整1h价格收盘向一侧，而 TakerBuyRatio 连续向相反方向变化，当前1h突破上一根高/低确认反转。
- 四核心币：BTC 329笔 PF 0.848；ETH 370笔 0.704；BNB 330笔 1.080；XRP 367笔 0.869。
- 合计：**1396笔，标准化PF 0.863，仅1/4正，1.824次/币/周**；年度PF 2023 0.931、2024 0.925、2025 0.788、2026 0.820。
- 结论：跨币/跨年稳定负期望；**冻结v80 / 不反向、不改2h/4h窗口、不加阈值或过滤器 / 不入库**。
- 审计归档：`strategy_templates/research/price-taker-multihour-divergence/2026-09-30-early-gate/`。

## 2026-09-30 — v80：Previous-Week Range Acceptance
- 项目代码支持 `1w` interval，但当前本地历史库缺完整1w序列；BTC DatasetBuilder 在收益计算前即报 historical K-line gaps。
- **未查看任何收益/PF，data-blocked 冻结**；不自动补库、不改引擎。
- 归档：`strategy_templates/research/previous-week-range-acceptance/2026-09-30-early-gate/`。

## 2026-09-30 — v81：Keltner Wide-Excursion Re-entry
- 项目原生 4h/1d Setup；KC(50,3.75) 宽通道极端后，最新完成4h仍在 KC(50,2.75) 窄通道外，当前价格重入，且前一完整日方向同向。
- 四核心币合计仅 **4笔，0.005次/币/周**；远低于最低频率要求。四笔结果不作为alpha证据解释。
- 结论：**按频率冻结v81 / 不调2.75/3.75、period50、10-bar窗口或日线过滤 / 不入库**。
- 审计归档：`strategy_templates/research/keltner-wide-excursion-reentry/2026-09-30-early-gate/`。

## 2026-09-30 — v81：Rolling-7d Range Acceptance
- 项目原生1d+1h Setup；过去7个完整日线区间被完整1h接受突破后，下一小时继续确认。
- 四核心币：819笔，标准化PF **0.867**，**0/4正**，1.070次/币/周；2025 PF **0.590**。
- **early gate失败并冻结**；不调5d/10d、不加过滤器、不反向。
- 归档：`strategy_templates/research/rolling-7d-range-acceptance/2026-09-30-early-gate/`。

## 2026-09-30 — v82：4h Triple-EMA Trend Start
- 项目原生 4h 现代版本；EMA3/7 与 EMA7/15 最近4根内完成层级交叉，EMA3继续同向，RSI6/14不过热/过冷；固定 TP8/SL6。
- 四核心币总共仅 **4笔，0.005次/币/周**。aggregate PF 2.035 由4笔构成，不作为正alpha证据。
- 结论：**按频率冻结v82 / 不调EMA周期、交叉窗口或RSI阈值 / 不入库**。
- 审计归档：`strategy_templates/research/triple-ema-trend-start/2026-09-30-early-gate/`。

## 2026-09-30 — v82：ATR Term-Structure Expansion
- 项目原生1h/1d ATR结构；ATR14(1h)*sqrt(24)/ATR14(1d) 从<=1上穿>1，按触发1h方向并由下一小时突破确认。
- 四核心币：758笔，PF **1.090**，3/4正，0.991次/币/周；年度PF 1.034/1.080/1.161/1.077。
- breadth/年度方向尚可，但未达到预注册PF>=1.15；**early gate失败并冻结**，不调ATR周期/倍率/阈值。
- 归档：`strategy_templates/research/atr-term-structure-expansion/2026-09-30-early-gate/`。

## 2026-09-30 — v83：Hourly Trend Strength Breakout / Standard Exit
- 复核现有 `hourly-trend-strength-breakout-v2.json`：入场逻辑原样保留，原非标准 CLOSE 禁用，退出统一固定 TP8/SL6。
- 四核心币总共仅 **4笔，0.005次/币/周**；PF 1.880、3/4正均不具统计意义。
- 结论：**按频率冻结v83 / 不调ADX/DI/ATR/Donchian/Supertrend入场阈值 / 不入库**。
- 审计归档：`strategy_templates/research/hourly-trend-strength-breakout-standard-exit/2026-09-30-early-gate/`。

## 2026-09-30 — v83：Low-Activity Taker Accumulation Breakout
- 连续2个完整1h主动流同向偏置且QPS均低于更早8h均值，随后突破两小时区间。
- 四核心币：3800笔，PF **0.846**，**0/4正**，4.966次/币/周；四个年份PF全部<1。
- **early gate明确失败并冻结**；不调0.5、2h/3h或QPS阈值，不反向。
- 归档：`strategy_templates/research/low-activity-taker-accumulation/2026-09-30-early-gate/`。

## 2026-09-30 — v84：ADX20 Trend Start
- 1h ADX14 从<=20上穿>20，DI方向决定多空，当前小时突破触发K极值确认。
- 四核心币：779笔，PF **1.080**，3/4正，1.018次/币/周；2024 PF **0.759**。
- **early gate失败并冻结**；不调ADX18/25、不加EMA/volume/funding过滤。
- 归档：`strategy_templates/research/adx20-trend-start/2026-09-30-early-gate/`。

## 2026-09-30 — v85：Rolling-4h VWAP Reclaim
- **DSL-blocked before returns**：KLinePrice 仅暴露 QuoteAssetVolume(`Amount`) 与 quote-volume-per-second(`Qps`)，没有 base volume，无法表达真实 VWAP。
- 未查看收益，不改引擎、不写DB。
- 归档：`strategy_templates/research/rolling-4h-vwap-reclaim/2026-09-30-early-gate/`。

## 2026-09-30 — v86：Daily Taker Regime Transition
- 前一完整日线TakerBuyRatio跨0.5后，当前日只做同方向1h突破确认。
- 四核心币：1367笔，PF **0.852**，**0/4正**，1.787次/币/周；四个年份PF全部<1。
- **early gate明确失败并冻结**；不调0.5、不加EMA/ADX/QPS/funding过滤。
- 归档：`strategy_templates/research/daily-taker-regime-transition/2026-09-30-early-gate/`。

## 2026-09-30 — v81/v82/v83/v84/v86 Fixed TP8/SL6 Exit Audit Correction
- 发现 Backtest `RunConfig` 的 TP/SL 是 close-rule ROI gate，不会在 close expression 为 false 时强制退出；因此此前这5个 family 的条件 CLOSE 不是严格固定 TP8/SL6。
- 已保留旧结果到各 research bundle 的 `legacy/conditional-exit/`，并保持 entry 规则完全不变，统一用 `ROI >= 8 || ROI <= -6` 重跑。
- canonical strict-exit：v81 Rolling-7d 1069笔 PF **0.891** 0/4正；v82 ATR Term 936笔 **0.853** 0/4正；v83 Low-Activity Taker 5401笔 **0.792** 0/4正；v84 ADX20 852笔 **1.011** 3/4正；v86 Daily Taker 3710笔 **0.789** 0/4正。
- 结论：五个 family 均继续冻结；此前条件退出下的弱正结果不再作为研究证据。后续新研究统一显式写固定 TP8/SL6 CLOSE。

## 2026-09-30 — v87：QPS Term-Structure Expansion
- 1h QPS 从<=前一完整日线QPS切到>日线QPS，按触发1h方向并由下一小时突破确认。
- 严格固定 TP8/SL6：4777笔，PF **0.829**，**0/4正**，6.243次/币/周；四年全部PF<1。
- **early gate明确失败并冻结**；不调baseline倍率、不加过滤器、不反向。
- 归档：`strategy_templates/research/qps-term-structure-expansion/2026-09-30-early-gate/`。

## 2026-09-30 — v88：Record Net Aggressor Flow Continuation
- 1h净主动成交额 `2*TakerBuyAmount-Amount` 的绝对值创此前8h新高，按flow符号并由下一小时突破确认。
- 严格固定TP8/SL6：4902笔，PF **0.827**，**0/4正**，6.407次/币/周；四年全部<1。
- **early gate明确失败并冻结**；不反向、不调lookback/倍率、不加过滤器。
- 归档：`strategy_templates/research/record-net-aggressor-flow/2026-09-30-early-gate/`。

## 2026-09-30 — v89：1h Full-ATR Displacement Continuation
- 完整1h实体幅度>=信号发生前ATR14[2]，按实体方向，下一小时突破信号K极值入场。
- 严格固定TP8/SL6：5570笔，PF **0.825**，**0/4正**，7.280次/币/周；四年全部<1。
- **early gate明确失败并冻结**；不反向、不调ATR倍率、不加过滤器。
- 归档：`strategy_templates/research/1h-full-atr-displacement/2026-09-30-early-gate/`。

## 2026-09-30 — v90：Rolling-24h Return Zero-Cross
- 因果重建 `NowSymbolPercentChange` 的滚动24h收益穿越0轴，并由上一完整1h极值突破确认。
- 严格固定TP8/SL6：4295笔，PF **0.822**，**0/4正**，5.613次/币/周；四年全部<1。
- **early gate明确失败并冻结**；不调±1/2%阈值、不改窗口、不加过滤器。
- 归档：`strategy_templates/research/rolling-24h-return-zero-cross/2026-09-30-early-gate/`。

## 2026-09-30 — Hourly Volume Retest / Strict TP8-SL6 Early Gate
- 复用既有 `hourly-volume-retest-strength-v4` entry：1h volume/OBV impulse → 缩量受控回踩 → reclaim，配合4h EMA34与1h ADX/DMI；不改任何 entry 参数。
- 结果前完成退出语义审计，canonical CLOSE 明确写为 `ROI >= 8 || ROI <= -6`。
- 四核心币：1771笔，标准化PF **0.853**，**0/4正**，2.315次/币/周；年度PF 2023/2024/2025/2026 = 0.754/0.947/0.903/0.784。
- **early gate明确失败并冻结**；不调volume/ATR/ADX/EMA34/retest阈值，不扩六币。
- 归档：`strategy_templates/research/hourly-volume-retest-standard-exit/2026-09-30-early-gate/`。

## 2026-09-30 — Remaining Conditional-Exit Audit Correction
- 对今天剩余已有收益的条件退出 family 做固定 TP8/SL6 审计；entry 规则完全不变，旧证据保存到各 bundle 的 `legacy/conditional-exit/`。
- `1h-exhaustion-wick-reversal`：516笔，PF **0.682**，0/4正，0.674次/币/周。
- `previous-day-range-acceptance`：2856笔，PF **0.860**，0/4正，3.733次/币/周。
- `price-taker-multihour-divergence`：1517笔，PF **0.828**，0/4正，1.983次/币/周。
- 结论：三条在 canonical strict TP8/SL6 下均明确失败，继续冻结；今天所有已有收益的 `2026-09-30-early-gate` bundle 退出语义已统一审计。

## 2026-09-30 — Strict TP8/SL6 Exit Audit Correction
- Engine语义确认：RunConfig TP/SL 只是 close-rule ROI gate，不会在 CLOSE=false 或不满足时强平。
- 以下此前条件CLOSE结果作废，canonical 全部改为 `ROI >= 8 || ROI <= -6`：v81 Rolling-7d PF **0.891**(0/4正)；v82 ATR Term PF **0.853**(0/4正)；v83 Low-Activity Taker PF **0.792**(0/4正)；v84 ADX20 PF **1.011**(3/4正)；v86 Daily Taker PF **0.789**(0/4正)。
- 原结果保存在各自 research bundle 的 `legacy/conditional-exit/`；这些方向均维持冻结。

## 2026-09-30 — v87：QPS Term-Structure Expansion
- canonical 严格 TP8/SL6 (`ROI >= 8 || ROI <= -6`)：4777笔，PF **0.829**，0/4正，6.243次/币/周；年度 PF 0.763/0.884/0.848/0.790。
- 四币与四年均稳定负期望，冻结；不调 baseline multiplier/window、不加过滤器。
- 归档：`strategy_templates/research/qps-term-structure-expansion/2026-09-30-early-gate/`。

## 2026-09-30 — Strict TP8/SL6 Audit Correction (v77-v80)
- v77 Previous-Day Range Acceptance canonical：**2856笔，PF 0.860，0/4正**；四年全部<1，冻结。
- v78 Legacy 3m Exhaustion：旧条件退出结果作废；strict rerun 因 BTCUSDT 3m 本地历史大缺口而 **data-blocked**，未查看严格收益，不补库。
- v79 1h Exhaustion Wick Reversal canonical：**516笔，PF 0.682，0/4正**，冻结。
- v80 Price-Taker Multi-hour Divergence canonical：**1517笔，PF 0.828，0/4正**，冻结。
- 原条件退出结果均保存在各 research bundle 的 `legacy/conditional-exit/`。

## 2026-09-30 — v88：1h DMI Crossover Breakout
- +DI/-DI 标准交叉，下一小时突破触发K极值确认；strict TP8/SL6。
- 四核心币：2990笔，PF **0.774**，0/4正，3.908次/币/周；年度PF 0.701/0.808/0.826/0.727。
- **early gate明确失败并冻结**；不加ADX/EMA过滤，不调周期，不反向。
- 归档：`strategy_templates/research/dmi-crossover-breakout/2026-09-30-early-gate/`。

## 2026-09-30 — v89：1h EMA Spread Re-acceleration
- EMA20>EMA50 时 spread 增量由<=0转>0做LONG，SHORT镜像；下一小时突破触发K确认；strict TP8/SL6。
- 四核心币：3376笔，PF **0.870**，0/4正，4.412次/币/周；年度PF 0.757/0.924/0.950/0.808。
- **early gate明确失败并冻结**；不调EMA周期、不加过滤器、不反向。
- 归档：`strategy_templates/research/ema-spread-reacceleration/2026-09-30-early-gate/`。

## 2026-09-30 — v90：3h Price-OBV Divergence Reversal
- 3h价格与OBV方向背离，下一小时突破最近1h极值确认；strict TP8/SL6。
- 四核心币：4224笔，PF **0.795**，0/4正，5.521次/币/周；年度PF 0.802/0.840/0.776/0.732。
- **early gate明确失败并冻结**；不改3h窗口、不加过滤器、不反向。
- 归档：`strategy_templates/research/price-obv-3h-divergence/2026-09-30-early-gate/`。

## 2026-09-30 — v63/v67/v74 Strict TP8/SL6 Audit Correction
- 发现早期 v62–v76 测试时误把 RunConfig TP8/SL6 当成强制退出；实际它只 gate CLOSE 规则。按用户要求不补旧 research bundle，仅对曾出现阶段性正结果的 v63/v67/v74 做一次 strict 复核。
- v63 Engulfing Native Trigger strict：**2451笔，PF 0.843，0/4正**；年度PF 0.796/0.815/0.916/0.854。旧老10币 PF≈1.38 阶段结论作废。
- v67 Taker Flow Cross strict：**1986笔，PF 0.902，0/4正**；年度PF 0.854/0.956/0.958/0.797。旧老10币 PF≈1.47 阶段结论作废。
- v74 Confirmed Pivot strict：**976笔，PF 0.882，0/4正**；年度PF 0.874/0.843/0.883/0.952。旧 core4 PF1.210 结论作废。
- 三个 family 全部冻结；后续所有新研究统一显式 CLOSE=`ROI >= 8 || ROI <= -6`。

## 2026-09-30 — v100：4h Quote-Volume-Weighted Return Pressure
- 4个完整1h的 return×QuoteAssetVolume 求和穿越0轴，下一小时突破确认；strict TP8/SL6。
- 四核心币：6967笔，PF **0.824**，0/4正，9.105次/币/周；年度PF 0.805/0.859/0.833/0.765。
- **early gate明确失败并冻结**；不改窗口、不加过滤器、不反向。
- 归档：`strategy_templates/research/quote-volume-weighted-return-pressure/2026-09-30-early-gate/`。

## 2026-09-30 — v101：ADX20 × ATR-Term Expansion Confluence
- Discovery仅用 SOL/DOGE/LTC/AVAX/UNI/ZEC；strict TP8/SL6。
- **485笔，PF 0.763，0/6正，0.423次/币/周**；年度PF 0.689/0.679/0.933/0.751。
- discovery明确失败，冻结；**fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未评估，保持干净**。
- 归档：`strategy_templates/research/adx-volatility-expansion-confluence/2026-09-30-discovery/`。

## 2026-09-30 — Strict TP8/SL6 Exit Audit Correction (v77-v80)
- v77 Previous-Day Range Acceptance canonical：2856笔，PF **0.860**，0/4正，四年均<1，冻结。
- v78 Legacy 3m Exhaustion：旧条件退出结果作废；严格版本因本地3m历史缺口在收益计算前失败，标记 **data-blocked**，不补库。
- v79 1h Exhaustion Wick Reversal canonical：516笔，PF **0.682**，0/4正，冻结。
- v80 Price-Taker Multihour Divergence canonical：1517笔，PF **0.828**，0/4正，冻结。
- 以上旧条件CLOSE结果仅保留在各自 research 的 `legacy/conditional-exit/`，不再作为策略证据。

## 2026-09-30 — v88：Daily Open Reclaim
- 严格 TP8/SL6：5700笔，PF **0.777**，0/4正，7.450次/币/周；年度 PF 0.708/0.798/0.831/0.730。
- 四币/四年均稳定负期望，冻结；不反向、不加过滤器。
- 归档：`strategy_templates/research/daily-open-reclaim/2026-09-30-early-gate/`。

## 2026-09-30 — v89：TTM Squeeze Release
- Bollinger(20,2) 收进 Keltner(20,1.5) 后释放，严格 TP8/SL6。
- 四核心币：1438笔，PF **0.876**，0/4正，1.879次/币/周；年度 PF 0.966/0.775/0.954/0.821。
- early gate 明确失败，冻结，不调参数/不加过滤器。
- 归档：`strategy_templates/research/ttm-squeeze-release/2026-09-30-early-gate/`。

## 2026-09-30 — v102：Daily Inside-Day 4h Acceptance
- 新6币 discovery（SOL/DOGE/LTC/AVAX/UNI/ZEC），严格 TP8/SL6：975笔，PF **0.858**，0/6正，0.850次/币/周；年度 PF 0.988/0.841/0.838/0.721。
- discovery 失败，fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取；冻结。
- 归档：`strategy_templates/research/daily-inside-day-4h-acceptance/2026-09-30-discovery/`。

## 2026-09-30 — v103：Daily Outside-Day 4h Continuation
- 新6币 discovery，严格 TP8/SL6：323笔，PF **0.907**，2/6正，0.281次/币/周；年度 PF 1.367/0.910/0.626/0.696。
- PF/breadth/frequency均未过门，fresh holdout未读取；冻结。
- 归档：`strategy_templates/research/daily-outside-day-4h-continuation/2026-09-30-discovery/`。

## 2026-09-30 — v104：Daily NR4 4h Acceptance
- 新6币 discovery，严格 TP8/SL6：1435笔，PF **0.878**，1/6正，1.250次/币/周；年度 PF 1.081/0.846/0.860/0.692。
- discovery失败，fresh holdout未读取；冻结，不改NR7、不加过滤器。
- 归档：`strategy_templates/research/daily-nr4-4h-acceptance/2026-09-30-discovery/`。

## 2026-09-30 — v105：Daily 20d Liquidity Sweep Reversal
- 新6币 discovery，严格 TP8/SL6：345笔，PF **0.796**，0/6正，0.301次/币/周；年度 PF 0.815/0.601/0.946/0.902。
- discovery明确失败，fresh holdout未读取；冻结，不调20d窗口。
- 归档：`strategy_templates/research/daily-20d-liquidity-sweep-reversal/2026-09-30-discovery/`。

## 2026-09-30 — v88：Daily Open Reclaim
- canonical 严格 TP8/SL6：5700笔，PF **0.777**，0/4正，7.450次/币/周；年度 PF 0.708/0.798/0.831/0.730。
- 四币与四年均稳定负期望，冻结；不反向、不加过滤器。
- 归档：`strategy_templates/research/daily-open-reclaim/2026-09-30-early-gate/`。

## 2026-09-30 — v89：TTM Squeeze Release
- canonical 严格 TP8/SL6：1690笔，PF **0.842**，0/4正，2.209次/币/周；年度 PF 0.955/0.783/0.850/0.763。
- 标准1h Bollinger(20,2) inside Keltner(20,1.5) squeeze-release 明确失败，冻结，不扫参数。
- 归档：`strategy_templates/research/ttm-squeeze-release/2026-09-30-early-gate/`。

## 2026-09-30 — v106：Classic Daily Pivot R1/S1 Acceptance
- 新6币 discovery，strict TP8/SL6：4846笔，PF **0.815**，0/6正，4.222次/币/周；年度 PF 0.819/0.790/0.820/0.857。
- discovery明确失败，fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取；冻结，不改R2/S2/Camarilla/过滤器。
- 归档：`strategy_templates/research/classic-daily-pivot-r1s1-acceptance/2026-09-30-discovery/`。

## 2026-09-30 — v107：Daily Full-ATR Displacement Continuation
- 新6币 discovery，strict TP8/SL6：550笔，PF **0.743**，1/6正，0.479次/币/周；年度 PF 0.895/0.837/0.583/0.690。
- frequency通过但expectancy/breadth明确失败，fresh holdout未读取；冻结，不调ATR倍率、不加过滤器、不反向。
- 归档：`strategy_templates/research/daily-full-atr-displacement-continuation/2026-09-30-discovery/`。

## 2026-09-30 — v108：Funding-Price 8h Divergence Transition
- 新6币 discovery，strict TP8/SL6：6671笔，PF **0.815**，0/6正，5.812次/币/周；年度 PF 0.822/0.800/0.864/0.744。
- discovery明确失败，fresh holdout未读取；冻结，不调8h窗口/funding阈值、不加过滤器、不反向。
- 归档：`strategy_templates/research/funding-price-8h-divergence-transition/2026-09-30-discovery/`。

## 2026-09-30 — v109：Funding Crowding Acceleration Fade
- 新6币 discovery，strict TP8/SL6：1091笔，PF **0.796**，0/6正，0.951次/币/周；年度 PF 0.745/0.868/0.793/0.715。
- discovery明确失败，fresh holdout未读取；Funding新变体路线暂停。
- 归档：`strategy_templates/research/funding-crowding-acceleration-fade/2026-09-30-discovery/`。

## 2026-09-30 — v106：Classic Daily Pivot R1/S1 Acceptance
- 新6币 discovery（SOL/DOGE/LTC/AVAX/UNI/ZEC），严格 TP8/SL6：4846笔，PF **0.815**，**0/6正**，4.222次/币/周；年度 PF 0.819/0.790/0.820/0.857。
- discovery 明确失败；fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取。冻结，不做 R2/S2、Camarilla/Fibonacci pivot 或过滤器变体。
- 归档：`strategy_templates/research/classic-daily-pivot-r1s1-acceptance/2026-09-30-discovery/`。

## 2026-09-30 — v107：Previous-Day Midpoint Reclaim
- 新6币 discovery，严格 TP8/SL6：7106笔，PF **0.826**，**0/6正**，6.191次/币/周；年度 PF 0.804/0.793/0.903/0.782。
- discovery 明确失败；fresh holdout 未读取。冻结，不改 midpoint、不加 25%/75% levels 或过滤器。
- 归档：`strategy_templates/research/previous-day-midpoint-reclaim/2026-09-30-discovery/`。

## 2026-09-30 — v108：Three-Hour Same-Sign Streak Continuation
- 新6币 discovery，严格 TP8/SL6：13874笔，PF **0.779**，**0/6正**，12.088次/币/周；年度 PF 0.799/0.745/0.810/0.751。
- discovery 明确失败；fresh holdout 未读取。冻结，不改2-bar/4-bar streak、不加过滤器、不反向。
- 归档：`strategy_templates/research/three-hour-same-sign-streak/2026-09-30-discovery/`。

## 2026-09-30 — ID121 / v54 Exact TP8-SL6 Candidate Audit
- 重要语义修正：Engine 的 RunConfig TP/SL 是 CLOSE gate，不是无条件强平；原 ID121/v54 条件 CLOSE 因此不等于固定 TP8/SL6。此次仅将 CLOSE_LONG/CLOSE_SHORT 改为 `ROI >= 8 || ROI <= -6`，entry、universe、eligibility、费用/滑点均不变。
- **ID121 canonical**：773笔，标准化 PF **1.211**，**12/15正**，频率 **0.533次/币/周**；2024/2025/2026 PF = **1.241 / 1.179 / 1.246**。LONG PF 1.232，SHORT PF 1.188。**保留为正式候选**。
- **v54 canonical**：830笔，标准化 PF **1.163**，11/15正，频率 **0.572次/币/周**；年度 PF = 1.157 / 1.128 / 1.214。
- 配对归因：共同770笔 PF **1.214**；ID121-only 3笔 PF 0.568（样本极小）；**v54-only 60笔 PF 0.653**，2024/2025/2026 = 0.650/0.543/0.825，三个年份均负期望。
- 结论：v54 的增频来自明显负 expectancy 的新增 cohort，并把总 PF 从 1.211 拉低到 1.163；按“不为频率接受负 expectancy”约束，**v54 从正式候选降级/冻结**。旧 ID121≈1.395 / v54≈1.396 的候选 PF 不再作为 exact TP8/SL6 证据。
- 归档：`strategy_templates/research/id121-v54-exact-exit/2026-09-30-audit/`。

## 2026-09-30 — ID121 Exact TP8-SL6 Early OOT (2021H2–2022)
- 使用 canonical ID121 exact-exit entry，严格 CLOSE=`ROI >= 8 || ROI <= -6`；只读本地历史，不改参数。
- BTC：42笔，PF **0.604**；ETH：37笔，PF **0.569**。
- 合计：79笔，PF **0.587**，**0/2正**，0.522次/币/周；2021H2 PF **0.303**，2022 PF **0.698**。
- 结论：退出语义修正没有修复 pre-2023 失效。**ID121 保留为当前 2024+ regime 候选，但不能视为跨全周期稳定策略**；不根据早期 OOT 反调参数。
- 归档：`strategy_templates/research/id121-exact-exit-early-oot/2021h2-2022-audit/`。

## 2026-09-30 — v110：Donchian 20h Midpoint Regime Cross
- 新6币 discovery，严格 TP8/SL6：10146笔，PF **0.815**，0/6正，8.840次/币/周；年度 PF 0.829/0.810/0.852/0.738。
- discovery 明确失败；fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取。冻结，不改20h lookback、不加过滤器、不反向。
- 归档：strategy_templates/research/donchian-20h-midpoint-regime-cross/2026-09-30-discovery/。

## 2026-09-30 — v111：Daily Quote-Volume Expansion Continuation
- 新6币 discovery，严格 TP8/SL6：2743笔，PF **0.814**，0/6正，2.390次/币/周；年度 PF 0.759/0.930/0.741/0.790。
- discovery 明确失败；reserved holdout 未读取。冻结，不改20d volume baseline、不加过滤器、不反向。
- 归档：strategy_templates/research/daily-quote-volume-expansion-continuation/2026-09-30-discovery/。

## 2026-10-01 — v107：Donchian 20h Midpoint Regime Cross
- 新6币 discovery，严格 TP8/SL6：10146笔，PF **0.815**，0/6正，8.840次/币/周；年度 PF 0.829/0.810/0.852/0.738。
- discovery 明确失败，fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取；冻结，不调10h/40h或加过滤器。
- 归档：`strategy_templates/research/donchian-20h-midpoint-regime-cross/2026-09-30-discovery/`。

## 2026-10-01 — v108：4h Directional Excursion Imbalance
- 新6币 discovery，严格 TP8/SL6：15268笔，PF **0.811**，0/6正，13.303次/币/周；年度 PF 0.788/0.784/0.861/0.802。
- discovery 明确失败，fresh holdout 未读取；冻结，不调2h/8h窗口、不改1.0阈值、不反向。
- 归档：`strategy_templates/research/directional-excursion-imbalance/2026-10-01-discovery/`。

## 2026-10-01 — v34 Pullback Exact-Exit Attribution
- 同一15币 exact TP8/SL6：BASE禁用pullback 372笔 PF **1.006**、0.256次/币/周；FULL启用pullback 718笔 PF **0.922**、0.495次/币/周。
- paired FULL-only：356笔，PF **0.843**，normalized net -0.565；2024/2025/2026 PF = 0.718/0.746/1.208。
- 结论：`funding_pullback_resume_short_v34` 是**负 expectancy 增频**，正式冻结；不调Funding/EMA/ADX/触发阈值。
- 归档：`strategy_templates/research/v34-pullback-exact-exit-attribution/2026-10-01-audit/`。

## 2026-10-01 — Daily Full-ATR Displacement Continuation
- 新6币 discovery，严格 TP8/SL6：550笔，PF **0.743**，1/6正，0.479次/币/周；年度PF 0.895/0.837/0.583/0.690。
- discovery明确失败，fresh holdout未读取；冻结。
- 归档：`strategy_templates/research/daily-full-atr-displacement-continuation/2026-09-30-discovery/`。

## 2026-10-01 — v112 Daily First-4h Opening Range Breakout
- 新6币 discovery，严格 TP8/SL6：8807笔，PF **0.799**，0/6正，7.674次/币/周；年度PF 0.809/0.771/0.842/0.762。
- discovery明确失败，fresh holdout未读取；冻结，不改opening-range长度。
- 归档：`strategy_templates/research/daily-first-4h-opening-range-breakout/2026-10-01-discovery/`。

## 2026-10-01 — v107：Donchian 20h Midpoint Regime Cross
- 新6币 discovery，严格 TP8/SL6：10146笔，PF **0.815**，0/6正，8.840次/币/周；年度 PF 0.829/0.810/0.852/0.738。
- discovery 明确失败；fresh holdout ALGO/INJ/LDO/PENDLE/PYTH 未读取。冻结，不调10h/40h、不加过滤器、不反向。
- 归档：`strategy_templates/research/donchian-20h-midpoint-regime-cross/2026-09-30-discovery/`。

## 2026-10-01 — v108：v67 Entry Strict TP8/SL6 Audit
- 复用 v67 entry，不改任何入场参数，只把 CLOSE 纠正为固定 `ROI >= 8 || ROI <= -6`。
- 新6币 discovery：3830笔，PF **0.854**，0/6正，3.337次/币/周；年度 PF 0.842/0.888/0.823/0.868。
- 原 v67 条件退出版本的强表现没有在严格 TP8/SL6 下保留；fresh holdout 未读取，冻结该 entry。
- 归档：`strategy_templates/research/v67-entry-strict-exit-audit/2026-10-01-discovery/`。

## 2026-10-01 — v113：v63 Entry Strict TP8/SL6 Audit
- 复用 v63c entry，仅将 CLOSE 改为固定 `ROI >= 8 || ROI <= -6`。
- 新6币 discovery：4864笔，PF **0.824**，0/6正，4.238次/币/周；年度 PF 0.790/0.823/0.854/0.828。
- 原 v63 条件退出版本的高 PF 不可保留；fresh holdout 未读取，冻结。
- 归档：`strategy_templates/research/v63-entry-strict-exit-audit/2026-10-01-discovery/`。

## 2026-10-01 — v114：Bollinger Band-Walk Continuation
- 新6币 discovery，strict TP8/SL6：4366笔，PF **0.875**，0/6正，3.804次/币/周；年度 PF 0.887/0.826/0.872/0.960。
- discovery明确失败，fresh holdout未读取；冻结，不改1/3根带外收盘，不扫Boll参数。
- 归档：`strategy_templates/research/bollinger-band-walk-continuation/2026-10-01-discovery/`。

## 2026-10-01 — v115：Keltner 20/2 First-Breakout Continuation
- 新6币 discovery，strict TP8/SL6：4597笔，PF **0.832**，0/6正，4.005次/币/周；年度 PF 0.839/0.811/0.864/0.810。
- discovery明确失败，fresh holdout未读取；冻结，不扫 Keltner multiplier、不加过滤器、不反向。
- 归档：`strategy_templates/research/keltner-20-2-first-breakout-continuation/2026-10-01-discovery/`。

## 2026-10-01 — ID121 Exact TP8/SL6 September Forward OOS
- canonical ID121 exact-exit，保持 production eligibility/state，从原起点连续回放，只统计 2026-09-01~09-12 15:59 UTC 新 entry。
- **9笔，PF 0.533，7 SL / 2 TP，1/15正，全部LONG；0.360次/币/周**。ZEC 3笔 PF1.312；UNI 3笔 PF0.518；AVAX/BNB/ONDO各1笔亏损。
- canonical历史773笔的连续9笔窗口共765个：PF<=0.533 有119个（**15.56%**）；<=2胜/9笔占 **13.20%**；PF q10=0.331、q25=0.585、median=1.066。
- 结论：这是需要持续记录的 forward warning，但仍属于历史中并不罕见的坏窗口；**ID121继续保留为2024+正式候选，不针对这9笔调参**。
- 归档：strategy_templates/research/id121-exact-exit-forward/2026-09-01_2026-09-12-1559z/。

## 2026-10-01 — v116：12h Breakout Retest Resume
- 新6币 discovery，strict TP8/SL6：2914笔，PF **0.786**，0/6正，2.539次/币/周；年度 PF 0.755/0.779/0.797/0.813。
- breakout→1h retest hold→resume 明确失败，fresh holdout未读取；冻结，不调lookback/retest长度、不加过滤器。
- 归档：`strategy_templates/research/12h-breakout-retest-resume/2026-10-01-discovery/`。

## 2026-10-01 — ID121 Canonical Exact-Exit Robustness Audit
- 773笔 canonical exact-exit，PF **1.211**。leave-one-symbol-out PF 范围 **1.148~1.258**；去掉最大贡献 XRP 后仍 PF1.148。
- 25个月中15个月净正；9个季度中8个净正，唯一负季度 2026-Q2 PF **0.858**。
- XRP/ETH/BTC 占正贡献约 **61.6%**、占组合总 normalized net 约 **69.6%**，存在头部贡献集中但不是单币依赖。
- 10,000次自然月 block bootstrap：PF q05 **1.015**、q10 1.057、median 1.207；PF<=1 仅 **3.71%**。
- 结论：支持 ID121 在 **2024+ regime** 存在 block-level edge；但早期 OOT 2021H2-2022 PF0.587 仍禁止宣称 all-regime 稳定。
- 归档：`strategy_templates/research/id121-exact-exit-robustness/2026-10-01-audit/`。

## 2026-10-01 — ID121 Fresh-Symbol Holdout
- canonical ID121 exact TP8/SL6，首次对未参与其调参/验证的 ALGO/INJ/LDO/PENDLE/PYTH 做一次性 symbol holdout。
- **238笔，PF 0.848，0/5正，0.629次/币/周**；2025 PF 0.746，2026 PF 0.968。
- LONG 117笔 PF **0.851**；SHORT 121笔 PF **0.845**，两侧均失败，不是单边拖累。
- 结论：ID121 只能保留为原15币/2024+ cohort 候选，**不能再视为跨新币通用策略**；不使用此 holdout 反调参数。
- 归档：`strategy_templates/research/id121-fresh-symbol-holdout/2026-10-01-audit/`。

## 2026-10-01 — v117：Daily Volume Shock One-Day Lag
- 新6币 discovery，严格 TP8/SL6：2490笔，PF **0.818**，0/6正，2.170次/币/周；年度 PF 0.690/0.856/0.781/0.971。
- 1日滞后 continuation 明确失败；不再扫2d/3d lag，不加过滤器、不反向。
- 归档：`strategy_templates/research/daily-volume-shock-one-day-lag/2026-10-01-discovery/`。

## 2026-10-01 — v117：Daily Volume Shock One-Day Lag
- 新6币 discovery，严格 TP8/SL6：2490笔，PF **0.818**，0/6正，2.170次/币/周；年度 PF 0.690/0.856/0.781/0.971。
- 1日滞后 continuation 明确失败；不再扫2d/3d lag，不加过滤器、不反向。
- 归档：`strategy_templates/research/daily-volume-shock-one-day-lag/2026-10-01-discovery/`。

## 2026-10-01 — v118：Daily 3-Bar Streak Reversal
- 新6币 discovery，strict TP8/SL6：3629笔，PF **0.792**，0/6正，3.162次/币/周；年度 PF 0.732/0.779/0.831/0.826。
- discovery明确失败，fresh holdout未读取；冻结，不改2日/4日 streak、不加过滤器、不反向。
- 归档：`strategy_templates/research/daily-three-bar-streak-reversal/2026-10-01-discovery/`。

## 2026-10-01 — v29 Entry Strict TP8/SL6 Audit
- 旧 v29/ID114 entry 不变，仅 CLOSE 改为固定 `ROI >= 8 || ROI <= -6`。
- 新6币 discovery：242笔，PF **0.989**，4/6正，但仅 **0.211次/币/周**；年度 PF 1.404/0.865/0.726/1.188。
- expectancy 与频率均不过门，正式冻结；fresh holdout未读取。
- 归档：`strategy_templates/research/v29-entry-strict-exit-audit/2026-10-01-discovery/`。

## 2026-10-01 — v119：Daily Engulfing + 4h Reversal Confirmation
- 2023–2024 discovery only，strict TP8/SL6：1716笔，PF **0.809**，0/6正，2.739次/币/周；2023/2024 PF 0.760/0.845。
- discovery失败；2025 OOS1、2026 OOS2均未读取。冻结，不加过滤器、不改确认周期。
- 归档：`strategy_templates/research/daily-engulfing-4h-reversal/2026-10-01-discovery/`。

## 2026-10-01 — v120：Daily + 4h ROC Alignment
- 2023–2024 discovery only，strict TP8/SL6：1181笔，PF **0.837**，1/6正，1.885次/币/周；2023/2024 PF 0.828/0.843。
- discovery失败；2025 OOS1、2026 OOS2均未读取。冻结，不扫ROC周期、不加过滤器。
- 归档：`strategy_templates/research/daily-4h-roc-alignment/2026-10-01-discovery/`。

## 2026-10-01 — v122：Daily 20d Turtle Breakout + 4h Acceptance
- 2023–2024 discovery only，strict TP8/SL6：657笔，PF **0.844**，1/6正，1.049次/币/周；2023/2024 PF 0.909/0.796。
- discovery失败；2025 OOS1、2026 OOS2未读取。冻结，不改20d窗口、不加过滤器。
- 归档：`strategy_templates/research/daily-20d-turtle-breakout-4h-acceptance/2026-10-01-discovery/`。

## 2026-10-01 — v123：1h EMA20/50 Crossover
- 2023–2024 discovery only，strict TP8/SL6：1015笔，PF **0.851**，1/6正，1.620次/币/周；2023/2024 PF 0.832/0.867。
- discovery失败；2025 OOS1、2026 OOS2未读取。冻结，不扫EMA周期、不加过滤器。
- 归档：`strategy_templates/research/ema20-50-1h-crossover/2026-10-01-discovery/`。

## 2026-10-01 — v118：Daily 3-Bar Streak Reversal
- 新6币 discovery，严格 TP8/SL6：3629笔，PF **0.792**，0/6正，3.162次/币/周；年度 PF 0.732/0.779/0.831/0.826。
- discovery 明确失败；fresh holdout 未读取。冻结，不改2日/4日 streak、不加过滤器、不反向。
- 归档：strategy_templates/research/daily-three-bar-streak-reversal/2026-10-01-discovery/。

## 2026-10-01 — v124：1h MACD(12,26,9) Signal-Line Cross
- 2023–2024 discovery only，strict TP8/SL6：4362笔，PF **0.797**，0/6正，6.962次/币/周；2023/2024 PF 0.824/0.778。
- discovery 明确失败；2025 OOS1、2026 OOS2未读取。冻结，不扫MACD周期、不加过滤器。
- 归档：strategy_templates/research/macd-1h-signal-cross/2026-10-01-discovery/。

## 2026-10-01 — v50 Entry Strict TP8/SL6 Audit
- 原 entry 不变，仅 CLOSE 改为固定 TP8/SL6；2023–2024 新6币 discovery：142笔，PF **1.066**，4/6正，但仅 **0.227次/币/周**。
- 2023 PF 0.785、2024 PF 1.263；expectancy/频率/跨年稳定性均不过门。2025/2026 OOS未读取，冻结。
- 归档：strategy_templates/research/v50-entry-strict-exit-audit/2026-10-01-discovery/。

## 2026-10-01 — v52 Entry Strict TP8/SL6 Audit
- 原 entry 不变，仅 CLOSE 改为固定 TP8/SL6；2023–2024 新6币 discovery：151笔，PF **1.298**，4/6正，但仅 **0.241次/币/周**。
- 2023 PF 0.978、2024 PF 1.533；LONG 38笔 PF **1.483**，SHORT 113笔 PF **1.240**。正 expectancy 并非单侧假象，但频率/跨年稳定性不过门。
- 2025/2026 OOS未读取；冻结，不调2–3 ATR定义或其它 entry 参数。
- 归档：strategy_templates/research/v52-entry-strict-exit-audit/2026-10-01-discovery/。

## 2026-10-01 — v125：1h 1-2-3 Swing Reversal
- 2023–2024 新6币 discovery，strict TP8/SL6：1608笔，PF **0.779**，0/6正，2.566次/币/周；2023/2024 PF 0.839/0.727。
- discovery 明确失败；2025/2026 OOS未读取。冻结，不改pivot几何、不加过滤器、不反向。
- 归档：strategy_templates/research/one-two-three-swing-reversal/2026-10-01-discovery/。

## 2026-10-01 — v125：Funding Zero-Cross Momentum
- 2023–2024 discovery only，strict TP8/SL6：369笔，PF **0.718**，0/6正，0.589次/币/周；2023/2024 PF 0.520/0.932。
- funding settle 后仅1h~2h事件窗口，避免同一funding周期重复触发；discovery明确失败。2025/2026 OOS未读取。
- 冻结，不调funding幅度、不反向成crowding trade、不改事件窗口。
- 归档：strategy_templates/research/funding-zero-cross-momentum/2026-10-01-discovery/。

## 2026-10-01 — ID121 Second Fresh-Symbol Holdout Feasibility
- 预注册第二批 untouched 5币 cohort：排除原15币+第一批5币；QuoteVolume>=500万；本地1m历史<=2023-01-01起且覆盖到2026-08-31 23:59；禁止按收益挑币。
- 当前本地缓存仅 **LITUSDT 1个**满足：QuoteVolume约5303万，历史44个月。
- **coverage-blocked before returns**：不足5币，不运行单币伪holdout，不读取LIT的ID121收益，不擅自补历史/写DB。
- 归档：strategy_templates/research/id121-second-fresh-symbol-holdout/2026-10-01-feasibility/。

## 2026-10-01 — v128：1h Body Compression -> Expansion Release
- 2023–2024 discovery only，strict TP8/SL6：6433笔，PF **0.758**，0/6正，10.267次/币/周；2023/2024 PF 0.744/0.769。
- discovery明确失败；OOS未读取。冻结，不调body倍率/压缩长度、不加过滤器。
- 归档：strategy_templates/research/body-compression-expansion-release/2026-10-01-discovery/。

## 2026-10-01 — v125：Funding Zero-Cross Momentum
- 仅在 funding settle 后 1h~<2h 触发，避免同一结算周期重复事件；NowTime 仅用于 event age，不使用 modulo。
- 2023–2024 discovery，strict TP8/SL6：369笔，PF **0.718**，0/6正，0.589次/币/周；2023/2024 PF 0.520/0.932。
- discovery 明确失败；2025 OOS1、2026 OOS2未读取。冻结，不调 funding 阈值/事件窗、不反向。
- 归档：strategy_templates/research/funding-zero-cross-momentum/2026-10-01-discovery/。

## 2026-10-01 — v129：1h Three-Return Acceleration
- 2023–2024 discovery only，strict TP8/SL6：2166笔，PF **0.810**，0/6正，3.457次/币/周；2023/2024 PF 0.898/0.745。
- discovery失败；2025/2026 OOS未读取。冻结，不改2/4-return、不加幅度阈值、不反向。
- 归档：strategy_templates/research/1h-three-return-acceleration/2026-10-01-discovery/。

## 2026-10-01 — v130：4h Double-Inside Compression Breakout
- 2023–2024 discovery only，strict TP8/SL6：49笔，PF **0.441**，0/6正，仅0.078次/币/周；2023/2024 PF 0.479/0.407。
- expectancy 与频率同时失败；2025/2026 OOS未读取。冻结，不改inside-bar数量、母K参考或确认逻辑。
- 归档：strategy_templates/research/4h-double-inside-compression-breakout/2026-10-01-discovery/。

## 2026-10-01 — v131：OI Expansion × Taker Alignment Early Gate
- Binance USD-M metrics，同一4h窗口：OI从<=0转为扩张，方向取4h mean log taker long/short ratio。
- 固定 quarterly 24 target-days / 6币：870事件；1h **+0.0479%**、4h **+0.0786%**、12h **+0.2470%**；4/6币4h为正。
- breadth通过但预注册4h经济门槛要求>=+0.10%，因此 **early gate失败**；不因12h更高而改gate，完整2023-2024未下载。
- 冻结，不调OI/taker阈值、窗口、方向或币种。
- 归档：strategy_templates/research/oi-expansion-taker-alignment/2026-10-01-early-gate/。

## 2026-10-01 — v65 Entry Strict TP8/SL6 Audit
- 原 v65 LONG/SHORT entry 完全不变，仅将 CLOSE_LONG/CLOSE_SHORT 改为固定 `ROI >= 8 || ROI <= -6`。
- 2023–2024 新6币 discovery：3557笔，PF **0.861**，0/6正，5.677次/币/周；2023/2024 PF 0.792/0.921。
- 旧 PF1.136 的接近候选表现未能保留；2025/2026 OOS未读取，正式冻结。
- 归档：strategy_templates/research/v65-entry-strict-exit-audit/2026-10-01-discovery/。

## 2026-10-01 — v130：4h Double-Inside Compression Breakout
- 2023–2024 discovery only，strict TP8/SL6：49笔，PF **0.441**，0/6正，0.078次/币/周；2023/2024 PF 0.479/0.407。
- expectancy 与频率均明显失败；2025/2026 OOS未读取。冻结，不改inside-bar数量/压缩定义、不加过滤器、不反向。
- 归档：strategy_templates/research/4h-double-inside-compression-breakout/2026-10-01-discovery/。

## 2026-10-01 — v64 Entry Strict TP8/SL6 Audit
- 原 v64 LONG/SHORT entry 完全不变，仅将两条 CLOSE 修正为固定 `ROI >= 8 || ROI <= -6`。
- 2023–2024 新6币 discovery：4479笔，PF **0.788**，0/6正，7.148次/币/周；2023/2024 PF 0.735/0.835。
- 旧 PF1.135 的近门槛表现未保留；2025/2026 OOS未读取，正式冻结。
- 归档：strategy_templates/research/v64-entry-strict-exit-audit/2026-10-01-discovery/。

## 2026-10-01 — ID121 vs v52 Precompression Attribution（新6币）
- 同一 SOL/DOGE/LTC/AVAX/UNI/ZEC、2023–2024、strict TP8/SL6：ID121 211笔 PF **1.059**、0.337次/币/周；v52 151笔 PF **1.298**、0.241次/币/周。
- exact common 150笔 PF **1.316**；被 v52 过滤掉的 ID121-only 61笔全部LONG，PF **0.613**；其中2024被过滤31笔 PF **0.382**。
- 说明固定2–3ATR pre-range在该6币样本确实删除负expectancy LONG，但v52仍因频率0.241与2023 PF0.978不过原gate；不解锁OOS、不调参数。
- 归档：strategy_templates/research/id121-v52-precompression-attribution/2026-10-01-audit/。

## 2026-10-01 — ID121 vs v52 Original-Cohort Attribution
- 原15币 canonical cohort：ID121 773笔 PF **1.211**、12/15正、0.533次/币/周；v52 509笔 PF **1.213**、10/15正、0.351次/币/周。
- exact common 500笔 PF **1.217**；ID121-only 273笔全部LONG，PF **1.200**。
- 被过滤 cohort 年度：2024 PF **0.845**（过滤有帮助）、2025 PF **1.481**（过滤反而删掉大量好交易）、2026 PF **1.006**。
- 结论：2–3ATR pre-range 效果明显 regime-dependent，不能作为稳定过滤器；v52继续冻结。ID121 fresh-symbol holdout失败结论不变。
- 归档：strategy_templates/research/id121-v52-original-cohort-attribution/2026-10-01-audit/。

## 2026-10-01 — v132：Intrahour Partial-QPS Previous-Hour Breakout
- 新机制：当前尚未完成的1h已经累积到 >= 前8个完整1h平均QPS，同时1m首次收盘突破上一完整1h高/低；利用 Engine 的 partial 1h 聚合，不用 NowTime%、趋势/资金费率/taker/外部数据。
- 固定 discovery：SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；strict exit `ROI >= 8 || ROI <= -6`；gate PF>=1.15、>=4/6正、>=0.30次/币/周且2023/2024 PF都>1。
- 结果：**7188笔，normalized PF 0.832938，0/6正，11.471956次/币/周**；LONG 3533笔 PF0.786282，SHORT 3655笔 PF0.879678。
- 年度：2023 3131笔 PF0.834772；2024 4057笔 PF0.831508。逐币PF全部<0.865；4175 SL / 3013 TP。
- 结论：频率充足但跨币/跨年稳定负期望，属于高频追逐过冲；**冻结v132 / 不反向、不调QPS倍率/8h窗口、不叠加taker/趋势过滤 / OOS未读 / 不入库**。
- 归档：`strategy_templates/research/intrahour-partial-qps-breakout/2026-10-01-discovery/`。

## 2026-10-01 — v133：1m QPS Record-Burst Continuation
- 项目原生新机制：当前完整1m QPS严格创最近60分钟新高，按该分钟实体方向做 continuation；不用partial-hour、上一小时突破、trend/taker/funding/外部数据。
- 固定 discovery：SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；strict exit `ROI >= 8 || ROI <= -6`；不扫描窗口/倍率。
- 结果：**17440笔，normalized PF 0.812907，0/6正，27.834017次/币/周**；LONG PF0.810856，SHORT PF0.814634。
- 年度：2023 PF0.807354；2024 PF0.816619；逐币PF 0.771215~0.846128。
- 结论：quote-volume burst高频但稳定负期望；**冻结v133 / 不调30m/120m窗口、不加QPS倍率、不反向、不加趋势/taker过滤 / OOS未读 / 不入库**。
- 归档：`strategy_templates/research/1m-qps-record-burst-continuation/2026-10-01-discovery/`。

## 2026-10-01 — v134：Intrahour Taker-Aligned Previous-Hour Breakout
- 项目原生新机制：使用当前尚未完成1h的 `TakerBuyRatio[0]` 作为实时主动流方向；LONG要求 >0.5 且1m首次收盘突破上一完整1h高点，SHORT镜像。仓库审计确认此前没有 LONG/SHORT entry 使用 `TakerBuyRatio[0]`。
- 固定 discovery：SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；strict exit `ROI >= 8 || ROI <= -6`；不叠加 QPS/趋势/funding，不扫描0.5附近阈值。
- 结果：**13912笔，normalized PF 0.821814，0/6正，22.203374次/币/周**；LONG 7153笔 PF0.803503，SHORT 6759笔 PF0.841424。
- 年度：2023 5624笔 PF0.830443；2024 8288笔 PF0.815927。逐币PF 0.767462~0.901145。
- 结论：实时主动流与突破同向仍是高频稳定负期望；**冻结v134 / 不扫0.52/0.48或0.55/0.45、不加QPS/趋势/funding过滤、不反向 / OOS未读 / 不入库**。
- 归档：`strategy_templates/research/intrahour-taker-aligned-breakout/2026-10-01-discovery/`。

## 2026-10-01 — v135：Intrahour Range-Expansion Previous-Hour Breakout
- 项目原生新机制：当前尚未完成1h的实时 high-low range 已经大于上一完整1h range，同时1m首次收盘突破上一小时高/低；不用QPS/taker/funding/趋势/外部数据。
- 固定 discovery：SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；strict exit `ROI >= 8 || ROI <= -6`；range ratio 只用自然1.0边界，不扫描倍率。
- 结果：**10446笔，normalized PF 0.817519，0/6正，16.671683次/币/周**；LONG PF0.792320，SHORT PF0.840750。
- 年度：2023 PF0.821409；2024 PF0.814619；逐币PF 0.791800~0.863969。
- 结论：与v132 partial-QPS、v134 partial-taker一致，live intrahour acceleration 三种独立表征均高频稳定负期望；**冻结v135并暂停该子方向 / 不扫range倍率、不叠加过滤、不反向 / OOS未读 / 不入库**。
- 归档：`strategy_templates/research/intrahour-range-expansion-breakout/2026-10-01-discovery/`。

## 2026-10-01 — Binance Loan / VIP Loan Asset Removal → SHORT：Feasibility
- Binance 官方 CMS catalogId=49 完整扫描 2023-2024：74篇 Loan 相关标题；明确 removal/forced-close 共4批。
- 非稳定币事件：PEPE；IRIS/IQ/OAX/JUV/MULTI/ARDR/ATM/MLN；PLA，共 **10 token-events / 10币**。BUSD loan+collateral 全面退出按稳定币排除。
- 在未读取任何事件后收益前执行 production eligibility：事件时 USD-M 历史>=730天 + 前24h QV>=500万。**10/10 全部在第一项失败**（合约不存在或历史不足2年），eligible=0。
- 结论：**feasibility blocked / 未看收益 / 冻结**。不降低2年门槛、不用事后历史、不和 Margin/Spot removal 合并凑样本、不入库。
- 归档：`strategy_templates/research/binance-loan-asset-removal-short/2023-2024-feasibility/`。

## 2026-10-01 — Directional Price-Impact Asymmetry：Early Gate
- 新机制：过去24个完整1h分别计算上涨/下跌方向的 `sum(|log return|)/sum(QuoteVolume)`，取 `log(upside impact/downside impact)`；零上穿LONG、零下穿SHORT，下一1h open。区别于总Amihud illiquidity与taker price-impact，不使用外部数据。
- 固定 discovery：SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；gate=12h signed mean >=+0.20%、>=4/6正、>=0.30事件/币/周且两年均正。
- 结果：**9907 events，12h signed mean +0.0004%，3/6正，15.811次/币/周**；1h +0.0097%，4h -0.0018%。
- 年度12h：2023 **+0.0278%**，2024 **-0.0290%**；经济幅度几乎为零且跨年翻转。
- 结论：**early gate失败 / 冻结 / 不扫12h/48h窗口、不加阈值或过滤、不反向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/directional-price-impact-asymmetry/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Simple Earn Asset Removal → SHORT：Feasibility
- Binance 官方 CMS catalogId=49 完整扫描 2023-2024：共 **40篇** Simple Earn 标题；直接 removal/delist/cease/discontinue/close/suspend/redeem 标题 **0篇**。
- 宽松 Update/Notice/Support/Migration 复核仅命中 `Notice Regarding CYBER Simple Earn Flexible Products`；正文只是解释 CYBER 赎回/流动性事件，并写未来低流动性 token **可能**被移除，没有宣布任何实际 Simple Earn removal。
- 因此明确 asset-removal events = **0**，在 production eligibility 和任何事件后收益之前即 coverage fail。
- 结论：**coverage blocked / 未看收益 / 冻结**。不把未来假设语句当事件、不用单币搜索补样本、不与 Spot/Margin/Loan delist 合并、不入库。
- 归档：`strategy_templates/research/binance-simple-earn-asset-removal-short/2023-2024-feasibility/`。

## 2026-10-01 — Binance Deposit/Withdrawal Resumption → LONG：Feasibility
- Binance Maintenance Updates 官方目录 catalogId=157 完整扫描 2023-2024；只接受标题明确表示充值/提现/网络服务已经 resumed/reopened/restored 的确认事件，排除计划升级中“完成后将恢复”的语句。
- 全部仅找到 **1事件 / 1币**：2023-05-29 TORN 恢复 Ethereum/BSC deposits。
- 低于预注册最低8事件/8币，因此 production eligibility（>=2年 USD-M + QV>=500万）和任何事件后收益均未读取。
- 结论：**coverage blocked / 冻结**。不把计划维护公告或其它交易所恢复事件并入凑样本、不入库。
- 归档：`strategy_templates/research/binance-deposit-withdrawal-resumption-long/2023-2024-feasibility/`。

## 2026-10-01 — Token Unlock Supply-Shock：Exact-Timestamp Recheck
- 复用此前冻结的 production-eligible cohort：9事件/8币（FTM、IMX、AXS、RON、APT×2、ID、FET、SEI），不改事件、不看新收益。
- 公开 replication repo 的真实 `01_binance_token_unlock_events_2023_2025.csv` 只有 `unlock_date`，**没有 `unlock_timestamp_utc`/`timestamp`**；但同仓库 `DATA_SOURCES.md`/`DATA_DOCUMENTATION.md` 又声称 T 为链上 UTC 小时级时间戳并要求从 File01读取 timestamp，文档与文件不一致。
- Git 历史确认该CSV自2026-04-20首次公开以来只有一个版本，不存在更早带timestamp的公开revision。Tokenomist虽提供 timestamped unlock-event 产品，但当前公开/可访问历史无法完整覆盖固定2024-2025 cohort。
- 结论：**timestamp coverage仍blocked / 不用00:00 UTC假设、不按可搜到timestamp的事件缩样本、不从重复周期推断 / exact 1m replay未跑 / 无新收益读取 / 不入库**。
- 归档：`strategy_templates/research/token-unlock-supply-shock/2026-10-01-timestamp-recheck/`。

## 2026-10-01 — Binance Unplanned Transfer Suspension → SHORT：Feasibility
- Binance Maintenance Updates 2023-2024完整扫描只找到1个非计划 suspension family：2023-07-05 Multichain incident，正文明确影响 POLS/ACH/BIFI/SUPER/AVA/SPELL/ALPACA/FARM 8币。
- 在未读任何事件后收益前执行 production eligibility：公告时 USD-M 历史>=730天 + 前24h QV>=500万；**8/8 全部在历史门失败，eligible=0**。
- 公告还说明部分 deposits 在2023-05-24已先暂停，因此不擅自把7月公告回填到5月作为可交易信号。
- 结论：**feasibility blocked / 未看收益 / 不降低2年门槛、不混入计划维护、不回填不可审计早期时点 / 不入库**。
- 归档：`strategy_templates/research/binance-unplanned-transfer-suspension-short/2023-2024-feasibility/`。

## 2026-10-01 — Binance Corporate-Action Support → LONG：Feasibility
- Binance 官方 CMS catalogId=49 完整扫描 2023-2024 共 **1071标题**；corporate-action 关键词候选30篇。
- 按预注册规则只保留首次 `Binance Will Support...` token swap/migration/rebranding/redenomination，剔除 completed/update/BNB Beacon generic migration 后剩 **14事件/14币**：MATIC/FRONT/RNDR/STRAX/PLA/TVK/TOMO/MC/AVA/QUICK/COCOS/SXP/BNX/GTO。
- 在未读取任何事件后收益前做 production eligibility：事件时 USD-M 历史>=730天 + 前24h QV>=500万。仅 **MATIC/TOMO/SXP 3/14** 合格；其余11个因合约不存在或历史不足2年排除。
- 低于预注册最低8事件/8币，**coverage blocked / 未看收益 / 冻结**。不降低2年门槛、不把completed公告或新ticker事后并入、不与KuCoin family合并、不入库。
- 归档：`strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility/`。

## 2026-10-01 — v136：Lagged Quote-Volume → Price Lead
- 新机制：过去24个完整历史pair估计 `Δlog QuoteVolume_t -> next-hour return` 的 centered covariance，用当前已完成小时 `Δlog QV` 乘该关系预测下一小时；predictor 零上穿LONG、零下穿SHORT。
- 新6币 2023-2024 early gate：**59539 events，12h signed mean -0.0090%，仅1/6正，95.023次/币/周**；1h -0.0061%，4h -0.0060%。
- 年度12h：2023 -0.0069%，2024 -0.0112%。事后 LONG/SHORT 分化不用于删方向救活。
- 结论：**early gate失败 / 冻结 / 不扫12h/48h、不加阈值、不删SHORT、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/lagged-quote-volume-price-lead/2026-10-01-early-gate/`。

## 2026-10-01 — v137：24h Signed Price-Volume Imbalance Regime
- 文献启发但不做跨币排序：每个完整1h定义 `2*TakerBuyQuoteVolume-QuoteVolume`，过去24h均值零上穿LONG、零下穿SHORT，下一1h open；单币自身信号，无Benchmark。
- 新6币 2023-2024 early gate：**4877 events，12h signed mean +0.0356%，5/6正，7.784次/币/周**；1h -0.0039%，4h -0.0117%。
- 年度12h：2023 +0.0073%，2024 +0.0665%；虽breadth/频率/年度符号通过，但远低于预注册 +0.20% economic gate。
- 事后 LONG 12h +0.2064%、SHORT -0.1348%，但禁止据此删SHORT救活。
- 结论：**early gate失败 / 冻结 / 不扫窗口或阈值、不做ratio normalization、不删方向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/signed-price-volume-imbalance-regime/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Convert Asset Removal → SHORT：Feasibility
- Binance 官方 CMS 2023-2024 完整标题集中共有 **50篇 Convert 标题**；按 removal/cessation 语义复核得到4个非addition候选。
- BIDR/RUB 为法币相关退出，BUSD为稳定币wind-down，NBS为其它生命周期事件后的余额转换；**明确普通crypto Convert-removal事件=0**。
- 因此在production eligibility和任何收益读取之前即 coverage fail。
- 结论：**coverage blocked / 未看收益 / 冻结**。不与Spot/Margin/Loan removal或普通delist合并凑样本、不入库。
- 归档：`strategy_templates/research/binance-convert-asset-removal-short/2023-2024-feasibility/`。

## 2026-10-01 — v138：Lagged Taker-Flow → Price Lead
- 新机制：过去24个完整历史pair估计 `signed taker flow_t -> next-hour return` centered covariance，用当前完整1h主动流乘该关系预测下一小时；predictor零上穿LONG、零下穿SHORT。区别于既有flow autocorrelation与同小时price-flow divergence。
- 新6币 2023-2024：**50484 events，12h signed mean +0.0007%，3/6正，80.572次/币/周**；1h -0.0004%，4h -0.0010%。
- 年度12h：2023 -0.0063%，2024 +0.0076%，几乎无经济幅度且跨年翻转。事后LONG/SHORT分化不用于删方向。
- 结论：**early gate失败 / 冻结 / 不扫窗口或阈值、不删方向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/lagged-taker-flow-price-lead/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Auto-Invest Asset Addition → LONG：Feasibility
- Binance 官方 CMS 2023-2024 完整审计得到 **13篇 Auto-Invest/Recurring Buy 标题**。
- 排除维护、支付选项、比赛、入口调整、指数计划再平衡后，真正新增非稳定币可定投资产只有2023-07-05一批：MAV/PENDLE/WBETH/COMBO/IQ，共 **5事件/5币**。
- 原始 coverage 已低于预注册最低8/8，因此**未读取production eligibility或任何事件后收益**。
- 结论：**coverage blocked / 冻结**。不与Earn/Convert/Spot listing或支付选项合并凑样本、不入库。
- 归档：`strategy_templates/research/binance-auto-invest-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance Futures Position-Limit Adjustment：Feasibility
- 官方CMS 2023-2024完整标题审计仅命中1篇 position-limit：2023-05-15 将 Position Limit Adjustment 功能扩展到 COIN-M；没有任何合约级before/after cap调整。
- **qualifying token-events=0**，未读取行情或收益。
- 结论：**coverage blocked / 冻结**。不把已研究的leverage-tier/notional-bracket调整重复包装成position-limit事件、不入库。
- 归档：`strategy_templates/research/binance-futures-position-limit-adjustment/2023-2024-feasibility/`。

## 2026-10-01 — Binance Corporate-Action First Support → LONG：Feasibility
- Binance 官方 CMS catalogId=49 完整扫描2023-2024 **1071标题**；swap/migration/rebranding/redenomination 机械候选30篇。
- 按预注册只保留首次 `Binance Will Support...`，排除 `Has Completed`、后续update与BNB全网迁移，得到 **14事件/14币**：GTO/BNX/SXP/COCOS/QUICK/AVA/MC/TOMO/TVK/PLA/STRAX/RNDR/FRONT/MATIC。
- 未读收益前执行 production eligibility（事件时USD-M历史>=730天、前24h QV>=500万）：仅 **SXP/TOMO/MATIC 3/14** 通过，其余11个因合约不存在或历史不足2年排除。
- 结论：**coverage gate失败 / 未看收益 / 冻结**。不降低2年门槛、不拿完成公告补事件、不与KuCoin/cross-exchange corporate action合并、不入库。
- 归档：`strategy_templates/research/binance-corporate-action-support-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance Pay Asset Addition → LONG：Feasibility
- Binance 官方 CMS 2023-2024 完整标题审计中仅 **3篇 Binance Pay 标题**：Pay ID迁移、选定Pay服务费更新、Binance Charity支持Binance Pay。
- 没有任何标题明确宣布新增某个普通crypto作为 Binance Pay spending/payment asset；**qualifying token-events=0**。
- 因原始事件覆盖已为0，未读取production eligibility、行情或任何事件后收益。
- 结论：**coverage blocked / 冻结**。不把通用Pay产品变化解释为token级utility事件、不与Earn/Convert/Card/Spot事件合并凑样本、不入库。
- 归档：`strategy_templates/research/binance-pay-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-01 — v139：Directional Taker-Flow Energy Asymmetry
- 新机制：每个完整1h定义 `flow=2*TakerBuyQuoteVolume/QuoteVolume-1`；过去24h分别平方累加正/负flow，取 `log(buy_energy/sell_energy)`；零上穿LONG、零下穿SHORT，下一1h open。
- 新6币 2023-2024 early gate：**4742 events，12h signed mean -0.0024%，2/6正，7.568次/币/周**；1h -0.0276%，4h -0.0185%。
- 年度12h：2023 -0.0076%，2024 +0.0029%。事后LONG +0.0531%、SHORT -0.0578%仅作审计，不允许删方向救活。
- 结论：**early gate失败 / 冻结 / 不扫窗口、平方指数或阈值、不删方向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/directional-taker-flow-energy-asymmetry/2026-10-01-early-gate/`。

## 2026-10-01 — Funding Sign Persistence Reversal：Early Gate
- 新机制：仅正常约8h funding cadence；连续第3次同号settlement首次形成时做 crowding reversal：正funding streak→SHORT，负funding streak→LONG。3次固定为一个正常24h funding cycle，不看幅度。
- 新6币 SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；下一完整1h open入场；2024年末会跨入2025的12h endpoint直接排除，OOS保持未读。
- 结果：**580 events，12h signed mean +0.0168%，3/6正，0.9257次/币/周**；1h -0.0131%，4h +0.0424%。
- 年度12h：2023 **-0.0253%**，2024 **+0.0697%**；LONG 143 events +0.0424%，SHORT 437 events +0.0084%（仅诊断，禁止删方向救活）。
- 结论：频率过门，但经济幅度、breadth、跨年一致性均失败；**冻结 / 不扫2/4/5 streak、不加funding幅度阈值、不删方向、不加trend/taker/OI、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/funding-sign-persistence-reversal/2026-10-01-early-gate/`。

## 2026-10-01 — v140：Conditional Return-Sign Markov Predictor
- 新机制：最近24个完整1h sign transition构造2×2状态转移矩阵；按当前sign估计下一小时条件期望，predictor零上穿LONG、零下穿SHORT，下一1h open。
- 新6币 2023-2024：**33866 events，12h signed mean -0.0065%，2/6正，54.050次/币/周**；1h -0.0053%，4h -0.0032%。
- 年度12h：2023 +0.0091%，2024 -0.0226%。事后LONG/SHORT分化仅作审计，不删方向。
- 结论：**early gate失败 / 冻结 / 不扫transition窗口或概率阈值、不删方向、不反向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/conditional-return-sign-markov/2026-10-01-early-gate/`。

## 2026-10-01 — v141：1h Range-Overlap Value Migration
- 新机制：相邻完整1h high-low区间 overlap ratio 相对前24对均值首次下穿，视为price acceptance突然迁移；当前range midpoint上移LONG、下移SHORT，下一1h open。
- 新6币 2023-2024：**26175 events，12h signed mean -0.0266%，2/6正，41.775次/币/周**；1h -0.0114%，4h -0.0373%。
- 年度12h：2023 -0.0145%，2024 -0.0387%，两年均负。
- 结论：**early gate失败 / 冻结 / 不扫overlap lookback或固定阈值、不加body/wick过滤、不删方向、不反向、不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/range-overlap-value-migration/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Holder Airdrop Support → LONG：Feasibility
- Binance 官方 CMS catalogId=49 完整扫描2023-2024 **1071标题**；airdrop/distribution 标题候选12篇。
- 按预注册排除 Launchpool/HODLer/BNSOL Super Stake/Wallet营销奖励、completion、稳定币分发、挖矿奖励与通用政策后，仅剩 **1个明确外部holder-airdrop事件：CHZ holders → 1000PEPPER**。
- 原始语义 coverage 仅 **1事件/1 holder token**，低于最低8/8；因此 production eligibility、QV 与任何公告后收益均未读取。
- 结论：**coverage blocked / 未看收益 / 冻结**。不把HODLer/Super Stake/稳定币补偿或cross-exchange airdrop并入凑样本、不入库。
- 归档：`strategy_templates/research/binance-holder-airdrop-support-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance P2P Asset Addition → LONG：Feasibility
- 复用已归档的 Binance CMS 2023-2024 完整1071-title corpus；其中 **29篇 P2P 标题**。
- 按预注册只认“普通crypto首次加入P2P”，地区/法币corridor扩展不重复计token事件；USDC/FDUSD/TUSD按稳定币排除。最终只有 **WLD 1事件/1币**。
- 原始语义coverage已低于最低8/8，因此production eligibility、QV和任何事件后收益均未读取。
- 结论：**coverage blocked / 冻结**。不拆地区市场伪造样本、不加入稳定币、不与Spot/Convert/Pay事件合并、不入库。
- 归档：`strategy_templates/research/binance-p2p-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance Token Trading-Fee Change：Feasibility
- 2023-2024 Binance 官方 fee-title 审计按预注册定义保留 **8个独立普通crypto政策批次**；fee relief start/expand→LONG，fee relief end/reduce→SHORT；同一公告无论含多少token只计1批。
- coverage gate 固定要求>=8个 production-eligible 独立批次。BETH zero-maker-fee 批次在事件月2023-04及2年前cutoff月2021-04均无 `BETHUSDT` Binance Vision USD-M 1h文件（404），因此最多只剩7批可能合格。
- 结论：**coverage blocked / 未看任何事件后收益 / 冻结**。不拆公告伪造独立样本、不降低批次门槛、不并入P2P/法币/global fee政策、不入库。
- 归档：`strategy_templates/research/binance-token-trading-fee-change/2023-2024-feasibility/`。

## 2026-10-01 — Order-Flow Coherence：Early Gate
- 新微观结构：过去60个完整1m，`signed_flow=2*TakerBuyQuoteVolume-QuoteVolume`，`coherence=|Σflow|/Σ|flow|`；首次由<=0.5上穿>0.5，净流正LONG、负SHORT。区别于小时净TakerBuyRatio和lag-1 flow autocorrelation。
- 新6币2023-2024：**33067 events，12h signed mean -0.0751%，0/6正，52.7745次/币/周**；1h -0.0228%，4h -0.0359%。
- 年度12h：2023 -0.0396%，2024 -0.1067%。事后方向拆分：LONG 9558 events +0.1007%，SHORT 23509 events -0.1466%，但禁止据此删SHORT救活。
- 结论：combined frozen family 明确失败；**不扫30m/120m窗口、不扫0.4/0.6阈值、不删方向、不反向、不加trend/QPS/funding / 不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/order-flow-coherence/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Token-Burn Support → LONG：Feasibility
- 复用 Binance CMS 2023-2024 完整1071-title corpus；burn关键词仅 **4篇**，全部为第22–25次 **BNB Auto-Burn 已完成报告**。
- 按预注册只认“首次 Binance 将支持 protocol-level burn/supply reduction”，completion与同一BNB机制重复执行均排除；**qualifying events=0**。
- coverage 在production eligibility和任何收益之前失败。
- 结论：**coverage blocked / 未看收益 / 冻结**。不把BNB季度重复burn拆样本、不把完成报告改造成前瞻信号、不与外部治理/tokenomics burn合并、不入库。
- 归档：`strategy_templates/research/binance-token-burn-support-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance Legacy Staking Asset Addition → LONG：Feasibility
- Binance CMS 2023-2024 完整corpus中有19篇 staking 标题；按预注册只认 legacy Binance/Locked/DeFi Staking 的普通token首次新增，排除ETH/NFT/Loan、removal、Simple Earn、BNSOL/SOL liquid staking、Babylon BTC On-chain Yields。
- 最终仅 **CVX 1事件/1币**（2023-06-14 DeFi Staking Adds Support for CVX）。
- 原始coverage低于8/8，因此production eligibility、QV和任何事件后收益均未读取。
- 结论：**coverage blocked / 冻结**。不合并新liquid/on-chain staking产品、不用APR促销或ETH/NFT事件凑样本、不入库。
- 归档：`strategy_templates/research/binance-staking-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-01 — Binance Proof-of-Reserves Asset Addition → LONG：Feasibility
- Binance 官方 CMS catalogId=49 的 2023-2024 完整1071-title corpus中，精确 PoR 标题仅3篇：zk-SNARK方法升级、2023-03-08 Eleven New Tokens Supported、2024 collateral-info升级。
- 按预注册定义，只有 **2023-03-08 一篇**属于“首次新增 PoR 资产”批次；其余2篇是方法/验证信息升级。
- coverage gate要求 >=8合格token、>=8 unique且 **>=3独立批次**。独立批次仅1，因此在production eligibility、QV和任何收益读取前即失败；不会把一批11个token伪装成11个独立事件。
- 结论：**coverage blocked / 未读市场数据与收益 / 冻结 / 不入库**。
- 归档：`strategy_templates/research/binance-proof-of-reserves-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-01 — Snapshot Governance Approval → LONG：Feasibility
- 复用已归档的626个安全映射 Snapshot proposals；严格二元语法得到43个 clean binary proposals，其中正向选项胜出41、负向胜出2。
- **41个 positive wins 全部来自 LDO / lido-snapshot.eth 单一token、单一space**（2023=24，2024=17）；预注册要求 >=4 token + >=4 independent spaces，因此在 production eligibility 前即失败。
- 结论：**coverage blocked / 未读取 USD-M 历史资格、QV 或任何公告后收益 / 冻结 / 不入库**。不把大量LDO提案伪装成跨币证据，不放宽安全space映射。
- 归档：`strategy_templates/research/snapshot-governance-approval-long/2023-2024-feasibility/`。

## 2026-10-01 — v142：Interpretable Two-Split Fast-Move Discovery
- 目的：在统一线性模型失败后，仅允许一个极小、可解释的非线性交互；2023 train、2024 validation，2025/2026保持未读。六币共用同一对称模型，无symbol特征；固定9个项目原生、DSL可翻译特征。
- label严格围绕4x TP8/SL6 fast move：下一1m open按5bps adverse slippage入场，未来最多24h仅用1m close判TP/SL；TP先到reward +8，SL先到-6，未决样本不参与拟合/评估。
- 2023 resolved **91,379/104,844**；固定depth-2 tree最终只使用 ret12_side / ret1_side / ret4_side。4个leaf的train mean reward分别 **+0.2993 / -0.0287 / +0.0307 / -0.3116**。
- 最佳leaf仅+0.2993，低于预注册train selection阈值+1.0，因此 **selected leaves=0**；没有合法规则可进入2024 selected-validation。2024 pool虽已按冻结流程生成，但selected样本=0；2025/2026未读。
- 结论：**冻结v142 / 不加树深、不扩quantile、不降reward gate、不减leaf size、不换特征、不改24h label / 不生成strict Engine策略 / 不入库**。
- 归档：`strategy_templates/research/interpretable-two-split-fast-move/2026-10-01-discovery/`。

## 2026-10-01 — v143：Funding Volatility Expansion Fade
- 新机制：仅正常约8h funding cadence；最近3次settlement（约24h）funding标准差 / 此前21次（约7天）标准差首次由<=1上穿>1时，按最近3次funding合计符号做crowding fade：正→SHORT、负→LONG。为隔离已研究的interval-compression，参与当前/上一ratio的所有funding gap预先要求7.5h–8.5h。
- 新6币 SOL/DOGE/LTC/AVAX/UNI/ZEC，2023-2024；下一完整1h open入场；2025/2026未读。
- 结果：**863 events，12h signed mean +0.2197%，5/6正，1.3773次/币/周**；1h -0.0054%，4h -0.0069%。LONG 149 events +0.6627%，SHORT 714 events +0.1272%（方向拆分仅审计，不允许删方向）。
- 年度12h：2023 **-0.0358%**，2024 **+0.4721%**。经济幅度/breadth/频率均过门，但预注册“两年均正”失败，判定明显regime-dependent。
- 结论：**冻结v143 / 不扫3/21窗口、ratio阈值、cadence tolerance、不删方向、不反向、不加过滤 / 不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/funding-volatility-expansion-fade/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Borrow / Margin Interest-Rate Adjustment：Feasibility
- 复用 Binance 官方 CMS catalogId=49 的2023-2024完整1071-title corpus；检索 interest rate / hourly interest / borrowing rate / loan rate。
- 仅命中 **1篇**：2023-02-20 `Binance Margin Introduces Dynamic Interest Rate Updates`，属于通用动态利率机制公告，不是token级before/after利率调整。
- **qualifying token-events=0**；因此production eligibility、行情和任何事件后收益均未读取。
- 结论：**coverage blocked / 冻结**。不从当前动态利率规则反推历史token利率、不把Margin/Loan新增资产并入凑样本、不入库。
- 归档：`strategy_templates/research/binance-borrow-interest-rate-adjustment/2023-2024-feasibility/`。

## 2026-10-01 — v144：Confirmed 1h Williams-Fractal Breakout
- 新OHLC结构：标准5-bar Williams fractal，只使用左右两侧都已完成后确认的局部swing；每个完整1h从过去24h选择最近已确认high/low fractal，fresh close上破LONG、下破SHORT，下一1h open。
- 新6币2023-2024：**10035 events，12h signed mean -0.0776%，0/6正，16.0157次/币/周**；1h -0.0036%，4h -0.0418%。
- 年度12h：2023 -0.0689%，2024 -0.0859%。LONG 5154 events +0.0171%、SHORT 4881 events -0.1776%仅作事后审计，禁止删SHORT救活。
- 结论：**early gate明确失败 / 冻结 / 不扫12h/48h horizon、不改3/7-bar fractal、不加retest/trend/volume过滤、不删方向、不反向 / 不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/confirmed-williams-fractal-breakout/2026-10-01-early-gate/`。

## 2026-10-01 — v145：Quote Ease of Movement(14) Zero-Cross
- 新机制：每个完整1h计算 `EOM=(midpoint_t-midpoint_{t-1})*(High-Low)/QuoteVolume`，固定14根均值；零上穿LONG、零下穿SHORT，下一1h open。只用Engine已有High/Low/Amount，可直接DSL化。
- 新6币2023-2024：**12085 events，12h signed mean -0.0201%，2/6正，19.2875次/币/周**；1h -0.0116%，4h -0.0355%。
- 年度12h：2023 +0.0078%，2024 -0.0487%；LONG -0.0268%，SHORT -0.0134%。
- 结论：**early gate失败 / 冻结 / 不扫EOM周期、阈值或平滑方式、不加过滤、不删方向、不反向 / 不进strict Engine / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/quote-ease-of-movement-14/2026-10-01-early-gate/`。

## 2026-10-01 — Binance Portfolio Margin Collateral-Asset Addition → LONG：Feasibility
- 复用 Binance 官方 CMS catalogId=49 的2023-2024完整1071-title corpus；Portfolio Margin / collateral asset 宽筛34篇，但绝大多数是collateral-ratio调整、Loan/VIP Loan collateral或通用program变化。
- 真正直接“新增资产到Portfolio Margin支持”的标题只有 **1篇：2024-12-19 BFUSD**；BFUSD属于稳定币型margin/reward asset，不是目标普通crypto token。
- 因此 qualifying ordinary-crypto events = **0**，低于8事件/8币门槛；production eligibility与任何收益均未读取。
- 结论：**coverage blocked / 冻结**。不把ratio调整、Loan collateral、通用program变化或BFUSD类margin单位并入凑样本、不入库。
- 归档：`strategy_templates/research/binance-portfolio-margin-collateral-asset-addition-long/2023-2024-feasibility/`。

## 2026-10-02 — DeFiLlama Protocol TVL Momentum
- 新基本面机制：复用既有 protocol→token 唯一映射 universe；每个协议的7个完整UTC日 TVL log-growth 零上穿 LONG、零下穿 SHORT，下一UTC日 USD-M open；动态资格仍为历史>=730天 + signal-day QV>=500万。
- 初始 current-REST loader 因 CVX/SXP/BTCST 等历史/下架合约返回0/400产生survivorship bias，判为 invalid preflight 并单独保留；正式结果改用 Binance Vision 历史月档，**信号、universe、gate均未改变**。
- corrected discovery 2023-2024：**1479 events / 17触发币，7d signed mean -0.7793%，4/17正，0.8331次/币/周**；1d -0.0640%，3d -0.4467%。
- 年度7d：2023 **-0.4588%**，2024 **-1.0865%**；LONG 735 events -0.8306%，SHORT 744 events -0.7285%。
- 结论：经济幅度、breadth、两年方向均明确失败；**冻结 / 不扫3d/14d/30d窗口、不加TVL-size/category过滤、不删方向、不residualize、不反向 / 2025+未评估 / 不进strict Engine / 不入库**。
- 归档：`strategy_templates/research/defillama-protocol-tvl-momentum/2026-10-02-discovery/`。

## 2026-10-02 — v146：20h Quote-Volume-Weighted Close Reclaim
- 项目原生机制：过去20个完整1h用 QuoteAssetVolume 加权 Close 得到 QVWC20；已完成1h close 新穿越 QVWC20 后，当前价再突破 trigger-bar 高/低确认；无EMA/ADX/funding/taker/time过滤，strict exit `ROI >= 8 || ROI <= -6`。
- 新6币 2023-2024 strict Engine：**6202笔，normalized PF 0.822533，0/6正，9.8983次/币/周**；LONG 3230、SHORT 2972。
- 年度：2023 2618笔 PF **0.813249**；2024 3584笔 PF **0.829465**。逐币PF全部0.795~0.869；3617 SL / 2583 TP。
- 结论：高频但跨币/跨年稳定负期望；**冻结v146 / 不扫QVWC周期、不替换true VWAP、不加过滤、不删方向、不反向 / 2025+未读 / 不入库**。
- 归档：`strategy_templates/research/quote-volume-weighted-close-20h-reclaim/2026-10-01-discovery/`。
