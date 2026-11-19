# CODE_REVIEW_UF-6-rereview-2 · F1/P1 整改复查记录

> 日期：2026-11-19。范围：UF-6 复审 F1/P1 路由注册缺失。基线 [原始独立复审](CODE_REVIEW_UF-6-rereview.md)；实施报告 [CODE_REVIEW_UF-6.md](CODE_REVIEW_UF-6.md)。
>
> **本文件为针对原复审 Finding 的整改与复测记录，并非另一个独立审查者的审计结论。原始「不通过」报告完整保留。** 当前仅证明当前源码及离线模拟环境修复情况，不能替代 UF-ALL 独立审查或真实主网/数据库验收。

## 一、原始 Finding 和修复

独立报告明确指出 `routers/router.go:189` 因为 Go 源码中出现 **字面量 `\n\t`**，使 UF-6 的 `GET /orders/summary` 后面的 UF-3 `GET /orders/anomalies` 被包含在 `//` 行注释中，因此路由没有实际注册（404）。该问题合法通过 Go 编译、Vet、单独控制器测试，却会直接破坏 UF-3 异常订单定位功能，是 P1 阻断问题。

**修复文件：** `routers/router.go`。恢复两条**独立**的 Beego 注册调用：

```go
web.Router("/orders/summary", &controllers.OrderController{}, "get:Summary")     // UF-6 all-filter PnL once per filter
web.Router("/orders/anomalies", &controllers.OrderController{}, "get:Anomalies") // UF-3 anomalous managed orders, cycles and Agent proposals
```

**新增永久测试：** `routers/router_uf6_rereview_test.go`，`TestUF6RereviewAllOrderRoutesRegisteredAndProtected`。这不是调用控制器的单测，而是通过实际 `web.BeeApp.Handlers.ServeHTTP` 访问路由表；不需要登录凭证时的预期响应是 401、而不存在的路径是 404。原 F1 的 404 会使测试失败。测试中无数据库访问、不会调用 Binance、没有创建/撤销任何订单。

## 二、实证：实际路由解析和身份校验

| 未授权 GET | 实际 HTTP | 预期 | 结论 |
| --- | ---: | ---: | --- |
| `/uf6-not-a-real-route-404` | 404 | 404 | 哨兵有效 |
| `/orders/anomalies`（UF-3） | 401 | 401 | **PASS，原缺失路由已恢复** |
| `/orders/performance`（UF-5） | 401 | 401 | PASS |
| `/orders/summary`（UF-6） | 401 | 401 | PASS |
| `/orders`（旧接口） | 401 | 401 | PASS |
| `/orders/123/details`（子订单/Fill） | 401 | 401 | PASS |
| `/futures/entry-diagnostics`（UF-4） | 401 | 401 | PASS |

解释：401 表示该 GET 路由已由 Beego 解析，随后由控制器的鉴权检查拒绝没有有效 `user` 的请求；本测试不模拟已授权用户访问数据库。它同时覆盖 UF-3/4/5/6 的跨阶段路由回归，防止再次出现语法合法而注册丢失的问题。

## 三、复测门禁

| Gate | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestUF6RereviewAllOrderRoutesRegisteredAndProtected$' ./routers` | **PASS** |
| `go test -count=5 -p=2 -run '^TestUF6' ./service/orderhistory ./controllers ./routers` | **PASS** |
| 7 个关键包 `go test -count=5 -p=2` | **PASS** |
| 7 个关键包 `go test -race -count=2 -p=2` | **PASS**，macOS `LC_DYSYMTAB` linker warning 非致命 |
| 88 个非历史研究 Go 包 `go test -count=1 -p=2` | **PASS** |
| 88 包 `go vet` | **PASS** |
| 独立 `go build -o /tmp/go_binance_futures_uf6_f1_rereview .` | **PASS** |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| `dist` 对后端 `static` 文件比较 | **PASS** |
| `gofmt`、Prettier、两仓 `git diff --check` | **PASS** |

## 四、整改范围和限制

- **后端业务仅改动** `routers/router.go` 注册分隔；新增 `routers/router_uf6_rereview_test.go` 测试；不改 `Store.List`、`Store.Summary`、`orders` Controller 逻辑或 UF-3 的 `Anomalies` 实现。
- 前端逻辑无需改动；原 UF-3 异常列表组件仍沿用 `GET /orders/anomalies`。
- 未连接生产 MySQL、未部署 ARM、未启用或提交 Binance 下单，不修改 `conf/app.conf`、不运行 `sync db`，不清理旧订单，不提交 Git。
- 原 UF-6 的 SQLite 100/1000/10000 基准和结果一致性证据继续有效，但这次改动**不是新性能优化**，不声称真实 MySQL 查询速度改善或生产已完成验收。
- 后续 UF-ALL 独立复审仍需覆盖交易安全边界、全部六个 UF 阶段的联动与真实部署前置条件。

## 五、复查判定

**F1/P1 已在当前工作区修复，HTTP 路由注册永久测试及全部源码/离线工程 Gate 复测 PASS。可视为 F1 整改闭环，并可进入 UF-ALL 独立整体复审；这不是 UF-ALL 已通过或生产环境已获准发布。原独立复审报告原样保留。** 用户未授权实盘操作或发布，因此不执行相关生产动作。
