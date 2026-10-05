# 对照研究核对：技能评审副本，未晋级

2026-10-04北京时间13:29:59实际SkillMax apply完成；13:34后实际结构校验返回 `Skill is valid!`。本次按项目Skill Maxing要求改进现有custom-strategy，不创建新的安装技能。

## 已验证

- 评审管理标签为 `custom-strategy-comparison-audit-20261004`，副本位于 `/Users/zhz/.skillmax/candidates/custom-strategy-comparison-audit-20261004`；frontmatter仍为 `name: custom-strategy`、`metadata.trusted: false`。标签不是新安装技能。
- 实际命令为 `skillmaxxing optimize apply --skill custom-strategy-comparison-audit-20261004 --skill-dir <project>/.agents/skills/custom-strategy --edits research_specs/skillmax-comparison-audit-edits.json --base 5 --min 1 --total 1 --step 0 --json`，budget5/applied5/rejected0。新SKILL SHA为 `c4b98786ae66d199bba559c7e4a6b589411f9e5bac64a6aff6f34337ada63e1e`。
- 基于当前正式目录重建，保留用户新加入的metrics CSV每小时最大有效create_time选择规则，不回退旧顺序覆盖实现。相对正式SKILL仅增加原两个seed/mark待评估条目、恢复阶段总结条目，以及三条窄核对步骤：退出改动必须完整重撮合；共享控制仅归一化已证明等价的路径身份；方向变更需要明确两侧数值预期和独立oracle。
- 正式SKILL SHA仍为 `270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd`，conf/app.conf仍为 `7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa`。原receiver/mark草案SHA `688b411fac8fe4afbb5137b7d0fcfb1f7bb4db818a97b65b5b4af0e32313d5c9` 和stage-summary草案SHA `c78899be03929afbac370b4a59f0d43e3dde0e9d4b6900d42fb437d364550072` 均未改变。
- `quick_validate.py`仅证明结构有效；实际diff只显示上述预期新增说明。真实RG13路径核对修正、RG14完整退出重撮合、RG15独立oracle失败保留/修正属于任务证据，不等同技能baseline/candidate行为评估。

## 待评估

`research_specs/skillmax-comparison-audit-eval.json`有7个agent-judge任务：seed窗口、真实mark覆盖、active恢复、paused恢复、退出路径、路径身份、方向oracle。没有新独立baseline/candidate回答、judge结果或strict-win；全部行为结果pending，aggregate为未判定，不能伪造分数或复用无关旧评分。没有新增/消息委派subagent，没有promote或修改正式SKILL。

晋级前需在获准的行为评估中对相同原始材料进行实际比较，并从当时最新正式目录重建，核对被评估SHA与完整目录差异；用户批准前保持trusted:false。该待办不阻断边界内策略研究。
