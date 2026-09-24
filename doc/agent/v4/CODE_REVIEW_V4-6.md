# V4-6 Phase 审计报告：Finalization

- **审计对象**：V4-6（Finalization）**工作区未提交**改动。基线 `HEAD = 2e2d075`（`feat: ai agent v4-5`，V4-5 已提交）
- **审计方式**：原地**只读**审计 + 实跑 Gate（本轮改动量小且新增逻辑均有永久测试，未建隔离副本、未写临时探针；用户仓库未被改动）
- **审查类型**：review-only
- **结论**：**交付物已完成、§7 自动化 Gate 全部通过；但按 Phase 文档自身的 §5/§10，Phase 仍标记为"进行中 🚧"，等待真实长期运行观察**。本轮未发现 P0/P1；发现 3 项 P3 + 3 项上轮遗留。
- **审计日期**：2026-09-24（20:00）

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-6 范围 | 原 **Plugin System 已移出 V4**（`06-phase-v4-6-plugin-system.md` 与 `07-phase-v4-7-finalization.md` 已删除），Finalization 重编号为 V4-6 ✓ 文档已同步 |
| 本阶段代码/文档交付 | ✅ 完成（Skill Studio 校验接入 Observability、Summary 新增 `llm_errors`/`skill_validation_errors`、看板展示、README/v4 README 同步） |
| §7 自动化 Gate | ✅ 后端全部通过（含 8 个 V4 相关包的 `-race`）；前端按报告为通过（本轮以构建产物交叉核对） |
| Phase 状态 | 🚧 **按文档自身标准仍为"进行中"**：`v4-6-implementation-report.md` §5 与 v4 README 均明确"真实长期运行观察尚未完成"。该项只能由用户实盘运行一段时间后确认，无法用自动化 Gate 替代 |
| DB / 配置 | ❌ 无 Schema 变更、无需 `sync db`、未改 `app.conf`（`conf/` 未出现在改动列表）✓ 与报告 §3 一致 |
| 是否存在 P0/P1 | ❌ 未发现 |

---

## 2. 验收清单逐项核对

### §2 Chat / Skill 检查

| 项 | 判定 | 证据 |
| --- | --- | --- |
| Conversation 多 Skill 行为稳定 | ✅（沿用 V4-1 证据） | V4-1 审计 + 修复提交 `b7cbc54`（`general_chat` 接入 `ChatAdapter`）；本阶段未改动该路径 |
| Model 切换可追踪实际模型 | ✅（沿用） | Task 记录 `provider/model/model_config_id`（V4-1 探针 P3 实证） |
| Chat 删除不留下不可访问 UI 状态 | ✅（沿用） | V4-1：删除走既有 `DeleteChat`（running 拒绝、Task/Audit 保留）+ 新增 bindings 清理；前端二次确认与 `deleteRunning` 文案齐备 |
| Skill Draft / Published Version 边界清晰 | ✅（沿用 V4-2） | V4-2 探针：发布前草稿不在 Store/ActiveVersions；发布生成 immutable revision；重复发布 `Duplicate` 复用 |
| Skill Studio 发布结果与上传 Import 结果一致 | ✅ | 两者都走同一 `Importer.install` + `ParsePackage`（`draft.go:565-571`、`importer.go:110-138`）；V4-6 只在控制器**追加观测记录**，未改导入语义（`agent_skill_draft.go:154-184`） |
| scripts 仍不会被执行 | ✅ **复测** | `git grep os/exec -- agent/` → **0 命中**（V4-2 结论仍成立） |

### §3 Selector 检查

| 项 | 判定 | 证据 |
| --- | --- | --- |
| smart local selector 确定性 | ✅（沿用 V4-3） | 三级稳定排序 + 永久测试；V4-3 复评探针（确定性/tie-break/轮转覆盖） |
| 不调用 LLM | ✅ | 选择器链路（`scanner` + `feature/strategy/coin`）无 LLM 依赖 |
| 不新增逐 Symbol Binance REST | ✅ **复测** | `go list -deps ./scanner` 中 Binance 依赖 = **0**（结构性证明） |
| 旧 selector 配置仍兼容 | ✅（沿用） | `GetCoinStrategy` 保留 `coin1～6`；V4-3 复评 N1 记录"测试路径跟随 `FutureStrategyCoin`"的行为变更，属**已文档化**的取舍 |

### §4 Binance API 检查

| 项 | 判定 | 证据 |
| --- | --- | --- |
| 看板可查看真实 API 用量 | ✅ | 前端产物 `observability-CFTM4DUD.js` 调用 `system/binance-api-usage`（只读、不调 Binance） |
| Top endpoint 与权重来源清晰 | ✅（沿用 V4-4） | 快照含 `top_endpoints_by_count/weight`、`limit_source`（exchange_info/reference）、`estimated_weight` |
| StartTrade 多币开仓不再线性重复高权重账户查询 | ✅（沿用 V4-5） | V4-5 探针：5 候选一轮全账户 positionRisk/openOrders 各 **1 次**（旧各 6 次）；候选内为 symbol-specific |
| Background/Historical 预算紧张时让路 | ✅（沿用 V4-5） | critical 下 P3 defer、P2 在 warning 串行且取槽后复检 |
| 429/418 不形成 retry storm | ✅（沿用 V4-5） | throttle 期内 P0 也在发送前被拒；`Retry-After` 驱动冷却窗口 |
| Mutation timeout/429 不导致 duplicate order | ✅（沿用 V4-5） | `execution.go:154` 预算拒绝 → 订单置 failed 且**不进入** lookup/reconcile（永久测试断言 submit/lookup 计数） |
| User Data WS/local snapshot 有 freshness/fallback | ✅（沿用 V4-5 复评） | 门控=真实连接状态（active generation + 该代 full sync + ≤35min），四分支探针实测；不健康时回退 REST |

### §5 System Dashboard 摘要

| 项 | 判定 | 证据 |
| --- | --- | --- |
| Chat / Model errors | 🔶 部分 | 新增 **LLM errors**（`llm_errors`：`type=llm_call && status=error`）；Chat 级失败仍只出现在既有 `errors[]` 分类中，未单列（P3，见 V4） |
| Skill validation errors | ✅ **实证** | 新增 `skill_validation_errors`（`type=skill_validation && status=error`），永久测试 `store_test.go` 断言两计数器各为 1；前端产物引用两个新字段 ✓ |
| Binance Used Weight / Order Count | ✅ | 前端渲染 `exchange_limits` 与 `budgets`（`used_weight_1m`/`order_count_10s` 各 2 处引用） |
| 429 / 418 | ✅（沿用 V4-5） | 窗口/端点/限流日志/限额状态四层可见 |
| Deferred / Coalesced API calls | ✅ | 前端引用 `deferred_requests`/`coalesced_requests`（各 2 处） |
| 不做 Prometheus/Grafana 强依赖 | ✅ | 全部为进程内内存指标 + 只读快照接口 |
| 「用量区域 5 秒静默刷新、手动刷新只刷新该区域」 | ✅ **产物实证** | `window.setInterval(oe, 5e3)`（`oe` = 只拉 `system/binance-api-usage` 的 fetcher），卸载时 `clearInterval`；与 `system/health` 的 fetcher 分离 ✓ |
| 「Exchange 头计数与 Budget 共用 75 秒 stale 语义」 | ✅ **代码 + 永久测试** | `collector.go:259-264` 在快照时对超过 `budgetStaleAfter`(75s) 的 `used_weight_1m/order_count_*` 归零；测试 `TestExchangeLimitsSnapshotClearsStaleResponseCounters` |

### §6 文档

| 项 | 判定 | 证据 |
| --- | --- | --- |
| 项目 README 多语言 | ✅ | `README.md` / `README.EN.md` 各更新 1 行（系统看板能力清单含 LLM 错误、Skill Studio 校验错误、Used Weight/Order Count、429/418、Budget、Deferred/Coalesced/Cache/WS 命中）；README 中 `Budget/预算` 5 处、`429` 1 处、`Skill Studio` 1 处 |
| V4 implementation reports | ✅ | `v4-0-baseline` + `v4-1`～`v4-5` + `v4-6` 共 **6** 份齐备；v4-5 报告已于 19:29 更新（补入 WS 强制基础设施说明） |
| Skill Studio 使用说明 | ✅ | `v4-2-implementation-report.md` §11 人工测试 A–H 组（新建/校验/路径安全/发布/激活/编辑新版本/allowed-tools/scripts） |
| Binance API Budget 说明 | ✅ | `v4-4`/`v4-5` 报告 + Phase 文档 §11 `05-phase-v4-5-binance-api-budget.md` |
| 删除文档的引用一致性 | ✅ | v4 README 的 V4-6 行已指向 `./06-phase-v4-6-finalization.md`，进度块为 `V4-6 Finalization 🚧`；`grep` 未发现对已删 `plugin-system`/`07-*.md` 的悬空引用 |

### §7 最终 Gate（本轮实跑）

| 命令 | 结果 |
| --- | --- |
| `go test -count=1 ./...` | ✅ **63 个包 ok / 0 FAIL** |
| `go test -count=1 -race ./agent/app ./agent/conversation ./agent/portableskill ./agent/observability ./scanner ./service/binanceapiusage ./feature/api/binance ./spot/api/binance` | ✅ 8 个包全部 ok（1.3s～3.2s） |
| `go vet ./...` | ✅ 无输出 |
| `go build ./...` | ✅ PASS |
| `git diff HEAD --check` | ✅ 无异常 |
| 前端 `pnpm typecheck` / `pnpm build` | ⚠️ 无法直接复跑（源码在独立仓库）；**产物侧确认**：`static/` 已含重建后的 `observability-CFTM4DUD.js`，且消费了 V4-6 新字段 → 与报告"已同步 dist→static"一致 |

### §8 项目约束

| 约束 | 判定 |
| --- | --- |
| 不修改 `app.conf` | ✅ `conf/` 未出现在本次改动列表 |
| 测试结束后不留下进程 | ✅ 本轮全部为前台命令，无后台进程 |
| 不回滚其它未提交修改 | ✅ 审计全程只读（未建副本、未写探针、未改动任何文件） |
| ARM 环境保持兼容 | ✅ 本阶段无平台相关代码；Go 侧仅新增纯逻辑与测试 |
| 个人使用优先 | ✅ 未引入 Prometheus/Grafana 或企业级治理组件 |

---

## 3. 本轮实证要点

1. **新计数器正确性**：`summary.go:209-282` 从**已按 `created_at` 窗口过滤且无分页上限**的 observation 集合中统计（`summary.go:176-178` 无 `Limit` → 不会出现"分页泄漏导致少算"）；永久测试插入 1 条 `llm_call/error` + 1 条 `skill_validation/error` 并断言两计数器各为 1（同时把 `traces.Total` 期望从 2 修正为 3，说明作者注意到新增观测会进入 Trace 统计）。
2. **Skill 校验失败入库形态**：`recordSkillDraftValidation`（`agent_skill_draft.go:244-259`）写 `type=skill_validation`、`error_type=skill_validation_error`、`error=<校验信息>`；`Validate` 与 Publish 前置校验**都**调用（计数不会漏 publish 路径）✓；未新增统计表 ✓（沿用 `agent_observations`）。
3. **看板刷新语义**：5 秒静默刷新仅作用于 Binance 用量区域，且该接口只读内存快照（不触发 Binance 请求 → 刷新不增加 API 用量）✓。
4. **75 秒 stale 语义**：collector 快照对超龄响应头计数归零，与 Budget 的 `budgetStaleAfter` 同源，避免旧 Used Weight 长期显示 ✓（有永久测试）。

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **V1** | P3 | **Phase 状态自认"进行中"**：代码/文档/Gate 均完成，但 `v4-6-implementation-report.md` §5 与 v4 README 均标注 🚧，理由是"真实长期运行观察尚未完成" | 这是**诚实且合理**的标记；建议明确观察清单（例如：连续运行 ≥24h 后检查 429/418 计数、deferred 比例、WS 重连次数、Skill 校验错误、Chat/Model 错误），以便把该 Phase 收口 |
| **V2** | P3 | 校验错误文本会被持久化到 `agent_observations.error`：内容为解析器消息（如 `unsupported Agent Skills frontmatter field "private-key"`、`skill name "x" must match parent directory "y"`）——**不含草稿正文/密钥**，但会包含用户自定的 Skill 名与字段名 | 建议在文档说明该观测字段的内容边界（或对超长错误做截断），避免未来把更长的解析上下文写进去 |
| **V3** | P3 | 新增计数与既有 `errors[]`（按 `error_type` 聚合）存在信息重叠，看板同时展示可能造成"同一失败被数两次"的观感 | 前端为两个独立区块时可接受；建议文案上区分"分类计数"与"重点摘要" |
| **V4** | P3 | §5 要求 "Chat / Model errors"，实现为 `llm_errors`（模型调用失败）+ `skill_validation_errors`；Chat 会话层失败（如消息启动失败）未单列 | 影响小（仍可在既有 errors 分类看到）；若需要，可后续补 `chat_error` 计数 |
| V5 | P3 | 上轮 V4-5 复评遗留未处理项：**N1**（删除 Config 三字段后旧库遗留孤儿列 / MySQL strict 下的 INSERT 说明——v4-5 报告 §17 仍只写"无 schema 变更、无需 sync db"，未提孤儿列）；**N2**（`market_data_guard` 固定节流移除后的"节流权移交"说明）；**N3**（v4-5 报告仍仅 1 处 "spot"，未覆盖 spot 侧约 700 行新缓存） | 均为文档补充项，不阻塞；建议在 V4 收尾文档里一并补上 |
| V6 | 观察 | V4-3 复评 N1（模拟盘路径跟随 `FutureStrategyCoin`）、V4-2 遗留（发布=管理员级提示词变更的文档说明、`DraftStore.List` 静默跳过损坏 draft.json）仍未处理 | 长期跟踪 |
| — | 正面 | 本轮改动**没有**触碰任何业务逻辑：仅新增观测写入 + 统计字段 + 文档/README；`scripts` 不执行、Selector 无 Binance 依赖、ownership 安全检查、预算拒绝语义等 V4 关键约束复测全部仍然成立 | — |

---

## 5. 审计边界与未验证项

- **未验证**：真实长期运行观察（Phase 自认的最后一个条件）——需用户以新二进制实盘运行一段时间后采集：429/418 计数、deferred/coalesced 比例、WS 重连与 full sync 间隔、Skill 校验错误、Chat/Model 错误。
- **未验证**：前端源码与 `pnpm typecheck/build`（仅以 `static/` 产物交叉核对；本轮未做源码级审查）。
- **未验证**：真实浏览器交互（看板 5 秒自动刷新与手动刷新按钮的实际点击行为——已从产物确认定时器与 fetcher 分离，未做端到端点击验证）。
- **未验证**：MySQL 侧（V5/N1 的孤儿列与 strict 模式场景）。
- 本轮未创建审计副本、未写临时探针（改动量小且新增逻辑已有永久测试），用户仓库与工作区均未被改动。

---

## 6. 附：V4-6 交付物清单

| 文件 | 变更 | 作用 |
| --- | --- | --- |
| `agent/observability/summary.go` | +2 字段 +14 行 | Summary 新增 `llm_errors` / `skill_validation_errors`（按看板时间窗口统计） |
| `agent/observability/store_test.go` | +12 行 | 永久测试：两个新计数的取值断言（并修正新观测进入 Trace 后的总数期望） |
| `controllers/agent_skill_draft.go` | +22 行 | Skill Studio `Validate` 与 Publish 前置校验接入 Observability（`type=skill_validation`，失败 `error_type=skill_validation_error`） |
| `README.md` / `README.EN.md` | 各 1 行 | 系统看板能力清单补充（LLM 错误、Skill 校验错误、Used Weight/Order Count、429/418、Budget、优化命中） |
| `doc/agent/v4/README.md` | 更新 | V4-6 指向 `06-phase-v4-6-finalization.md`，标注 🚧；说明 Plugin System 已移出 V4 |
| 删除 | 2 份 | 原 `06-phase-v4-6-plugin-system.md`、`07-phase-v4-7-finalization.md`（Plugin System 移出 V4，Finalization 重编号为 V4-6） |
| 新增文档 | 2 份 | `06-phase-v4-6-finalization.md`（本阶段文档）、`v4-6-implementation-report.md` |
| `doc/agent/v4/v4-5-implementation-report.md` | 更新 | 补入 WS 强制基础设施说明（复评建议的部分落实） |
