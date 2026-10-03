# PV4：日线方向一致的单维冻结对照

2026-10-03北京时间18:38–18:40，在观察PV4收益前冻结。基础AF0及PV3所有现存入口/指标/平仓原样；只对PV3补充LONG追加闭合日线PlusDI[1]>MinusDI[1]，SHORT反向。两侧对称，不加ADX强度或EMA门槛、不改ratio/Qps/no-chase/资金费参数，仍不读forming taker[0]；价格和已观察实时量[0]不改。用来检验PV3的4h方向与日线方向冲突是否造成部分坏入口，不能从旧账本族亏损推导删除交易后的新盈利。

日线DMI原子v29已有，并非新优势；381个既有portable入口去空白全程序比较没有精确同对入口，不是任意等价改写的形式化去重证明。两完整新JSON在temp_strategy/20261003-pullback-daily-flow/：

| 文件 | SHA-256 |
| --- | --- |
| 00-daily-aligned-flow-family.json | ebe1cbb4ff9fec951301f5692be6ea871ed17e1df47429f53c1512f796c65fee |
| 01-v29c-daily-aligned-flow-pullback.json | 02b9e4d38cc5c7116b0f4a396387a534a6b120eb30c35de0f0ae34b38df40cbd |

helper顺序base LONG/supplement LONG/base SHORT/supplement SHORT；明确-exit-source base，无guard/hash绑定；全部四类启用、9个technology及原AF0两个关闭对象完全原样。普通16/28盈利与-12亏损需趋势/动量/反向冲量确认；仅-20灾难例外可ROI-only；外部5/5资格不等于自动退出。

完整同条件重撮合AF0/PV3/PV4×BTC/ETH/SOL/XRP，共12次，UTC2022-09-01..2026-08-31、1461日、每币至少188笔/0.9周频、逐币净正、四个完整Sep–Aug稳定、跨币泛化。保留PF/DD/成本/方向/族集中度，开发失败不观察冻结AAVE/ATOM/ETC/LINK收益，不选盈利币、不改门槛。初始1000/币、现金10%保证金复利、8倍、双边各0.0005与5bps、历史资金费、外部5/5；原standard v7/1m与canonical repaired v2数据hash不变，结果另存，不覆盖PV2/PV3。

真实Expr检查覆盖精确PV3→PV4 raw唯一差异、闭合DI同向/反向/相等/0/epsilon与[0]相反无影响、原ratio/入口/关闭与本地有序匹配；不能把局部模型当实际私有selector或盈利。闭合taker接收路径PV3真实3833分钟诊断仅可在源码/数据hash完全相同后复用；PV4新日线DI规则仍需本轮验证。完整会计/全账本新增ratio条件及分钟活动审计单独执行，若开发门槛通过才扩大前向/未见币/逐接收者验证，不以未覆盖检查作成功证据。

只有app.conf注释arm三库程序只读元数据/v29复查，不操作App/配置/生产/前端/DB写/启用，没有新增仓库测试文件。生产taker[0]修复仍无明确人类授权，不修/换引擎；新的closed实验可安全继续，goal active不缩减。

## 完成结果（冻结参数不变）

12次完整四年运行结束、8个原AF0/PV3对照完整复现。PV4四币频率通过，但BTC/ETH净亏、四币都有亏损年度，联合失败，不扩大未见收益验证/发布/写库/启用。1472 Expr通过；755补充入口独立canonical信号/身份及原9指标真实fresh历史receiver+完整entry表达式验证0失败；1068笔交易活动无零量填单，2620笔会计一致。完整顺序cache/私有selector/live/forward/订单簿等范围不冒称通过。见2026-10-03-pullback-daily-flow-summary.md。
