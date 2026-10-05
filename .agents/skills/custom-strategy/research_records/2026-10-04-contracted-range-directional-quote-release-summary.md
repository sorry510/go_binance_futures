# RG13：相邻闭合范围收缩后主动 quote 同向释放 — 完整研究总结

## 当前结论

- 2026-10-04 北京时间 10:17：12 组完整 49 月撮合已实际结束（主 25653 terminal exit 0），包含 8 组共享对照，3953 笔账目；RG13 自身 2623 笔。RG13 **invalidated**，没有合格可发布策略。
- 四币交易频率均超过每币 0.9 次/周，但 BTC、SOL、XRP 净亏，四币均存在负完整 Sep–Aug 年及负完整日历年。不得挑币、删侧、改变成本或以累计收益代替四年稳定。
- 本轮归因不是新策略独立收益。独立族 JSON 没有单独回测；完整组合才是四币研究对象。未读取 AAVE/ATOM/ETC/LINK 验证收益。

## 冻结方案及有效执行范围

- 只替换 RG12 两补充入口为 recent [2:6] 四闭合小时相对 older [6:10] 四小时的严格范围收缩、合法 closed[1] 同向主动 quote 多数/放量、首次越过 recent 边界 0.10 ATR；previous[2] 对 [3:7] 尚未释放。弱 ADX 或强四小时 EMA/DI 同向允许，日线反向排除、实体/实时保留范围保持。
- AF0 两基础完整入口、原 9 指标配置、whole uniform RG4 完整多空关闭、风险参数、真实引擎逻辑、时间范围、数据/成本身份均不变。没有使用 MarketCondition、forming Taker[0] 或新增指标。
- 时间为 UTC 2022-09-01 至 2026-10-01 exclusive，1491 日 / 213 周，每币至少 192 笔。四完整年按 Sep–Aug 分段，额外 2026-09 另列；日历 2023/24/25 及 2026 Jan–Sep 同时报告。
- 风险/执行：8 倍、每次当前现金 10% 保证金、两边费用 0.0005 和不利滑点各 5bps、资金费原始时序、外部正负 5% 门控、AutoStop=false；不减少成本补救收益。

| 币 | 笔数 | 次/周 | 净 USDT | PF | 最大回撤 | 平均持仓小时 |
|---|---:|---:|---:|---:|---:|---:|
| BTCUSDT | 557 | 2.615 | -83.602 | 0.980 | 31.62% | 22.354 |
| ETHUSDT | 674 | 3.164 | 327.572 | 1.063 | 40.56% | 15.093 |
| SOLUSDT | 745 | 3.498 | -584.351 | 0.900 | 72.19% | 10.104 |
| XRPUSDT | 647 | 3.038 | -43.353 | 0.990 | 58.48% | 13.050 |

每币初始资金与复利路径独立，不将此表相加当作一个可交易组合收益。

| 币 | 2022-09..2023-08 | 2023-09..2024-08 | 2024-09..2025-08 | 2025-09..2026-08 | 额外 2026-09 |
|---|---:|---:|---:|---:|---:|
| BTCUSDT | -65.577 | 82.014 | -104.647 | 41.425 | -36.817 |
| ETHUSDT | -204.963 | 24.378 | 421.013 | 97.559 | -10.416 |
| SOLUSDT | -183.952 | 284.695 | -508.825 | -201.794 | 25.525 |
| XRPUSDT | 30.851 | -442.125 | 64.593 | 214.225 | 89.103 |

上述年度均为原完整撮合的退出时间归因，不是重新初始化的独立年度组合。

## 实际验证及首次检查器错误

- 收益前预声明与原 4958 Go/Expr 合成检查、真实前端导入 validator VM、限定 300 文件 / 677 启用全入口身份检查均保留，不冒充市场盈利证据。
- 原会计检查实际 exit 1，只有两项 control snapshot changed。只读递归差异证明：旧候选 path 相对、新候选 path 绝对，其他字段一致，当前文件 SHA 也一致。首次错误输出原样保留。
- 单独 v2 检查器仅将候选 path 相对固定工作区 resolve 并 realpath 比较，其他 snapshot 字段仍全部严格比较；没有忽略哈希、策略、指标、名称或版本。v2 actual exit 0，12 组 / 3953 笔 / 8 完整共享对照复现 / errors 0；未重跑主回测。
- 开仓 13724 actual exit 0：2337 补充实际原 receiver / canonical 配对，186960 闭合字段、11685 四小时值、7011 日线值、4674 ATR 值、14022 范围值，actual 200 input / 199 closed 种子，0 失败。所有合法多数、收缩、首次释放、方向/OR、原完整 Expr 都核验。
- 平仓 20478 actual exit 0：2623 正常、forced-end 0；old 1463、added 1175、both 15、added-only 1160、0 失败。使用原 Position/ROI/外部门控以及完整统一 RG4，不把强制结束当正常关闭证明。
- 成本 90677 actual exit 0：3953 笔、8094 资金费应用、2452 缺 mark 的观测分钟价回退；RG13 自身 2623 / 4666 / 1436。原实际现金复利/下一分钟活动/双边成交价及手续费/结算包含规则 0 失败、零活动成交 0。**精确交易所资金费结算 mark 仍 pending**，不能把算术一致称为 exact venue costs。

## 原交易归因与失败机制

| 币 | 补充弱组笔数 / 净 | 补充强组笔数 / 净 | 弱组 added-only 平仓净 |
|---|---:|---:|---:|
| BTCUSDT | 258 / -35.711 | 242 / -542.740 | -455.112 |
| ETHUSDT | 333 / 71.229 | 268 / 269.523 | -331.597 |
| SOLUSDT | 319 / -738.617 | 334 / -11.904 | -848.679 |
| XRPUSDT | 330 / -255.803 | 253 / -290.937 | -707.773 |

- 2337 补充全配对；按真实 closed 4h ADX20 的弱/强 OR 边界事前固定归因，没有搜索阈值。弱组与强组都不一致盈利；不能据此直接生成 strong-only 或 weak-only 盈利宣称。
- 补充 SHORT 四币净归因均负；LONG 只有 ETH 为正。BTC、SOL、XRP 补充合计 gross 已负，不能只怪费用。ETH 累计正但首完整年负，原账目去掉最佳五筆的描述性归因转负；不代表删除交易后的新净收益。
- 弱趋势 added-only 原交易四币净归因负，很多持仓仅数小时。候选失败可能包含弱趋势实时越过上一根高低点触发的噪声退出，但这是待进一步 canonical 闭合价格/流量确认诊断的机制，不从亏损直接推导反向盈利，不删除旧亏损组。
- 下一步若继续实际 active：仅对原关闭时可观察的 canonical 闭合方向/合法主动 quote 多数做预声明诊断，再冻结一个统一多空的关闭确认变化；不得用未来小时回填过去、网格挑阈值或未经完整重撮合称新策略有效。当前 RG14 尚未生成。

## 文件身份

- 族：temp_strategy/20261004-contracted-range-directional-quote-release/00-contracted-range-directional-quote-release-family.json；SHA 4258b05982c8d2703f9d03373b9007da3b0ecce1e9c20efad966de5a71262ac7。
- 组合：temp_strategy/20261004-contracted-range-directional-quote-release/01-v29c-contracted-range-directional-quote-release.json；SHA a1eb5a1e0080a8e75e9c9da199a4e055d8ab759904ab34875f2316dd49b3cae5；version bea53f9acfc843d4007effa54d358ad77a3c825df6ff659f9e4111aa40ff5f1f。
- 所有结果根目录：/Users/zhz/Library/Caches/go-binance-strategy-research/results/
- 20261004-rg13-development4-canonical-repaired-v2-funding-tail-v1.json：SHA 4e1d633b28c10e73bda00bd2ad955e0376a5df37c4e2fe50cbe1c773236ff7ae。
- 20261004-rg13-accounting-summary-v2.json：SHA 0880cb82c89e7f7d72b40cfab595ecaed794d33921d7f45e00fec485ec915f18。
- 20261004-rg13-expr-checks.json：SHA 9cf5a104f6af0cca1f60c986bd418811aa75cdc2deaa56e0d4525a24563731a7。
- 20261004-rg13-canonical-actual-seed-open-signal-audit.json：SHA f254c1bba230533bb7da0c03c3747334a836440cdbeed507af3a1e1cc5099850。
- 20261004-rg13-execution-original-cost-fallback-audit.json：SHA d96be07b2c8f4c1c0466a7a3ab3b6c6bb2565299f4bda80e5896415642fd79bf。
- 20261004-rg13-original-position-close-signal-audit.json：SHA 1e9db5cb51346bd2607591e93a70afc077a3090a051479198535be6806ce024f。
- 20261004-rg13-natural-entry-regime-attribution.json：SHA 4e3199946f134b508cd03d1b5174d1c9d7207e0c11fcb89345626c5d055a5517。
- 20261004-rg13-accounting-summary.json：SHA 5a4802e7d51ed9a2580ffa1fb3eab271c1dbb0ab33616114e3279989356aa002。
- 自然分组 helper 冻结 SHA 71e8f91698c49a930bc9b4095db57d1db1e9437fb5bffc9ea4af31ba3a2dca8b；在执行分组结果前已创建并 node --check 通过。

## 边界及阶段恢复

- conf/app.conf、生产引擎/environment/indicator cache、前端、数据库模板/策略分配/交易启用/订单均没有修改；没有操作 App 或增加仓库测试文件，未删除任何材料。
- 最新 ARM 只读 snapshot 58634 exit 0，10:04:26..30 三库 templates/results 为 17/222、17/219、17/8，v29 SHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0 byteexact旧。这是元数据和 v29 身份，不是 449 条新 forward 全量分析。
- 本阶段所有实际主/审计已 terminal；不重复 poll 旧句柄或重跑完整已结束回测。完整目标仍 active 且未达标；不能因为阶段结束自行 complete/paused/blocked。
- 下次开始前先总结本报告与检查点：完整组数、硬门槛、失败原因、文件、真实在运行进程和下一步。正式 skill 保留外部 metrics 新增行；两个 SkillMax 草案行为 gate pending，未晋级。
