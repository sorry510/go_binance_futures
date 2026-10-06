# RG32 单一10bps滑点压力测试：新收益前协议

冻结时间2026-10-06 01:51:40 UTC（北京时间UTC+8）；goal active。原RG32主及全部审计实际terminal0，61旧freeze/七原结果SHA当前exact；没有活主/审计，新压力主输出不存在。原49月5bps收益已观察；本协议先于10bps新回放/新收益。

## 目的与唯一改动

固定同一AF0/RG31/RG32完整JSON×BTC/ETH/SOL/XRP，12完整顺序run。只将每侧adverse slippage由5bps加倍为10bps；原current-cash10%保证金×8、每侧fee.0005、outer5/5、历史funding及原缺mark分钟Close回退、49月窗口、所有rules/order/九指标/data/source/engine不变。

新的cache外独立Go回放副本与原harness逐字节比较，逆转唯一SlippageBps literal后完全相等；go build实际terminal0，新binary已哈希。生产engine/environment/cache/loader/conf全部不变。沿用完整原校验/采集逻辑，不假造新runtime version；engine仍backtest_engine_v7，**不同RunConfig与输出**准确区分压力场景。MAIN不使用overlay。

这是一次非适应性双倍成本检验，不是滑点或ROI网格，不再按结果寻找刚好过线的bp值。完整再撮合体现entry价/ROI/退出时点/后续entry/资金费/cash复利变化；不能静态从旧PnL扣费或删除高占比/亏损交易。

## 预声明判定与未完成事项

同原窗口UTC2022-09-01 inclusive到2026-10-01 exclusive、1491天/213周，每币至少171笔及≥.8/完整周，不要求每币盈利。固定四1000独立钱包组合net正、四完整Sep–Aug年组合正、日历2023/24/25组合正、去最大贡献币仅集中度描述net正。年份和聚合不变，所有单币负年/方向/集中度披露。

压力场景数值通过，只能说明更严假设下的稳定性，不证明真实盘口/10bps实际可成交或精确mark。若不通过，则记录成本余量不足，保留原5bps已通过的事实，不声称成本压力失败推翻原算术；不会降低原声明门槛或读取未阅AAVE/ATOM/ETC/LINK来选结果。精确funding mark、历史盘口、完整真实成本与跨币发布仍pending。

原容量诊断2215行/自身871行全匹配、0缺行/重复/活动0；自身entry p95 .4252531114%、max8.0176312843%。整分钟quote是事后活动，不是盘口深度，不能据此把5bps判为真实或设置事后筛币/侧/年条件。

## 后审计范围

主实际终止后才运行独立完整会计及开/关/all12成本审计。会计保持ledger chronology、snapshot/rule hash、gross/fee/funding/net、213周、完整年/额外Sep、方向等原检查；只允许config.slippage_bps10vs原5。八AF0/RG31控制是相同source/data/engine的压力对照，**不声称重现原5bps trade/PnL**；原5bps八控制完整复现证据已保存。

开/关使用原已冻结binary和**新实际压力账本**、相同fullversion和portable；关闭old comparator仍完整pre-strong RG20（不是父RG31）。所有正常/forced分类及新added-only相对范围明确。成本binary读取study.Config，逐笔重建当前cash数量、10bps下一分钟fill、双侧fees、funding/mark回退和分钟activity，all/own/control所有失败保留。三后审计实际terminal后才natural/phase及完整人类总结。

Go/Expr91276/0是当前同一冻结strategy已完成的分支预检，本次不复制成新执行结果；API3333未连接是历史实际blocked，不自动启动App/服务。所有新helper Node --check实际0。没有新策略参数、生产/前端/conf/_test.go/DB/模板/分配/启用/交易/globalmemory/新委派变化。

## 唯一主命令

```text
rtk proxy /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_replay -conf conf/app.conf -strategy-files temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json,temp_strategy/20261006-strong-hourly-loss-structure-exit/01-v29c-strong-hourly-loss-structure-exit.json,temp_strategy/20261006-strong-hourly-joint-loss-structure-exit/01-v29c-strong-hourly-joint-loss-structure-exit.json -symbols BTCUSDT,ETHUSDT,SOLUSDT,XRPUSDT -start 2022-09-01 -end 2026-09-30 -execution-source public-archive -indicator-source public-archive -minute-repair verified-archive -cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2-funding-tail-v1 -archive-cache-root /Users/zhz/Library/Caches/go-binance-strategy-research/public-canonical-repaired-v2/archives -output /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-slippage10bps-development4.json
```

只能启动一次并追踪真实返回句柄；partial/checkpoint/timeout不是terminal。必须按实际句柄结束再审计，不能把原60518或旧审计当压力进程，也不能因观察超时重启。

## 收益前冻结文件

```text
7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa  conf/app.conf
ae88603241a5c6c840017398019a41ac1ef658b18a76cde619e51add8a05b6ad  service/backtest/engine.go
b98434951e4d7a0c044f133f7538ecfe931869376a74e7252ea6e7b7b14a3cd5  service/backtest/environment.go
1a930117edc927bd7233c95edecdef4a672344f2084e301e3c7715a22ee063a0  service/backtest/indicator_cache.go
aef51d7cadaa5b561b4fdb893c4da3a2e089c3b69ecd47263c8553a0b35804b8  feature/strategy/line/technology.go
270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd  .agents/skills/custom-strategy/SKILL.md
b44a3ee66e0f2aa7e6360d33f14c55cd54a09db49ff7139996657c7525a31b95  .agents/skills/custom-strategy/research_specs/20261006-rg31-strong-hourly-loss-structure-exit.json
b27cb840b0ee92bc165b64fed7b8e5574946d50cf7922f4e419b31aa78e6d04c  temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json
12a1a4c924e7c9bed118337ed883d123de42734ad7d59451522b964568eb5f15  temp_strategy/20261004-extreme-anchored-reclaim-followthrough/01-v29c-extreme-anchored-reclaim-followthrough.json
6f625a8a9a953b649f165e253986e3cbc0715c746e0bd0d6ea31843c196ebffe  temp_strategy/20261006-strong-hourly-loss-structure-exit/00-strong-hourly-loss-structure-exit-family.json
9074f901841467acd32fe99bab042b82fe82bae90d0cefa81effd45e4d57adb0  temp_strategy/20261006-strong-hourly-loss-structure-exit/01-v29c-strong-hourly-loss-structure-exit.json
21cd7e45d5d132671ebd36abd986a2a93dd81733cad3b15ba0838c4cef9012b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_replay_funding_tail_copy.go
a340335a5c780078e071fa2cf285662fd3205b7b530253216ab628853c2adf63  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/pv5_arm_data_funding_tail_copy.go
5945f0be3780af2f8aee5b2744f22b82715932f4f6c51d0c87c1322e561dc3c8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_exit_expr_checks.go
817ed68036d722ebdcacf760742f2825d2a11fb89575968a94937bb5917c01b8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg20_closed_signal_audit.go
b06f898db75ae3de85ba7dee9d75b588880c2a588fb2aaff785557ef41193e82  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg20_closed_signal_audit
8aa504d7eb079a0cc7f73542bc626866c1a0026213e50493e3598a8af51315c9  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_signal_receiver_bridge.go
b705eeebf7649c01a05d025027b55c50cd04b38955b98f0a47a71861acf804d2  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg13_receiver_overlay.json
6ec4126ff2684dfeafb46e862793ad161cebd7767effb3799478826ac954f389  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg4_close_receiver_bridge.go
e594a079993d40feb1fce5d28cba85e2fe57f3ff1e08456fd2195f99f5deca08  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_close_signal_audit.go
f64aecc71217ae2d29697e16473eadb812e6bd4ca80c1c38975c944bf10ca998  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_close_signal_audit
86cb7f9a28c9890edfa750c512a9039a987b2845e3974e9c5ef0a7a39a4ed14c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_close_receiver_bridge.go
ca9f05173f3b12414fec857b19752fbad5ebc3f75abfc1e38244b01cdb6b2ef8  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_close_receiver_overlay.json
9edfadbbc08f772d31b2ae592deca38d78505699d82cd485f15b19e8f62bb1ed  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_accounting_audit.mjs
8e84eb4e98318192a5fc98cb644b459cf20c2274d926df041dfc7fc41cd05d05  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_regime_attribution.cjs
5fa342d51c7919ba061097c967882a3746419ab65c03e45e3126c3c5904a03e4  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_report_evidence.cjs
216ebf6afbd278008cbb1c38194896cac7ba3b8f4d502169c454af950de70b02  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_portable_preflight.cjs
ada03ce7f3b86c2196f4804442516e85d6b95add30cc2ea70391d558be68c24b  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg10_execution_cost_audit
21c2d006f12511a758e65a8bac8abe54e3f572879c5316c42c34cca2f0c72d64  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg31_local_v29_snapshot.go
088ff9709704c6ea8485e269c640cd3bd22d9e40227cf09f58c80aa3e9a19788  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg31-expr-checks.json
80f1795c3c6e6468d957f145db7957b57cd03e8a895d3da0b983cde50bc549f0  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg31-frontend-contract-checks.json
fb9182554e275569f55391e0e2f23398076442939ff5a0b2a987bfe4867823aa  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg31-portable-close-identity-scan.json
1488ee8e66670ed49f367461af78f3f64f85088686978ae5ee447ec42a7363e0  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg31-api-preflight-blocked.json
cee8416487fd4622870863f5d880790fc9852a0e7c6bf43695650e3503cc842a  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg31-local-v29-snapshot.json
1e0e06693a190a20fde834978feb26cf9ab04cadf10f6a5fbb147f943aad7c48  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg30-development4-canonical-repaired-v2-funding-tail-v1.json
9c94aeb1210e4e4757b5b05a1193d0e868a5791e27457451d7200e02eb6957ee  /Users/zhz/work/binance/go_binance_futrues_new_ui/src/utils/technology.ts
dbae210e400ba86d36f3a2b72ede38be8aba0777e4cd1dfeb8b5518ea21820d9  temp_strategy/20261006-strong-hourly-structure-exit/01-v29c-strong-hourly-structure-exit.json
9d556165a29899694dcf7ddae952a9e8572ddf9c23153147c4379918014ccdf8  feature/strategy/line/line_custom.go
c4aed02a320a10457c861a34c10878c6e97cf97f6f6e46eca22f826c573c2217  feature/strategy/line/parse_technology_config.go
1ad15bbbb231bc43eb77d7e9da9bfa21bf135bee151f4000be2a4dca0357f104  technology/define.go
028e296ea0e2129ec01b0485525050705f13cc309694c892056e67cc71a0fb82  STRATEGY.CN.md
68f30d11d7df75cfe91742e7f60c16d31d6be426ec6f3998781e0bcc1e5ad5f1  .agents/skills/custom-strategy/research_records/2026-10-06-strong-hourly-loss-structure-exit-summary.md
01314bc035edc90fd0c65f86a7d5bda31dce430cc51fdc49a03c0be856a5163b  .agents/skills/custom-strategy/research_records/2026-10-06-strong-hourly-loss-structure-exit-protocol.md
9932d0314d1d5bc325c23bc1c36ba7d2faa50a2a6d79a9fe55c83d007dd8846a  .agents/skills/custom-strategy/research_specs/20261006-rg32-strong-hourly-joint-loss-structure-exit.json
839c4d33e23ce0786cdd72ca705512ff468daf737bcd744ce1330b808acf4f7d  temp_strategy/20261006-strong-hourly-joint-loss-structure-exit/00-strong-hourly-joint-loss-structure-exit-family.json
51b182991491ef122942bc5f29a33f4a28e2439234da03da7359d31584112638  temp_strategy/20261006-strong-hourly-joint-loss-structure-exit/01-v29c-strong-hourly-joint-loss-structure-exit.json
55bde931447473aae002d9baedbddbc63036c0819035a46289ffeff819344c45  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_exit_expr_checks.go
e8d5f300c123a2018ffff8e430892679cb3f71a22b70327e9712aa688b6feab0  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_close_signal_audit.go
9dde960aef9b0036832c6729eec756fbd17e93c99968f83cac2f2b9571ce52fa  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_close_receiver_bridge.go
d49892fcaed06da8d9ddf2efc31026bc2f033c0159ae3900ecedac8ff75b4668  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_close_receiver_overlay.json
43fca7a2d345d965237ee1b1e957459051b45b5f9f50f211cc4d6c271f9a6932  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_portable_preflight.cjs
37713626b60736144fda893a641d678c17e172d924a8e8ec8493a811ec8fec4c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_local_v29_snapshot.go
ec4c7821438dcb0fa908e69cd68be5060143bfea56ada58c2aa8f504f3e3c739  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_close_signal_audit
8cb4f98057356dc0b2b30811067a97080966d4591734ec23a2e5a86bab778777  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_accounting_audit.mjs
b94fef80aff9107779097b2912b65d9f2550dc37f11950aaf7edd7b25a1c900f  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_regime_attribution.cjs
fd57a43663eaf4403be850f1ac25497a290354df53b84f54241f3500f66757d1  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_report_evidence.cjs
e7be471628c69429ff1d3540f825d37d57578925615e747efe9a70f84d9e4dae  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-expr-checks.json
3b74ccba2060b2117f29a09f3a17837074bc8fa200a1dc5ff6ae66e34264c07e  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-frontend-contract-checks.json
bbc4495840cacce2e0468828acc845203d622a5d9cd49094b15512f726197513  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-portable-close-identity-scan.json
163eca0b9a8a945e089ab2b67445d29c7bd1a8946cc8dda3c86e594f7232e474  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-api-preflight-blocked.json
b20068f15f46365b7f104854c7950cf3e86c83d7663f08a817b5c5f8faac4882  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-local-v29-snapshot.json
8aaf7a60f881ef54ed6e6fe9213b1240fc104e457af55bfce8a4135133f66cd6  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_replay_copy.go
181fcbfbfb3a90a985150d6b23df4cad3ffb08b8239592672a3311869817f22c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_replay
f2c0abeadf3b603e0a36ec4e22fec54a566df3f0c78cfa7ddb8376e0affe34ef  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_accounting_audit.mjs
45355a43c46ff552344a7449ba4b6f925033df16d253e959491c38227bb24763  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_regime_attribution.cjs
9ed6dd907a15f93d9c2d449330b66706bc12c9ba6ba08ac35e9ae692a116fb1c  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_slippage10bps_report_evidence.cjs
05b465ff70930572ca343d77db01f539963afb0ab7afc99fd05d7eca3ac90a1b  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-development4-canonical-repaired-v2-funding-tail-v1.json
25071c979e0b3d66f56eaa69097a3ca995a11ae18ae4fd844b899554f4d2c9be  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-accounting-summary.json
7d8df6572292e497443ba4aa0e38e759906d01fa988c88a73e9e5066e4ebd553  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-canonical-actual-seed-open-signal-audit.json
5d6551e36f0f038a23dea78a78e2dba4dd1fdda495a88d570f2397e5af49e1fa  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-original-position-close-signal-audit.json
7a740473798629e4cca8f304b9d539cebbdae4100eaeb89852d22f5c86ada47e  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-execution-original-cost-fallback-audit.json
8c9ee876d9270544283268db678c52263a1e83a9414e3e1b99b6fa0cb2bba0e7  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-natural-entry-regime-attribution.json
c699329ec9bf3bccf8017212dd29fdc291fab56184e24449848dbb12f1de7a5b  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-phase-evidence-summary.json
3aaadb34693240501cfdf70d65aba0adacfeee45fd004e781c9b8ccc521aa945  /Users/zhz/Library/Caches/go-binance-strategy-research/verification/rg32_fill_participation_diagnostic.cjs
a9e3e107ecf1211f2cd6c6552230bca50e9bfea9201601198dcfdda14579d6a9  /Users/zhz/Library/Caches/go-binance-strategy-research/results/20261006-rg32-fill-participation-diagnostic.json
1418289cf61bab0fea727a87545cf8d61fdde5848b319fb387f33e42c17405bf  .agents/skills/custom-strategy/research_records/2026-10-06-strong-hourly-joint-loss-structure-exit-summary.md
b2059f909944cfffb27786ab320e6a1ae2fa596d33e08d72b1ad2c099d4abc1f  .agents/skills/custom-strategy/research_records/2026-10-06-rg32-fill-participation-protocol.md
```

冻结77个唯一文件；正式skill与所有用户dirty保持。SkillMax草案如有后续改动，仅trusted:false待审，不改此收益前合同或原证据。原参数候选留temp_strategy，不复制到发行strategy_templates。

