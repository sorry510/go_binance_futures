# RG4：弱趋势小时结构确认退出（首次收益前冻结）

2026-10-04北京时间00:44保存完整候选，00:51:41.576核对身份；2806合成检查之后、完整收益之前写本协议。目标active；RG3完整12run/3445笔已失败，不重复其原主。

## 完整候选和唯一变化

目录temp_strategy/20261004-weak-hourly-structure-exit/：族00-directional-volume-escape-hourly-exit-family.json SHA19a2d6f8e0e375d5b216fd2d509296b29f4d971733fa343d00046d276334103c /version75c21f09024b9c85a1c125c8d9ff44f4f14c0cf2e9c6cb7e86cf97a575781c32；组合01-v29c-weak-hourly-structure-exit.json SHA62c14f8161688d27d24e6f8c59fc85c4028b55130cf9312a269e7a06d2b0c7c9 /versionea83eeadbb6e3bba3d341f84441440cfdddd7dae5ec2ef45bef4b416910db700。

族4/组合6规则，完整RG3开仓规则及名字/原9指标/周期/其他字段逐对象相同，不加开仓过滤，不删短/币别选择。只改两侧统一关闭：保留原所有确认分支和-20灾难例外，追加`(ROI>=5 || ROI<=-5) && closed4hADX<20 && current1hPriceBreak`；多为Close[0]<Low[1]、空为Close[0]>High[1]，等值不触发。这是当前弱趋势小时价格结构的全模板统一退出，不按开仓族/hash路由，避免未授权的forward hash接收缺口。forming价格[0]有意保留，不读forming taker0或新增指标/系统变量。

弱趋势小时持仓时间尺度的假设来自RG3完整失败与补充持仓统计，但不由其静态PnL算出新收益。原开仓仍有强/弱两个区域；新关闭ADX分类取当前已闭合4h，不保证与开仓时状态相同，明确会对变成弱趋势的base仓位生效。

399旧portable/817同侧完整close程序无精确去空白重复；开仓与RG3故意完全重复。精确去重非语义/alpha优势证明。

## 已运行矩阵

1091实际terminal exit0，verification/rg4_expr_checks.go及results/20261004-rg4-expr-checks.json：2806/0，实际Go structs/Expr/局部有序模型。包括所有入口原样、原基础可达、只增加新关闭维度的整对象断言；两侧ROI{-20,-12,-5,-4.999,0,4.999,5,16,28}、弱ADX0/19.999999/20/25、结构无破坏/等值/破坏、空/错误/实际hash，使用原close程序同环境的实际结果与独立新branch布尔值核对union。含静态/重复案例，非2806独立市场样本、缓存/private/API/live/forward/盈利证明。

| 两侧条件 | 关闭 |
| --- | --- |
| 仅ROI到+5/-5/+16/+28/-12、确认不成立 | false |
| ROI±5外且闭合ADX<20且小时结构向相反方向破坏 | true，新正常分支 |
| 新结构等值、ADX>=20或ROI在(-5,+5) | 新分支false，原确认分支按原逻辑 |
| 原>=16趋势失败或动量+反向冲击、<=-12趋势/动量失败、>=28动量/反向冲击 | true，完整保留 |
| ROI<=-20 | true，唯一ROI-only灾难例外 |

外部profit/loss5/5是调用资格，不是在5%必定平仓；AutoStopOrder=false，内区间不运行close。新正常分支仍需要实际价格结构和弱趋势，不把ROI阈值伪装成市场证据。

## 冻结撮合和硬门槛

AF0/RG3/RG4×BTC/ETH/SOL/XRP，12完整run各1000初始现金。UTC2022-09-01含至2026-10-01不含（1491天/213周，每币至少192笔）；四完整Sep–Aug年、2026-09和日历2023/24/25/2026 Jan–Sep全部报告，45月cohort非新初始化。每币>=0.9次/周、净正、四年稳定、跨币泛化联合门槛不变，开发窗口已观察而非未观察时间验证。AAVE/ATOM/ETC/LINK仍未观察、资格待核对，开发失败不读其收益。

原standard v7/1m、已观察分钟close决策/下一分钟Open成交；当前可用现金10%保证金/8倍、双边各fee0.0005/slip5bps、真实funding时间/率和原缺mark已观察结算分钟Close回退、外部5/5、灾难-20不变。新close是策略逻辑变化，不是生产engine或成本变化。

复用public-canonical-repaired-v2-funding-tail-v1，public-archive/public-archive、verified-archive修复版本20261003-v2；四币data_hash及原replay/data helper/suffix manifest SHA与RG3协议一致，仍逐次完整重算cache身份。RG3自身3762结算/1104mark回退，真实费率不等于精确交易所mark，不补零/制造精确mark或隐改runtime。新study另报实际回退，成本资格受限；分钟非零活动非订单簿容量保证。

## 预声明审计、保护、恢复

主完成后对8个AF0/RG3完整旧对照、全部会计/年度/日历/方向/族/集中度、现金复利quantity/fills/fee/实际funding逐笔复核。补充开仓仍以canonical闭合小时/原fresh receiver的实际200input/199closed seed验证ATR/ADX/EMA与量价。正常关闭另以exit_time−1观察分钟、历史真实entryPrice/quantity/side、原Position接收器/grossROI/外部gate，执行原close与新完整close并比较新结构branch；不拿净profit/保证金替代原mark-price-denominator ROI。强制end_of_data单列，不误当表达式触发。fresh不证明全顺序cached/private/API/live/forward或独立数学；关闭branch归因不是删除分支后的反事实收益。

主输出results/20261004-rg4-development4-canonical-repaired-v2-funding-tail-v1.json。启动后检查点记实际handle，先poll同handle/完成run数，不因超时重复启动。所有失败JSON保留temp_strategy，大输出/辅助程序/诊断overlay在仓库外缓存，真实overlay源目标必须不存在。

保护config/engine/environment/cache SHA与RG3收尾一致，正式技能1.0.8/trusted:false不变，新receiver-seed/mark技能行为pending，无promote。最新三库metadata仍00:11:54.640/57.451/00:12:00.033，不重复声称444行新forward复查。禁止App/DB写入/分配/交易启用/生产或前端/config/新仓库测试文件。下次开始先阶段总结，核对goal/实际进程/冻结身份；若paused停止、不自行恢复；没有合格策略、不complete或因困难/局部缺数据标blocked。
