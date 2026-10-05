# receiver递推种子与资金费mark覆盖：技能候选待行为评估

2026-10-03末/10-04初按Skill Maxing优先更新现有custom-strategy，而非创建近重复。正式版本仍1.0.8/trusted:false，SHA b36ba1e915711e54aab4e5ca3bb3567be95e737336d1c2af6e27e12b54a82f77。没有promote，没有通用性能或盈利改善结论。

## 已执行

- research_specs/skillmax-receiver-seed-funding-mark-edits.json包含两个窄补充：从实际receiver取递推输入窗口/种子而不是live最小值；分别审计真实资金费时点/费率与缺mark回退覆盖。
- 实际skillmaxxing optimize apply（base4/min1/total1/step0）应用1/rejected0，生成/Users/zhz/.skillmax/candidates/custom-strategy，候选SKILL.md SHA 688b411fac8fe4afbb5137b7d0fcfb1f7bb4db818a97b65b5b4af0e32313d5c9。
- quick_validate.py真实返回Skill is valid，仅证明结构语法；正式SKILL未变。
- research_specs/skillmax-receiver-seed-funding-mark-eval.json冻结两个agent-judge任务及证据/行为rubric，而非字面关键字评分。
- 缓存verification/20261003-receiver-seed-funding-mark-pending-rollouts.json为空数组。实际CLI score返回两个pending:true、aggregate:null、缺行为回答；perTask的0只是未判定占位，不是已评估失败或strict win。

## 尚未执行与晋级门槛

没有授权启动独立agent行为rollout，本轮不新建subagent，不伪造独立回答/分数，不复用旧1.0.8其他任务的5/8→8/8。当前没有同任务baseline/candidate实际行为比较、独立judge、strict-win gate或promote。

未来可完成适当获授权的行为评估；只有真实strict win才能晋级。晋级前必须基于最新live目录重建candidate，保持被评估SKILL SHA，并确认仅预期技能差异，避免旧snapshot覆盖后来报告或外部45月默认/研究改动。新改技能仍trusted:false，用户认可前不宣称trusted。

本记录是保守待办，不把结构通过当效果通过，也不让技能评估待办阻断当前可安全继续的策略研究。
