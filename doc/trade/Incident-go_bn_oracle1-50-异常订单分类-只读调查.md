# go_bn_oracle1 「50 条异常订单」只读调查与本地分类修复

> 调查日期：2026-11-19。服务器数据库：`go_bn_oracle1`。**数据库仅使用 `SELECT`、`information_schema` 读取及 `sql.TxOptions{ReadOnly:true}` 的只读事务；所有事务结束回滚。未执行任何 INSERT/UPDATE/DELETE/DDL、迁移、结算回填、重启或实际交易。**
>
> 本报告记录服务器现状；本地前后端变更并不等于服务器已经部署，也不能据此宣称 Binance 账户余额/资金费核实完成。

## 一、观察数据

| 只读指标 | 数量 |
| --- | ---: |
| `futures_managed_positions` 持仓周期 | 56 |
| `main/auto_strategy` active/open | 6 |
| `main/auto_strategy` closed/incomplete | **50** |
| closed/incomplete 结算处理时间已填写 | 0 |
| `futures_managed_orders` 总数 | 127 |
| OPEN filled / CLOSE filled / OPEN canceled | 56 / 50 / 21 |
| 已成交 Managed 订单的 Fill 累计数量不匹配 | **0/106** |
| `futures_trade_fills` 总数 | 158 |
| 未经 REST 核实 Fill | **0** |
| closed/incomplete 中具备开仓单 + 平仓单 | **50/50** |
| closed/incomplete OPEN/CLOSE Fill 总数量完全平衡 | **50/50** |
| closed/incomplete 发生非零 BNB 手续费 | **50/50** |
| 状态为 `reconcile_required` 或未终结的 Managed 订单 | 0 |
| `reconcile_required` Managed 持仓 | 0 |
| Agent `execution_uncertain/protection_failed` 提案 | 0 |

费用资产聚合：

- 已记录的 CLOSE BNB 手续费 Fill 85 笔，OPEN BNB 手续费 Fill 62 笔；金额记录在 **BNB**，不可直接与 USDT 数值相减。
- 另有 11 笔 OPEN 的 `commission_asset=USDT`，手续费金额均为零。
- 全部本轮显示的真实 Fill 的 `pnl_asset` 为空；`settlement_processed_at=0`。数据字段和当前结算状态一致，不代表无真实实现盈亏。
- 50 个已关闭周期都带有 BNB 手续费；不排除额外的财务核验证据缺失，因此**不将 BNB 的存在等同于结算唯一失败原因**。

## 二、直接原因

当前 UF-3 异常列表的 SQL 原来包括：

```sql
WHERE p.status='reconcile_required'
   OR (p.status='closed' AND p.settlement_status='incomplete')
```

它把「账本已平仓但**财务未核实**」无差别计入「**订单执行异常**」，所以页面报告 50 条。

结算代码 `service/futuresownership/settlement.go::verifiedSettlementFacts` 会拒绝将非结算资产的 BNB 手续费当作 USDT/USDC 扣费：必须有可信的交易时刻 BNB 汇率与资产折算证据才可以认证净收益；这是一项应当保留的安全约束。结算未通过时 `SettlementStatus` 仍为 `incomplete`，不会回填 `pnl_asset`，也不会使用未经核实的净收益更新连续亏损风控计数。

**结论：50 条是财务结算待认证周期，不是发现 50 笔开仓/平仓执行失败。** 本地真实成交账本数量匹配与 REST Fill 核实不等价于独立的币安主网再次对账。

## 三、本地代码修改（只调整展示归类，不改任何结算状态）

| 位置 | 变更 |
| --- | --- |
| `service/orderhistory/anomalies_uf3.go` | 新增可选 `ExcludeSettlementPending`，将已关闭但 `incomplete` 的周期从「默认执行/对账异常」列表中单独区分；独立计数 `settlement_pending_total` 和其中拥有非零 BNB 手续费的 `with_bnb_commission_total`，均基于 SQL 只读事实；明确分类不是已结算认证 |
| `controllers/orders.go` | 新增可选 GET 参数 `exclude_settlement_pending=true`；非法值 HTTP 400；默认不传参数保持 UF-3 原 API 语义，不破坏已有客户端 |
| `src/api/order.ts` | 补充可选参数和财务待核实数字类型 |
| `src/views/order/AnomalyPanel.vue` | 新「执行/对账异常」默认列表剔除 **仅作为账面状态的已平仓待结算周期**；明确展示财务待核实数量、含 BNB 手续费数量，单击「查看财务待核实周期」仍可看到所有原始记录和关联订单/Fill |
| `locales/zh-CN.yaml`、`en.yaml` | 明确区分交易执行异常与财务结算待核实，并警示不能直接计算已核实净收益 |
| `service/orderhistory/anomalies_oracle1_readonly_test.go`、`controllers/orders_oracle1_anomaly_readonly_test.go` | 本地隔离 SQLite：默认 API 向后兼容、执行异常未丢失、财务待结算不丢失、BNB 独立计数、账户/Owner/方向隔离、只读查询不回写状态或 Fill、非法参数拒绝 |

**重要：** 即使某个已关闭周期除 BNB 手续费外还有其他严重的结算证据异常，也继续留在「财务待核实」的显式分类和计数中。此次修复只是消除**订单执行异常数**的语义误导，不是忽略这些未结算状态；未进行自动 FX 折算、历史账本修复、风险判定放宽或迁移。

## 四、可选后续工作（本次没有实施）

如果未来需要这 50 个周期在 UF-5「实盘策略效果（已核实）」中产生精确 USDT 净收益，须另外确定可信的 **BNB 历史手续费折算方法及价格来源（对应每笔成交的时间）**，并在严格回归后讨论结算补算。不能用当前 BNB 现价、假设 1:1 汇率、手工改 `settlement_status` 或强制将其归类为 `settled`。

服务器仍保留原有 50 个 `closed/incomplete` 周期。只有经用户单独授权部署本地改动之后，线上 UI 才会看到新的「执行/对账异常」与「财务待核实」分离展示。

## 五、离线验收

| 验证 | 结果 |
| --- | --- |
| 服务器 MySQL 只读事务：新计数 SQL | **PASS：50 待结算、50 含 BNB 手续费** |
| 本地 SQLite：默认/显式分类、Main/Lead/Owner/Side 隔离、BNB 计数与无写入检查 | **PASS** |
| 本地 Go 五个核心包 `go test -count=5 -p=2` | **PASS** |
| 本地 Go 五个核心包 `go test -race -count=2 -p=2` | **PASS**（macOS 链接器有非致命告警） |
| `go vet`、独立 `go build` | **PASS** |
| 前端 TypeScript / Prettier / `pnpm build` | **PASS** |
| 本地前端 `dist` 与后端 `static` 一致性 | **PASS** |
| 两仓 `git diff --check`、`conf/app.conf` 无修改 | **PASS** |

针对账户隔离及交易执行逻辑的源码回归**只使用本地 SQLite 隔离数据库**，没有在服务器运行测试、迁移、对账或改写；服务器侧仅执行有只读事务保护的查询。服务器数据库原始 `incomplete` 状态保持不变。
