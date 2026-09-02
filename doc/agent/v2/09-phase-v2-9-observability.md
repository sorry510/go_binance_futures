# Phase V2-9：Observability、Trace 与运营页面

## 目标

从“任务成功/失败”升级到可定位质量、上下文、工具、外部依赖、权限和成本问题。

## Trace

完整链路：Request -> Skill Resolve -> Context -> Plan -> Model Route -> LLM -> Tool/MCP -> Evidence -> Validation -> Approval -> Final。

每个节点记录 task/step、skill revision、prompt version、model config、tool source、remote MCP server、duration、tokens、status、error type。

## 长期指标

- Skill/Model/Prompt 版本成功率和成本。
- Context tokens、裁剪率、Memory 命中。
- Native/MCP Tool latency、cache、partial、timeout。
- 第三方 MCP Server availability、protocol/catalog/schema changes。
- Skill import validation failures、active revision、rollback。
- Evidence 覆盖、Repair 分布、Eval Score。
- Risk Proposal accept/reject。

## Web 页面

### 任务中心

Step Timeline、Context Summary、Tool/Evidence、模型路由、Skill revision、MCP 来源、Eval 信息。

### MCP 管理

第三方 HTTP MCP 连接、鉴权状态、协议、健康、Catalog、权限、错误、启停。

### Skill 管理

导入、验证、版本、文件浏览、权限、active revision、rollback。

## 安全

默认展示摘要，不展示 Secret 或敏感 Tool 原始数据。所有下载/查看操作遵守权限和脱敏。

## 验收

- [ ] 单个失败 Task 可以定位到具体 Step/Provider/Tool/MCP。
- [ ] MCP 与 Skill revision 变化有历史记录。
- [ ] 运营页面不泄漏 Secret。
- [ ] AI 子系统故障与交易主循环指标分离。
