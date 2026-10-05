# QPS 浮点边界与维度缩放：待审批技能草案

2026-10-05 RG25 首版 actual Go/Expr 5930 passed/4 failed，只是 raw quote90% 构造在字面 Float64 QPS 运算顺序下比门槛低1ULP的错误预期；独立 numeric oracle 与 Expr 都返回 false。v2 保留原失败实例，另外同步缩放全部 quote/taker quote/QPS 使字段级等号可表示，实际5962/0/terminal0。候选、阈值、生产运算未变。RG24 另有 fixture 未同步新 EMA20 价格维度的失败证据，亦已保留。

依项目 Skill Maxing 和 skill-creator 窄更新规则，不新建安装近似技能。仅在原 custom-strategy-cost-algebra-20261004 草案的 QPS 段后新增一条 instruction，范围是独立 oracle 字面操作顺序/可表示等号与上下边界/全部引用字段维度同步/保留失败且禁止修生产或阈值迎合fixture/证据强度。

实际命令 skillmaxxing optimize apply --skill custom-strategy-qps-fixture-20261005 --skill-dir /Users/zhz/.skillmax/candidates/custom-strategy-cost-algebra-20261004 --edits <project>/research_specs/skillmax-qps-fixture-edits.json --json 返回 apply1/rejected0，预算4仅机械编辑次数，不是行为胜出分数。新标签事前不存在，原父草案未覆盖。

候选目录 /Users/zhz/.skillmax/candidates/custom-strategy-qps-fixture-20261005；仍 name:custom-strategy/metadata.trusted:false。SHA4949dabb6786cca9468497d7e229e1dc0d129c3b039af4a13d9865eca5197769。quick_validate actual terminal0（只结构检查），独立逐字比较证明相对父草案仅增加一段指令。父SHA5eccefb08492b12bd29036b89fc0f3bc464b2d43fdb16e704f8cc674bebd8ae6、正式SKILL SHA270d20e8f4d673342a085bdebff90c830b27c714dd61dffa40b65de7f92757bd、protected conf SHA7461e8e3a38d70331835e9a61cd48e6651fc216b527163b0e58a38e67f8daafa未变。

research_specs/skillmax-qps-fixture-eval.json 增加三个现实请求：字面QPS与quote边界差异、EMA/OHLC/ATR与taker/quote/QPS完整维度同步、fixture通过与盈利及失败留存边界。原9成本范围和3代数行为请求继续保留。独立 baseline/candidate 行为、score、gate、strict-win 与用户批准仍 pending；没有新委派、评分、promote或安装，不能把Expr研究结果当技能行为评分。

当前技能引导已产生完整的未审批可审阅草案，不改变实际回测、风险、币、频率、年度与成本门槛。不插库/分配/启用/下单，也不修改正式技能、生产、前端、配置、全局记忆或添加仓库测试文件；不阻断本轮研究。

