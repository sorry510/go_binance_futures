# CODE_REVIEW_UF-6-rereview · 订单历史查询性能复审

> 复审日期：2026-11-19。后端基线：`master` / `f31a7e6 feat: UF-5`。前端基线：`0323d5be5169fcc005442df86828ecb04f7517d7`。
>
> **判定：❌ UF-6 不通过，禁止宣告 UF-ALL 完成，禁止进入生产验收。**
>
> 阻断原因不是 UF-6 自身的结果一致性或性能拆分，而是改动中意外删除了已交付的 UF-3 `/orders/anomalies` 路由。详见 F1。

---

## 一、复审范围与方法

### 1.1 已审查材料

- `doc/trade/开发方案-UF-U本位合约交易六项精简优化.md` 第八节（UF-6）
- `doc/trade/CODE_REVIEW_UF-6.md`
- 中英文 README 的 UF-6 说明
- `service/orderhistory/history.go`、`controllers/orders.go`、`routers/router.go`
- `service/orderhistory/history_uf6_test.go`、`history_uf6_benchmark_test.go`、`controllers/orders_uf6_test.go`
- 前端 `src/api/order.ts`、`src/views/order/index.vue`、`locales/zh-CN.yaml`、`locales/en.yaml`
- 静态产物 `static/index.html` 与 `static/static/js/*`

### 1.2 审查纪律

- 未修改业务代码/配置/策略/静态源码；未提交、未推送、未执行 `sync db`、未连接 Binance、未下单、未重启服务、未触碰仓位。
- 临时探针只写在隔离副本 `/tmp/phase_audit_uf6`，复评后清理。
- 审计期间清空 `BINANCE*` / `COIN*` / `SPOT*` / `FAPI*` / `DAPI*` / `UF6_BENCH_MODE` 等环境变量。

---

## 二、基线与等价性

```text
HEAD f31a7e6da26d7e697563ac9b73f7a21b140b4302
工作区：86 D + 7 M + 90 ?? = 183 项
```

隔离副本 SHA256 等价校验：

| 副本 | 结果 |
| --- | --- |
| `/tmp/phase_audit_uf6` | `paths=1295 checked=1205 missing=0 diff=0 deleted_confirmed=86` |
| `/tmp/phase_audit_uf6_ui` | `paths=300 checked=298 missing=0 diff=0 deleted_confirmed=0` |

说明：工作区同时包含 UF-3～UF-6 累积的静态资源重建。本报告不对历史阶段的静态重建归属作否定判断，只判定 UF-6 当前状态。

---

## 三、Findings

### F1 / P1：`/orders/anomalies` 路由被并入注释，UF-3 已交付功能回归

**位置**

```text
routers/router.go:189
```

**现状代码（实际原文）**

```go
web.Router("/orders/summary", &controllers.OrderController{}, "get:Summary")              // UF-6 all-filter PnL once per filter\n\tweb.Router("/orders/anomalies", &controllers.OrderController{}, "get:Anomalies")          // UF-3 read-only anomalous managed orders, cycles and Agent proposals
```

**原因**

UF-6 在插入 `/orders/summary` 路由时，把换行写成字面量 `\n`，同一行剩余内容全部落入 `//` 行注释。因此 `web.Router("/orders/anomalies", ...)` 不再执行，路由未注册。

**影响**

1. UF-3 已通过的只读异常订单定位接口 `GET /orders/anomalies` 变为 404。
2. 当前前端 `AnomalyPanel` 仍保留并会调用该接口，打开异常面板即失败。
3. 这不是 UF-6 性能优化的正确性损失，而是前一阶段已验收功能的回归；按既有复审标准属于 P1。
4. `go build`、`go vet`、全量测试均无法发现该问题，因为源码在 Go 语法层面完全合法。

**实证**

隔离副本中加载 `routers` 包并访问 Beego 全局路由表：

```text
GET /no-such-route        -> 404（sentinel，证明 404 判定有效）
GET /orders/summary       -> 401（已注册，JWT 生效）
GET /orders/performance   -> 401（UF-5 路由仍在）
GET /orders               -> 401（旧兼容路由仍在）
GET /orders/anomalies     -> 404（应为 401，实际未注册）
```

**要求修复**

把 `routers/router.go:189` 拆成两条独立语句：

```go
web.Router("/orders/summary", &controllers.OrderController{}, "get:Summary")     // UF-6 all-filter PnL once per filter
web.Router("/orders/anomalies", &controllers.OrderController{}, "get:Anomalies") // UF-3 read-only anomalous managed orders, cycles and Agent proposals
```

修复后必须重跑本报告全部 Gate 与探针，特别是路由探针必须得到 `/orders/anomalies` 的 401，并由独立报告记录闭环；不得只改代码后直接宣告 UF-ALL 完成。

---

## 四、UF-6 自身验收核对（F1 之外）

### 4.1 结果一致性：通过

| 验收点 | 证据 | 结论 |
| --- | --- | --- |
| Legacy `GET /orders` 默认完全兼容 | `Store.List` 保留完整 `filtered_pnl` 与 `pnl_unpriced_count`；`include_summary` 空值按 true 处理 | 通过 |
| 只有显式 `include_summary=0/false` 走快路径 | 其他值返回 400 fail closed | 通过 |
| 快页不计算全范围 | `ListPageOnly` 返回空 `FilteredPnL`、0 `PNLUnpriced`、`SummaryIncluded=false`，页内 `PageVerifiedNet` 与同分页 legacy 相同 | 通过 |
| 独立汇总与 legacy 全筛选一致 | `Summary` 复用同一 `project` + `sumDisplayedPnL`，USDT/USDC、无行情价、手工覆盖、账户/Owner/状态/日期过滤均逐项对照 | 通过 |
| 汇总与分页无关 | 页码变化不改变 `Summary` 的 `total`、`filtered_pnl`、`pnl_unpriced_count` | 通过 |
| 无服务端财务缓存陈旧 | Fill 价格更新后下一次 `Summary` 立即反映新结果 | 通过 |
| 失败不伪造零 | 非法过滤 400；取消 Context 返回错误；中途取消不返回部分汇总；加载失败前端显示错误而非零收益 | 通过 |
| 行情价时间不伪造 | 无本地行情时 `price_updated_at=0`；`calculated_at` 明确为本地汇总时间 | 通过 |
| 路由与 JWT | `/orders/summary` 仅 GET + JWT；非法参数 400 | 通过；但 F1 影响另一个已交付路由 |

### 4.2 前端缓存与请求策略：通过

- `summaryKey()` 仅由账户、Owner、币种、方向、状态、结算、起止时间构成，不含 `page` / `limit`。
- `fetchPage()` 与 `updatePageSize()` 调用 `load(..., false)`，翻页/改每页条数不重算全筛选汇总。
- 首次加载、筛选变化、主动刷新、批量操作成功后才刷新汇总。
- `requestToken` / `summaryRequestToken` 双序号防止旧响应覆盖；失败时不保留旧筛选的成功汇总。
- 文案明确 `calculated_at` 与 `price_updated_at` 是本地时间，不是 Binance 标记价实时核实时间。

### 4.3 性能声明：部分可复现，不等于生产结论

- 实施报告明确初屏仍需完整汇总，收益主要来自第 2、3 页复用。
- SQLite 合成基准使用无 Order/Fill 的 100/1000/10000 周期数据，隔离测量重复投影规模效应；不等于真实 MySQL 8、真实 Fill 或 QPS。
- 本轮 Gate 中 `TestUF6SyntheticBenchmark` 在未设置 `UF6_BENCH_MODE` 时按设计跳过；未把实施方单次测量当作独立生产性能验收。
- 未执行真实 MySQL 8 `EXPLAIN` / P95 / ARM Linux；不得据此宣称生产性能达标。

---

## 五、离线工程 Gate

| Gate | 结果 |
| --- | --- |
| 两仓 `git diff --check` | PASS |
| `gofmt -l` UF-6 相关 Go 文件 | PASS |
| `go list ./...` | 88 / 88 包 |
| `go vet` 88 包 | PASS |
| `go test -count=1` 88 包 | PASS，75 ok + 13 no test + 0 FAIL |
| UF-6 专项 `-count=5 -run '^TestUF6'` | PASS |
| 7 个关键包 `-count=5 -p=2` | PASS |
| 同 7 包 race `-count=2 -p=2` | PASS；macOS `LC_DYSYMTAB` warning 非致命 |
| `go build` | PASS |
| 前端 `pnpm typecheck` / `pnpm build` | PASS；Browserslist 过期提示非致命 |
| `dist/` 与后端 `static/` | PASS，`diff -qr` 零差异 |

**关键结论：上述 Gate 全 PASS 仍不能覆盖 F1。Go 合法性与路由注册语义是不同层面的验收。**

---

## 六、隔离探针

| 探针 | 结果 |
| --- | --- |
| `TestAuditUF6PageOnlyIsStrictlyLessThanLegacy` | PASS：快页不携带全筛选；同分页列表/页内净收益与 legacy 一致；独立汇总与 legacy 全筛选一致 |
| `TestAuditUF6SummaryFilterChangesAreNotCachedByPagination` | PASS：账户/Owner/状态隔离；分页不影响汇总身份；无服务端财务缓存错配 |
| `TestAuditUF6SummarySeesLateFillWithoutServerCache` | PASS：Fill 变更后下一次汇总反映新值 |
| `TestAuditUF6AllOrderRoutesRegistered` | **FAIL（预期暴露 F1）**：sentinel/summary/performance/list 均 401，`/orders/anomalies` 为 404 |

说明：前三个一致性探针在 `-count=5` 下 PASS；路由探针一次运行即稳定复现 F1。探针中曾出现两次构造错误（固定 client_order_id 冲突、把不同 limit 的页内净收益互比），均在隔离副本内修正后 PASS，不作为代码 Finding。

---

## 七、观察项与边界

1. **O1** 后端保留旧兼容接口的 O(N) 全范围汇总成本；旧客户端不传 `include_summary=0` 仍承担原成本。实施报告已明确。
2. **O2** 初屏必须同时请求当前页与完整汇总；不能把「分页快」误述为「首屏整体快」。
3. **O3** SQLite 合成基准排除了真实 Fill 关联；真实 MySQL 8、ARM Linux、线上索引与大表表现仍需专门验收。
4. **O4** 前端汇总缓存是组件内内存键，页面销毁/进程切换不保留；这是设计边界，不是跨设备持久真相。
5. **O5** UF-6 的 `filtered_pnl` 是行情估值展示，不是 UF-5 的 REST 已核实净收益；两者不可混用。

### 7.1 历史未决事项不因本轮改变

- CM-7 F8/P1 仍未闭环，禁止宣告 CM 完成。
- 现货扫描 F1、06-7 F1 仍待裁决。
- 生产统一 Gate 仍需：库只读预检 → MySQL 8 EXPLAIN/P95 → ARM Linux + static + schema 成套回滚 → 持久化授权 → 主网小额。

---

## 八、最终结论

```text
UF-6：❌ 不通过（F1/P1：/orders/anomalies 路由被注释掉，UF-3 功能回归）
禁止：宣告 UF-ALL 完成；禁止进入生产验收。
下一步：修复 routers/router.go:189 的字面量 \n，重跑全部门禁与探针，
        另起 rereview-2 记录 F1 闭环后再判 UF-ALL 源码准入。
```

本轮没有修改用户业务代码；只新增本报告。审计副本、探针、日志和编译产物将在报告写入与记忆同步后清理。
