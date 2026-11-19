# Code Review：BT-3 回测存储与 IO 优化

> 审查对象：BT-3 工作区改动（基线 `HEAD d6c16d4 feat: bt-2`，BT-2 已提交）
> 审查依据：`doc/trade/BT-3-回测存储与IO优化开发记录.md`、`doc/trade/回测引擎优化方案-精简版.md` 及复审报告
> 审查方式：隔离副本 `/tmp/phase_audit_bt3`（1224 文件 / 800 Go，SHA256 全等）+ 全套离线 Gate + 独立审计探针（数据层 / 引用计数 / 并发竞态）
> 未改用户业务代码、未提交、未跑 `sync db`、未连真实 Binance、未写生产库。

---

## 一、结论

**BT-3：✅ 源码 / 离线 Gate 通过；建议「有条件进入真实历史样本验收准备」，禁止宣告空间优化达标或生产通过。**

- 无 P0 Finding，无阻断性 P1。实现兑现了方案 BT-3 的三项核心改法：归档有界共享缓存、默认停止新增高精度明细写库、保留旧数据只读兼容；并主动落实了首轮方案审查 F2（不直接关写库、先改命中路径）与 O1（UM/CM 隔离）。
- 数据层一致性、SHA256 完整性、引用计数、同进程并发保护经开发方测试 + 本次 5 组独立探针验证。
- 1 项 P2 观察（O1 跨进程共享目录无锁）、若干预期长需评估；1 项 P3（端到端空间/性能未量化，须在真实历史样本上验收）。
- 开发记录自报「整库 `go build ./...` 未通过」经核实属实，但失败全部来自未跟踪的冻结研究工具，与 BT-3 无关；隔离副本（仅跟踪文件）build 通过。

**允许**：进入经单独授权的真实历史样本空间/性能对照准备。**禁止**：据此宣称生产库空间已下降特定百分比或端到端提速。

---

## 二、改动范围与等价性

| 文件 | 类型 | 用途 |
|---|---|---|
| `public_data.go` | M | `MaxCacheBytes` / 持久化 CacheDir；`cachedArchive` 增加磁盘恢复与 mtime 更新；`downloadArchive` 用 `.part` 临时文件 + 原子保存 manifest；`Close` 改为 release + prune |
| `archive_cache.go` | A | `archiveTarget` URL 哈希定位；`holdArchive`/`releaseArchives` 全局引用计数；`loadDiskArchive` SHA256 + size 双校验；`saveArchiveManifest`；`pruneVerifiedArchives` 字节预算 LRU |
| `resolution_provider.go` | M | `writeSparse` 开关（默认 provider=false，legacy=true）；UM-only 拒绝 CM；统计口径注释 |
| `bt3_cache_test.go` | A | 冷热复用、损坏重取、零新增 SQL、UM/CM 隔离、派生秒线一致、来源失效 fail-closed、LRU、活跃引用保护 |

副本 4 文件 SHA256 全等。`store.go:49` 生产路径确认使用 `DefaultAdaptiveResolutionProvider`（本阶段 writeSparse=false）。

---

## 三、离线 Gate（全部通过）

| 检查 | 结果 |
|---|---|
| `git diff --check` | ✅ 通过 |
| gofmt（4 文件） | ✅ 无未格式化 |
| `go vet ./service/historicalmarket ./service/backtest` | ✅ 通过 |
| `go build ./...`（隔离副本，仅跟踪文件） | ✅ 退出码 0 |
| `go build ./service/...` | ✅ 通过 |
| `go test -count=1` 全量 88 包 | ✅ 76 ok + 12 no test + 0 FAIL |
| `go test ./service/historicalmarket -run '^TestBT3' -count=5 -p=2` | ✅ 通过 |
| `go test -race ./service/historicalmarket ./service/backtest -count=1` | ✅ 通过（LC_DYSYMTAB warning 非致命） |

**整库 build 核实**：用户工作区 `go build ./...` 退出码非 0，失败项全部位于未跟踪的 `strategy_templates/research/**`（重复 main）与 `research_specs/**`（冻结工具缺符号/重复定义），均为历史冻结欠账，非 BT-3 变更；与长期记忆「research/** 未跟踪」一致。隔离副本用 `git archive HEAD` 只含跟踪文件，故 build 通过。开发记录第 35 行声明属实。

---

## 四、核心声明实证

### 4.1 默认停止新增高精度明细写库 —— ✅

- `DefaultAdaptiveResolutionProvider`：`CacheDir=<tmp>/adaptive-um-archives`、`MaxCacheBytes=2GiB`、`writeSparse=false`；`NewAdaptiveResolutionProvider`（legacy）保持 `writeSparse=true`。
- 独立探针验证：默认 provider 确为 writeSparse=false + 2GiB + 专用缓存目录；legacy 构造器 writeSparse=true。
- `SecondBars` 的 ZIP 1s 路径、Trades 派生秒线路径、`Trades` 路径三处 Store 调用均由 `if provider.writeSparse` 守卫（resolution_provider.go:170/196/251），覆盖完整。
- 数据层对照：开发方测试 `TestBT3FileModeKeepsRowsAndEvidenceWithoutNewSQLInserts` 实测文件模式新增数据库行 0，冷/热 `Rows` 与 `EvidenceHash` 一致；派生秒线模式另有 `TestBT3DerivedSecondsMatchLegacyWriteMode` 证明 write/file 两模式 rows、EvidenceHash、ArchiveSHA256、Resolution 一致。这些是方案验收「结果一致，缓存命中统计可不同」的直接证据。

### 4.2 归档完整性与复用安全 —— ✅

- 下载走 `.CHECKSUM` 校验上游 SHA256，原子 rename 到 target，再写 `.sha256` sidecar（sha + size）。
- 磁盘命中 `loadDiskArchive` 重算文件 SHA256 并核对 size 与 sidecar，任一不符返回 miss 并触发重下载。
- 独立探针：① 篡改 sidecar size → 不被当作缓存命中，触发重取（PASS）；② 开发方测试覆盖篡改 ZIP 内容 → 重新下载验证；③ 来源不可用时明确报错、不静默产出精确结果（`TestBT3MissingSourceWithoutCacheFailsExplicitly`）。

### 4.3 容量预算与清理范围 —— ✅

- `pruneVerifiedArchives` 仅处理带 `.zip.sha256` 且命名符合 `SHA256(URL)[:8]-filename` 的成对归档，校验 sidecar 与文件大小一致才纳入预算；不认识的文件（如 `notes.txt`）跳过。
- 独立探针 + 开发方测试：文件模式临时片段 > 24h 才清理；LRU 按 mtime 淘汰最旧；`notes.txt` 等无关文件不被删除；小预算下活跃 client 的归档不被淘汰。

### 4.4 引用计数生命周期 —— ✅

- 全局 `activePublicArchives.paths` 记录同进程活跃引用；`holdArchive` 在下载启动与磁盘命中时注册，`releaseArchives` 在 `Close()` 释放，`prune` 跳过 `paths>0` 的文件。
- 独立探针 A：下载失败（checksum mismatch）路径中引用曾 +1（duringFailedFetch=1），`Close()` 后净归零（afterFailClose=0），无泄漏；正常路径同样归零。
- 独立探针 E：20 个 client 共享同一 CacheDir 并发 Fetch/Close，`-race -count=3` 通过，无数据竞争、无引用泄漏。

### 4.5 UM/CM 隔离 —— ✅

`SecondBars` / `Trades` 在稀疏缓存未命中后加入 `market != MarketFuturesUSDT` 硬拒（resolution_provider.go:157/233），开发方 `TestBT3UMArchiveDoesNotServeCoinM` 验证 COIN-M 请求不发任何公共请求且明确报错。落实了方案审查 O1 与项目 CM/UM 隔离铁律。

---

## 五、观察项

- **O1（P2，跨进程边界）**：`activePublicArchives` 是**进程内**引用计数；开发记录第 42 行已明确「其他进程同时共享同一目录未做跨进程锁验证」。两个独立进程共用同一 `adaptive-um-archives` 目录时，A 进程 prune 可能淘汰 B 进程正在读取的文件（B 的 hold 对 A 不可见）。当前部署为单服务实例，属可接受限制；若未来多实例共享磁盘缓存，须引入文件锁或按实例分目录。**建议纳入生产发布 Gate 的部署形态确认项**。
- **O2（P3，端到端量化缺失）**：开发记录第 37 行的「新增数据库行 0、归档请求 0」是 2 根 1s + 2 笔 Trades 的合成证据。方案第 24 行要求「固定相同任务比较新增磁盘字节数、重复运行不重复下载、删缓存后能按来源重新获取」——前两项已在合成层验证，第三项（删缓存重取）由损坏重取探针间接覆盖；但**真实 MySQL 8 行数、真实日 ZIP 大小、完整历史区间的净空间收益与首次磁盘 SHA256 扫描 IO 的成本未被测量**。须在真实历史样本验收中补齐。
- **O3（P3，统计口径）**：`ArchiveCacheHits` 现兼表「跨回测磁盘命中 + 现有客户端 ZIP 命中」，开发记录第 13 行已明确定义；下游若按旧口径解读会误判。建议在结果元数据/文档保持一致说明，避免与「下载次数」混淆。

---

## 六、边界与放行

- 源码 / 离线 / SQLite 合成 / 单进程并发通过 ≠ 真实 MySQL 8、真实日 ZIP 体积、完整历史区间、多进程共享、生产库或实盘通过。
- BT-1 / BT-2 限制延续：真实 Mark Price 归档未验证/未接入、开仓仍 1m 信号+下分钟开盘、约 2 秒调度延迟未模拟；BT-3 不改 ROI / 成交 / 指标算法与 `EngineVersion` / `ResolutionModel`，符合「只改存储 IO」的边界。
- 未发现改动触及实盘交易流程、`conf/app.conf`、生产库；未执行 `sync db`。
- **放行**：BT-3 源码 / 离线完成；进入真实历史样本的空间/性能对照需单独授权（只读预检、MySQL 8 EXPLAIN/P95、备份后缓存目录操作）。BT-3 综合验收后另起独立 Code Review。

> 审查探针仅写在隔离副本，报告落盘后清理；用户工作区保持 BT-3 交付时的改动状态。
