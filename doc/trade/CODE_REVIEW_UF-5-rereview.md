# CODE_REVIEW_UF-5-rereview · 已核实 Main 实盘策略效果统计复审

> 复审日期：2026-11-19。后端基线：`master` / `d2c2f28 fix: update trade`。前端基线：`dff4af2090a114973f821c31352278afd8485440`。
>
> **判定：✅ UF-5 源码 / 离线 SQLite / 前后端构建 Gate 通过。**
>
> **允许：进入 UF-6「订单历史查询性能」代码开发。**
>
> **禁止：据此宣告真实 MySQL 8、真实 Binance REST 私有账户、Testnet/Demo/Mainnet、ARM Linux、生产发布或持久化授权 Gate 已通过。**

---

## 一、复审范围与方法

### 1. 已阅读材料

- `doc/trade/开发方案-UF-U本位合约交易六项精简优化.md`
- `doc/trade/CODE_REVIEW_UF-5.md`
- `doc/readme/zh-CN/02-合约交易与订单.md`
- `doc/readme/en/02-futures-and-orders.md`
- UF-3 / UF-4 历轮复审报告及项目长期 Gate 记录

### 1.1 逐文件审查范围

| 范围 | 文件 |
| --- | --- |
| 统计核心 | `service/orderhistory/performance_uf5.go` |
| 永久测试 | `service/orderhistory/performance_uf5_test.go`、`controllers/orders_performance_uf5_test.go` |
| 接口与路由 | `controllers/orders.go::Performance`、`routers/router.go` |
| 证据链复用 | `service/orderhistory/history.go::summarize / fillRow / dec / val / activatedAt / closingIntent` |
| 前端源码 | `src/api/order.ts`、`src/views/order/PerformancePanel.vue`、`src/views/order/index.vue`、`locales/zh-CN.yaml`、`locales/en.yaml` |
| 静态产物 | 后端 `static/index.html`、`static/static/js/*`（含 `PerformancePanel-*`、订单页入口 chunk） |

### 1.2 审查纪律

- 未修改用户业务代码、配置、`strategy_templates/`、`static/` 源码；未执行 `sync db`、未连接 Binance、未下单、未重启服务、未触碰仓位。
- 未执行任何 Git 提交、推送、切换、检出或清理工作区操作。
- 所有临时探针只写在 `/tmp/phase_audit_uf5`，复评结束后清理。
- 审计期间显式清空 `BINANCE*` / `COIN*` / `SPOT*` / `FAPI*` / `DAPI*` 等环境变量，避免误触外网或真实账户。

---

## 二、基线与隔离副本等价性

### 2.1 当前基线

```text
HEAD d2c2f2896fbd56f249b6c3e0ec3818a550f8f0c9
工作区：83 D + 6 M + 90 ?? = 179 项
```

该工作区同时包含 UF-3 / UF-4 遗留的静态资源重建结果，以及 UF-5 新增后端、前端、文档和静态产物；本轮仅判定 UF-5 相关部分，不把历史阶段的静态重建误记为 UF-5 业务改动。

### 2.2 等价校验

| 副本 | 校验结果 |
| --- | --- |
| `/tmp/phase_audit_uf5` | `paths=1287 checked=1200 missing=0 diff=0 deleted_confirmed=83` |
| `/tmp/phase_audit_uf5_ui` | `paths=300 checked=298 missing=0 diff=0 deleted_confirmed=0` |

说明：校验基于 `git ls-files` 清单和 `git status --porcelain -z` 的 M/A/??/D 记录，逐文件 SHA256 对比；未使用全仓 `os.walk`，避免历史 SIGTERM(137)。`conf/app.conf` 仅在本地存在时复制到副本供离线测试使用，不进入报告正文，也未输出其中任何配置值。

---

## 三、静态审查结论

### 3.1 未发现阻断性 Finding

本轮未发现 P0 / P1 / P2 Finding。UF-5 的实现与自评声明的边界一致：只读、Main 限定、证据严格、资产隔离、指标定义明确、前端不自动轮询。

### 3.2 关键不变量核对

| 验收点 | 实现证据 | 结论 |
| --- | --- | --- |
| 账户与 Owner 固定 | `Performance` 查询固定 `Filter("account_id","main").Filter("owner","auto_strategy")`；响应 DTO 固定 `AccountID/Owner`；API 不接收 `account_id/owner` | 通过 |
| 只统计已结算真实周期 | 查询窗口使用 `closed_at`；复检 `status=closed`、`settlement_status=settled`、`managed_qty≈0`、`ClosedAt>=CreatedAt`、`SettlementProcessedAt>=ClosedAt && <=end` | 通过 |
| 不依赖历史展示覆盖 | 不调用 `List`/`project`；`summarize(..., includeFills=false, ...)`；不读取 `history_status_override` / `history_completeness_override` / `symbols.close` | 通过 |
| REST 成交严格核实 | 复用 `summarize` 的身份、订单、Fill、数量、手续费、PnL 资产、T0 和结算符号校验；额外 `performanceExactFlat` 要求 OPEN/CLOSE Fill 数量精确相等 | 通过 |
| 部分 Fill 聚合 | 多次 OPEN 与多次 CLOSE Fill 均按周期汇总；`fullCycle` 与 105 周期分批测试均有永久覆盖 | 通过 |
| 资产隔离 | `PnLAsset` 仅允许 `USDT` / `USDC`，分资产计算；手续费资产与 PnL 资产不一致即排除，无假设汇率 | 通过 |
| 指标口径 | `big.Rat` 精确累加；零收益单独计数；无亏损样本时 PF 为 `null` + `profit_factor_note=no_loss_samples`；回撤按 `(closed_at,id)` 排序的累计已结算净收益峰谷金额差 | 通过 |
| 策略规则版本 | 仅识别严格 `auto_strategy:<symbol>:<64位小写 hex>`；其余归为 `unknown`；DTO 只返回 Hash，不返回 `source_ref` | 通过 |
| 日期与 T0 | 7/30 天按 UTC `closed_at >= start && <= end`；`BINANCE_ORDER_HISTORY_T0_MAIN_MS` 未配置或周期早于 T0 均排除；解析失败 fail closed | 通过 |
| API 安全 | 路由仅 `get:Performance`；JWT `authorized()`；`days/symbol/position_side/rule_hash` 白名单；非法输入 400，DB 错误 503 且错误文本泛化 | 通过 |
| 只读性 | Store 层只执行 `Count/All/Raw QueryRows`；不使用任何 Update/Insert/Delete；探针用 SQLite 触发器实证零写入 | 通过 |
| 前端边界 | 面板默认收起；仅展开后主动查询；`requestNumber` 防止旧响应覆盖；无自动轮询；无下单或状态修复入口 | 通过 |

### 3.3 与实施报告的声明核对

| 实施报告声明 | 复审核对 |
| --- | --- |
| 「只统计 Main / `auto_strategy`、已结算、REST Fill 已核实」 | 静态代码与探针一致 |
| 「USDT/USDC 分资产、不换算」 | 静态代码与永久测试、探针一致 |
| 「PF 无亏损样本返回 null」 | DTO 为 `*string`，无负样本时保持 nil，探针与永久测试覆盖 |
| 「累计已结算净收益回撤不是账户权益回撤」 | 字段名、README、i18n、UI tooltip 四处均明确披露 |
| 「人工历史覆盖不改变统计」 | 实现不读取覆盖字段；永久测试用真实覆盖值实证 |
| 「105 周期分批不漏不重」 | 永久测试覆盖，离线 Gate 通过 |
| 「前端构建与 static 一致」 | 本复审重新构建并 `diff -qr` 为零差异 |

---

## 四、离线工程 Gate

| Gate | 结果 | 证据 |
| --- | --- | --- |
| 两个仓库 `git diff --check` | PASS | 无输出 |
| `gofmt -l` UF-5 相关 Go 文件 | PASS | 无输出 |
| `go list ./...` | 88 / 88 包 | 未包含研究目录（本次 go list 本身为 88） |
| `go vet` 88 包 | PASS | `/tmp/uf5_audit_vet.log` |
| `go test -count=1` 88 包 | PASS | `75 ok + 13 no test + 0 FAIL` |
| UF-5 专项 `-count=5 -run '^TestUF5'` | PASS | `./service/orderhistory ./controllers` |
| 7 个关键包 `-count=5 -p=2` | PASS | controllers / feature / feature/api/binance / orderhistory / coinmquant / coinmexecution / systemhealth |
| 同 7 包 `go test -race -count=2 -p=2` | PASS | macOS `LC_DYSYMTAB` linker warning 非致命，退出码 0 |
| `go build -o /tmp/go_binance_futures_uf5_audit .` | PASS | 无输出 |
| 前端 `pnpm typecheck` | PASS | `tsc --noEmit && vue-tsc --noEmit --skipLibCheck` |
| 前端 `pnpm build` | PASS | Vite 6.3.3，2564 modules，Browserslist 过期提示非致命 |
| `dist/` 与后端 `static/` | PASS | `diff -qr` 无输出；两侧各 87 个 JS chunk |

说明：以上均为源码、离线 SQLite、Mock 与构建证据，不等于真实 MySQL 8 查询计划、真实 Binance REST 私有账户数据或生产部署通过。

---

## 五、隔离审计探针

探针文件：`/tmp/phase_audit_uf5/service/orderhistory/audit_uf5_probe_test.go`，`-count=5` 全部 PASS。

| 探针 | 攻击点 | 实证结果 |
| --- | --- | --- |
| `TestAuditUF5ForeignFillCannotBeCountedForTargetCycle` | 把外资周期的 close Fill 改挂到目标周期，并伪造目标 `client_order_id` / `order_id` | 两个周期均不纳入；目标记 `identity_mismatch`，供体记 `missing_or_mismatched_fills` |
| `TestAuditUF5MixedPnlAssetCannotBeAveraged` | 同一周期一笔 Fill 的 `pnl_asset` 改为 USDC，其余为 USDT | 不产生 USDT/USDC 混合收益，记 `unknown_or_mixed_asset` |
| `TestAuditUF5ZeroForeignFeeStillNotSilentlyConverted` | 把一笔手续费改成非零资产 BNB 且金额为 0 | 保守排除，不把异币零手续费静默折算或忽略 |
| `TestAuditUF5ReportDoesNotWriteLedger` | 在三个账本表安装 INSERT/UPDATE/DELETE 触发器后连续生成 3 次报告 | 触发器计数为 0；报告只读 |
| `TestAuditUF5DrawdownUsesClosedAtThenIDForTies` | 三笔交易同一 `closed_at`，按 ID 顺序为 `+10/-5/-5` | 回撤为 10，而非按盈亏重排后的 5；排序按 `(closed_at,id)` |
| `TestAuditUF5RuleFilterDoesNotCountRejectedRulesAndNeverLeaksSourceRef` | 用规则 Hash 过滤两个周期，并序列化响应 | 候选/纳入均为 1、排除为 0；响应不含 `source_ref` 或原始来源串 |

---

## 六、观察项与边界

1. **O1（非阻断）** `Performance` 在分页批次之间没有跨请求事务快照。若后台交易在统计期间持续写入，可能出现不同批次间的数据时点差异；实施报告已声明「报告在请求时以数据库读取结果生成，可能与下一次刷新不同」。这不等同于账本错账，但真实并发场景仍需 MySQL 8 与生产连接下验证。
2. **O2（非阻断）** 非零资产但金额为零的手续费也会被保守排除。代码注释明确该口径（`even a nonempty, zero-valued different fee asset is not assumed convertible`），本轮探针已实证。实施报告未单独展开这个极边界，但不构成口径矛盾。
3. **O3（非阻断）** 本次 `static/` 变化是 UF-3 / UF-4 / UF-5 UI 的整体重建；复审确认 `dist → static` 完全同步，但不应把所有删除/新增 chunk 归因于 UF-5 单阶段。
4. **O4（非阻断）** 规则 Hash 过滤在取回日期候选后按精确 Hash 内存过滤，因此 `candidate_cycles` 指「通过规则过滤后的候选」；这不会造成 SQL 注入或错计排除原因，但与「先按 Hash 过滤数据库再计数」的语义不同，调用方应按当前响应字段理解。
5. **O5（Gate）** UF-5 只解决「已核实实盘策略效果」的只读展示，不替代 UF-6 的订单历史查询性能优化；旧的全量收益筛选仍含估值口径，两者不可直接比较为同一收益。

### 6.1 历史未决事项不因本轮改变

- CM-7 F8/P1（保护配置签名预检期竞态）仍未闭环，禁止宣告 CM 阶段完成。
- 现货扫描 F1、06-7 F1 仍待裁决。
- 生产统一 Gate 仍需：库只读预检 → MySQL 8 EXPLAIN/P95 → ARM Linux + static + schema 成套回滚 → 持久化授权 → 主网小额；禁止用 Testnet 或离线 PASS 跳过。

---

## 七、最终结论

```text
UF-5：✅ 源码 / 离线 SQLite / 前后端构建 Gate 通过
允许：进入 UF-6（订单历史查询性能）代码开发。
禁止：据此宣告真实 MySQL 8、真实 Binance 私有账户数据、
      Testnet/Demo/Mainnet、ARM Linux 或生产发布已完成。
```

本轮没有修改用户业务代码；只新增本报告。审计副本、探针、编译产物和日志将在报告写入与记忆同步后清理。
