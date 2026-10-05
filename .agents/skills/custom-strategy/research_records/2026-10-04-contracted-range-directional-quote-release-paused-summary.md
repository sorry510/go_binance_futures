# RG13 阶段性总结：已暂停，4/12完整run保留

观察时间：2026-10-04北京时间09:20:32..09:21。实际get_goal返回paused，updatedAt=1791075654；本轮最初09:00:43查为active，之后09:19实际再查才发现paused，发现后停止研究并仅保存/收尾，不自行恢复或改goal状态。不是完整目标达成。

## 已经可用的研究材料

两个完整候选在temp_strategy/20261004-contracted-range-directional-quote-release/：独立入口族4规则、AF0组合6规则，都包含long/short/close_long/close_short。新补充使用相邻闭合四小时范围相对收缩、闭合主动quote同向多数和放量、首次边缘释放，weak OR strong EMA/DI方向，原9指标/基础/wholeRG4关闭/风险/成本/日期/门槛保持。身份和完整表达式在收益前协议及JSON，不把合成检查称盈利证明。
- 34800实际terminal exit0：4958真实Go/Expr合成及本地有序规则模型检查，0失败。
- 真实当前前端validator隔离Node VM，两完整JSON issue=null/9指标/四规则类型和shape通过，无App/UI/build/API操作。
- 限定300配置/677启用入口whole-code whitespace-normalized检查0重复，不是语义/alpha/global证明，未读其它研究收益。
- opening和closing外部overlay helper已build exit0，尚未运行实际交易核验；accounting和report helper仅node --check通过。
- 收益前协议2026-10-04-contracted-range-directional-quote-release-protocol.md、helper和全部已有检查结果保留。

## 实际运行/暂停结果

主6856于09:17:10.955启动12完整49月run，09:19:35之后因实际goal paused，对只属于本輪且参数/output路径确认的child62200发送SIGTERM。随后原6856actual terminal exit1，日志signal: terminated，wrapper62195/go62196随child退出；09:20再次限定pgrep无进程。不得再poll旧句柄或声称主exit0、12完整run完成、全审计通过。

输出本身是逐个完成run保存的checkpoint，不另有.checkpoint.json：
/Users/zhz/Library/Caches/go-binance-strategy-research/results/20261004-rg13-development4-canonical-repaired-v2-funding-tail-v1.json
暂停后SHA ad2aaff2cd7a728bfe7c08b1c4a1142f5da8a934f4e17bc7bb97b6094a05ec02。JSON有效，4完整run/977执行账目：
- BTC AF0：103；BTC RG12：213；BTC RG13：557。
- ETH AF0：104。
- ETH RG12当时进度90%被终止，没有写入完整run；其余8个run未完成（包含该ETH RG12）。不把进度当完整、不能分拆旧控制结果补齐。

当前只知道未独立审计的主日志BTC RG13：557笔、2.615次/周、净-83.602USDT、DD31.62%。因此没有通过盈利资格；其余RG13币/年度/跨币审计还未完成，不宣布新候选整体invalidated或有效。完整会计/开/关/成本输出尚不存在，不做进一步策略分析/改参数。

## 下次恢复动作（只在恢复授权/实际goal状态允许后）

第一条先此阶段总结并重新核实际goal/进程/保护/当前SKILL；若仍paused，不启动研究、不自行resume或update_goal。
现有固定pv5 replay主源码148..218确认：相同-output自动读已保存JSON并校验日期/config/interval/candidate版本/source/repair；同dataHash完成run跳过。故沿原冻结命令及相同输出路径继续，跳过BTC全部和ETH AF0，重算被中断ETH RG12并完成其余8run；不重复已完成4run、不改候选或阈值。
之后才执行预声明12run会计及8共享完整控制复现、全部自身开/关receiver及成本验证，再总结失败模式。精确资金费settlement mark缺口仍pending；AAVE/ATOM/ETC/LINK收益未读，不变门槛。

## 范围和保护

本輪仅程序读取选定注释arm，元数据22785 exit0：分别17/222、17/219、17/8；v29 SHA b511b61fab163f5ef551e9a94d180aba2a82fd4d44a297316a67e28c26edf2a0与旧字节相同。不是449条完整新forward内容分析。
09:20:32 config/engine/environment/indicator_cache SHA保持收益前；两个pending技能SHA保持且不晋级。正式SKILL由外部新增1行，当前SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd，和初始b36不同，本轮未编辑或回退它，下次使用必须重新完整阅读。其他dirty研究不动。
无App、DB写入/分配/启用/下单、生产/前端/conf.app变更或新仓库测试文件；未删除任何文件，停止进程已完成的checkpoint/缓存/所有JSON仍可恢复。
