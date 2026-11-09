# Code Review 复审：方案 01「合约订单审计与部分成交」01-0 ～ 01-3（修复后）

- 复审时间：2026-11-08
- 依据：首轮报告 `CODE_REVIEW_01-0_01-3.md`、整改记录 `CODE_REVIEW_01-0_01-3_修复记录.md`、方案文档第七～十节
- 审计对象：`feat/trade-upgrade`（HEAD 仍为 `49ba0e5`）**全部工作区未提交改动**
- 方式：`git archive HEAD` + 拷贝全部未提交文件建隔离副本 `/tmp/phase_audit_r2`，实跑 Gate 并写 2 个针对性探针；**未改业务代码、未 commit/push、未对真实数据库执行 `sync db`、未请求真实 Binance**
- 本报告不修改首轮报告的历史离线结论

---

## 一、结论先行

| 维度 | 结论 |
| --- | --- |
| 首轮 F1～F13 修复 | **10 项完全修复，3 项部分修复**（F2 / F4 / F11）；无一项被误标为已修而实际未动 |
| 新增永久测试 | **8 个 `TestStageReview*`** 评审响应型测试，断言扎实、覆盖主要修复点 |
| 离线 Gate | ✅ 全绿（编译 / 测试 / race / vet / diff-check） |
| 新增 v23 schema | ✅ 迁移函数本身**经探针实证正确**（幂等、legacy 回填、共享 DDL 17 列） |
| 实盘 Gate / T0 | ❌ **仍未通过、未授权**；F1 真实成交证据、保护单重挂、生产 schema 升级均未完成 |
| 是否建议合并 | **建议合并**，但 **R2-N1 / N2 / N3 应在部署前处理**，其中 N2、N3 是实盘可用性的硬约束 |

> ⚠️ 复审发现修复本身质量较高，但**引入了 2 个新的可用性缺口**（N2 永久 incomplete 无恢复路径、N3 撤销保护单后无通用重挂），并有 **1 处验收矩阵声明与实测不符**（N1）。三者均不阻塞离线合并，但阻塞实盘授权。

---

## 二、差异对照（首轮基线 → 本轮）

| 项 | 首轮 | 本轮 | 变化 |
| --- | --- | --- | --- |
| 已跟踪改动文件 | 20 个，+850/−63 | 24 个，+1226/−72 | +4 文件、+376 行 |
| 新增文件/目录 | 7 个 | 11 个 | 新增 `command/trade_settlement_migration.go`、`service/tradefills/schema.go`、`service/internal/testtrades/`、`service/futuresownership/execution_missing_fill_test.go` |
| DB Schema | v22 | **v23** | `appversion/version.go` 22→23 |
| 关键增量 | — | `execution.go` +140→+227、`reconcile.go` +64→+140、`reconcile_test.go` 0→+143、`db_update_test.go` +76→+90 | 修复集中在这四处 |
| 未变模块（沿用首轮结论） | — | `feature/feature.go`、`models/futures_managed.go`、`models/tableStruct.go`、`manual_claim.go`、`account_client.go`、`index.go`、`user_data_mirror.go`、`service.go` | 逐段核对无语义漂移 |

---

## 三、F1～F13 修复对照表

| 编号 | 首轮问题 | 复审结论 | 代码 / 测试证据 |
| --- | --- | --- | --- |
| **F1** P1 | WS `n` 口径未验证、可能永久 reconcile_required | ✅ **代码防护完成；实盘 Gate 仍未完成** | 新增 `rest_verified`（v23）；WS 为 provisional，`FromREST` 为权威，REST 覆盖 `commission/commission_asset/realized_pnl/trade_time` 并置 1；迟到 WS 不回退；数量/价格冲突仍强制失败；差异打「报警」Warning。`ledger.go:246-290`、`TestStageReviewRESTAuthoritativeAndMissingWSCommission` |
| **F2** P1/P2 | 风控只在完整结算触发、缺口即静默 | ⚠️ **部分修复** | 新增 `ReconcileSummary.IncompleteAged` / `OldestIncompleteAgeSec` 与 15 分钟限频「报警」日志、超 10000 积压仍 fail-closed；`TestStageReviewOldIncompleteSettlementSurfacesAlarmCounters`。但**仍无主动推送/人工兜底开关**，且新 `rest_verified` 门禁使 incomplete 更易发生（见 N2） |
| **F3** P2 | broker 未实现 `OrderTradeReader` 静默跳过 | ✅ **已修** | `reconcileTradeFills` 改为返回显式错误并保留 `reconcile_required`；`var _ OrderTradeReader = BinanceOrderBroker{}` 编译期断言；所有测试 broker 显式提供模拟 REST；`TestStageReviewNoTradeReaderMustFailClosed` |
| **F4** P2 | 已挂保护单不按剩余量缩减 | ⚠️ **已修但引入新敞口** | `cancelExcessProtections` 按 `requested_qty - filled_qty > managed_qty` 调真实 `Broker.Cancel`，确认成功才置 CANCELED，失败可重试；两个永久测试覆盖撤销成功/失败重试。但**除 `notice_auto_order` 外无通用自动重挂**（见 N3） |
| **F5** P2 | v22 无显式迁移、MySQL 补列未实证 | ✅ **已修** | `command/trade_settlement_migration.go`：v22 五列显式幂等补列 + 旧行回填 `legacy`/0，v23 `rest_verified` 补列；`db_update.go` 三个分支齐全。**探针实证**两次执行幂等、legacy 回填 `'legacy'`、`settlement_processed_at=0`、config 计数 0、共享 DDL 17 列、v23 列已存在时正确跳过 |
| **F6** P2 | 6 份 DDL 硬编码、无一致性断言 | ✅ **已修** | 唯一 `tradefills.SchemaDDL`；生产迁移 + 5 个 SQLite 夹具共 6 处引用，静态检索确认无残留硬编码 DDL |
| **F7** P2 | 零手续费/缺 `N` 中断整个 WS 持久化 | ✅ **已修** | `FromWS` 把空 commission 归一为 `"0"`、允许空 `commission_asset`；只有 `RESTVerified` 才强制要求资产；`TestStageReviewRESTAuthoritativeAndMissingWSCommission` 首段实证 |
| **F8** P3 | `decimal()` 未限制整数位 ≤18 | ✅ **已修** | `ledger.go:35` 拆为 `fraction>18 || 整数有效位>18`；`TestStageReviewDECIMAL3618RejectsNineteenIntegerDigits` 含最大合法值回归 |
| **F9** P3 | 结算 Fill 查询未过滤 account_id | ✅ **已修** | `verifiedSettlementFacts` SQL 加 `AND account_id=?` |
| **F10** P3 | marginAsset 用 TTL 缓存、口径不统一 | ✅ **取舍已文档化** | 保留低频缓存（合约静态元数据），缓存不可用 fail-closed 保持 incomplete；与新鲜 REST 仓位快照用途不同 |
| **F11** P3 | 空快照被当作可信零仓位 | ⚠️ **部分修复** | 新增 `uncertainEmpty` 阻止「完全空快照 + 未解决订单」据此关闭/结算；`TestStageReviewEmptySnapshotUnresolvedOrderCannotProveFlat`。**部分缺失快照未覆盖**（见 N4） |
| **F12** P3 | Lead read-only observer 与 Fill 写接口潜在冲突 | ✅ **已修（守卫）** | `TestStageReviewLeadReadOnlyStreamCannotPersistTradeFills` 扫 `stream.go` 禁止 `persistFuturesAccountEvent(` / `tradefills.Insert(` / `orm.NewOrm(` / `.Exec(`；守卫强度见 N9 |
| **F13** 阶段边界 | 旧 `order` 表仍写入 | ✅ **保持不变** | `insertOpenOrder/insertCloseOrder`、`UpdateOrderStatus`、旧控制器/页面未动，符合 01-4/01-5 顺序约束 |

> 备注：修复记录第 13 行称「`TestSyncDatabaseInitializesAndIsIdempotent` 验证 v22 迁移」，**实测不成立**（见 N1）；第 28 行称 `go_bn_test` MySQL 2026-11-08 11:09 完成 v22→v23，本轮**未复跑用户数据库**，按用户自述记录，不作为本复审实证。

---

## 四、实证过的隐式契约（本轮新增探针，副本内执行）

| 探针 | 方法 | 结果 |
| --- | --- | --- |
| v22/v23 迁移 | 预建「pre-v21」表（无 settlement 列、无 ledger）→ 插历史 closed 行 → 连跑 `ensureV22SettlementColumns` / `ensureV23RESTVerificationColumn` 各 2 次 → 查回填值与共享 DDL | ✅ **PASS**：两次执行无错；历史行 `settlement_status='legacy'`、`settlement_processed_at=0`；config 计数 0；`SchemaDDL` 含 `rest_verified`，重建后 17 列；v23 在列存在时正确跳过 |
| 已成交订单后的 Fill 恢复 | 构造完整 closed 周期（Fill 全 `rest_verified=1`）→ 把一笔改回 0 并老化 16 分钟 → 跑 `ReconcileOwner` | ⚠️ **确认缺口**：`ACTIVE_ORDERS_AFTER_FILL=0`；`AFTER_RECONCILE: status=incomplete, processed_at=0, unresolved=1, incomplete_aged=1, rest_verified=0` → **NO_AUTO_RECOVERY**（详见 N2） |
| 生产依赖闭包 | `go list -deps . \| grep -c` | `tradefills=1`、`futuresownership=1`（真实接线）；`service/internal/testtrades=0`（测试辅助未进生产二进制）✅ |

> 探针随副本删除，建议将两条关键断言转为永久测试。

---

## 五、本轮新发现（R2）

| 编号 | 级别 | 发现 | 建议 |
| --- | --- | --- | --- |
| **R2-N1** | **P2** | **v21/v22/v23 迁移分支无任何自动测试覆盖**。`command` 包仅 3 个测试，`db_update_test.go` 里 `SyncDatabase(` 调用最高为 **19**；全包 `go test -v` 日志无 v20+ 跃迁。即 v21 建表、v22 五列、v23 `rest_verified` 三个分支**从未被 `SyncDatabase` 路径执行过**；整改记录对此的验证声明不成立。我的探针只证明迁移函数本身正确，**不等于有回归保护** | 在 `db_update_test.go` 补 `SyncDatabase(20..23)` 序列与幂等断言，并断言 `rest_verified` 列存在、旧行回填 `'legacy'`；把「版本测试上限 = `appversion.DatabaseSchemaVersion`」固化为断言，避免每次升版漏测 |
| **R2-N2** | **P2** | **`filled` 订单之后到达/丢失 REST 确认的 Fill 无自动补确认路径**。`liveOrderStatus` 只含 pending/submitted/partially_filled/reconcile_required；`ActiveOrders` 不含 filled → 不再进入 `Executor.Reconcile` → `reconcileTradeFills` 不会重跑 → `rest_verified=0` 的 Fill 永远无法确认 → `verifiedSettlementFacts` 以 "trade fee and realized PnL not confirmed by Binance REST" 拒绝 → **周期永久 incomplete，只剩 F2 告警**。探针已实证全套链路 | 在 `ReconcileOwner` 的 `unsettled` 循环里，对 reason 为「未 REST 确认」的周期按 `position_id` 主动补一次 `userTrades` 确认（或把该周期关联订单临时置回 `reconcile_required` 以复用现有对账）；补永久测试 |
| **R2-N3** | **P2** | **F4 修复把「超量挂单」风险转成了「撤销后无止损保护」敞口**。`cancelExcessProtections` 撤销超量 TP/SL 后仅打「报警」日志要求 owner 重挂；除 `notice_auto_order` 有既有修复逻辑外，无通用自动重挂。两个永久测试只断言「已撤销 / 失败可重试」，**未断言新保护单存在**。整改记录第 34 行已自认该 Gate 未关闭 | 实盘授权前逐 Owner 审计保护单恢复路径；至少为所有 Owner 提供「按剩余量重挂」或明确的人工接管 SOP 与告警升级；补一条「撤销后仍有有效保护」的回归测试 |
| **R2-N4** | P3 | F11 的 `uncertainEmpty := len(accountPositions)==0 && summary.OrdersUnresolved>0` 只覆盖**完全空快照**，且依赖「活动订单对账失败」这一间接信号；**部分缺失快照**（REST 返回部分币种）不会被捕获，仍可能把缺失 key 当零仓位 | 增加「快照覆盖度」校验（如本次快照 symbol 集合与 managed/unsettled 集合比对），而非仅判断空/非空 |
| **R2-N5** | P3 | `FromREST` 对空 `CommissionAsset` 直接 fail-closed（`validate` 要求 `RESTVerified && asset 非空`），与 F7 对 WS 的宽容**不对称**。若官方 `userTrades` 对零手续费成交也返回空 asset，REST 确认会失败并落入 N2 的永久 incomplete | 对 REST 空 asset 按合约结算资产补齐或显式标记 `fee_unknown` 并保持 incomplete，但不要阻断确认流程；补测试 |
| **R2-N6** | P3 | `incompleteAlertAt` 是进程级全局 map，**无清理/无容量上限**；每个历史周期 ID 长期驻留，长期运行缓慢增长 | 结算成功后删除对应 key，或改用带 TTL/LRU 的限频结构 |
| **R2-N7** | P3 | `service/internal/testtrades` 假 REST 恒返回 `Commission="0"` / `RealizedPnl="0"`，使「REST 权威覆盖费用/PnL」链在多数集成测试里**无法覆盖非零费用差异**；全局 `history` map 无重置接口，存在跨测试污染风险。已确认不在生产依赖闭包（计数 0） | 提供 `Reset()` 与可注入的费用/PnL 参数，让 01-3 净收益回归走真实数值；把该包标注为 test-only（如移入 `internal/testdata` 或加 build tag） |
| **R2-N8** | P3 | v22 显式迁移只 `ADD COLUMN`，**未创建** `settlement_status` / `settlement_processed_at` 索引；仍依赖 `RunSyncdb` 在 MySQL 上补索引（项目既有事实：MySQL 补列/补索引未验证）。`UnsettledClosedPositions` 按 `settlement_status` 过滤，历史量大时可能退化 | 在显式迁移里补索引存在性检查与创建；MySQL 8 授权验证时一并核对 `SHOW INDEX` |
| **R2-N9** | P3 | F12 守卫是**字符串黑名单**（扫 `stream.go` 的 4 个危险串），可被重命名、接口分派、跨包 helper 绕过；按审计方法论，白名单/依赖闭包断言更强 | 增加「`service/leadaccount` 只读宿主不依赖 `tradefills`」的 `go list -deps` 断言，或改为允许符号白名单 + 反例自测 |

---

## 六、独立复跑的 Gate 记录

| 项目 | 命令 | 结果 |
| --- | --- | --- |
| 全仓编译 | `go build ./...`（副本） | `EXIT=0` |
| 主程序构建 | `go build -o /tmp/r2_main_bin .` | `EXIT=0` |
| 目标包测试 | `go test -count=1` 9 包 | **全 ok** |
| Race | `go test -count=1 -race` 3 包 | **全 ok**；仅 macOS `ld: malformed LC_DYSYMTAB` 噪音 |
| Vet | `go vet` 9 包 | `EXIT=0` |
| 空白错误 | `git diff --check`（用户仓库） | `EXIT=0` |
| 迁移覆盖审计 | `go test -count=1 -v ./command` | PASS，但**最高版本 19**（见 N1） |
| 探针 | 2 项（command / futuresownership，`-run '^TestProbeR2'` 单独执行） | 均 PASS，其中 1 项确认缺口 |
| 永久测试 | 首轮 18 个 01 阶段测试 + 本轮 **8 个 `TestStageReview*`** | 全部随目标包测试通过 |

> 环境说明：副本由 `git archive HEAD` 生成，不含未跟踪的 `strategy_templates/research/**`，因此全仓编译为绿；`controllers` 既有 ORM 重复注册/模板测试问题不在本轮范围。

---

## 七、实盘 Gate：仍未关闭（不得上线 / 启用 T0）

1. **F1 真实部分成交证据仍缺**：需人工授权抓一笔真实多次部分成交的 WS `l/L/n/N/rp/T` 与 REST `userTrades` 逐 `tradeId` 比对，确认新账本与官方一致。代码防护不能替代实盘证据。
2. **N2 / N3 未解决**：REST 确认的自动恢复路径、保护单撤销后的重挂，二者直接决定实盘能否正常结算与是否有止损保护。
3. **N1 迁移无回归保护**：生产 schema 需从 v22 升到 **v23**（新增 `rest_verified`）；整改记录称仅在 `go_bn_test` 完成 v22→v23，本轮未复跑、生产 `go_binance` 未动。
4. **F2 仅为应用内 Warning + `ReconcileSummary` 字段**，无主动推送、无人工覆盖开关；`incomplete` 超 15 分钟需人工检查账户绑定 REST、真实 Fill、手续费资产、Algo 映射与持仓快照，不得手工改 `settlement_status=settled`。
5. **01-4 / 01-5 未实施，T0 未启用**：旧 `order` 读写仍在，前端 `incomplete/settled` 不可见，新主子表 API 未实现。
6. 生产 schema 变更、二进制替换、重启与实盘验证按方案 §四 01-6 均需**单独授权**；生产库 v22 结构与 5 条未归属非零镜像仓位须上线前独立只读复核。

**离线通过 ≠ 实盘可用。** 本复审的通过结论仅覆盖本地代码、SQLite 隔离测试、静态检索与探针证据。

---

## 八、审计纪律声明

- 未修改用户任何业务代码、未 commit/push、未改 `conf/app.conf`、未触碰 `strategy_templates/` 与 `static/`。
- 副本 `/tmp/phase_audit_r2` 中的探针在报告完成后随副本删除；用户仓库仅新增本复审报告。
- 副本 `conf/app.conf` 指向本地测试库，因此**全程未执行 `sync db`**、未连接真实 Binance、未执行任何生产写操作。
- 对用户自述的 MySQL v22→v23 升级与生产只读核查，按「未独立复跑」标注，未当作本轮实证。

---

## 九、复审整改后状态（2026-11-08 18:15，北京时间）

本报告为 R2 修复前审计快照，原始结论保持不变。R2-N1～N9 的实际整改、回归测试、数据库确认与生产剩余 Gate 记录于 [CODE_REVIEW_01-0_01-3-rereview_整改记录.md](CODE_REVIEW_01-0_01-3-rereview_整改记录.md)。特别注意：已完成订单的 REST Fill 自动补查现已修复；agent_trade 补齐了 Owner 级保护单重挂，但其他 Owner 的真实保护管理和真实交易所样本仍需独立验收。
