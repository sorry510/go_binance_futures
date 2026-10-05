# RG6完整研究总结：价格同向没有一致改善稳定性

2026-10-04北京时间02:12:52核对完成。主21928/开80315/成本43265/关87284均实际terminal exit0，12完整49月run/2635执行账目（含共享对照），8旧控制完整重现、会计errors0。目标active，所有联合门槛未通过，不能发布或进入未观察验证币。

| 币 | 笔数 | 周频 | 净USDT | PF | 回撤% | 四完整Sep–Aug净USDT |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| BTCUSDT | 256 | 1.202 | 410.136 | 1.228 | 19.404 | -13.747 / 45.435 / 37.355 / 412.335 |
| ETHUSDT | 274 | 1.286 | 227.615 | 1.114 | 29.833 | -160.663 / -73.482 / 258.262 / 259.970 |
| SOLUSDT | 297 | 1.394 | -414.104 | 0.820 | 45.466 | -276.131 / 59.893 / -33.660 / -126.843 |
| XRPUSDT | 241 | 1.131 | 18.503 | 1.010 | 27.376 | 21.232 / -189.554 / 148.229 / 57.051 |

四币频率>=0.9，但SOL净负，四币负完整年，联合失败，裁决invalidated（对本研究全部发布要求）。价格确认相对RG5改善ETH/XRP、恶化BTC/SOL；XRP总净略正不能掩盖年度失败、利润集中或精确成本未补齐。没有把某币失败剔除或选择每币最佳版本。

| 币 | 日历2023/24/25/2026 Jan–Sep净 | 追加2026-09净 | 去最佳5笔净 | 补充LONG数/净 | 补充SHORT数/净 |
| --- | --- | ---: | ---: | --- | --- |
| BTCUSDT | 170.754 / -104.155 / 194.389 / 225.685 | -71.243 | 122.095 | 90/-218.804 | 68/55.264 |
| ETHUSDT | 3.552 / -92.613 / 215.040 / 256.543 | -56.471 | -124.392 | 97/69.242 | 78/-170.681 |
| SOLUSDT | -14.987 / -271.988 / -115.757 / -18.987 | -37.363 | -730.688 | 68/-229.175 | 97/-255.135 |
| XRPUSDT | -55.481 / 156.765 / -222.558 / 187.840 | -18.456 | -340.833 | 42/-54.188 | 112/-272.857 |

年度/方向/族/45月cohort仅exit-time归因，非独立初始化或删组反事实。SOL全部gross -208.431已经负，不是只调费用能补救。弱4h方向及小时价量同向仍不保证日线环境同向；下一轮预声明只增加对强度已到原v29日线阈值20且DI相反的排除，保留RG6全部其他条件与全部RG4关闭。这是待检验日线冲突机制，不把目前数据归因成已证因果、静态账目筛选或删短方向。当前RG7尚未生成/测试。

## 已完成审计及未完成资格

652补充/52160 canonical闭合字段/failed0，实际原9配置fresh receiver200input/199closed原ATR/ADX/EMA函数另算；逐笔Open[9]/Close[2]和前八小时quote/主动额有效性、加权方向、形成价格与活动也核对。不是独立数学、全顺序缓存/private/API/live/forward或订单簿证明。

正常close1068/forced-end0/failed0；old true509/RG4 added563/both4/added-only559，原Position、gross mark-price-denominator ROI/外部gate/完整关闭程序和独立新结构条件核对。归因不是去分支后的反事实PnL。

全部2635笔/4578真实funding应用/1187原观察分钟Close mark回退，zero activity0/arith failed0；RG6自身1068笔/1702应用/444回退。资金费率/时刻完整不等于交易所精确mark，真实成本资格仍缺，不补零/伪造mark/修engine。完整成本/年度/跨币硬门槛不放宽，AAVE/ATOM/ETC/LINK未观察收益仍未读，资格待核。

原UTC2022-09-01..2026-10-01(不含)/1491天213周每币至少192笔；现金10%/8倍、两侧各fee0.0005/slip5bps、funding原算法、外部5与5/-20唯一灾难例外和signal-confirmed普通close全部原样。关闭矩阵同首次协议：ROI普通阈值无确认false，弱ADX与方向结构破坏且±5资格true，等值/内区间新增分支false，原16/−12/28确认及−20例外保留；AutoStopOrder=false。无API/live/forward验证。

## 文件身份、保护、恢复

完整族/组合temp_strategy/20261004-closed-price-flow-alignment-escape/及收益前protocol保留，3266合成Expr检查0失败，375旧portable/1626全程序比较无精确重复但非语义/alpha证明。原9配置/基础/关闭对象完全相同。数据和helpers SHA冻结，与RG5原标准引擎相同；8 AF0/RG5控制逐笔/metrics/年度/source全重现。

- results/20261004-rg6-development4-canonical-repaired-v2-funding-tail-v1.json SHA ed964b8c87b1ae3b81d591e52b40bcefc0cc62c19cb2db5f019fa657d4c6f1d1
- results/20261004-rg6-accounting-summary.json SHA 146a9228c00c9480beda579e8eb9d064b181b85bc34e391d44cbe9bd18e10379
- results/20261004-rg6-expr-checks.json SHA 2e5131629db63c875450ffd0f4e1c0698172a1c39715f400edcbbc733dda7262
- results/20261004-rg6-canonical-actual-seed-open-signal-audit.json SHA 1937b29e980336dde2f6a50d34809680813d9f2ecc42caa3d9c6de85947345a6
- results/20261004-rg6-execution-original-cost-fallback-audit.json SHA f64fb879096b47d2c32a8bd502cf98ad3750796052533fbaa1a3ac76e2091d4a
- results/20261004-rg6-original-position-close-signal-audit.json SHA e70cb94be87323cff191756df83fdcee25236ad5c82eeb6e172457877c3e1c80

元数据截止仍01:39:58.217/01:40:03.394/01:40:08.651：模板17/17/17、结果221/218/7、v29三库相同，不是446行新forward内容复查。02:09:13保护conf/app.conf、engine/environment/indicator_cache/正式SKILL SHA和全部四新候选不变，git diff --check通过，其他用户研究改动保留。无App/DB写入/分配/启用、生产/前端/config/新仓库测试文件；正式skill1.0.8/trusted:false，新SkillMax行为pending未晋级。下次开始先阶段总结核goal/真实进程，不重跑已terminal主，不自行complete/paused/blocked。
