# CODE_REVIEW_UF-3 · U 本位异常订单定位

> 日期：2026-11-19。后端分支 `master`；依据：[UF 六项简化优化方案](开发方案-UF-U本位合约交易六项精简优化.md)。
>
> **阶段结论：完成源码和离线 Mock/SQLite/前后端构建验收；真实 MySQL 8/ARM/生产账户尚未验收。** UF-3 是只读异常定位，不是自动对账或异常修复。原有 `CODE_REVIEW_UF-1.md`、`CODE_REVIEW_UF-2.md` 保留。

## 一、交付与文件

| 范围 | 交付 |
| --- | --- |
| `service/orderhistory/anomalies_uf3.go` | 新增独立只读异常联合查询，支持 Main/Lead 账户、Owner、币种、LONG/SHORT、类别、分页；保留异常来源、系统真实状态、Managed ID、clientOrderId、exchangeOrderId、Proposal ID、更新时间及最后对账时间 |
| `controllers/orders.go`、`routers/router.go` | JWT 鉴权的 `GET /orders/anomalies`；分页上限 100、类型/账户/Owner 严格校验；数据库失败返回泛化服务不可用，不把原始 SQL 或敏感详情给浏览器 |
| `src/api/order.ts` | UF-3 TypeScript 响应与过滤参数类型 |
| `src/views/order/AnomalyPanel.vue` | 在合约订单页新增可展开只读异常面板，支持分类筛选、分页及异常详情；关联 `position_id` 时通过既有 `/orders/:position_id/details` 展示订单和真实成交 Fill；没有周期 ID 的异常 OPEN 直接显示客户端订单 ID、交易所 ID 和原始状态 |
| `src/views/order/index.vue` | 使用现有合约订单页，无新顶级菜单；原历史列表、批量操作和收益计算路径保留不动 |
| `src/views/ai/observability.vue` | 系统看板的主要异常数字可点击定位至合约订单异常面板，并预设对应异常分类 |
| `locales/zh-CN.yaml`、`en.yaml` | 中英文异常类别、来源、对账/Fill、只读范围、详情等独立文案 |
| `service/orderhistory/anomalies_uf3_test.go`、`controllers/order_anomalies_uf3_test.go` | SQLite/Mock 回归、鉴权/参数边界/账户隔离/分页/分类/无周期 OPEN/真实 Fill 证据 |

## 二、异常分类与系统健康统计的严格口径

`service/systemhealth::checkTrade` 既有汇总保留原样：

- `execution_uncertain`、`protection_failed` 是 `agent_trade_proposals.status`，**不是**所有 Futures Managed Order 的状态。本次分别映射 `proposal_execution_uncertain`、`proposal_protection_failed`，Owner 显示为 `agent_trade`、账户为 Main。
- `reconcile_required` 是 Main Managed Position + Managed Order，且需要将符合官方 Fill 待核实条件的订单从硬错误口径中区分。本次提供组合过滤 `reconcile_required`，对单条记录显示 `managed_position_reconcile` 或 `managed_order_reconcile`。
- `fill_verification_pending` 使用与健康统计一致的主要条件：`order.status=reconcile_required`，请求量等于成交量、有 exchangeOrderId、有关联 active/closed 周期、存在 Fill 但部分 `rest_verified=0`、Fill 量总和精确匹配。**缺失或数量不符的 Fill 不降级为普通待核实，而保留为硬对账异常。**
- 为便于调查，额外显示 10 分钟以上未终结的 `pending/submitted/partially_filled` Managed 订单（`aged_managed_order`），以及 `closed+settlement_status=incomplete` 周期。这些超出健康统计原有计数范围，因此**异常列表总数不等于系统健康汇总数**。
- 分类取数据库真实 `status/settlement_status`，完全不读 `history_status_override` 或 `history_completeness_override`；人工历史展示修改无法消灭真实风险。
- 组合/明细按账号过滤，Main 查不到 Lead，Lead 查不到 Main 的 Proposal。

## 三、导航与操作边界

1. 系统看板 `执行结果待确认` → `proposal_execution_uncertain`；`保护单失败` → `proposal_protection_failed`。
2. 系统看板 `执行/归属待对账` → 组合 `reconcile_required`；`成交核验待补` → `fill_verification_pending`。
3. 进入 `/futures/list?anomalies=1&category=...&account_id=main` 后自动展开异常面板；用户也可在合约订单页自行展开，筛选 Lead、Owner、币种、方向或类别。
4. 点击某条记录进入只读异常详情：展示原始 Managed 订单/周期/Proposal 标识、意图、订单量、状态、核实时间；存在周期 ID 时读取其子订单及 Fill。没有周期 ID 的 OPEN 只显示该订单本身，**不虚构周期、不替它执行对账**。
5. 本阶段没有新增 POST、DELETE、修复、认领、撤单、真实 OPEN/CLOSE 操作，也没有执行自动 REST 对账、资金或仓位变更。异常查询故障显示明确错误，**不返回伪造空列表**。

## 四、永久测试证据

| 目标 | 测试及验证 |
| --- | --- |
| Main/Lead/Owner 隔离 | `TestUF3AnomalyAccountIsolationAndOrphanOpen`；同币种的独立 Agent Proposal 不被误当 Managed Order |
| 无 `position_id` 的 OPEN | 独立 `managed_order` 行仍被列出，ClientOrderID、Owner、Symbol、PositionSide 保留 |
| 系统真实状态优先 | 写入与异常相反的人工历史状态/完整性覆盖后，异常仍可检索 |
| 真实 REST Fill 待核实 vs 数量异常 | `TestUF3AnomalyRealRESTFillPendingMatchesHealthClassification`，更改 Fill 数量后立即回到强对账分类 |
| 只读过滤及类别 | `TestUF3AnomalyCategoryAndOwnerFiltering`，包括 Main/Lead、Owner、币种、方向及 Agent/Managed 类别 |
| 未终结订单阈值和分页 | `TestUF3AnomalyPagedStableAndStaleOnlyAfterTenMinutes`，10 分钟内仍在提交中的订单不会误报超时 |
| 请求输入安全 | `TestUF3AnomalyInvalidFiltersFailBeforeAnySQL`、`TestUF3AnomalyCanceledReadIsNotSuccessfulEmpty` |
| JWT 与控制器输入 | `TestUF3AnomalyRouteRequiresJWTAndValidFiltersBeforeDB`，未经登录返回 401，不合法过滤参数在 SQL 前返回 400 |

- `TestUF3HealthSummaryReconcileLinkCombinesPositionsAndOrders`：系统看板的组合待对账入口命中 Managed 仓位与订单，不混入 Agent Proposal / 结算不完整。

## 五、工程 Gate

| Gate | 验证结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestUF3' ./service/orderhistory ./controllers` | **PASS** |
| `go test -count=5 -p=2` 五个关键 Go 包 | **PASS** |
| 88 个非历史研究 Go 包单次测试 | **PASS** |
| 88 包 `go vet` | **PASS** |
| `go test -race -count=2 -p=2` 五个关键包 | **PASS**（macOS `LC_DYSYMTAB` linker warning 非致命） |
| `go build -o /tmp/go_binance_futures_uf3 .` | **PASS** |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS**（构建约 4.28 MB） |
| `rsync -a --delete dist/ backend/static/` + `diff -qr` | **PASS** |
| `gofmt` 与 Git diff 检查 | **PASS** |

## 六、未执行 Gate 和限制

- 未连真实 Binance FAPI、私有 WS，未发送实际 Binance 订单，不涉及生产 MySQL 8、ARM Linux 部署与服务重启；测试使用隔离 SQLite/本地 HTTP Mock。
- 系统健康里的细分 Fill 计数（活动/历史）没有增加独立导航过滤，仍会通过共同 `fill_verification_pending` 入口查看；点击不会保证页面条数与其单项数字相等。
- 浏览器显示最后对账时间仅依据本地 `last_reconciled_at`；为零时显示本地更新时间，**并非宣称刚刚核实了交易所**。
- 本功能用于快速定位和排查，并非替代真实 REST Fill 对账与资金异常处理；清理/修改历史展示仍按原有隔离权限逻辑运行，不影响实际交易。
- 本轮未操作 `conf/app.conf`、数据库迁移或 Git commit；额外已有 `CHANGELOG.md` 修改不属于 UF-3，不擅自覆盖。

## 七、阶段判定

**UF-3 的源码范围为只读异常定位、既有页面集成和中英文提示，不引入新的交易执行机制或大型监控平台。** 各项工程 Gate 已全部通过，可以进入 UF-4「为什么没有开仓」的代码开发，真实环境发布仍需单独验收。
