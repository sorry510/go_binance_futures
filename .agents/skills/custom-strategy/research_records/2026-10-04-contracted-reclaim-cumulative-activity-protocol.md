# RG17：扫边收回增加当前累计成交额门槛 — 收益前协议

## 冻结与唯一变更

2026-10-04 北京时间14:20建立：RG17主回测未启动，未读取任何RG17收益；上一RG16主及审计均终止，成本失败原件保留。

本轮只在RG16两个完整补充入场程序尾部追加同一条件：kline_1h.Qps[0] >= mean(kline_1h.Qps[1:9]) * 0.90。不更改其他变量、判断、价格[0]、参数、指标、规则次序、基础入口与平仓整对象。原positive Amount[0]仍保留。

已完整读取PV5的00-recovery-cumulative-volume-family.json并核对现有live GetLineFloatValues、historical klinePriceSeries、klineQPSDurationSeconds和cachedKlineQPSDurationSeconds/cachedKlinePriceSeries。当前与闭合1h均除名义3599.999秒；门槛是观察到的当前累计quote至少达到前8根闭合[1..8]平均quote的90%，不是经过时间归一速率，不保证下一分钟容量或alpha。不使用尚未修复的forming taker字段。

依据仅为RG16原始SOL seq244零成交活动填单及同信号已观察累计quote约0.0035%闭合均值。该失败保留，不删除净额、不修改canonical/生产engine、不根据事后收益加回。加入门槛后的持仓挤占、资金复利与退出必须从头完整顺序撮合，不能静态删交易作为新收益。

RG16几何完整冻结：LONG反向Low[1]严格扫过recent[2:6]Low−0.10ATR再Close[1]收回Low+0.10ATR；SHORT严格High扫过High+0.10ATR再Close收回High−0.10ATR。前一[2]按[3:7]/ATR[2]相同收回拒绝；recent4h区间严格收缩；闭合quote[1]超过前8[2..9]非负均值；合法买quote多侧<半、空侧>半；closed4h weak OR aligned EMA/DI；closed强反向日线排除；极值到收盘0.15..2ATR与当前−0.15..+0.35ATR保持。下限0.15已被严格扫边+收回蕴含，不另调参数。

## 已完成的收益前检查

- 独立族4规则、组合6规则，均含long/short/close_long/close_short与原9指标/1h4h1d。
- 族temp_strategy/20261004-contracted-reclaim-cumulative-activity/00-contracted-reclaim-cumulative-activity-family.json：SHA6f77d651f4b64a97ec64ab88d636842ad21b3b9eb57e84c173a6209e3481d25d，version92edc06f2044523be89c92bf6d477d0a3bbccba676f6a22b38b3b2e2da7f9ce3。
- 组合01-v29c-contracted-reclaim-cumulative-activity.json：SHA68e4a73b854cf2207d8b832085b78bdd10bc2ab3364fb077114863e4ae5cffd6，versionfb20107ca0e87d184f6c9e915a78456d05d37ab1a26eb279310f845324a1344c。
- actual Go/Expr 26873 terminal exit0，4414passed/0failed，结果SHA7bfd2eef8c29f5fa8096c2398e715c7128affa75a77864b7e4d6e8d8864618d2。完整程序与上一程序+唯一条件exact；独立数值oracle、90%相等/低于/高于、QPS0/1/8参与与9/10不参与、nominal/clock、原价量方向/几何/收缩/首次/ADX日线/完整uniformRG4关闭/ROI-only拒绝/灾难例外/AF0可达均检查。刻意扰动QPS字段案例明确为synthetic，不冒充canonical。合成与本地顺序模型不是private evaluator、历史顺序、API/live/forward或盈利证明。
- actual frontend TypeScript validator隔离VM两issue=null/9指标/四类型/shape有效，SHA4da4ef21ba4981ef57d9bf46481f728f9a0c57720b5c055386e94c82680caeb0；未操作App/前端UI或production build。限定307旧portable697启用entry/0完整同侧精确重复，identity SHA99388074b43211d0f9a7fb933c1179b334f375e3fc5ce8a5c9c16c7bffcdf8c2。排除自身、research、diagnostic/audit/>128KiB，非全局语义或alpha新颖证明，未读收益。
- 新opening helper32879 actual build terminal0，源码SHA87cc9d66d6d7ef5938a35e48c9ae46fc9bd11526fee53bd085af2c8feed986a4、binarya2d59bea01b71ee314f29ad61639c3886485d9077761b622cf1101d68fce06c7。独立canonical已观察分钟累计quote、闭合[1..8]mean、原receiver current/mean QPS和门槛纳入全部新增信号核验；桥接仍仅原新鲜环境导出，不改生产。

## 程序只读数据库元数据

ARM snapshot49705 actualterminal exit0，read_only=true；go_binance/go_bn_oracle1/go_bn_oracle2分别17模板与222/219/8测试记录，各有独立截止时间。三库v29技术/策略语义hash一致，当前go_binance导出的raw v29 SHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0与冻结原DB基线exact。AF0是有意研究v29C变体，不冒称rawDB模板原样。
快照与导出results/20261004-rg17-arm-{metadata,v29}-snapshot.json；只读取元数据和v29，不声称本轮重新分析所有forward结果。未写入/分配/启用数据库。

## 完整回测、成本与发布门槛不变

- AF0/RG16/RG17组合×BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT，UTC2022-09-01..2026-10-01 exclusive，完整49月/1491天/213周/12run。每币0.9次/周至少192笔，四完整Sep–Aug年与额外Sep、完整日历2023/24/25以及Jan–Sep2026全部报告；不选币、侧、年份或降频率。
- 原backtest_engine_v7/standard_1m、分钟close观察→下一分钟open执行、原200input/199closed指标种子、初始现金1000/每笔当前cash10%保证金/8倍、双边fee0.0005/各侧不利5bps、真实funding与缺精确mark的原分钟Close回退、outer5/5门控/AutoStop=false均保持。所有基础与补充使用whole uniform RG4确认退出和唯一ROI−20灾难例外，不做hash分流。
- canonical四币SHA c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457 / 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371 / 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818 / 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。funding tail54条/7overlap/manifest78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547完全保持。
- 主输出results/20261004-rg17-development4-canonical-repaired-v2-funding-tail-v1.json自身是逐run恢复checkpoint。真实terminal前不重复启动，所有12run完成再裁决。
- 全12账本算术及8AF0/RG16完整共享控制对上一RG16逐字段exact，候选path只固定workspace/realpath等价，其他snapshot字段不得排除。全部开仓新gate、正常关闭Position/ROI/wholeRG4/outergate、forced、现金数量、资金费及当前/退出分钟活动全量独立审计；旧RG16已知零活动预计原样保留，须分别明确all-study/自身失败，不能把全量失败改0。
- 自身净/频率/所有完整年共同通过才取得未阅验证资格，AAVE/ATOM/ETC/LINK收益仍未读。精确结算mark/订单簿真实滑点容量仍pending，回退算术正确不等于真实成本全部通过。
- 收益前预设仅以入口closed4h ADX20原OR分weak/strong，对全部补充与真实关闭逐笔配对描述；不扫阈值、不删组造反事实PnL、不推inverse edge。

## 权限、技能与阶段恢复

无App、DB写、分配、启用、下单、production/frontend/config变化、新仓库测试文件或材料删除。conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa；正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd和用户metrics CSV规则保留。已有SkillMax三pending副本不晋级/不重复创建；无需因为已有活动审计的新增失败实例复制技能。

目标实际active，无合格策略；本续接开头已给阶段总结。下次若达到限制恢复，仍先实际阶段性总结、核get_goal/真实进程；阶段结束不能自行complete/paused/blocked。

