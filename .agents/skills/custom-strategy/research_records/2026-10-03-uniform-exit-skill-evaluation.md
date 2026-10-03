# 统一平仓对照：技能评估

已按skill-creator及项目Skill Maxing评估单段补充，不改生产/数据库。edit spec为research_specs/skillmax-uniform-exit-controls-edits.json；optimize apply在受管副本应用1项、quick_validate通过，live原技能没有修改。

独立baseline与candidate评估均只收到同一现实请求、对应技能和基础/补充完整JSON，允许源码检查；没有给预期答案或其他agent输出。请求要求平仓对象完全原样，并询问静态删旧交易及局部selector成功是否可证明净收益/端到端。

评价5个可验证行为：正确使用-exit-source base且无两种guard；完整close对象/基础入口/technology身份；helper六条顺序与准确匹配身份；单仓/当前现金复利需要完整重撮合、不能静态删单；局部模型不能证明API/历史/前向接收全部兼容。baseline和candidate均5/5。两者还均识别uniform exits不受既有forward hash注入缺口影响，不能为此无故增加guard。

真实命令optimize gate --current 1 --candidate 1 --best 1返回reject（exit1），严格无提升，未promote，技能仍1.0.6/trusted:false。补充文本及评估保留为未晋级研究草案，不把说明更详细当作可验证行为胜出。新发现的当前主动成交缓存语义缺口应作为另一项有证据的高价值修正，而不是覆盖这次tie结果。
