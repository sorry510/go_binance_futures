# Phase V3-9：Agent Studio

## 目标

把现有 Portable Skill、Skill Management、Team 配置和测试能力整合成一个适合单用户维护的 Web Studio。

## Skill Studio

- 在 Web 创建/编辑标准 `SKILL.md`。
- 支持 Metadata、Prompt、allowed-tools、Chat Enabled 等配置。
- 保存前做格式与权限校验。
- 支持 Draft、Test、Publish、Rollback。
- 发布后仍进入现有 Skill Store / Permission / Runtime，不建立旁路。

## Team Studio

- 选择已存在的 Chat/Runtime Skill 组成 Team。
- 配置角色、输入映射、输出 Schema、执行顺序或有限并行。
- 配置 Supervisor 和 Team Budget。
- 提供 Test Run，能查看所有 Child Task 与 Trace。
## 简化原则

- 不做多人协作编辑、组织空间、审批发布。
- 当前登录用户就是唯一维护者。
- Studio 只是更方便地编辑现有标准对象，不绕过现有 Permission。
- `allowed-tools` 仍然只是权限请求；Trade Tool 不能通过 Skill 文本自行获得。

## UI

建议在现有 `AI → Skill 管理` 基础上增加编辑器和测试入口，并新增 Team 配置页；不需要再做复杂的“开发者门户”。

## 验收 Gate

- Web 创建的 Skill 能导出为兼容标准的 `SKILL.md`。
- Draft Skill 不会进入正式 Chat/Runtime。
- Test Run 使用与正式运行相同的 Runtime/Tool/Permission 边界。
- Publish/Rollback 后版本可追踪，旧 Task 仍能识别原版本。
- Team Studio 不能通过配置越权获得成员 Skill 未授权的 Tool。

## 本阶段不做

- 不允许在 Web 直接上传并执行任意 Go/Shell 代码。
- 不做第三方 Skill Marketplace。
- 不做多人审批和企业权限管理。
