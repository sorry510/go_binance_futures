# 币安合约自动带单 — Stage 5 代码审计报告

- **审计对象**：`doc/trade/币安合约自动带单-Stage5-{1..6}-实施记录.md` 对应的 **Stage 5 Lead User Data WS、启动恢复、API 预算与告警**（工作区未提交）。基线 `HEAD = 8622ff2`（`feat: stage 4`，Stage 4-1～4-6 已提交）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库与 `strategy_templates/` 未被改动）
- **审查类型**：review-only
- **结论**：**Stage 5 的 6 个子阶段离线实现完成**（WS 生命周期 / 账户镜像 / 启动恢复与持久 CAS / 预算与限速 / 告警与监督器 / 并发联合回归），安全不变量（WS 仅只读投影、代际与水位防护、待对账必须持久证明才清、跨账户隔离）均**结构性成立**；**Gate 5 的真实环境验收按记录与我的复核一致：未通过**。生产仍**零接线**（生产二进制依赖闭包不含 `service/leadaccount`）。未发现 P0/P1；新增 4 项 P3（3 项沿用）。
- **审计日期**：2026-10-11

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| 5-1 WS 生命周期 | ✅ `explicit Run(ctx)/Stop()`、每次连接新 generation、ListenKey 过期/keepalive/异常触发重建、Lead 与 Main 连接管理分离、**先建 WS 再读 REST 且启动窗口内有事件即判该代不可信并重新快照**、不在导入/启动时自动连接 ✓ |
| 5-2 账户镜像 | ✅ 仅只读投影（绝不认领订单或写生产 Ownership 表）；`Seed` 需完整 REST 快照（空列表合法、nil 账户拒绝）；订单按 OrderID/时间/累计成交量单调 + **终态不可复活**；`ACCOUNT_UPDATE` 逐腿/逐资产水位校验（拒绝回滚）；`ALGO_UPDATE` 不能冒充成交证据 ✓（本轮探针 8 类拒绝全过） |
| 5-3 启动恢复/持久 CAS/权威对账 | ✅ `ReserveSubmission` 在**任何 Broker HTTP 写之前**按 `account_id + ClientOrderID` 原子 CAS（`execution.go:188` 早于提交），未抢到者只查单不重试 ✓；`OutstandingOrders` 读本账户全部未决单 + 安全行数上限（不再截断 200/500）✓；恢复器读**含 CLOSED** 的全部受控持仓 + REST 全量，同 ClientOrderID Lookup 幂等写回；未知挂单/重复/不一致即阻断 ✓；启动顺序 DB/REST 恢复审计 → 私有 WS ✓；**仅"较新且无缺口镜像 + 持久证明"才清 pending**，且不得清亏损/身份熔断或设 Stage 7 ✓ |
| 5-4 预算/WS 效率/账户限速 | ✅ **复用**既有全局 `binanceapiusage` 预算与 429/418 降级（不建第二套 IP 预算）✓；`metrics.go` 区分 WS 事件数与 REST 快照数，并把 `UnverifiedSavedRequests` 标 `json:"-"` —— **明确拒绝把 WS 消息数虚报为省下的 REST 请求** ✓；per-Key ≤18/10s（普通开仓 ≤16，≥2 预留给 CLOSE/Cancel）✓；`lead_http_fault.go` 把 429/418 直接反馈 Guard（无 Secret/签名 URL/Body），且**规避 transport 内反向取执行器锁的自锁** ✓ |
| 5-5 告警与监督器 | ✅ 有界非阻塞队列 + 单 worker（`running` 守卫）；背压**丢弃只计数、绝不改变风险阻断**；发送失败不复活开仓；≤3 次/连续 429-418、间隔 ≥2 分钟、恢复后预算重开；`Module=lead_trading` 复用既有 DingDing/Slack 通道且**仅显式 Run 才启动**；`supervisor.Run` 整合"显式恢复 → 私有 WS → Guard → HTTP 回调 → 通知队列"，ctx 取消只停本账户 ✓ |
| 5-6 联合回归与 Gate 5 口径 | ✅ 9 包 `-race`/vet/build/diff 全绿（我独立复跑 ✓）；生产依赖闭包 `service/leadaccount` = **0** ✓；记录明确"真实 WS/Testnet/双账户 2s 负载/MySQL 8 恢复/预算证据未执行 → 不能标产线通过" ✓ |
| 生产是否可能被启用 | ❌ 仍不能：无生产调用方、无启动钩子、`LeadSupervisor` 公开面仅 `{Run, Status}`（探针反射）、`RiskController` 无 Enable/Resume、`ReadOnlyOrderBroker` 写恒锁 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 子阶段逐项复核

| 子阶段 | 记录声明要点 | 我的复核 |
| --- | --- | --- |
| 5-1 | 显式生命周期、代际隔离、WS→REST 顺序与缺口判不可信 | ✅ `NewLeadUserStream(client, mirror)` + `Run/Stop/Status`；`BeginGeneration` 立即清空并置未就绪；`ApplyEvent` 要求 `ready && generation 匹配`（探针实证）；缺口规则由永久测试 `TestStage56PreSnapshotWSMessageForcesResync` 覆盖 ✓ |
| 5-2 | 只读投影、单调与水位、终态不可复活、Algo 不作证据 | ✅ 探针：未就绪/失败种子/nil 账户拒绝；成交回退、时间回退、终态→非终态复活、超量成交、`ALGO_UPDATE` 全部拒绝；终态后订单移出 open 集；代际切换后再接收旧代事件被拒；**旧代 `Invalidate` 不影响现役镜像**、现役 `Invalidate` 强制重同步 ✓ |
| 5-3 | 持久 CAS + 全量未决单 + 含 CLOSED 对账 + 清 pending 的严格前置 | ✅ 静态核对 CAS 位置（`service/futuresownership/execution.go:188`，`!reserved` 时拒绝重试）；恢复器接口与报告字段齐全；清 pending 依赖私有证明函数（外部不可直接调用）✓ |
| 5-4 | 复用全局预算、不虚报节省、per-Key 槽位预留、429/418 回调 | ✅ `LeadUsageMetrics` 字段与注释明确"不把观测到的 WS 事件当作省下的 REST 请求"；`lead_order_limiter.go` 的 16+2 策略与 Stage 1 老测试同步更新（记录 §汇总 已说明）✓ |
| 5-5 | 有界队列、非阻塞、丢弃不改阻断、限频、显式通道 | ✅ 探针：容量 2 时第 3 次 publish 即背压且 `Dropped()==1`；Main 账户/未知故障码被拒不占容量不计数；**worker 卡死时 Publish 仍立即返回**；发送失败不影响 worker 与状态；ctx 取消使 worker 退出 ✓ |
| 5-6 | 并发联合回归 + Gate 5 未通过口径 | ✅ 我独立复跑 9 包 race 全绿；依赖闭包 0；无 3333 遗留进程 ✓ |

---

## 3. 本轮实证过的隐式契约（探针，全部通过）

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PR-S5-1** 镜像代际与水位 | 8 类拒绝 + 代际切换 + `Invalidate` 语义 | ✅ 事件永不产生就绪；失败/空种子拒绝；成交与时间回退拒绝；终态不可复活；超量与 ALGO 事件拒绝；新代未种子前拒绝；**旧代事件被拒**；旧代 `Invalidate` 不误伤现役、现役 `Invalidate` 触发重同步；拒绝计数进入指标（8） |
| **PR-S5-2** 告警队列 | 容量/背压/丢弃计数/非法事件/worker 卡死/取消 | ✅ 超容量即背压且计数；非法事件不占容量不计数；**stalled worker 不阻塞发布**；sender 报错不影响 worker 生命周期；取消即退出 |
| **PR-S5-3** 监督器公开面 | 反射 + 构造约束 + `Status` 纯读 | ✅ 公开面恰为 `{Run, Status}`；拒绝 Main 账户与 nil 参数；未运行时 Guard=disabled、stream 未运行、usage 全 0 |

> 既有永久测试覆盖：5-1 的 `TestStage51MirrorRequiresCompleteLeadSnapshotAndGeneration` / `...StreamLifecycleReconnectKeepaliveAndStop` / `...ClientRejectsMainAccount`、5-2 的单调与部分更新测试、5-3 的 `TestStage53DurableCASOnlyOneWorkerCanReserve`（32 并发）/`...RestartResolvesKnownFillButDoesNotEnableTrade`/`...MissingExchangeEvidenceStaysBlockedAcrossRestart`/`...IncludesClosedHistoryAndRejectsUnknownExchangeOrders`/`...OnlyNewerNoGapWSAndDurableProofClearsPending`、5-4 的限流与来源测试、5-5 的告警限频/背压/并发/脱敏、5-6 的联合回归。**本轮探针补的是**：镜像水位与代际的**全类拒绝矩阵**、告警队列的**非阻塞与取消语义**、监督器**公开面与默认值**。

---

## 4. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 -race`（leadaccount / feature-api-binance / futuresownership / feature / line / scanner / backtest / command / controllers，`-skip` 研究模板用例） | ✅ **9 包全部 ok**（leadaccount 2.20s、backtest 6.37s、command 4.22s 等） |
| `go vet`（同 9 包） | ✅ 干净 |
| 根包生产 `go build` | ✅ 成功（临时二进制已删除） |
| `git diff HEAD --check -- '*.go'` | ✅ 干净 |
| `go list -deps . \| grep -c service/leadaccount` | ✅ **0**（生产二进制不含 Lead 执行/WS/监督代码） |
| 端口 3333 遗留监听 | ✅ 无 |
| 探针 PR-S5-1～3 | ✅ 全部通过（含 `-race`）；副本去掉探针后原测试 ok |
| Schema / 路由 / 配置 / 前端 | ✅ 无变更（仍 v19；本阶段未新增迁移） |
| 真实 Binance（Lead 私有 WS、Testnet 断线/ListenKey/Algo）、MySQL 8 恢复、双账户 2s 真实负载、API 预算证据 | ❌ **未执行**（Gate 5 真实环境项，记录已如实声明） |

---

## 5. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | P3（接线口径） | 6 项能力齐备但**全部未接线**（依赖闭包 0、无启动钩子）。记录标题含"真实 Lead 私有 User Data WS 生命周期/实现"，易被误读为"已在运行" | 记录 §限制 已写明"未连真实 ListenKey、生产主循环没有自动启用" ✓；建议把 5-1~5-6 记录的统一口径改为"**离线实现，未接线**"，并在 Stage 6/7 明确接线清单与授权门（同时保留 `LeadSupervisor` 公开面白名单测试，防新增 `Start/Enable`） |
| **F2** | P3（可观测性缺口） | `LeadAlertQueue` 的背压丢弃只累加 `Dropped()`，未出现在 `LeadUsageMetrics`/System Dashboard → 运维可能不知道告警已丢 | 建议 Stage 6 把 `Dropped()` 与队列长度并入看板 Lead 区块（与 `metrics.go` 的现有字段同路径），并在丢弃时按故障严重度至少保留一次日志 |
| **F3** | P3（性能，沿用） | 恢复器/镜像的 REST 全量读取与平仓校验仍会**重复拉取 exchangeInfo**（4-4 F3 未处理） | 建议复用 V4-5 已建立的长 TTL 交换信息缓存（Lead 规则源加 TTL），避免每次对账/平仓各一次公共请求 |
| **F4** | P3（沿用） | 4-5 的 F1（`RuntimeStatus()` 写内部镜像）、4-4 的 F1~F4（恢复证据无落库/Algo FINISHED 判定/exchangeInfo 缓存/动态过滤器）、4-3 的 F3/F4、4-2 的 F1/F2、4-6 的 F1/F2/F3/F5（AST 扫描加固、`GetPosition` 只返回活动仓位、Gate 措辞、三项外部确认） | 均未处理；建议在 Stage 6/7 逐项收口，尤其"恢复证据落库"（Stage 5 已具备 CAS，但恢复结果仍只回写 Ownership，未形成持久恢复审计流水） |
| — | 正面 | ① 三条不变量用**结构性**手段固定：WS 只更新只读投影（不碰 Ownership）、证据有缺口即不可信（代际 + 水位 + 启动窗口）、待对账必须"持久 CAS + 完整对账 + 无缺口镜像"三重证明才清除；② `UnverifiedSavedRequests` 明确拒绝虚报 API 节省（与 5-4 记录口径一致）；③ 告警队列与 429/418 回调都刻意规避阻塞/自锁，符合"告警不等于放行、通知失败不解锁"；④ 生产依赖闭包仍为 0，公开方法白名单测试 + 全仓 AST 扫描双保险 ✓ |

---

## 6. 审计边界与未验证项

- **未验证**：真实 Lead 私有 WS 连接/断线重连/ListenKey 到期、Testnet 断线实验、真实 Algo STOP/TP、双账户真实 2 秒网络负载、MySQL 8 上的重启恢复与跨进程 CAS（本阶段仅 SQLite 内存库）。
- **未验证**：真实 API 预算/权重的 A-B 证据（记录明确不作数值承诺 ✓）。
- **未验证**：真实 DingDing/Slack 通道的时延与超时（仅 Fake Sender 验证 worker 生命周期 ✓）。
- **未验证**：前端与 MySQL（无变更）。
- 本审计未发起任何真实 Binance 请求、未改动用户仓库任何文件（探针仅存在于已删除的 `/tmp` 副本）。

---

## 7. 附：Stage 5 交付物清单与建议下一步

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `service/leadaccount/stream.go` | 249（新） | Lead 私有 User Data WS 生命周期：显式 Run/Stop、代际、ListenKey 重建、WS→REST 缺口判不可信 |
| `service/leadaccount/mirror.go` | 333（新） | 只读账户镜像：完整 REST 种子、订单/余额/仓位水位与单调校验、终态水位、拒绝 ALGO 冒充 |
| `service/leadaccount/reconciler.go` | 174（新） | 显式启动恢复器：全量未决单 + 含 CLOSED 的受控持仓 + REST 全量，幂等写回、异常即阻断 |
| `service/leadaccount/supervisor.go` | 127（新） | 监督器：恢复 → WS → Guard → HTTP 回调 → 告警队列；仅 `{Run, Status}` |
| `service/leadaccount/alerts.go` | 106（新） | 有界非阻塞告警队列 + 限频 + 复用既有通道（`Module=lead_trading`） |
| `service/leadaccount/metrics.go` | 47（新） | Lead 五分钟用量/429/418 + WS 事件/REST 快照计数；不虚报节省 |
| `service/futuresownership/service.go` / `execution.go` | 调整 | `ReserveSubmission` 持久原子 CAS（写前预留），未抢到者不重试 |
| `feature/api/binance/lead_stream_client.go` / `lead_http_fault.go` | 30 / 46（新） | Lead ListenKey 清理 + 经代理的 WS Dial；429/418 → Guard（脱敏、无反向取锁） |
| 测试 | `stream_test.go`、`reconciler_test.go`、`telemetry_alerts_test.go`、`stage56_integration_test.go`、`lead_http_fault_stage5_test.go`、`lead_order_limiter_stage5_test.go` | 见 §3 说明 |
| 文档 | — | `Stage5-1` ~ `Stage5-6` 实施记录 + 方案文档同步 |

**建议下一步**：① 统一 Stage 5 记录口径为"离线实现、未接线"，并把 `Dropped()` 纳入看板（F1/F2）；② 按计划进入 **Stage 6**（Portfolio 身份/资产人工对照界面、强鉴权凭证与启停 API、额度/故障/用量面板），期间保持 `enabled=false`；③ 三项外部确认不要遗漏（Stage 3 Main 在线业务验收、Stage 2 MySQL v19 迁移、Gate 0-LIVE 的 Portfolio 身份与只读权限）；④ **Gate 5 的真实环境项**（真实 WS/Testnet 断线/双账户负载/MySQL 8 恢复/预算证据）需在获得授权与测试条件后补验，补验前不得以本阶段结论作为启用 Lead 实盘的依据。
