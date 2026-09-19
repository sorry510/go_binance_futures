# Phase V4-6：ChatGPT-Compatible Plugin System

## 1. 定位

本阶段最后实施，因为它依赖 V4-1 多 Skill Chat 和 V4-2 Skill Studio 稳定。

目标不是完整复制 ChatGPT，而是在本项目增加一个 Plugin Package 层，使插件能够组合现有：

```text
Skill
MCP Server
Skill + MCP Server
```

并优先兼容 2026 年当前 ChatGPT / Codex Plugin Package 结构。

## 2. 当前标准方向

当前 OpenAI Plugin Package 以根目录 `plugin.json` 为主要可移植 manifest，并兼容 `.codex-plugin/plugin.json`。

典型结构：

```text
my-plugin/
├── plugin.json
├── skills/
│   ├── skill-a/SKILL.md
│   └── skill-b/SKILL.md
└── mcp.json              optional
```

OpenAI 当前 Plugin 可以是：

- skills only。
- MCP only。
- skills + MCP。

可选 UI / lifecycle hook 不作为本项目首版兼容目标。

## 3. Plugin Domain

新增轻量 Plugin 概念：

```text
Plugin
- name
- version
- description
- package_hash
- status
- source
- manifest
```

Plugin 只负责“打包和安装边界”，真正运行仍复用：

- Portable Skill Runtime。
- MCP Client Gateway。
- Tool Permission。
- Chat Workspace。

不建设 Plugin 专用 Runtime。

## 4. Import

支持：

- ZIP 上传。
- server directory（沿用安全目录导入模式）。
- 可选 Git/URL 不在首版强制实现。

导入流程：

```text
unpack staging
→ validate plugin.json
→ validate skills
→ validate mcp.json
→ permissions preview
→ install
→ optional enable
```

## 5. Skill 安装

插件中的 `skills/` 复用 V4-2 / V2 Portable Skill parser。

必须保持：

- immutable version。
- package hash。
- permission request 不等于授权。
- scripts 默认不可执行。

## 6. MCP 安装

插件中的 MCP 声明映射到现有 MCP Client Config。

首版只支持本项目已经安全支持的 Transport / Auth 类型。

不能因为 plugin manifest 声明某 MCP 就跳过：

- domain validation。
- permission。
- auth 配置。
- Tool risk classification。

交易类远端 Tool 继续受现有 `RiskTrade` 禁用边界。

## 7. Chat 集成

安装 Plugin 后，其 Skill 可以被加入 V4-1 Conversation。

UI 可按 Plugin 分组展示：

```text
Plugin X
  ├─ skill-a
  └─ skill-b
```

MCP tools 仍由 Skill / Permission 决定是否可调用。

## 8. Web 管理

在现有 Skill/MCP 管理基础上增加“插件”页或 Tab：

- Installed Plugins。
- Version。
- Included Skills。
- MCP Servers。
- Permission Summary。
- Enable / Disable。
- Uninstall。

Uninstall 不允许破坏历史 Task / Conversation Audit。

## 9. Compatibility Scope

首版目标：

- 兼容基础 `plugin.json` package。
- 兼容 skills directory。
- 兼容基础 MCP mapping。
- 尽可能接受 `.codex-plugin/plugin.json` compatibility manifest。

首版不承诺：

- 完整 ChatGPT UI resource runtime。
- Codex lifecycle hooks。
- OpenAI Plugin Directory 自动安装协议。
- OpenAI review/publish workflow。

## 10. Gate

- 导入 skills-only plugin。
- 导入 MCP-only plugin。
- 导入 skill+MCP plugin。
- Skill 可进入 Chat Workspace。
- MCP Tool 继续遵守 Permission。
- malformed manifest / path traversal / oversized package 被拒绝。
- disable plugin 后所属能力不能继续被新 Task 使用。
- 历史 Task 仍可查看。

## 11. 本阶段不做

- 不兼容历史 `ai-plugin.json + OpenAPI` 旧插件体系。
- 不执行插件 hooks。
- 不执行 Skill scripts。
- 不绕过本项目现有安全模型。
