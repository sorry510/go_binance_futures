# CODE_REVIEW_01-ALL 复审整改记录（2026-11-09）

依据：[CODE_REVIEW_01-ALL.md](CODE_REVIEW_01-ALL.md)。本文件记录其发现项在审计后实际修改的源码、文档与测试结果。**原审计报告保留当时的 HEAD 8841d8a 快照，不以本文件倒改原始证据。**

**结论：ALL-F1/F2/F3/F4 已做代码/文档整改；ALL-F7 的前端构建已独立复验。ALL-F5 仍为真实生产硬 Gate，ALL-F6 的无关测试环境/策略研究冲突保留单独追踪。代码 Schema 仍 v23，实际 Main/Lead T0 均未设定，不能生产发布。**

## 一、各 Finding 整改

| Finding | 实际修改及结果 | 当前 |
| --- | --- | --- |
| **ALL-F1 · P1** 后端 `static` 已提交但文档互相矛盾 | 修订 `CODE_REVIEW_01-5_整改记录.md`、`01-6_本地验收记录.md`、正式开发方案及 `doc/订单产生与记录完整流程.md`：区分**当次验收尚未同步**与**随后 `8841d8a` 已提交 static**。新增「仓库内已存在新静态文件≠目标运行程序已同步更新」的发布约束，禁止对生产运行目录直接 `git pull` 导致二进制与页面半发布。当前工作区经本轮翻译改动重新构建 UI，`dist` 与仓库 `static` 106 文件逐项 hash 一致；没有向生产服务器部署。 | **文档和本地产物一致性已修；G5 BLOCKED** |
| **ALL-F2 · P2** 中英 README Schema v19 错误 | `README.md`、`README.EN.md` 的所有当前 Schema 要求从 v19 更新为 **v23**，明确 `appversion.DatabaseSchemaVersion` 为唯一真源。旧阶段编号保留为历史里程碑，生产 MySQL 8 尚未验收不改变。新订单页改为「代码和静态资产已在仓库、生产部署仍未授权」。 | **已修** |
| **ALL-F3 · P2** root CLI 守卫在干净 CI 启动 DB panic | 新增无副作用包 `internal/cliargs` 的 `IsSyncDatabaseCommand(args)` 及其独立表驱动测试；将 `main.go` 的注册归档 Order、`isCLICommand`、`runCommand` 三处判定统一使用该函数，删除原 root 中需要数据库 `init()` 的同类测试；01-4 全仓 AST 守卫检查条件同步为 `cliargs.IsSyncDatabaseCommand(os.Args[1:])`。在**没有任何 `app.conf` 的临时目录**复制此纯包后，`GOWORK=off go test ./internal/cliargs` PASS。原普通启动、`doctor` 和 `cleanup` 不会注册旧 `models.Order`；精确 `sync db` 仍是唯一例外。 | **已修并独立验证** |
| **ALL-F4 · P2** T0 只有文档无代码保护 | **明确选择「读历史必须经过账户级 T0 认证」方案**，新增 `service/orderhistory/history.go::ConfiguredHistoryT0`：Main/Lead 分别读取 `BINANCE_ORDER_HISTORY_T0_MAIN_MS` / `BINANCE_ORDER_HISTORY_T0_LEAD_MS`（必须是真实批准的 UTC Unix 毫秒）；默认未设置 = **不启用收益认证**。List 与 Details 经同一 `Store.project` 判断：未启用的周期标 `history_not_activated`，周期创建时间早于 T0 标 `pre_activation_position_cycle`，显示为 incomplete，隐藏主表净交易盈亏、名义收益率、手续费认证总额及子订单汇总毛盈亏/费用；保留只读原始 Fill 和 REST 标记。无效 T0 直接报错，绝不推断为 0 或进程启动时间。**该防护只作用于历史查询，不改变订单生成、仓位归属、持久化结算或亏损控制**。补充永久测试，覆盖未设置、设晚、设早、Main/Lead 相互隔离及非法值；跨阶段 Stage016 Mock 注入明确时间，不依赖机器环境。新增中英文 incomplete 原因文案并构建产物。 | **代码门禁已修；实际 T0/G6 仍 BLOCKED** |
| **ALL-F5 · P2** 真实 Binance/查询性能未验 | 没有实际真实 WS/REST 逐 TradeID 比对、真实方向级 FreshZero 行、保护单 opt-in 验证或生产 MySQL 8 实际行数/P95。依原方案保留生产 Gate，不作新 DDL、不启用主动重挂、不发 Binance 委托。 | **G1～G4 BLOCKED** |
| **ALL-F6 · P3** 全量测试环境/研究脚本红点 | 不删改 `strategy_templates/research` 多 AI 研究证据。审查报告中 root/tests 的 `init`、controllers 策略模板 fixture 与 leadaccount 偶发时序问题保留另案。本轮尝试 `go vet ./...`，因**其他研究目录存在重复 `main`/类型声明**返回非零；随即使用项目主程序及订单相关包范围的 Go Vet/Build 验收，结果 PASS。**未宣称全量测试或 Vet PASS**。 | **无关历史问题 OPEN** |
| **ALL-F7 · P3** 前端未重做 Build | 已实际执行 `pnpm exec eslint --max-warnings 0 src/api/order.ts src/views/order/index.vue`、`pnpm typecheck`、`pnpm build`：均 PASS。含新中英文 T0 原因的 106 个 `dist` 文件已安全同步到**本地代码仓** `static` 并逐文件 SHA256 验证一致。 | **本地构建已复验** |

## 二、T0 代码级门禁与发布约定

> **两个环境变量目前都不能填写猜测值；没有批准的真实启用时刻时，历史收益认证默认关闭。**

```text
BINANCE_ORDER_HISTORY_T0_MAIN_MS=<人工批准的 Main 实际启用 UTC Unix 毫秒>
BINANCE_ORDER_HISTORY_T0_LEAD_MS=<人工批准的 Lead 实际启用 UTC Unix 毫秒>
```

- 当前只读接口：`GET /orders`、`GET /orders/:position_id/details`。保留老 Managed 周期用于审计、风险归属和原始 Fill 可追踪；但是即使持久化 `settlement_status=settled`、REST 数据全部正确，**未启用/前 T0 的周期不对外认证其新历史收益**。页面显示明确原因，`page_verified_net` 不含这些周期。
- T0 必须为每个账户分别批准、记录在持久的发布清单中，并由受控环境配置管理在进程重启后提供**完全相同**的值。不得把旧 Managed 订单创建时间、首次本地测试时间、进程启动时刻或本次代码编译时间当作 T0。
- 不改变 `futures_managed_positions` / `futures_managed_orders` / `futures_trade_fills` Schema；不重新判定真实结算、也不因未设置 T0 停止既有仓位的保护/平仓/完整成交留存。
- **真正 G6 仍未达标**：未取得 Main 和 Lead 各自生产启用时间、最终部署哈希、批准记录及真实 Binance 验证。开发测试中 `ActivatedAt` 函数仅作为测试夹具注入，不应供生产绕过审批。

## 三、回归证据

- `go test -count=1`：`internal/cliargs`、`service/orderhistory`、`service/futuresownership`、`service/tradefills`、`service/agenttrade`、`service/leadaccount`、`feature/api/binance`、`scanner`、`feature/strategy/coin`、`feature`、`command`、`models` 均 PASS；`appversion` 无测试文件；`controllers` 的 Stage014/015/016 专项 PASS。
- `go test -race -count=1`：`internal/cliargs`、`orderhistory`、`futuresownership`、`tradefills`、`feature` 相关 Stage014/015/016/R3 回归 PASS（macOS `ld LC_DYSYMTAB` 为非致命链接器 warning）。
- `go vet` 对正式入口、相关业务包全部 PASS，`go build .` 主程序 PASS，`GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build` PASS；`git diff --check` PASS。
- **独立 CI 模拟**：将 `internal/cliargs` 和最小 `go.mod` 放入不含数据库配置的 `/tmp`，`GOWORK=off go test -count=1 ./internal/cliargs` PASS。
- 前端新中英文 T0 状态：ESLint、TypeScript/Vue Typecheck、Vite Build PASS；前端 `dist` / 后端本地 `static` **106/106 文件 hash 相同**，当前 `index.html` SHA256=`151ae56f036498b458a97dd71d33ef7388d94a5d614ce60b3230acc26b56c97a`。
- **全仓不全绿的独立问题仍存在**：`go vet ./...` 在研究脚本重复声明处失败；报告中的 root/tests 配置依赖、controllers 策略 fixture 等没有因为本轮 T0 与 CLI 修正而消失；勿把目标包 PASS 写作全仓 PASS。

## 四、部署和安全界线

1. `8841d8a` 是**整改前审计基线的已提交代码**；本文件、T0 门禁、CLI 门禁与本次新翻译/静态资源是**其上的本地未提交更改**。未执行新的 commit/push。
2. 本地 Git 中已包含新订单 UI 和更新后的 `static`，**不代表生产实际执行的是同版本二进制**；禁止运行目录直接 `git pull` 或只复制其中一端。必须在独立 release 目录内校验二进制 SHA256、全套静态资源 hash、生产 v23 DB 结构及 T0 清单，获批维护窗口里完整切换并验证回滚。
3. 原 `conf/app.conf` 保持不变；未执行生产 `sync db`，未修改 `go_binance` 或真实 Binance 订单，未启用 `BINANCE_AGENT_PROTECTION_AUTO_REPAIR`，未重启部署程序。
4. G1（真实 Binance WS/REST）、G2（方向级归零）、G3（保护单）、G4（生产 MySQL 8 性能）、G5（协同发布）及 G6（真实 T0）仍 BLOCKED；G7（独立全仓测试红点）仍 OPEN。

**整改完成不等于允许部署。**
