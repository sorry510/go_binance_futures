# 关闭版本谱系：SkillMax待审批窄修订

按项目AGENTS和skill-creator，更新既有custom-strategy待审批草案的版本谱系，不创建独立重复技能、不修改正式SKILL或晋级。父current-goal-gates草案保留，版本修订r2仍name=custom-strategy、trusted:false。

- 父 `/Users/zhz/.skillmax/candidates/custom-strategy-current-goal-gates-20261006/SKILL.md` SHA690a5617528080158e9f7eb684857ae6352070ce59dd1fb02c09483eb89220cd。
- r2 `/Users/zhz/.skillmax/candidates/custom-strategy-current-goal-gates-20261006-r2/SKILL.md` SHAa08fd782aa5d67925710f3628b4068895c3df1d9db038da75e80200725c2260a。
- 正式项目SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd不变。

实际skillmaxxing optimize apply applied1/rejected0；--base4是机械编辑预算，非行为得分。系统quick_validate.py最初python别名不存在未执行，随后python3实际exit0/Skill is valid。逐字节核验只插入声明的一个段落，其余父内容保持、旧父保留；正式合同/全部77成本压力冻结文件不受草案影响。

新增方法源于实际RG31/RG32完整关闭审计：分别记录当前、直接父版本和实际audit comparator的完整snapshot/hash；old/new/both/added-only按实际关闭环境配对，若comparator是更早祖先，要明确统计相对祖先而不是立即父。局部OR→AND须独立验证父/当前真值表和所有未变旧分支，不能全局修改OR；added-only负PnL不能直接证明删除止损会盈利。

唯一edits spec是research_specs/skillmax-close-lineage-edits.json；三个现实只读请求在skillmax-close-lineage-eval.json，原完整JSON/Go helper/raw结果为输入。真实baseline/candidate独立行为比较仍pending；没有行为成绩、严格改进证明、score/gate/promote/批准或新委派。旧成本、QPS、当前门槛评估请求及父修正全部保留。

目前可复用的是待审批窄修订和可复核结构检查，不将quick_validate或字节一致性当成严格行为胜出。没有globalmemory写入、DB/生产/frontend/conf/_test.go或交易操作。

