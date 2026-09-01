# Phase V2-4：Eval、Replay 与 Prompt/Skill Version

## 目标

建立 Agent 自动质量体系，使 Runtime、Prompt、Model、Context、MCP Tool 和 Imported Skill 的变化都能量化回归。

## Eval Case

Case 保存输入、固定 Tool/MCP Fixture、预期结构、关键事实、禁止事实、方向范围、评分规则和标签。

首批覆盖四个 V1 Skill；之后每个新 Skill 在启用前必须提供 Eval Case。

## 新增维度

除结构/事实/Evidence/Repair/Token/耗时外，增加：

- Tool selection accuracy。
- stale/missing data honesty。
- MCP Tool failure recovery。
- Imported Skill instruction compliance。
- Skill Router selection accuracy。
- Prompt injection / permission escalation resistance。

## Replay

Native Tool 与 MCP Tool 都可替换成 Fixture Adapter；Replay 不连接真实交易 API。历史 Task 保存当时的 tool catalog hash 和 skill package hash，避免“同名能力已变化”导致不可复现。

## Shadow

候选模型、Prompt、Skill revision 可在生产输入上 shadow 运行，但不能影响通知、数据库写操作或交易。

## CI Gate

关键 Case 退化、权限越界、事实错误超过阈值时阻止上线。主观语言风格不作为唯一 Gate。

## 验收

- [ ] 核心 Skill 有自动 Eval。
- [ ] Model/Prompt/Skill revision 可对比。
- [ ] MCP 和 Portable Skill 有专门回归 Case。
- [ ] 关键退化可阻止发布。
