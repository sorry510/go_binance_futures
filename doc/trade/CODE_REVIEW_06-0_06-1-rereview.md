# CODE_REVIEW_06-0_06-1：反馈整改与复评记录

日期：2026-11-11  
基线：`feat/trade_spot`，原报告 `CODE_REVIEW_06-0_06-1.md`（未覆盖，原文保留）。  
范围：仅方案 06 的 06-0/06-1 问题整改与测试，不提前进入 06-2、06-3、06-4、06-5。

> **2026-11-11 勘误（06-2 Code Review 后）**：本报告最初记录了
> `main.go.initializeRuntimeDatabase` 会调用 `VerifyV25SpotQuantSchema`。
> 用户随后**明确要求删除** `main.go` 中的 `dbVersion >= 25` Schema 检查，
> 并已执行删除。因此，下文表格「06-F1 / runtime 已修复」以及第二区「运行态也拒绝启动」
> 是**删除前的历史描述，在当前代码中不成立**，不得作为发布证据。
> 当前的 Schema 完整性校验仅在显式运行 `./go_binance_futures sync db` 时触发，
> 正常服务启动只比较数据库版本号。遵循用户的简化要求，**不恢复启动时检查**。
> 发布前必须执行 [06 发布前 Schema 验证检查单](开发方案-06-发布前Schema验证检查单.md)
> 并保存结果；未完成该检查前，生产发布 Gate 仍 BLOCKED。


## 一、逐条处理

| ID | 原级别 | 处理情况 | 代码与验证 |
| --- | --- | --- | --- |
| 06-F1 | P1 阻塞 | **已修复，采用“缺失即报错、拒绝启动”而非盲目补建** | `command/db_update.go`：当 DB 版本已为 v25 时，`SyncDatabase` 每次验证 Spot schema，验证通过才允许幂等检查/修复普通索引；不再因 `config.version=25` 直接宣告成功。首次 v24→v25 升级完成后，在保存新版本号前再验证。新 `VerifyV25SpotQuantSchema` 检查 2 开关列、四张表的全部预期字段以及 10 个财务 `DECIMAL(36,18)` 列。 |
| 06-F1 / runtime | P1 | **已修复** | `main.go` 的 `initializeRuntimeDatabase` 在版本门禁之后使用只读 `VerifyV25SpotQuantSchema`；旧方案 02 复用 v25 但缺新表的数据库无法正常启动新服务。 |
| 06-F2 | P2 | **已修复** | `GET /service/config` 返回统一的 `spot_quant_enable`（snake_case）；`PUT` 接受相同键。误用 `spotQuantEnable` 明确报错，不再静默忽略。 |
| 06-F3 | P2 | **针对新开关已修复；旧接口通用风险仍存在** | 单独的 `spot_quant_enable` 不可与任何其他字段混在同一次 PUT；`persistServiceConfigPatch` 通过 `o.Update(current, "SpotQuantEnable")` 单列更新，不重写 Futures/历史现货字段。旧的其他配置项仍遵循原通用 PUT 行为；避免在 06-1 大规模破坏配置中心。未新增复杂审计中心，符合个人使用原则。 |
| 06-F4 | P2 | **本次涉及的配置接口已修复** | GET/PUT 的成功与错误响应使用 `IndexController.respondServiceConfig → JSONResp`，不再由 `Context.Resp(map)` 根据 `Accept` 协商 XML/Proto。沿用已存在的全局 JWT 中间件；不在本阶段重构所有 Controller。 |
| 06-F5 | P2 | **预生产待验证** | 已有 MySQL 类型修正及类型拒绝逻辑保留；本次只对实验库运行只读 v25 Schema 校验，**没有在 MySQL 8 生产库测试 DDL/权限**。生产升级前需检查目标库实际版本、表、权限、备份和 SQL DDL。 |
| 06-F6 | P3 | **按阶段后移 06-3** | 当前只有表结构，没有业务写入/交易接口。06-3 的业务写入层必须校验 `side in {BUY,SELL}` 和交易所订单状态；是否使用 DB CHECK 需兼容数据库版本后再定，不提前扩大 Schema。 |
| 06-F7 | P3 | **已补永久回归** | `command/db_update_test.go` 中命名子测试 `V25VersionReuseMissingSpotLedgerFailsClosed`：删除新 Fill 表但保留 v25 标记，验证只读校验和 `sync db` 必须失败、版本未改、缺表未被静默重建、显式修复后重复升级通过。新增 `controllers/index_spot_quant_review_test.go`：实际 SQLite 字段隔离持久化、错误 `Accept` 仍返回 JSON；另补 mixed PUT / camelCase 非法输入测试。 |
| 06-F8 | P3 | **既有问题，不属于 06-1 整改** | 全仓库策略研究目录中的构建失败、独立全量测试环境差异仍需另项解决。本轮仅声明目标包及主程序构建通过，不声称 `go test ./...` 全绿。 |

## 二、F1 为什么选择拒绝而不是自动建表

旧方案 02 和当前方案 06 共用了物理版本号 **v25**，但表结构不同。若数据库已标记 v25 而缺 `spot_trade_fills`，可能是切换旧分支，也可能是真实成交记录表遭误删。盲目 `CREATE TABLE IF NOT EXISTS` 会制造一个空 Fill 表，把潜在的数据丢失隐藏成“升级成功”。

现在 `SyncDatabase(25)` 对已有 v25 数据库先**只读验证**，不兼容则明确报错；运行态也拒绝启动。这保证不会用新代码自动管理不完整的账本。需要修复这类数据库时，应先确认其来源并人工审查/备份，不能只更改 `config.version`。

对正常 v24→v25 的第一次升级仍使用现有幂等 DDL；升级成功后校验结构，再推进版本。对正常 v25 数据库重复升级仍会检查/补齐允许补的普通索引。测试覆盖健康与故障两种路径。

## 三、本次实测

| 验证 | 实际结果 |
| --- | --- |
| `go test ./command ./controllers ./models -count=1` | PASS，3 包 |
| `go test -race ./command ./controllers ./models -count=1` | PASS，3 包；macOS linker `malformed LC_DYSYMTAB` 警告为非致命 |
| `go vet ./command ./controllers ./models` | PASS |
| `go build -o /tmp/spot_quant_review_main_20261111 .` | PASS，未覆盖项目内既有可执行文件 |
| `git diff --check` | PASS |
| `TestSyncDatabaseInitializesAndIsIdempotent/V25VersionReuseMissingSpotLedgerFailsClosed` | PASS，缺表场景拒绝 + 版本不变 + 显式重建后健康重试 |
| `TestSpotQuantConfigUpdateOnlyChangesItsColumn` | PASS，真实临时 SQLite：新开关变化，其他 3 个配置保持不变 |
| `TestSpotQuantServiceConfigJSONIndependentOfAccept` | PASS，JSON/XML/Proto/plain text 的 `Accept` 都返回 JSON |
| 本地 `go_bn_test`（MySQL）只读 `VerifyV25SpotQuantSchema` | PASS，`config.version=25`，4 表、2 开关及 10 个 DECIMAL 字段均存在 |

**数据库边界**：本次对 `go_bn_test` 的在线检查只执行 `SELECT/information_schema` 元数据读取。SQLite 回归测试使用临时数据库。未在实验库或生产库执行新一轮 `sync db`、未操作 Binance 账户、未覆盖前端四项未提交文件、未执行 commit/push。

## 四、Gate 结论

- **06-0：PASS**，原审计的正向结论保留。
- **06-1：P1 阻塞项已闭环，代码及实验库只读 Gate PASS**。06-2 可以在本地代码/实验环境继续开发。
- **生产发布 Gate：仍 BLOCKED**，需单独做 MySQL 8 目标库预检、生产备份、授权升级、完整联调；当前系统仍无新 Spot 自动下单路径。
- **下一阶段遗留**：06-3 对 `side/status` 做写入验证，06-5 接口调用固定使用 `spot_quant_enable`；通用配置编辑的历史宽权限问题可另行精简处理，不扩成新审批/审计框架。
