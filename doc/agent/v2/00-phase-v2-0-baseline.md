# Phase V2-0：V1 Baseline、版本契约与 Replay 基线

> 状态：✅ 已完成

## 目标

在重构 Runtime 前，把 V1 当前行为固化成可重复测试的基线，保证 V2 每一步都能回答“兼容了什么、改变了什么、为什么改变”。本 Phase 不改变 Agent 的业务推理模式，不引入 Planner/Context Engine/Tool Runtime V2。

## 已落地的版本身份

每个 Task 在执行开始时冻结：

```text
runtime_version
skill_version
prompt_version
prompt_hash
model_config_id
input_contract_version
output_contract_version
skill_source
skill_source_version
```

`prompt_hash` 使用实际 System Prompt 的 SHA-256；`model_config_id` 保存创建 LLM Client 时使用的数据库配置 ID。Runtime 运行过程中继续使用已冻结的 System Prompt，不因后续配置变化修改当前 Task 的版本身份。

## V1 Skill 契约

| Skill | Skill | Prompt | Input Contract | Output Contract |
| --- | --- | --- | --- | --- |
| market_regime | 1.0.0 | 1.0.0 | `market_regime_snapshot_v1` | `market_regime_analysis_v1` |
| strategy_builder | 1.0.0 | 1.0.0 | `strategy_builder_input_v1` | `strategy_template_v1` |
| symbol_analysis | 1.0.0 | 1.0.0 | `symbol_analysis_input_v1` | `trading_plan_v1` |
| alert_analysis | 1.0.0 | 1.0.0 | `alert_analysis_input_v1` | `alert_v1` |

当前内置 Skill 的 `skill_source=native`、`skill_source_version=v1`。这两个字段为后续 Imported Agent Skill 与其他来源预留，不代表外部 Skill 自动可信。

## Replay 基础设施

新增 `agent/replay/`：

- `fixture.go`：版本化 JSON Fixture 加载。
- `client.go`：脚本化 LLM Client，不访问真实 Provider。
- `tools.go`：固定 Tool Fixture Store，不访问 Binance/生产数据库。
- `runner.go`：仍调用真实 `agent/runtime.Runner`。
- `diff.go`：Runtime/Skill/Prompt/Model/Contract 版本差异报告。

Replay 的关键原则是“固定数据源，真实 Runtime”。不能另外实现一套简化 Agent Loop，否则无法作为 Phase V2-1 Runtime 重构的兼容 Gate。

## 固定 Fixture

`agent/replay/testdata/` 当前包含：

```text
market_regime_success.json
strategy_builder_success.json
symbol_analysis_success.json
alert_analysis_success.json
runtime_llm_json_error.json
runtime_repair.json
runtime_tool_error.json
runtime_timeout.json
runtime_context_too_large.json
```

覆盖四个 V1 Skill 成功契约，以及 LLM JSON 错误、Repair、Tool Error、Timeout、Context Too Large 等 V1 Runtime 关键行为。

## Task / DB

扩展现有 `agent_tasks`，不新建 V2 Task 表。新增字段全部通过 Beego ORM Model + `orm.RunSyncdb` 自动同步，不增加字段 migration SQL。

历史 Task 新字段为空/0 时仍可读取；新 Task 会保存完整版本身份。Task List/Get API 直接返回 `task.Task`，因此版本字段自动进入现有 Task API，不需要平行 DTO。

## LLM 配置身份

`llm.Config` 在从 `llm_configs` 读取时保留数据库 ID；官方 Client 实现通过可选 `ConfigID()` 能力暴露该 ID。自定义/测试 Client 未实现时返回 0，不破坏现有 `llm.Client` 接口。

这样模型切换后，历史 Task 仍保存原 `model_config_id`，不会只剩 provider/model 字符串而失去配置来源。

## 测试基线

已经固定以下断言：

- 相同输入 + 相同 Tool/LLM Fixture 可重复得到相同结构结果。
- 四个 V1 Skill 的 input/output contract 不变。
- Prompt Hash 和 Prompt Version 可形成明确版本差异报告。
- queued Task 到 completed Task 的版本身份保持不变。
- ORM Store 可以完整保存/恢复版本字段。
- LLM database config ID 可以传递到实际 Client，再冻结到 Task。
- malformed Decision 进入 Repair；耗尽轮次仍保持 V1 `max_rounds` 行为。
- Tool Error 返回 Agent 后允许继续完成任务。
- Timeout 保持 `timeout` 失败阶段。
- Context 超限保持 `context_too_large` 失败阶段。

## 主要代码入口

```text
agent/skill/version.go
agent/runtime/version.go
agent/replay/
agent/task/task.go
agent/task/orm_store.go
models/agent_task.go
llm/config.go
llm/client.go
```

四个 V1 Skill 的版本声明分别保留在各自 Skill 实现中。

## 验收

- [x] 四个 V1 Skill 都有 Replay Case。
- [x] Task 可追溯 Runtime/Skill/Prompt/Model/Contract 版本。
- [x] Prompt 有版本号和实际内容 Hash。
- [x] 核心回归测试不依赖生产行情或真实 LLM Provider。
- [x] Runtime 关键错误/Repair 状态已有固定 Fixture。
- [x] 后续 Phase 可以复用同一 Replay Runner。
- [x] `go test ./...` 全量通过。

## 手动验收建议

V2-0 不改变 Agent 的分析能力，因此手动测试重点是版本身份和兼容性，而不是比较分析质量。

### 1. 新 Task 版本身份

在现有 Web 中执行一次“AI -> 单币分析”，等待任务完成后，通过任务中心或 `GET /agents/tasks/:taskId` 查看 Task。新 Task 应至少满足：

```text
runtime_version = 1.0.0
skill_version = 1.0.0
prompt_version = 1.0.0
prompt_hash = 64 位 SHA-256 十六进制字符串
model_config_id > 0（使用数据库 LLM 配置时）
input_contract_version = symbol_analysis_input_v1
output_contract_version = trading_plan_v1
skill_source = native
skill_source_version = v1
```

同一版本下连续执行两次 `symbol_analysis`，`prompt_hash` 应保持一致。

### 2. LLM 配置切换后的历史身份冻结

先用当前启用的 LLM 创建任务 A，记录 `model_config_id`；然后在 LLM 配置页面切换到另一条配置，再创建任务 B。预期：

- A 的 `model_config_id` 永远保持原值。
- B 使用新的 `model_config_id`。
- 再次查询 A，不会因为当前启用模型改变而被覆盖。

### 3. 历史 Task 向前兼容

查询 V2-0 上线前已经存在的 Agent Task。旧记录应仍可正常 List/Get；新增版本字段允许为空字符串或 0，不应导致页面、API 或 ORM 读取失败。

### 4. ORM 自动同步

正常重启一次后端，确认启动过程没有要求执行新的 `command/sql/version/*.sql`，并且新 Task 可以正常写入上述版本字段。V2-0 没有新增 migration SQL。

### 5. Replay 开发者手测

可在项目根目录执行：

```bash
go test ./agent/replay -v
```

该测试使用固定 LLM/Tool Fixture，不调用真实 LLM 或实时 Binance。成功表示四个 V1 Skill 契约以及 JSON Repair、Tool Error、Timeout、Context Too Large 等基线仍可重放。

以上 1～4 是推荐的人工验收；第 5 项主要用于开发阶段回归。

## Phase V2-1 的 Gate

V2-1 拆分 `DefaultRunner.Run()` 时，必须首先运行 `agent/replay`。除非 V2-1 文档明确声明并解释行为变化，否则上述 Replay Case 的 Task terminal status、关键 stage、版本身份、四个 Skill output contract 均视为兼容契约。

### LLM API Key 显式查看

`GET /llm/configs` 继续只返回 `has_api_key` 和掩码，不在普通列表中返回明文。编辑页面只有在用户主动切换“显示 API 密钥”时才调用受登录鉴权保护的 `GET /llm/configs/:id/api-key` 读取该配置的明文 Key；关闭显示后仅改变 UI 可见性。这样避免列表/自动刷新无意携带 Secret，同时允许管理员显式核对已配置 Token。
