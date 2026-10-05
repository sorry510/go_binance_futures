# RG3：弱趋势顺4h均线方向放量脱离（完整收益前协议）

2026-10-04北京时间00:22:22.440核对完整候选身份，首次收益之前冻结。本协议写于首次主研究之前、生成及1510项合成检查之后，不伪称早于生成。目标active，RG2完整12run/2833笔已审计但失败；不重复RG2原主，不改开发/验证门槛。

## 候选和单一变化

完整JSON在temp_strategy/20261004-weak-trend-directional-volume-escape/：

| 文件 | SHA-256 | snapshot version |
| --- | --- | --- |
| 00-directional-volume-escape-family.json | 81fd398a2c5c368a61c2f715443b78b6e7dd8970a577b66ee46eb311dc12f752 | cdc7dfcfddd8243e685333b29aab279fcdf54f00550db2ab4a20c6e91d399c77 |
| 01-v29c-directional-volume-escape.json | 567d3c7c14c6cce34c1a79d4424f32c7aabbb2eb13cb5a648365a63c115d0c32 | 2acec6db06143cb55376c2203559188aa39317030aad74907260bc50506ad72d |

RG2四币补充SHORT均亏、SOL补充LONG也亏，成交参与不保证中周期可持续。因此只加一个方向一致维度：LONG要求闭合4h EMA20[1]>EMA50[1]，SHORT要求<，相等拒绝。两侧对称，不删空头，不事后选币/年。RG2弱ADX<20、8小时首次极值脱离/0.10ATR、闭合quote高于前8平均/有效主动quote同向占比、实体0.15..2ATR、当前价反向0.15/同向0.35ATR和当前quote>0完整不变。原9指标/基础/统一退出对象身份不变，不新增周期、指标或变量，不改有意forming[0]或使用forming taker0/OpenStrategyHash。

族4/组合6规则，base LONG / supplement LONG / base SHORT / supplement SHORT。机械生成只改补充规则名字/上述code和候选显示名；现有variant脚本会对全部规则加suffix而改变base/close名字，所以此处用受限读取+apply_patch生成，并实际核对整对象身份。

397旧portable/855同侧全程序去空白比较无精确重复。完整读取相关v198弱趋势EMA20 Volume Reclaim入口：其是1h EMA偏离和forming成交恢复后1m均线收复，不是闭合小时极值脱离与4h均线方向。没有读其收益；精确去重不证明语义或alpha新颖。

## 实际合成检查和关闭矩阵

verification/rg3_expr_checks.go使用实际Go/Expr：30336 terminal exit0，1510通过/0失败，结果results/20261004-rg3-expr-checks.json。追加EMA方向−/0/+与相反forming EMA、only-closed-direction-dimension-changed整对象检查；沿用已隔离base的补充fixtures，并单独保留原base可达。闭合ADX/quote/taker/极值/实体/live边界、原指标/基础/关闭对象、禁用/局部有序模型、两侧ROI/hash矩阵均通过。含静态/重复案例，不是1510独立市场样本，非私有selector、cached receiver、API/live/forward或盈利通过。

| LONG/SHORT情况 | 统一关闭 |
| --- | --- |
| ROI仅到+5/-5/+16/+28/-12，市场确认不成立 | false |
| ROI>=16且趋势失败，或动量失败且反向累计量/实体冲击 | true |
| ROI<=-12且趋势或动量失败 | true |
| ROI>=28且动量失败或反向冲击 | true |
| ROI<=-20 | true，唯一ROI-only灾难止损例外 |

空/错误/正确开仓hash不影响统一关闭；外部profit/loss5/5仅调用资格，内区间不运行，AutoStopOrder=false，不承诺在5%必定平仓。

## 冻结研究设计与成本

- AF0/RG2/RG3三个组合×BTC/ETH/SOL/XRP，共12完整从1000重新撮合，族单独收益未计划，不能从组合静态删除成交重算。
- UTC2022-09-01含至2026-10-01不含（1491天/213周，每币至少192笔），四完整Sep–Aug年、2026-09增量及2023/24/25/2026 Jan–Sep日历归因齐备。较短45月cohort不是新初始化，也不替代用户四年门槛。
- 原standard v7/1m，已观察分钟close决策/下一分钟Open成交；当前现金10%保证金/8倍，双边fee0.0005/slip5bps，真实资金费时间/率，原缺mark→已观察结算分钟Close回退，外部5/5/关闭参数不变。
- 每币>=0.9次/周、净正、四年稳定、跨币泛化联合通过；过滤若降低频率仍原判失败，不调阈值/费用/币/年。开发周期已多轮观察，非untouched time holdout，参数仅在本候选利润前冻结。
- 未观察AAVE/ATOM/ETC/LINK名单不变，资格待核对，开发失败不读验证收益，不声称候选可盈利或发布。

相同cache public-canonical-repaired-v2-funding-tail-v1，public-archive/public-archive、verified-archive repair20261003-v2，真实tail manifest SHA78758a90cd0d7bbce2cb37f74272a38d5f4682a59b03f26ea4dfb652e1ee6547，每币54 suffix/7 overlap。replay/data复制SHA21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8 /a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63，完整cache hash和日期/周期/source每次重新验证。

四币data_hash沿用RG2完整协议：BTC c521ace1ceff44534353fbff6179a929402a8f6f96381482670cf43a1612f457；ETH 7c4bb4e7985cf9a0edb7b5eafdc72241aab954f2839c50d1cbf71271391b4371；SOL 64be83c45564b68a75e344bfa32712dfec20f498f62deea32da993a6f4ef8818；XRP 8aa4acab1e7e3fd8bec851d1e66356a715f358f5f7bc06f901c95e6000b774e3。

RG2自身5325结算/1452缺mark回退。真实费率与原引擎算术不等于完整精确交易所mark，完整成本资格仍受限。新轮另数回退，不补零资金费、合成精确mark或默改runtime，不因局部缺口停止安全同模型研究。分钟非零活动非订单簿保证。

## 审计、保护与恢复

主完成后核对2833笔旧研究中的8个AF0/RG2完整控制（新总成交数以实际结果为准）、所有新交易会计/年度/方向/族/集中度/现金复利fill/fee/真实funding与mark回退。对RG3全部补充信号entry_time−1复核canonical闭合量价/首次条件/live已观察quote和价、200 input/199 closed原ATR/ADX以及新闭合4h EMA20/50方向；fresh receiver不是全顺序cache/private/live/forward/独立数学证明。overlay目标必须实际不存在，不变生产cache。

主结果results/20261004-rg3-development4-canonical-repaired-v2-funding-tail-v1.json；每run落盘，启动后记录实际handle，不因超时另起。最新三库metadata截止00:11:54.640/57.451/00:12:00.033，17/17/17模板和221/217/6结果，v29摘要及go_binance完整导出语义相等；不是新444行内容分析，不再全量重复查询。

app.conf/engine/environment/cache保护SHA与RG2协议一致；live技能1.0.8/trusted:false未晋级，新receiver-seed/mark覆盖行为评估pending。禁止App、DB写入/分配/启用、生产/前端/config修改、新仓库测试文件。失败候选保留temp_strategy，大输出/临时辅助程序在研究缓存。quota或中断恢复前先阶段总结，再核对goal/实际进程/身份；若paused停止，不自行恢复。
