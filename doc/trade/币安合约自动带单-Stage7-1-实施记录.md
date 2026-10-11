# 币安合约自动带单 Stage 7-1 — 历史 Gate 与 Schema v19 静态核查

> 日期：2026-10-11，分支 `feat/lead-trading`。**仅离线代码和 Mock 验收，不允许真实 Lead 交易。**

## 实施及实际证据

本次明确区分 **源代码/SQLite 验收** 与 **真实 MySQL 8 生产 Gate**：

- `appversion.DatabaseSchemaVersion` 为 **19**；现有 `command/account_scope_migration.go` 提供账户维度的迁移实现，`command/account_scope_migration_test.go` 及 `command/db_update_test.go` 的数据库迁移测试随 `go test -race ./command` 通过。
- Stage 2 的五张表账户维度、Main 默认 owner/历史迁移、索引和 fail-closed 行为有现有 SQLite/Mock 测试，Stage 3 Main golden 与用户数据隔离测试随 `go test -race ./feature` 回归通过。
- 新增 `TestStage71SchemaVersionStaticGate`，使未来代码要求版本变动时必须重新审查。
- **没有**查看或修改实际 MySQL 8 `go_binance` 数据库版本，**没有**运行 `sync db`，所以生产 Schema 19、所有历史主账户订单无损、Main 真实交易回归三项仍为 **未验收**，不可仅以 SQLite 测试推导。
- 下一步获授权后应先备份、查看目标库当前版本/索引/主数据量，再用独立 `./go_binance_futures sync db` 进行迁移和复核，并通过 Main 在线业务回归。

## 工作边界

没有修改 `app.conf`，没有运行 `sync db`、修改真实业务库或触发任何真实 Lead Submit/Cancel/REST/WS 测试；没有自动提交/推送 Git，不回退未提交的用户修改。
