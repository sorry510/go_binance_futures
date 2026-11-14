# CODE_REVIEW_删除ListenRateEat：ListenRateEat / 吃资金费率套利链路删除审查

日期：2026-11-14（10:30 起）
后端基线：`master`，HEAD **`2700411 (tag: v2.2.0, origin/master)`**，工作区 **10 项未提交改动**（8 M + 2 D），本次审查对象即这批改动
前端基线：`go_binance_futrues_new_ui` HEAD `a13d44c`，工作区干净；本轮未修改前端
审计对象依据：`doc/TODO.md`「# 删除废弃无用的功能和代码 → [x] 删除 ListenRateEat 相关代码」
副本：`/tmp/phase_audit_eatrate`（只写探针，审计结束删除）
本报告为**新增文件**，未覆盖任何历史 `CODE_REVIEW_*` 报告。

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 删除完整性（路由 / 控制器 / 模型 / 定时逻辑 / SDK 封装 / 文档） | **✅ PASS**：分层核对无残留、无孤儿 Go 引用 |
| 离线 Gate（build / vet / 全量 test / race / 脚本 / diff-check） | **✅ PASS**：全绿，0 FAIL |
| 隔离探针 | **✅ 5 项 PASS**：路由 404、符号清零、syncdb 保留旧表、活跃链完好 |
| 前端源码 / dist / 后端 static 残留 | **✅ 无**：无 `/fund-rate/eat` 调用 |
| 本地开发库遗留数据 | **⚠️** `db/coin.db` 只读取证：`eat_rate_symbols` 表仍在、1 行 `enable=0` |
| **生产库遗留活跃仓位 / 现货库存** | **⛔ 未核验**（本次审计不触碰生产库）→ **F1 P1 条件触发风险** |
| 阶段判定 | **⚠️ 有条件通过**：可合并本地代码；**部署前必须完成遗留数据只读预检** |
| 实盘 / 发布 Gate | **⛔ 全部 BLOCKED**，与之前 06-7-ALL 结论一致，本次删除不改变任何既有 Gate |

一句话：**删除动作本身是干净的——598 行代码与 41 行 API 文档被完整移除，编译、全量回归、竞态与路由运行时行为均有实证；但被删掉的 `EatRateController.End()` 是历史上按 `eat_rate:<id>` SourceRef 平仓的唯一专用入口，而这类 managed 仓位既被自动退出链按 `funding_rate:` 前缀过滤、又被手动认领以「已 managed」拒绝。若生产库存在未平的历史套利仓位，删除后没有任何内置恢复路径。这不是本次 diff 引入的新代码缺陷，而是删除动作暴露的历史数据债务，必须在部署前以只读预检闭环。**

---

## 一、审计范围与基线

### 1.1 相对上一轮审查的基线变化

上一轮 06-7 整体复审在 `feat/trade_spot`、HEAD `2c7c2b8 feat: 06-7-all`。本轮已切到 **`master`**：

```
2700411 feat: update ui          (HEAD, v2.2.0)
dfc843d doc: update doc          ← 删除 121 个文件，含整个 doc/trade/ 历轮报告
ebdc814 Merge PR #66 from sorry510/feat/trade_spot
fc8d32c fix: update code         (feat/trade_spot)
08c86b4 feat: 06-7-all
```

`dfc843d` 同时给 `.gitignore` 增加 `/.local/`、`/research_specs/`。**master 上 `git ls-files | grep -ci CODE_REVIEW` = 32，全部位于 `doc/agent/*`；`doc/trade/` 目录已被删除且未加入 gitignore。** 本轮按既有约定重建 `doc/trade/` 存放报告（见 F5）。

### 1.2 本次工作区改动（10 files / +6 −598）

| 文件 | 变化 |
| --- | --- |
| `controllers/eatRate.go` | **D 424 行**：`EatRateController` 的 Get/Post/Edit/Delete/Start/End 及 ownership 辅助函数 |
| `rate/index.go` | **D 47 行**：`package rate` 唯一文件，`ListenRateEat()` |
| `routers/router.go` | −5：4 条 `/fund-rate/eat*` 路由 |
| `models/tableStruct.go` | −36：`EatRateSymbols` 结构体 + `TableName()` |
| `main.go` | −10：`orm.RegisterModel(new(models.EatRateSymbols))` + 注释态的 `rate.ListenRateEat()` goroutine |
| `feature/api/binance/index.go` | −35：`IncomeParams` + `GetIncome()`（`/fapi/v1/income` symbol-scoped 封装） |
| `.github/skills/api-reference.md` | −38 / +3：删除「13. 吃资金费率套利」，后续章节重编号 |
| `README.md` | −1：删除「增加吃资金费率功能」TODO |
| `doc/mcp.md` | ±1：从「保留 HTTP 但不通过 MCP 暴露」列表移除「资金费率套利」 |
| `doc/TODO.md` | +3：新增清理清单并勾选本项 |

### 1.3 易混淆点（已核实，未误删）

本次删的是 `rate.ListenRateEat`（**早已是 main.go 中整段注释、历史上从未运行**），**不是** `feature.ListenCoinFundingRate`。后者是 `main.go:548` 每 120 秒调度的活跃策略链，使用 `symbol_funding_rates` 表、`OwnerFundingRate` owner、`funding_rate:<symbol>` SourceRef，与 `eat_rate_symbols` / `eat_rate:<id>` 是两套独立数据。探针 05 六项检查全部为 `true`，确认该链未被波及。

### 1.4 隔离副本与等价校验

副本 = `git archive HEAD` + 按 `git status --porcelain -z` 同步 8 个 M 文件 + 删除 2 个 D 文件 + 副本内 `conf/app.conf.example → conf/app.conf` 兜底（不复制用户真实配置）。

```
TRACKED_GO 682  WORKTREE_GO 680  DELETED_GO ['controllers/eatRate.go','rate/index.go']
COPY_GO 680  IDENTICAL 680  MISSING 0  DIFFER 0  COPY_ONLY 0
```

剪枝说明：工作区另有 75 个 `.local/` 私有探针 Go 文件（未跟踪、Go 工具链默认忽略点目录），以及 `strategy_templates/research/**`、`research_specs/` 研究目录，均不进入副本；`git grep` 与 `git ls-files` 全部显式剪枝。审计过程中一次全仓 `os.walk` 触发 `SIGTERM(137)`（项目已知体积问题），改用 `git ls-files` 清单逐文件 SHA256 比对后通过——**属审计工具限制，非代码缺陷**。

---

## 二、删除完整性核对

| 层级 | HEAD | 工作区 | 证据 |
| --- | --- | --- | --- |
| HTTP 路由 | 4 条 `/fund-rate/eat*` | **0** | 探针 01：路由字面量 `187 → 183`，4 条旧路由全数消失；探针 03 运行时 4 条请求全部 `404`，同一 handler 的 sentinel 对照 `200` |
| 控制器 | 424 行 | 文件删除 | `git status` D；`EatRateController` 全仓 0 引用 |
| 定时任务 | `rate/index.go` | 文件删除 + main 注释删除 | `ListenRateEat` 文本 `head=true / current=false`；`go_binance_futures/rate` import 0 引用 |
| ORM 模型 | `EatRateSymbols` + `TableName` + `RegisterModel` | 全删 | AST 符号计数 `2 → 0`；`main.go` 中 `0` |
| SDK 封装 | `GetIncome` / `IncomeParams` | 全删 | AST `1 → 0` / `2 → 0`；独立的 `GetIncomeHistoryContext`（AccountClient，account-scoped）仍存在，计数 `1` |
| 控制器包级辅助函数 | `getSpotPrice` / `getFuturesPrice` / `validateEatRateFuturesOpenSlot` / `submitEatRateFuturesOpen` / `prepareEatRateFuturesClose` / `eatRateSourceRef` | 随文件删除 | `git grep` 在 `controllers/*.go` 与全仓 Go 文件中 **0 处残留引用** |
| SQL / 迁移 | 无 `eat_rate_symbols` DDL | 无 | `command/*` 与 `*.sql` 0 命中（表由 Beego `RegisterModel` 自动建） |
| 文档 | §13 章节、README TODO、mcp 列表项 | 已同步 | api-reference 章节标题 `1..14` 连续无断号 |
| 前端 | 后端 `static/` + 前端源码 + `dist/` | 无残留 | 三处 `git grep -E 'fund-rate/eat|eatRate|吃资金费率|资金费率套利'` 全部 0 命中 |
| Go 符号总残留 | — | **0** | 剪枝后 `git grep -i -E 'eat_?rate|fund-rate/eat'` 仅命中 `doc/agent/v3`、`doc/agent/v4` 历史报告与 `doc/TODO.md` 勾选项 |

---

## 三、离线 Gate 实证

全部在隔离副本内执行，显式清空 13 个 opt-in 许可 env（`BINANCE_TESTNET_*` / `SPOT_QUANT_TESTNET_*` / `SPOT_QUANT_VALIDATE_*` / `SPOT_LEDGER_MYSQL_DSN` / `SPOT_STRATEGY_LIVE_TRADING_APPROVAL`）。

| Gate | 结果 |
| --- | --- |
| `go build -o /tmp/... .` | **EXIT=0** |
| `go vet ./...` | **EXIT=0** |
| `git diff --check`（工作区） | **EXIT=0** |
| `go test ./... -count=1` | **EXIT=0，73 ok + 12 无测试包 + 0 FAIL**（探针写入前） |
| **探针删除后复验** `go test ./... -count=1` | **EXIT=0，73 ok + 12 无测试包 + 0 FAIL**；`go build` EXIT=0 |
| `go test -race -count=1`（10 个相关包：controllers / models / routers / feature / feature-api-binance / futuresownership / spot-api-binance / spotledger / spotexecution / root） | **EXIT=0，无 DATA RACE** |
| `./scripts/test-go.sh -short` | **EXIT=0** |
| opt-in 实网测试隔离 | 3 个真实 Testnet 用例（`TestSpotQuantLiveTestnetBUYSELLAndWS`、`TestStage0675RealTestnetAutomaticRunnerCycle`、`TestSpotQuantTestnetMarketPreflight`）全部 **SKIP**，标准回归不联网、不发单 |
| 前端 | 本轮无前端改动，未运行 `tsc` / `eslint` |

> 全量测试的红点归因沿用既有结论：用户工作区因 `conf/app.conf` 未入 Git 与未跟踪 research runner，直接 `go test ./...` 可能变红；副本用 `app.conf.example` 兜底且不含 research 目录，故 73 ok / 0 FAIL **不代表「用户工作区全仓测试通过」**。

---

## 四、隔离探针（5 项，全部 PASS）

探针只写在副本 `auditprobe/` 内，使用临时 SQLite 文件与 `httptest`，**不联网、不访问任何真实库、不下单**。

```
PROBE_EATRATE_ROUTES head_total=187 head_eat=4 current_total=183 current_eat=0
  HEAD_ROUTE /fund-rate/eat | /fund-rate/eat/:id | /fund-rate/eat/start/:id | /fund-rate/eat/end/:id
PROBE_EATRATE_SYMBOLS EatRateSymbols head=2 current=0 GetIncome head=1 current=0 \
  IncomeParams head=2 current=0 ListenRateEat head=true current=false
PROBE_EATRATE_INDEPENDENT_INCOME_READER GetIncomeHistoryContext=1
PROBE_EATRATE_RUNTIME_CONTROL GET /auditprobe/sentinel status=200
PROBE_EATRATE_RUNTIME_TREE contains_fund_rate_eat=false
PROBE_EATRATE_RUNTIME GET  /fund-rate/eat           status=404
PROBE_EATRATE_RUNTIME POST /fund-rate/eat           status=404
PROBE_EATRATE_RUNTIME POST /fund-rate/eat/start/1   status=404
PROBE_EATRATE_RUNTIME POST /fund-rate/eat/end/1     status=404
PROBE_EATRATE_SYNCDB force=false legacy_table_kept=true enabled_rows=1 registered_model_table_created=true
PROBE_EATRATE_CHAIN ListenCoinFundingRate_still_scheduled=true
PROBE_EATRATE_CHAIN funding_rate_open_still_active=true
PROBE_EATRATE_CHAIN auto_exit_filters_source_prefix=true
PROBE_EATRATE_CHAIN manual_claim_allows_funding_owner=true
PROBE_EATRATE_CHAIN manual_claim_rewrites_source=true
PROBE_EATRATE_CHAIN manual_claim_http_route=true
PROBE_EATRATE_ORPHAN legacy_source_ref="eat_rate:1" auto_exit_includes=false manual_claim_available=true
```

解读要点：

1. **sentinel `200` + 旧路由 `404`** 证明路由子系统确实在工作，404 不是「服务未初始化」造成的假阳性。
2. **syncdb 探针**复现 `command/db_update.go:420` 的 `orm.RunSyncdb("default", false, false)`：先建 `eat_rate_symbols` 表并写入 `enable=1` 一行，再注册现存模型跑 syncdb，结果 `symbol_funding_rates` 被创建（证明 syncdb 真的执行）、旧表与旧行**均保留**。即删除 `RegisterModel` 不会隐式 DROP 物理表或数据；`appversion.DatabaseSchemaVersion` 仍为 **25**，删代码不改 schema 是正确做法。
3. `manual_claim_available=true` **仅表示 HTTP 入口存在且 owner 白名单包含 `OwnerFundingRate`**；它**不能**覆盖已有活跃 managed 记录（见 F1），不要据此判定遗留仓位可自动恢复。

---

## 五、Findings

### F1 · P1 · 删除唯一专用平仓入口后，历史 `eat_rate:<id>` managed 仓位无内置恢复路径

**触发条件**：生产库存在由旧 `POST /fund-rate/eat/start/:id` 成功开出、且尚未通过 `End()` 平掉的仓位，即 `futures_managed_positions` 中 `owner='funding_rate'`、`source_ref LIKE 'eat_rate:%'`、`managed_qty>0` 的活跃记录；或 `eat_rate_symbols` 中 `enable=1` / `spot_quantity>0` 的记录。

**代码事实**（HEAD 保留代码 + 被删文件对照）：

| 位置 | 行为 |
| --- | --- |
| 被删 `controllers/eatRate.go::End()` / `prepareEatRateFuturesClose` | 唯一按 `eatRateSourceRef(row) == "eat_rate:<id>"` 严格校验并平掉合约腿 + 现货腿的入口 |
| `feature/ownership.go:56,76`（`syncStrategyExitPositions`，由 `feature/account_trade_cycle.go:60` 注入） | 对 `OwnerFundingRate` **只纳入** `strings.HasPrefix(position.SourceRef, "funding_rate:")`；`eat_rate:` 前缀被 `continue` 跳过 → 自动策略退出链不接管 |
| `service/futuresownership/manual_claim.go::claimVerifiedManualPosition`（:151-235） | 遍历同 symbol/side 的 managed 记录，`activePositionStatus && ManagedQty>0` 时直接返回 `position is already managed by ...`；非 closed 或有数量则返回 `unresolved ownership record; reconcile first`。**即使 owner 白名单允许 `OwnerFundingRate` 也无法覆盖已有 managed 仓位** |
| `service/futuresownership/reconcile.go` | 只把 managed 数量对账到交易所真实仓位 / 置 closed，**不改写 `source_ref`、不主动平仓** |
| `routers/router.go:27-29` | 仅 `/agents/trade/ownership`（视图）、`/reconcile`（对账）、`/manual-claim`（认领）三个接口，均不能平掉该类仓位 |

**探针证据**：`PROBE_EATRATE_ORPHAN ... auto_exit_includes=false`；探针 05 六项 chain 检查全 `true`（即活跃链、前缀过滤、认领入口状态与上述分析一致）。

**本地只读取证**：`db/coin.db`（开发库，`mode=ro&immutable=1`）有 `eat_rate_symbols` 表、1 行 `enable=0`，**无 `futures_managed_positions` 表** → 本地无活跃遗留仓位。生产库状态**未在本次审计中核验，也不应由审计动作触碰**。

**要求（部署前必须完成）**：

1. 在目标库执行只读预检并留证：
   ```sql
   SELECT id, account_id, owner, symbol, position_side, managed_qty, status, source_ref
     FROM futures_managed_positions
    WHERE managed_qty > 0
      AND (source_ref LIKE 'eat_rate:%' OR owner = 'funding_rate')
    ORDER BY id;

   SELECT id, symbol, enable, spot_quantity, futures_quantity, start_time, end_time
     FROM eat_rate_symbols
    WHERE enable = 1 OR spot_quantity > 0 OR futures_quantity > 0;

   SELECT id, position_id, symbol, client_order_id, status
     FROM futures_managed_orders
    WHERE status IN ('NEW','PARTIALLY_FILLED','PENDING_CANCEL')
      AND client_order_id LIKE '%eat_rate%';
   ```
2. 若命中：先在交易所人工平仓并按账本流程处置，或先提供**受控恢复入口 / 受控迁移**（含操作者授权与 append-only 审计），再部署本次删除；**禁止直接改库绕过账本**。
3. 若未命中：仍需保留 0 行结果作为发布证据。

> 定性：这是删除动作**暴露**的历史数据债务，不是本次 diff 写错代码。但对「删除代码」类改动，数据面预检是必须前置的部署 Gate。

### F2 · P2 · 遗留现货库存与 `eat_rate_symbols` 物理表无处置路径

旧 `Start()` 买入的现货数量记在 `eat_rate_symbols.spot_quantity`，删除控制器后**没有任何 API 能读取或精确卖出这部分库存**。`spot/api/binance/index.go:183` 的 `SellMarket` 仍存在，但仅由旧现货自动策略 `spot/spot.go` 按 `new_symbols` 配置调用，不是人工精确处置入口。探针 04 与 `command/db_update.go:420` 共同证明 `RunSyncdb(force=false)` 不会删除旧表或数据，物理表与历史行会永久保留。

**要求**：书面确认「保留 `eat_rate_symbols` 表、不做数据清理」是有意决定（与方案 01 旧 `order` 表物理保留的惯例一致），写入 `doc/TODO.md` 或发布说明；若目标是彻底清理，应提供显式受控的版本化迁移（备份 + `DROP TABLE` + 回滚），不得依赖 syncdb 隐式行为或人工裸 SQL。

### F3 · P2 · 本次改动缺少部署前只读预检与回滚留证，`doc/TODO.md` 已提前打勾

10 个改动文件中没有任何数据预检、盘点或回滚说明；`doc/TODO.md:36` 已写 `[x] 删除 ListenRateEat 相关代码`。离线 Gate 只能证明代码可编译，无法证明生产数据状态安全（F1/F2）。

**要求**：把 F1 的只读预检结果与 F2 的保留决策作为「完成」的前置条件；在此之前建议把该勾选项改为「代码删除完成，待生产数据预检」。同段还有未勾选的「删除 合约交易->交易复盘 相关代码」，说明本轮是系列清理第一步，**本次审查范围仅覆盖 ListenRateEat 链路，不包含交易复盘删除**。

### F4 · P3 · 文档一致性

- `README.md:792` 仍保留 `[ ] 增加对冲功能，实现空投保值 (现货买入和x倍合约等值对冲)`——描述的正是被删的 EatRate 功能（现货买入 + 等值合约做空），与删除决策矛盾，建议同步删除或改写为「已移除」。
- `.github/skills/api-reference.md` 章节 `1..14` 连续、无断号；`doc/mcp.md` 移除「资金费率套利」表述正确（该 HTTP 接口现已不存在）。
- `doc/agent/v3/CODE_REVIEW_V3-5-rereview.md`、`doc/agent/v4/v4-5-implementation-report.md` 等历史报告仍提及 `controllers/eatRate.go`、`rate.ListenRateEat`、`GetIncome`。按「不覆盖历史报告」纪律保留，属历史事实而非残留缺陷；如需可加一行「已删除」注记。
- `doc/data/income.js` 在 `GetIncome` 删除后不再被 `@returns` 引用，但 `GetIncomeHistoryContext` 仍返回 `futures.IncomeHistory`，样例仍有参考价值，不必删除。

### F5 · P3 · 审计链断裂（基线事实，非本次缺陷）

`dfc843d doc: update doc` 删除 121 个文件，其中包含整个 `doc/trade/` 的 01 / 02 / 06 / 06-7 历轮 Code Review 与开发方案。master 上仅剩 32 份 `doc/agent/*` 历史报告。影响：在 master 分支上无法直接引用 06 系列结论，需要切回 `feat/trade_spot` 分支或查 git history。本轮报告按既有约定重建 `doc/trade/` 目录；若不希望重建，可改放 `doc/` 根目录。

### F6 · P3 · 无新增永久回归测试

本次为纯删除，未新增测试；离线 Gate + 隔离探针足以证明当前状态，但无法阻止旧路由 / 符号在后续提交中被重新引入。对「删除废弃功能」而言可接受，列为观察项。

---

## 六、Gate 判定

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 删除完整性 / 无残留引用 | ✅ PASS | §二，`git grep` + AST + 运行时 404 |
| 离线 Gate（build/vet/test/race/脚本/diff-check） | ✅ PASS | §三，全部 EXIT=0 |
| 隔离探针（5 项） | ✅ PASS | §四 |
| 前端源码 / dist / static 同步 | ✅ PASS | 无旧端点调用，前端工作区干净 |
| 本地开发库遗留数据盘点 | ✅ PASS（只读） | `db/coin.db` 1 行 `enable=0`，无活跃 managed 仓位 |
| **生产库只读预检 + 遗留仓位 / 库存处置** | **⛔ 未执行 / BLOCKED** | F1 / F3，本次审计不触碰生产库 |
| 主网实单 / 生产 v25 检查单 / 成套发布与回滚 / 异常恢复入口 | **⛔ 全部 BLOCKED** | 沿用 `CODE_REVIEW_06-7-ALL.md` 结论，本次删除不改变任何既有 Gate |

---

## 七、后续准入条件（按顺序）

1. **F1 + F3 前置**：在目标库执行 §五 的三条只读预检 SQL，留证 0 行或命中结果；命中时先人工平仓 / 提供受控恢复入口。
2. **F2 决策**：书面确认 `eat_rate_symbols` 物理表与数据保留策略；如需清理则提供版本化受控迁移与回滚。
3. **F4 收尾**：同步 `README.md` 对冲 TODO；`doc/TODO.md` 勾选状态与预检结果对齐。
4. 完成 1–3 后再合并 / 发布本次删除；**不得因离线全绿而跳过数据面预检**。
5. 06-7 整体准入（方案契约裁决、生产 Gate、独立书面授权）仍按 `CODE_REVIEW_06-7-ALL.md` 执行，本轮无变化。

---

## 八、最终结论与审计边界

**⚠️ 有条件通过：认可「ListenRateEat 相关代码删除完整、离线可回归、可作为研发基线」，不认可「该删除任务已可无条件合并上线」。F1（P1，遗留 `eat_rate:` managed 仓位无内置恢复路径）与 F3（P2，缺部署前只读预检与留证）必须在部署前闭环；F2 / F4 / F5 / F6 为需决策或可接受的观察项。本次审查不改变 06-7 整体复审「不认可整体完成、生产与发布 Gate 全 BLOCKED」的结论。**

**审计边界**：未修改任何业务代码、未自动 commit/push、未改 `conf/app.conf`（副本仅用 `app.conf.example` 兜底）、未碰 `static/` / `strategy_templates/` / 前端源码；本地 SQLite 仅以只读模式盘点，**未对任何真实库执行 `sync db` 或预检 SQL**，探针中的 `RunSyncdb` 只作用于临时 SQLite 文件；未访问真实 Binance、未下单、未重启服务、未操作生产仓位。隔离探针只写在 `/tmp/phase_audit_eatrate/auditprobe/`，删除后复验干净副本 `go test ./... -count=1` 为 73 ok / 0 FAIL；副本与本次临时产物在审计结束删除。
