# CODE_REVIEW_CM-1 · 币本位交易独立模型、Schema v27、全局开关

> 2026-11-17。分支：`feat/m-coin-futures`。本阶段只做数据库结构、配置 API 与离线测试。无主网、无生产库修改，无新交易执行接口。

## 一、需求对照与实现

| 要求 | 代码位置 | 检查结果 |
| --- | --- | --- |
| DB Schema v26→v27，沿用人工 `sync db` | `appversion/version.go`、`command/db_update.go`、`command/coin_m_migration.go` | PASS |
| 配置中心只有一个新全局开关，默认 OFF | `models/tableStruct.go`：`CoinMQuantEnable` / `config.coin_m_quant_enable DEFAULT 0` | PASS（后端字段/API；UI 留 CM-3） |
| 每币启停与 U 本位一致 | 复用 `models.DeliverySymbols.Enable` / `delivery_symbols.enable` | PASS（复用字段；页面留 CM-3） |
| 不混用旧 `delivery_enable` | 新全局开关独立存储；旧值原样保留 | PASS |
| 老币种预算不被复用 | `delivery_symbols.budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0`；旧 `usdt` 不重写 | PASS |
| COIN-M 必要交易对元数据 | `delivery_symbols.contract_size`、`margin_asset`、`contract_type` | PASS |
| 独立业务账本 4 表 | `coin_m_managed_positions`、`coin_m_managed_orders`、`coin_m_trade_fills`、`coin_m_strategy_templates` | PASS |
| 精确数据类型 | 四表的合约张数、价格、收益、手续费和预算均 SQL `DECIMAL(36,18)`，DTO 使用 string；禁止 ORM 接管业务表 | PASS |
| 开平周期关联与去重 | `position_id` 关联；ClientOrderID 唯一；(symbol, exchange_order_id) 唯一；Fill 三元组唯一 | PASS |
| API 读写与字段隔离 | `controllers/index.go`：`coin_m_quant_enable` GET/PUT，0/1 校验，必须单字段更新 | PASS |
| 不改变 U 本位/Spot/Lead | 未接入 DAPI 下单、未更改现有调度、WS、订单归属和币安配置 | PASS |
| 数据库已宣告 v27 但结构损坏 | `VerifyV27CoinMSchema` 只读，缺表/列/DECIMAL/索引/唯一性/非法开关时拒绝通过；不静默重建丢失账本 | PASS |

**迁移实现说明：** 本项目 v25 已采用 Go 迁移 helper 配合版本管理。v27 的 DDL 位于 `command/coin_m_migration.go`，没有单独的 `command/sql/version/27.sql`；数据库版本仍由 `./go_binance_futures sync db` 推进，不能直接执行服务启动来假设完成迁移。MySQL 的 `coin_m_quant_enable` 与既有现货开关同样转换为 `TINYINT NOT NULL DEFAULT 0`，且转换前校验 0/1 值。

## 二、实际执行的验证

| 验证 | 结果 |
| --- | --- |
| 将 SQLite 内存库真实模拟为物理 v26（移除新字段，保留旧 delivery 行），然后执行 v27 | PASS |
| 旧 `delivery_enable=1`、币种 `enable=1` 与 `usdt=10` 不使新 CM 开关变 1，预算仍为 0 | PASS |
| v27 四业务表创建后为空 | PASS |
| 所有资金/张数字段为 `DECIMAL(36,18)` | PASS |
| ClientOrderID 重复提交和相同交易所 Fill 重复插入均被唯一索引拒绝 | PASS |
| 18 位小数的费用/PnL 插入并重新读取 | PASS |
| 重复 `SyncDatabase(27)` 不删除 CM、U 本位、现货历史数据 | PASS |
| 已标记 v27 但 Fill 表被删除，重复 sync fail closed、不自动新建表 | PASS |
| 已标记 v27，但唯一索引被替换为同名非唯一索引，重复 sync fail closed | PASS |
| 配置 API 仅接受单独的 0/1 数字，拒绝 null/string/bool/混合字段 | PASS |
| 更新新 CM 开关不会覆盖旧 `delivery_enable`、合约或现货开关 | PASS |
| `go test ./command -run '^TestSyncDatabaseInitializesAndIsIdempotent$' -count=5 -p=2` | PASS |
| 11 个相关 Go 包 `go test ... -count=1 -p=2` | PASS |
| `go build -o /tmp/go_binance_futures-cm1-v27 .` | PASS |
| `git diff --check` | PASS |

**边界与保留项：**

- 以上仅是 SQLite 离线迁移/单测；未在真实 MySQL 8、PostgreSQL、币安主网或 Demo 环境执行迁移/交易验收。
- 现阶段 **尚无「币本位交易」前端页面、配置中心 UI 卡片、DAPI 正式交易客户端、自动调度或实盘下单**。它们分别留到 CM-2、CM-3、CM-5、CM-6。
- 开启 `coin_m_quant_enable` 目前只会改变数据库标志；不会下单。未来执行器必须再次校验币种有效、预算 >0、真实币本位余额和持仓/Fill 安全条件。
- 现有历史 `delivery_symbols.enable=1` 可能存在。CM-3 应在页面明确展示并逐项确认；新全局开关和 `budget_usd=0` 是迁移阶段不自动开仓的保障。
- 不写生产库，不改 `app.conf`，不触碰现有后端可执行文件；仅构建 `/tmp` 内供本地验收的检查产物。
- 前端工作树及未关联仓库未作任何编辑；不清理其他 AI 的并行改动。

## 三、结论

**CM-1 通过本地代码审查与离线测试，可以进入 CM-2。**

CM-2 只做 COIN-M 行情、DAPI Client、WS 和测试/正式网络隔离，仍不能因开启开关就发出真实订单。首次真实 MySQL v27 升级需在计划维护窗口备份并人工执行 `sync db`，在 CM-2/CM-3 的环境与配置校验结束前不要部署此二进制至生产。
