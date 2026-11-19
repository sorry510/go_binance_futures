# CODE_REVIEW_UF-ALL-rereview · U 本位合约六项精简优化整体复审

> 复审日期：2026-11-19。开发方案：[UF · U 本位合约交易六项精简优化](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **最终判定：✅ UF-ALL 源码 / 离线 / 前后端构建整体复审通过。**
>
> **允许：进入「经单独授权的 MySQL 8 / ARM / 实盘小额验收」准备流程。**
>
> **禁止：把本判定解读为生产发布、真实账户安全或已完成实盘验收。所有真实交易所、真实账本与部署 Gate 仍为 BLOCKED。**

---

## 一、复审范围与基线

### 1.1 基线变化说明

复审开始时后端 HEAD 为 `f31a7e6 feat: UF-5`，工作区 183 项；审查过程中实施方提交了 UF-6，HEAD 前进到：

```text
后端 HEAD 8a9c2938edfdb6953debd8493ca2780027c71cc8  (feat: uf-6)
工作区残留：M CHANGELOG.md（非 UF 功能改动）
前端 HEAD 0323d5be5169fcc005442df86828ecb04f7517d7
前端工作区：4 项 UF-6 改动（locales/en.yaml、locales/zh-CN.yaml、src/api/order.ts、src/views/order/index.vue）
```

隔离副本已基于审查期间的最新 HEAD 重建，避免因基线漂移产出错误结论。本报告判定对象为当前已提交的 UF-6 源码 + 前端未提交的 UF-6 源码/静态产物。

### 1.2 已审查材料

- 六份实施自评：`CODE_REVIEW_UF-1.md` ～ `CODE_REVIEW_UF-6.md`
- 六份独立复审：UF-1/UF-2、UF-3、UF-4、UF-5、UF-6 原始复审、UF-6 F1 整改复测
- 开发方案全文与中英文 README
- 全部 UF 相关后端源码、控制器、路由、前端源码与静态产物

### 1.3 审查纪律

未修改业务代码/配置/策略/静态源码；未 commit/push、未运行 `sync db`、未连接 Binance、未下单、未重启服务、未触碰仓位。审计期间清空交易所/账户环境变量，临时探针只在副本中。

---

## 二、逐阶段 Findings 闭环清单

| 阶段 | 独立复审结论 | 阻断 Finding | 当前闭环状态 |
| --- | --- | --- | --- |
| UF-1 总开关只阻止新仓 | ✅ 通过 | 无 | 保持通过 |
| UF-2 配置失败 fail-closed | ✅ 通过 | 无 | 保持通过；非原子 TOCTOU 为已声明边界 |
| UF-3 异常订单定位 | ✅ 通过 | 无 | 保持通过；路由在 UF-6 曾被误注释，见下 |
| UF-4 未开仓原因诊断 | ✅ 通过 | 无 | 保持通过；进程内诊断边界已声明 |
| UF-5 已核实策略效果 | ✅ 通过 | 无 | 保持通过；只读统计不等于对账/修复 |
| UF-6 订单历史性能 | ❌ 原始复审不通过 | **F1/P1：`/orders/anomalies` 路由落入注释** | **✅ 已闭环**（见第三节） |

---

## 三、F1/P1 闭环独立验证

### 3.1 修复事实

```text
routers/router.go:189  web.Router("/orders/summary", ... "get:Summary")
routers/router.go:190  web.Router("/orders/anomalies", ... "get:Anomalies")
```

两条独立注册语句，字面量 `\n` 已移除。新增永久测试 `routers/router_uf6_rereview_test.go::TestUF6RereviewAllOrderRoutesRegisteredAndProtected`，通过真实 `web.BeeApp.Handlers.ServeHTTP` 检查路由表，而非仅调用控制器方法。

### 3.2 复测判定

实施方 `CODE_REVIEW_UF-6-rereview-2.md` 声明 F1 闭环。本轮整体复审不采信单方声明，独立执行路由探针：

| 未授权 GET | 期望 | 实测 |
| --- | --- | --- |
| `/uf-all-not-a-real-route-404`（sentinel） | 404 | ✅ 404 |
| `/orders/anomalies`（UF-3） | 401 | ✅ 401 |
| `/futures/entry-diagnostics`（UF-4） | 401 | ✅ 401 |
| `/orders/performance`（UF-5） | 401 | ✅ 401 |
| `/orders/summary`（UF-6） | 401 | ✅ 401 |
| `/orders`（旧接口） | 401 | ✅ 401 |
| `/orders/123/details`（子订单/Fill） | 401 | ✅ 401 |

**结论：F1/P1 已闭环，UF-3/4/5/6 全部只读路由均实际注册且受 JWT 保护。**

---

## 四、跨阶段不变量核对

### 4.1 只读边界

- 4 条 UF 只读路由全部 `get:` 绑定：`/orders/anomalies`、`/futures/entry-diagnostics`、`/orders/performance`、`/orders/summary`。
- 写方法探针：POST/PUT/PATCH/DELETE 到上述路径均返回 404/405/410，无任何写入口被路由；410 来自 01-4 旧 order 表 tombstone，属预期安全行为。
- 新增服务文件（`performance_uf5.go`、`anomalies_uf3.go`、`history.go` 的 `Summary`）仅执行 SELECT `QueryRows/QueryRow`；无 `Insert/Update/Delete`，无 `Submit*`/`Cancel*` 交易调用，无 binance 包调用。
- `controllers/orders.go::Delete()` 是既有历史人工删除接口，非 UF 引入。

### 4.2 财务口径隔离（跨阶段探针实证）

| 探针 | 验证 | 结果 |
| --- | --- | --- |
| `TestAuditUFAllValuationIsNotCertifiedPerformance` | 未结算周期即使有本地行情估值、且被人工覆盖为 closed/verified，UF-6 可展示估值，但 UF-5 绝不纳入已核实收益 | ✅ PASS |
| `TestAuditUFAllAccountIsolationAcrossReadStages` | UF-5 固定 Main，Lead 周期不进入已核实报告；UF-6 汇总按 account_id 过滤，Main/Lead 各见自身周期 | ✅ PASS |
| `TestAuditUFAllCanceledReadsFailClosed` | UF-3 异常查询、UF-5 已核实统计、UF-6 汇总在取消 Context 时均返回错误，不伪造成功空结果 | ✅ PASS |

结论：三个阶段的口径彼此隔离——UF-3 定位异常、UF-4 解释未开仓、UF-5 是 REST 已核实结算收益、UF-6 是行情估值展示。**UF-6 的 `filtered_pnl` 与 UF-5 的已核实净收益不可混用；人工历史覆盖无法把未结算周期提升为已核实收益。**

### 4.3 账户与交易安全

- UF-1/UF-2 只作用于 Main U 本位新开仓前置路径；OFF 仅阻断新 OPEN，不阻断已有 Managed 退出/恢复；配置错误 fail-closed。
- Lead、Spot、COIN-M、手动/未知 Owner 的账户隔离在各阶段复审与永久测试中保持；本轮未发现 UF 改动触及这些路径。
- 无新建数据库表、无 schema 迁移、无全局交易开关、无队列/调度器；`conf/app.conf` 未改。

---

## 五、隔离副本与工程 Gate

### 5.1 等价性

| 副本 | 结果 |
| --- | --- |
| `/tmp/phase_audit_ufall` | `paths=1212 checked=1208 missing=0 diff=0 deleted_confirmed=0` |
| `/tmp/phase_audit_ufall_ui` | `paths=300 checked=298 missing=0 diff=0 deleted_confirmed=0` |

后端工作区仅剩 `CHANGELOG.md` 修改（非 UF 功能），已同步进副本；等价性通过。

### 5.2 Gate 结果

| Gate | 结果 |
| --- | --- |
| 两仓 `git diff --check`、gofmt | PASS |
| `go list ./...` | 88 / 88 包 |
| `go vet` 88 包 | PASS |
| `go test -count=1` 88 包 | PASS：**76 ok + 12 no test + 0 FAIL**（routers 包因新增永久路由测试由 no test 转为 ok） |
| UF 专项 `-count=5 -run '^(TestUF[1-6]\|TestUF6Rereview)'` 5 包 | PASS |
| 8 个关键包 `-count=5 -p=2`（含 routers） | PASS |
| 同 8 包 race `-count=2 -p=2` | PASS；macOS `LC_DYSYMTAB` warning 非致命 |
| `go build` | PASS |
| 前端 `pnpm typecheck` / `pnpm build` | PASS（corepack pnpm 12.21.0；`ERR_PNPM_IGNORED_BUILDS` 为依赖预检警告，退出码 0） |
| `dist/` 与后端 `static/` | PASS，`diff -qr` 零差异 |

说明：全量测试包数为 76 ok + 12 no test（此前各阶段为 75+13），差异来自 UF-6 修复新增的 `routers` 永久测试使该包从无测试变为有测试，属预期变化。

---

## 六、观察项与未通过 Gate

### 6.1 阶段遗留观察（均不阻断源码准入）

1. UF-1/UF-2：最终读库到真实 POST 非原子；5 秒成功缓存非账户永久证明。
2. UF-3：只读异常定位不等于自动对账/修复；`last_reconciled_at` 回退为本地时间；10 分钟阈值基于本地时钟。
3. UF-4：进程内诊断非账户真相/Fill 证明/多实例监控；`order_submitted` 是 ACK 非 Fill。
4. UF-5：无跨请求事务快照；零额异币手续费保守排除；PF 无亏损样本返回 null。
5. UF-6：`filtered_pnl` 是行情估值非已核实收益；初屏仍需完整汇总；SQLite 合成基准不代表生产性能。

### 6.2 仍然 BLOCKED（真实环境与部署）

- 真实 MySQL 8：EXPLAIN、索引、P95、10k 有 Fill 全量数据、备份/回滚。
- 真实 Binance：FAPI 保证金/杠杆权限与错误响应、私有 WS、Testnet/Demo/Mainnet 小额 OPEN/CLOSE、部分成交/未明委托。
- ARM Linux 部署包 + `static` 成套发布与回滚演练。
- 生产持久化授权、配置变更审计、撤销与回滚。
- 真实 Lead / COIN-M / Spot / 手动仓位并行隔离。

### 6.3 历史未决事项不受本轮影响

现货扫描 F1、06-7 F1、CM-7 F8/P1 仍按既有报告处理，不因 UF-ALL 通过而关闭。

---

## 七、最终结论

```text
UF-ALL：✅ 源码 / 离线 / 前后端构建整体复审通过
F1/P1（UF-6 /orders/anomalies 路由回归）：✅ 已独立验证闭环
允许：进入经单独授权的 MySQL 8 / ARM / 实盘小额验收准备流程。
禁止：据此宣告生产发布完成、真实账户安全或已完成实盘验收。
```

本轮未修改用户业务代码；只新增本报告。审计副本、探针、日志和编译产物将在报告写入与记忆同步后清理。
