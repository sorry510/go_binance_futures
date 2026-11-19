# CODE_REVIEW_UF-3 · 复审报告

> 日期：2026-11-19。复审对象：当前工作区中的 UF-3「异常订单一键定位」实现；不覆盖实施方 `CODE_REVIEW_UF-3.md`。
>
> 开发契约：[UF · U 本位合约交易六项精简优化开发方案](开发方案-UF-U本位合约交易六项精简优化.md) §五、§九。
>
> **复审结论：UF-3 在「当前源码 + 离线 SQLite + 前后端构建」范围内 ✅ 通过，允许进入 UF-4 代码开发。**
>
> **真实 MySQL 8 / ARM Linux / Testnet / Mainnet / 生产发布 Gate 仍为 ⛔ BLOCKED。**
> 本轮未发现新的 P0/P1 Finding。UF-3 是只读异常定位，不是自动对账、异常修复或交易执行机制。

---

## 一、复审基线与范围

### 1.1 基线

| 项目 | 值 |
| --- | --- |
| 后端仓库 | `/Users/zhz/work/binance/go_binance_futures` |
| HEAD | `0230c06` (`master`, `origin/master`) |
| 工作区总改动 | 179 项：84 D + 5 M + 90 ?? |
| 非 UF-3 夹带 | `CHANGELOG.md` 新增 v2.2.5 记录，实施方自评已声明为既有改动，本轮不覆盖 |
| 后端 UF-3 改动 | `service/orderhistory/anomalies_uf3.go`、`controllers/orders.go`、`routers/router.go`、2 个 UF-3 测试文件、UF-3 自评与方案状态文档 |
| 静态产物改动 | 84 删除 + 1 修改 + 86 新增，均为前端重建同步结果 |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`，HEAD `b2508ab` |
| 前端 UF-3 改动 | 6 文件：`locales/{zh-CN,en}.yaml`、`src/api/order.ts`、`src/views/order/AnomalyPanel.vue`、`src/views/order/index.vue`、`src/views/ai/observability.vue` |

本轮未修改业务代码、未 commit/push、未改 `conf/app.conf`、未执行 `sync db`、未访问真实 Binance、未重启服务、未触碰仓位。

### 1.2 隔离副本

| 检查 | 结果 |
| --- | --- |
| 后端副本 | `/tmp/phase_audit_uf3` |
| 前端副本 | `/tmp/phase_audit_uf3_ui` |
| 后端期望文件 | 1184 |
| 后端实际文件 | 1184 |
| Go 文件 | 779 |
| static 文件 | 122 |
| SHA256 差异 | 0 |
| 缺失 / 多余文件 | 0 / 0 |
| 删除确认 | 84/84 |
| `conf/app.conf` in copy | false |
| `conf/app.conf.example` in copy | true |

等价性结论：隔离副本可代表当前工作区；所有测试、探针和构建均在隔离环境完成。

---

## 二、UF-3 静态审查

### 2.1 接口与路由

新增接口：

```text
GET /orders/anomalies
```

路由仅绑定 `get:Anomalies`：

```go
web.Router("/orders/anomalies", &controllers.OrderController{}, "get:Anomalies")
```

未发现 `post/put/patch/delete:Anomalies`。控制器处理链：

```go
if !ctrl.authorized() { return }
...
result, err := (orderhistory.Store{}).Anomalies(ctrl.Ctx.Request.Context(), orderhistory.AnomalyFilter{...})
if err != nil {
    if errors.Is(err, orderhistory.ErrInvalidQuery) {
        ctrl.respondError(400, "invalid anomaly filters")
    } else {
        ctrl.respondError(503, "managed order anomalies unavailable")
    }
    return
}
ctrl.success(result)
```

判定：

- JWT 缺失或 `Username` 为空 → 401 `order_history_auth_required`；
- 非法过滤参数 → 400，且参数在服务层 SQL 前再次校验；
- DB/上下文失败 → 503 泛化错误，不返回伪造空列表，不泄漏原始 SQL 或错误详情；
- 无新增 POST/DELETE/认领/撤单/对账执行入口。

### 2.2 分类与数据来源

`service/orderhistory/anomalies_uf3.go` 使用单一 `UNION ALL` 只读视图，来源限定为：

1. `futures_managed_orders`：`managed_order_reconcile`、`fill_verification_pending`、`aged_managed_order`；
2. `futures_managed_positions`：`managed_position_reconcile`、`settlement_incomplete`；
3. `agent_trade_proposals`：`proposal_execution_uncertain`、`proposal_protection_failed`。

Agent Proposal 行显式映射为：

```sql
'main' AS account_id, 'agent_trade' AS owner,
NULL AS position_id, NULL AS managed_order_id
```

因此不会把 Proposal 伪装成 Managed Order，也不会被 Lead 查询命中。

`fill_verification_pending` 与 `systemhealth::checkTrade` 的真实条件保持一致：

- `requested_qty > 0`；
- `ABS(filled_qty - requested_qty) <= 0.00000001`；
- 有非空 `exchange_order_id`；
- 关联周期 `status IN ('active','closed')`；
- 存在真实 Fill 行且至少一行 `rest_verified=0`；
- Fill 数量总和与订单成交量精确匹配。

数量缺失或不匹配的 Fill 保留为 `managed_order_reconcile`，不会降级成“软待核实”。

### 2.3 输入校验与分页

`AnomalyFilter.validate()` 全部为白名单校验：

- `account_id`：空默认 `main`，仅允许 `main` / `lead`；
- `owner`：`^[a-z][a-z0-9_]{0,31}$`，`all` 转为空；
- `symbol`：`^[A-Z0-9]{1,32}$`；
- `position_side`：仅 `LONG` / `SHORT` / 空；
- `category`：9 类白名单；
- `page`：1～1,000,000；`limit`：1～100。

排序固定为 `updated_at DESC, source ASC, managed_order_id DESC, position_id DESC, proposal_id DESC`，并有 `LIMIT/OFFSET`；不存在客户端拼接表名/列名。`source` 来自代码常量，非请求参数。

### 2.4 只读性与人工覆盖隔离

- 服务层注释与实现均限定为只读；无 `binance` import、无 `Submit*`、无 DML。
- `history_status_override` / `history_completeness_override` 未进入查询；人工历史展示修改无法消灭真实异常。
- 无 `position_id` 的异常 OPEN 作为独立 `managed_order` 行返回，`client_order_id`、`exchange_order_id`、原始状态与 Owner 保留，前端不虚构周期。

### 2.5 前端边界

`AnomalyPanel.vue` 只调用：

- `GET orders/anomalies`；
- `GET orders/:position_id/details`（已有只读详情接口）。

未发现下单、撤单、修复、认领、对账执行或 POST/PUT/DELETE 调用。系统看板仅通过 `router.push` 跳转到 `/futures/list` 并携带过滤参数，不触发自动修复。

---

## 三、Gate 结果

所有命令在隔离副本执行，环境变量显式清空；Go 包列表为 88 个可部署工程包，排除历史 `strategy_templates/research/`。

| Gate | 结果 |
| --- | --- |
| 工作区 `git diff --check` | ✅ exit 0 |
| `go vet`（88 包） | ✅ exit 0 |
| `go build -o /tmp/go_binance_futures_uf3_audit .` | ✅ exit 0 |
| 全量 `go test -count=1`（88 包） | ✅ 75 ok + 13 no test + 0 FAIL |
| UF-3 专项 `go test -count=5 -run '^TestUF3'` | ✅ exit 0 |
| 5 个关键包 `go test -count=5` | ✅ exit 0 |
| 5 个关键包 `go test -race -count=2` | ✅ exit 0 |
| 前端隔离副本 `pnpm typecheck` | ✅ exit 0 |
| 前端隔离副本 `pnpm build` | ✅ exit 0，4.28 MB |
| 重建 `dist` 与后端 `static/` | ✅ 122/122 文件一致，`diff -qr` exit 0 |

说明：

- race 中可能出现 macOS `LC_DYSYMTAB` linker warning；进程退出码为 0，判定非致命。
- 日志中出现 `COIN-M DAPI configured: environment= mainnet ...` 为初始化打印，不是审计期间真实 DAPI 请求。
- 审计脚本曾出现两次自身错误：`git status --porcelain=z` 参数拼写、探针中 `ctx.Reset()` 清掉 JWT claims / 无 ORM alias。均已在隔离副本内修正后重跑 PASS，不属于 UF-3 代码 Finding。

---

## 四、隔离探针

探针仅写入 `/tmp/phase_audit_uf3`，未污染工作区。

| 探针 | 覆盖点 | 结果 |
| --- | --- | --- |
| `TestAuditUF3ProposalNeverLeaksToLead` | Main Proposal 不泄漏到 Lead；Owner/source/ID 映射不混同 Managed Order | ✅ PASS |
| `TestAuditUF3AnomalyReadDoesNotMutateLedger` | 调用 `Anomalies` 前后 4 张表行数不变 | ✅ PASS |
| `TestAuditUF3ManualOverrideCannotHideManagedReconcile` | 人工覆盖为 closed/verified 后真实 `reconcile_required` 仍可检索 | ✅ PASS |
| `TestAuditUF3InvalidFilterStopsBeforeNilDB` | 非法 account 在 nil DB 下返回 `ErrInvalidQuery`，未触达数据库层 | ✅ PASS |
| `TestAuditUF3AnomalyRouteIsGetOnlyReadOnly` | 路由只注册 GET；handler 无 DML、无交易提交、无 binance 调用 | ✅ PASS |
| `TestAuditUF3UnauthorizedRequestNeverGetsData` | 无 JWT 返回 401，响应不含 anomaly payload | ✅ PASS |

结果：6 个运行节点全部 PASS；`go test -count=5 -run '^TestAuditUF3' ./service/orderhistory ./controllers` exit 0。

---

## 五、永久测试覆盖复核

实施方永久测试覆盖以下契约，本轮复核认为口径成立：

- `TestUF3AnomalyAccountIsolationAndOrphanOpen`：Main/Lead 隔离、孤儿 OPEN、Agent 来源独立；
- `TestUF3AnomalyCategoryAndOwnerFiltering`：类别、Owner、币种、方向组合过滤；
- `TestUF3HealthSummaryReconcileLinkCombinesPositionsAndOrders`：健康看板组合入口同时覆盖仓位与订单对账；
- `TestUF3AnomalyRealRESTFillPendingMatchesHealthClassification`：真实 Fill 待核实与数量不符的强对账区分；
- `TestUF3AnomalyPagedStableAndStaleOnlyAfterTenMinutes`：10 分钟阈值与分页越界；
- `TestUF3AnomalyInvalidFiltersFailBeforeAnySQL`：非法输入在 SQL 前失败；
- `TestUF3AnomalyCanceledReadIsNotSuccessfulEmpty`：取消 context 返回显式错误；
- `TestUF3AnomalyRouteRequiresJWTAndValidFiltersBeforeDB`：401/400 与 JSON 响应。

---

## 六、观察项与未通过 Gate

### 6.1 观察项（不阻断本阶段）

1. **O1**：`CHANGELOG.md` 的 v2.2.5 条目属于工作区既有改动，不是 UF-3；本轮未覆盖、未提交。
2. **O2**：系统健康中的 active/historical Fill 细分没有独立导航过滤；异常列表总数按设计不等于健康汇总数，页面需保持“当前可定位记录”与“系统健康汇总”的文字差异。
3. **O3**：`last_reconciled_at` 为零时前端展示 `updated_at`，这是本地时间，不代表刚刚完成交易所 REST 核实。
4. **O4**：`aged_managed_order` 的 10 分钟阈值基于本地 `updated_at`；真实生产时钟和写入语义未验证。
5. **O5**：SQLite 离线验证不能替代 MySQL 8 的 `EXPLAIN`、索引与 P95；大量异常行下的分页性能未实测。

### 6.2 仍然 BLOCKED

| Gate | 状态 |
| --- | --- |
| 真实 MySQL 8 账本与查询计划 | ⛔ 未验 |
| ARM Linux 部署包、`static` 成套发布、回滚 | ⛔ 未验 |
| 真实 Binance FAPI / 私有 WS / 账户数据 | ⛔ 未验 |
| Testnet / Demo / Mainnet 异常订单定位 | ⛔ 未验 |
| 生产持久化授权、审计、回滚演练 | ⛔ 未验 |
| 真实 Agent Proposal、Lead、多 Owner 混合数据 | ⛔ 仅 Mock/SQLite 验证 |

历史未决事项不因本轮通过而关闭：现货扫描 F1、06-7 F1、CM-7 F8/P1、UF 真实账户与 MySQL 8 Gate。

---

## 七、最终判定

```text
UF-3：✅ 源码 / 离线 SQLite / 前后端构建 Gate 通过

允许：进入 UF-4（为什么没有开仓）代码开发。
禁止：宣告生产可用、打开真实自动交易开关、把只读定位视为自动对账或异常修复、用离线通过替代真实 MySQL 8 / 实盘验收。
```

本复审不修改历史报告，不替代真实交易所与生产发布 Gate。若后续新增写操作、改变异常分类口径或引入新的前端构建产物，应另起复审并记录证据。
