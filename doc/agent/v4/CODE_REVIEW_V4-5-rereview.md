# V4-5 复评报告（代码再次修改后重新审查）

- **复审对象**：V4-5 **当前工作区/索引**版本（相对首轮审计 2026-09-24 16:28 的大幅扩展）。基线 `HEAD = 80ba5bd`（Merge PR #58）；V4-5 全部改动仍未提交
- **首轮报告**：`doc/agent/v4/CODE_REVIEW_V4-5.md`（24 文件 / +1266-424）
- **本轮规模**：**49 文件 / +4605-495**（新增：spot 侧 WS 缓存约 700 行 + Spot Rush/ExchangeInfo 缓存；删除：3 个 Config 字段与 `[ws]` 配置段与旧固定节流）
- **复审方式**：原地只读审计 + 隔离副本写临时探针（探针与副本已删除，用户仓库未被改动；探针中一处仅用于测试的 setter 只存在于副本）
- **结论**：**已完成，可进入 V4-6**。首轮 Gate 全部仍然成立（10 组探针复测通过）；**首轮 F1（空仓位快照不缓存）已修复**；本轮新增 **1 项 P2（配置/模型字段删除后的库兼容与文档同步）+ 3 项 P3**，无 P0/P1。
- **复审日期**：2026-09-24（18:26）

---

## 1. 上轮发现处理核对

| 首轮编号 | 问题 | 现状 | 证据 |
| --- | --- | --- | --- |
| **F1**（P2） | 空仓位账户下 2s Position 快照缓存失效（`len(rows) != 0` 判空） | ✅ **已修复** | `positionSnapshotEntry`/`openOrdersSnapshotEntry` 增加 `valid` 字段，读取条件改为 `entry.valid`（`account_read_cache.go:73-105`）；新增永久测试 `TestAllPositionsCacheEmptySnapshot`。**复评探针 RR-1 实测**：空仓账户 TTL 内 3 次全账户读取 → **真实请求 1 次**（首轮为 3 次）；同期 openOrders 仍 1 次 |
| F2（P3） | `Snapshot` 为 O(事件数)（20k 事件 / 120 键 = 7.58ms） | ⚠️ **未处理** | `collector.go` 本轮未变（仍 530 行）；看板为低频刷新，可接受，仍建议后续加 TTL 缓存或增量聚合 |
| F3（P3） | 文档"账户读取不再随候选数线性放大"口径过宽 | ✅ **Phase 文档已修正** | §3 改为"当 Futures User Data WS 尚未建立健康 mirror（例如启动/bootstrap、重连后等待 full sync）时…"；§5 改为"当强制启用的 WS 已完成当前 generation full sync 且 freshness 合格时" |
| F4（P3） | `EstimateWeight` 未列出路径按 1 计（权重为下界） | ⚠️ 未处理（口径一致） | 仍建议看板/报告标注"估计权重仅用于排序" |
| F5（P3） | Spot order 限额依赖 spot exchangeInfo 是否被拉取 | 🔶 **部分改善** | 新增 spot exchangeInfo 长 TTL 缓存 + `GetExchangeInfoFreshContext`（`spot/api/binance/index.go:65`），Spot Rush 会触发 fresh 拉取 → 限额会被写入（`index.go:87/98` 仍在 load 路径内 ✓）；仍未在启动时主动拉取 |
| F6（P3） | 跨阶段遗留（V4-2 两项、V4-3 R1） | ⚠️ 未处理 | 与本阶段无关，仅跟踪 |

---

## 2. 本轮改动总览（自首轮审计以来）

| 类别 | 内容 |
| --- | --- |
| **新增（spot 侧对齐 futures）** | `spot/api/binance/kline_ws_cache.go`（387 + 测试 96）：Spot Kline 规范缓存 + 组合 WS 增量 + 订阅管理；`spot/api/binance/market_cache.go`（237 + 测试 78）：Spot all-market ticker WS 快照 + ExchangeInfo 长 TTL 缓存；`spot/spot.go`：Spot Rush 改为 **listing-gated** ExchangeInfo 刷新（ticker WS 出现该 Symbol 才强制刷新，fallback probe 10s）；`main.go`：启动 spot WS 同步 |
| **futures 侧补强** | `feature/feature_rush.go`：Futures Rush 同样改为 listing-gated（fallback probe 5s）；`feature/strategy/coin/common.go`：legacy 选币的"最近订单"查询加 **10s 缓存 + singleflight**；`feature/api/binance/index.go`（+384）：账户/行情缓存接入与大量 `*Context` 变体 |
| **删除（配置/模型）** | `models/tableStruct.go` 移除 `WsFuturesEnable`/`WsSpotEnable`/`WsDeliveryEnable`；`command/db_update.go` 的 InitData INSERT 列同步收缩；`controllers/index.go` 配置响应移除三字段；`conf/app.conf.example` 移除 `[ws] futures_user_data = 1` 段 → **User Data WS 变为强制基础设施**（Phase 文档 §2/§5 已如实声明） |
| **行为替换** | `market_data_guard.go`：移除固定 **1000 权重/分钟**市场数据节流，只保留 `-1003` 封禁冷却；限速权全部移交全局 Budget Coordinator |
| **门控改造** | `futuresUserDataMirrorUsable()` 不再读配置开关，改为**纯真实连接状态**：当前 active generation 非 0 + 该 generation 已完成 full sync + full sync ≤35 分钟 |
| **健康检查** | `service/systemhealth`：Futures WS 检查不再读已删字段（恒执行，见 N5） |

---

## 3. Gate 复评（10 组探针全通过）

| 探针 | 复核内容 | 结果 |
| --- | --- | --- |
| **RR-1** 空仓位缓存（首轮 F1） | 空仓账户 TTL 内 3 次全账户读取 | ✅ 真实请求 **1 次**（修复生效） |
| **RR-2** 5 候选单轮放大（Gate 1/2） | 两轮完整 cycle（全账户 + 每候选 symbol-specific） | ✅ 全账户 positionRisk/openOrders 各 **1 次**（两轮仍是 1/1）；候选内 5+5 全部带 `symbol=` |
| **RR-3** 镜像门控（Gate 6，新条件） | 无连接 / 重连 generation 不匹配 / 36 分钟超龄 / 匹配且 34 分钟 | ✅ 前三者不可用，第四者可用；不再依赖已删除的配置开关 |
| **RR-4** cycle snapshot 语义 | 多空独立、零仓/撤销单不计、只认开仓方向、pending open 只记一次 | ✅ 全部符合 |
| **RR-8** 预算拒绝前置（Gate 3/4） | critical 下 P2 defer、P0 放行；429 后 P0 也被拒 | ✅ 被拒请求 **base 调用数 0** 且不计入用量；throttle 状态可见 |
| **RR-9** Order Count 守卫（Gate 3） | `order_count_10s = 10/10` | ✅ 下单发送前被拒（base 调用 0），P1 读取放行 |
| **RR-10** 采集器环形缓冲 | 20k 事件稳态 + 窗口内溢出 60k | ✅ 稳态 **1.49µs/请求**（首轮 328ns、V4-4 为 596µs）；`dropped=15001/truncated=true` |
| **RR-5** spot ticker 新鲜度 | 无 WS / 刚更新 / 10s 陈旧 | ✅ 三态正确（陈旧拒绝） |
| **RR-6** spot Kline 缓存边界 | TTL 命中、负缓存返回错误、过期失效、容量 | ✅ 容量实测 **256 = 上限**（最旧到期项淘汰） |
| **RR-7** spot WS 订阅上界 | 306 条订阅 + 1 条不活跃 | ✅ 返回 **256** 条（上限），不活跃订阅被淘汰 |

---

## 4. 本轮新增发现

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **N1** | **P2（部署兼容 + 文档）** | 删除 `Config` 三个字段与 `[ws]` 配置段，但**没有 DB 版本/迁移动作**（`appversion` 仍 18，无新 SQL）。已存在的库会保留 3 个孤儿列：SQLite 无害 ✓；但**MySQL strict 模式下**，若某条"整行 INSERT config"路径在保留旧列的库上执行，会因这些无默认值的 NOT NULL 列报 1364。当前唯一 INSERT 是 `InitData`（仅在 config 表为空时执行）→ 现实触发场景罕见（旧库 + config 被清空） | 建议：① 在 Phase 文档/README 明确"升级无需 `sync db`，旧库保留孤儿列且不影响运行"；② 如追求整洁，可加 `19.sql` 显式 DROP 这三列（MySQL 在线 DDL，注意与版本号联动）；③ 老 `conf/app.conf` 里残留的 `[ws] futures_user_data = 1` 属无害未知键，可在升级说明里提一句 |
| **N2** | P3（行为移交需知情） | `market_data_guard.go` 的**固定 1000 权重/分钟**节流被移除，市场数据（Kline/历史）限速完全交由 Budget Coordinator。差异：normal 状态（<70%）下历史/Kline 高频拉取**不再有任何固定节流**，只在 warning/critical 被 P2/P3 defer 挡住 | 与 V4-5 设计一致（统一预算）✓；建议在 Phase 文档/报告显式记录"节流权移交"，避免读者以为两道保护仍在 |
| **N3** | P3（文档同步） | `v4-5-implementation-report.md` **未随本轮更新**：全文仅 1 处 "spot"（矩阵里的旧行），未覆盖 spot 侧新增缓存（约 700 行）、未覆盖 WS 开关/`[ws]` 配置删除、未覆盖 market_data_guard 节流移除 | Phase 文档已更新 ✓，但实施报告是"静态调用矩阵 + 安全边界"的权威参考；建议同步（至少补 spot 行与配置删除说明） |
| **N4** | P3（观察） | spot Kline WS 订阅上界 256 条流（Binance 单连接上限 1024 ✓）、缓存 256 条、15 分钟不活跃淘汰、负缓存 2s —— 边界设计保守且已实测生效 | 无需改动；若未来 spot 全市场订阅需求上升，可评估提高上限 |
| **N5** | P3（语义变化） | `service/systemhealth` 的 Futures WS 检查在标志删除后**恒执行**：WS 无数据时现在报 warning（`no futures market update has been recorded`）而非 `disabled` | 与"WS 强制开启"一致 ✓；建议同步 V3-8 文档/看板文案，避免运维误判为故障 |
| — | 正面 | Spot 侧（ticker/Kline/ExchangeInfo）与 Futures 侧已对齐同一套设计（WS 优先 + TTL + singleflight + 上界 + 负缓存）；Rush 两侧都改为 listing-gated，消除了 100ms ExchangeInfo 轮询 | 与报告 §8 声明一致 ✓ |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **63 个包 ok，0 FAIL** |
| `go test -count=1 -race ./spot/api/binance ./feature/api/binance ./service/binanceapiusage ./feature` | ✅ 全部 ok（1.29s / 1.64s / 1.69s / 1.34s） |
| `gofmt -l`（改动 Go 文件） | ✅ 无输出 |
| 隔离副本 10 组探针（RR-1～RR-10） | ✅ 全部通过；副本去掉探针后三个包原测试 `ok` |
| 前端产物核对 | ✅ 已删配置（`wsFuturesEnable`/`wsSpotEnable`/`wsDeliveryEnable`/`ws_futures_enable`/`futures_user_data`）在 `static/` 中 **0 命中**，前后端一致 |

---

## 6. 审计边界与未验证项

- **未验证**：真实 MySQL 上"删除字段后旧库运行 + 清空 config 后重新初始化"的组合场景（N1 为静态推演 + SQLite 侧无影响）——如需确认，建议在测试库执行一次 `sync db` 后插入配置行观察。
- **未验证**：真实 WS 生命周期（spot/futures 的真实连接、重连、订阅切换）——门控逻辑用副本内 setter 写同一 active generation 原子验证；spot 订阅裁剪用直接构造订阅表验证。
- **未验证**：spot 主流程端到端（Spot Rush/Notice 真实触发）。
- **未复测**：`Snapshot` 成本（首轮 7.58ms；`collector.go` 本轮未变，结论沿用首轮）。
- 未对生产发起任何真实下单或配置修改。

---

## 7. 复评结论摘要

本轮把 V4-5 从"futures 为主"扩展为 **futures/spot 对称的 WS 优先缓存体系**，并把 User Data WS 从"可选开关"提升为**强制基础设施**（门控改为纯连接状态判定，安全性更强）；同时移除了旧固定节流，限速统一归 Budget Coordinator。首轮唯一 P2（空仓位快照不缓存）已修复并补了永久回归测试；10 组探针复测确认 §11 七条 Gate 全部仍然成立。剩余为 1 项 P2（配置/模型字段删除后的**库兼容说明与文档同步**）与 4 项 P3，均不阻塞进入 V4-6。
