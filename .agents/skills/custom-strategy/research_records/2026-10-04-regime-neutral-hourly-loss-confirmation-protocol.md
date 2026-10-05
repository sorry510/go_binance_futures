# RG8：全趋势小时结构确认止损（首次收益前冻结）

2026-10-04北京时间02:55:48，13793 actual terminal exit0、4738真实Go/Expr合成检查0失败后冻结，RG8完整收益尚未读取。目标实际active。RG7全部主/审计已终止，不重跑旧句柄。

## 原因、唯一变化及完整候选

RG7真实943笔账目和原Position close核验联合归因，结果保存仓库外 results/20261004-rg7-exit-path-attribution.json（SHA a4d69d755ca835e73afe00d79e9e3305af74ef0280511acddcae5afe6ea5f236）。基础入口ETH/SOL/XRP分别16/40/21笔退出时ROI<=−20；补充族的弱小时结构-only退出中位持仓约2.33–4.93小时，四币该组净负。该归因不是删组可实现PnL或私有短路日志，不能证明放宽退出必然提高收益。它指出需检验趋势强度仍高时结构失败的止损迟延，而非继续任意增加入场过滤。

唯一逻辑变化：在完整原RG7关闭程序追加 `ROI <= -5 && adx_4h_14.ADX[1] >= 20 && hourlyAdverseBreak`。LONG用forming Close[0]<上一闭合Low[1]，SHORT用Close[0]>上一闭合High[1]，严格破位。结合已有ADX<20弱结构亏损分支，−5外结构失败在全部强弱趋势下可确认平仓。盈利退出、原16/−12/28市场确认分支和−20唯一ROI-only灾难例外全不变。当前强度未必等于入场强度，统一退出不依赖OpenStrategyHash，所有基础和补充入口完整对象和原9指标配置相同。

假设：新止损覆盖可能减少强趋势局部失败后的灾难损失，亦可能增加噪声止损/费用或减少之后恢复的赢家；必须全撮合再裁决，不静态删账目估计收益。没有新增指标/周期/变量环境、修改普通ROI门槛或选择币/方向。

- temp_strategy/20261004-regime-neutral-hourly-loss-confirmation/00-regime-neutral-hourly-loss-confirmation-family.json SHA bda4685b2ebeb9c8a8ffb7cb0c60c59e5a1176080e3ccc2c26574fa06d705692 /version a1533adb221458fcc35fc1d86227b98a1ae622d5461ae633782b213148f49bb9
- temp_strategy/20261004-regime-neutral-hourly-loss-confirmation/01-v29c-regime-neutral-hourly-loss-confirmation.json SHA fd95c6d59bda9992c00d7515d435895788f011c78182c9a68c84b1a2c0783060 /version 1d5bb8e281e919f4519f4e84749381713c7ae12127ab8e69e637d9d3b187d5a6

族4条/组合6条启用规则，均含long/short/close_long/close_short。379旧portable/777同侧完整close程序比较无精确去空白重复；开仓故意与RG7完全一致，不是语义/alpha新颖性证明。全部失败中间候选仍保留temp_strategy。

## 关闭矩阵及验证范围

| 双侧情形 | 结果 |
| --- | --- |
| 普通ROI ±5/+16/+28/−12，没有原或小时反向确认 | false |
| ROI<=−5且小时严格反向破前Low/High，ADX弱/等于20/强 | true |
| ROI>=5且小时反向破位、ADX<20 | true，保留原RG4盈利结构分支 |
| 盈利小时破位且ADX>=20 | 新分支false，原16/28确认仍可成立 |
| ROI内区间或结构等值 | 新分支false，内区间外部门槛不调用 |
| ROI<=−20 | true，唯一无市场确认灾难例外 |

外部profit/loss5/5仅调用资格，不是±5必平；AutoStopOrder=false。4738检查含扩大ROI边界、闭合ADX0/19.999999/20/20.000001/25/50×三种结构×两侧×空/错/真开仓hash，以及原全部日线/八小时价格和quote方向/基础入口矩阵；matrix SHA 99a9a40a338d6fa49fb3a6c1567ec7bc2b8355106631817637eeecdb39bf4dbc。这是实际Go/Expr合成与本地有序规则模型，不是private/cache/API/live/forward或市场盈利证明。Data[0]故意保留，不新增forming taker0。

## 冻结完整撮合与硬门槛

AF0/RG7/RG8×BTC/ETH/SOL/XRP，共12完整49月run，每币初始1000。UTC2022-09-01含至2026-10-01不含（CLI end2026-09-30），1491天/213周/每币至少192笔。每币>=0.9次/周、净正、四年稳定、跨币泛化和真实成本联合要求不放宽；四完整Sep–Aug、额外2026-09、日历2023/24/25/2026 Jan–Sep单列。45月exit cohort只归因非独立初始化，49月开发已观察不称时间holdout。AAVE/ATOM/ETC/LINK未观察收益未读，开发失败不进入验证币。

原backtest_engine_v7/standard_1m、已观察分钟close/下一分钟Open、现金10%保证金/8倍、两侧各fee0.0005/slip5bps、真实历史funding和原缺mark观察分钟Close回退均不变。当前精确mark未齐，不称真实成本完整通过；不补零/造mark/改生产引擎。数据public-canonical-repaired-v2-funding-tail-v1，各币原data SHA、replay/data helper、suffix54/overlap7和manifest SHA与RG7相同，每次cache全身份/CRC核对。

## 主后审计及恢复

主保存 results/20261004-rg8-development4-canonical-repaired-v2-funding-tail-v1.json。核对8个AF0/RG7控制逐笔/metrics/年度/source精确重现、全账目/年度/方向/族/集中度；全部补充入口保持RG7原代码，仍按canonical闭合字段/实际200input199closed原指标函数重算，包括日线ADX/DI。全部新正常退出用原Position、observed exit_time−1、gross mark-price-denominator ROI/外部门槛、原完整RG7 close与新增强结构-loss条件核验；forced-end单列。fresh不是顺序/private/API/live/forward/独立数学/订单簿或删分支PnL。全现金复利qty/fill/fee/每次真实funding核验，缺mark回退次数单列。

新三库只读20187 terminal exit0，截止02:43:13.230/02:43:15.818/02:43:18.155：模板17/17/17、结果221/218/7，v29三库摘要相同、go_binance新全导出与RG5 parsed相同；不是446行新forward内容分析。原protected SHA全部相同，禁止App/DB写入/分配/启用/生产/前端/config/新仓库测试文件。正式skill1.0.8/trusted:false、新技能评估pending未晋级。诊断Go/overlay和大输出仅仓库外缓存，virtual源实际不创建。主启动后保存真实handle，先poll同handle，不因超时重起；下次先阶段总结核goal/真实进程，阶段结束不自行complete/paused/blocked。

