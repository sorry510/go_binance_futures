# BT-3 回测存储与 IO 优化开发记录（待 Code Review）

> 日期：2026-11-19；基线：`d6c16d4`（BT-2 已提交）；范围仅 `service/historicalmarket` 与相关测试。未操作生产库、未运行 `sync db`，未修改 `conf/app.conf` 或实盘交易代码。

## 一、实现内容

1. **保留旧稀疏缓存读取**：`SecondBars` / `Trades` 仍优先查询 `market_klines_1s` / `market_trades`；已有完整、有效的行缓存仍按原逻辑命中，不清理或迁移旧数据。
2. **默认停止新增高精度明细写库**：`DefaultAdaptiveResolutionProvider` 以 `writeSparse=false` 运行；新获得的 1s、Trades 及 Trades 推导秒线均只返回并使用已验证 ZIP，不再插入新稀疏行。保留 `NewAdaptiveResolutionProvider` 的旧写库行为用于独立对照/兼容，不增加前端配置。
3. **跨回测的持久化日 ZIP 缓存**：`public_data.go` 复用原下载、CHECKSUM、CSV 前向扫描代码。目录为配置项 `binance::public_data_cache_dir` 下的 `adaptive-um-archives` 子目录，未配置时使用 `./cache/tmp/adaptive-um-archives`，不再每次回测创建并删除整个日归档缓存。
4. **SHA256 校验与复用**：路径由来源 URL 哈希和 ZIP 文件名确定；下载时用上游 `.CHECKSUM` 验证并原子保存 ZIP 与 `.sha256` 校验元数据；新回测首次命中磁盘归档时重新计算文件 SHA256，损坏则重新下载/验证，不把错误文件当成缓存命中。
5. **容量和清理**：默认容量预算 **2 GiB**；完成回测并关闭客户端时，按 ZIP 最近使用时间淘汰最旧的已验证归档。运行中正在被本进程其他回测使用的文件不会被淘汰；中途可能临时超出预算。清理仅匹配 BT-3 的归档命名及有效元数据，过期 24 小时的 BT-3 临时下载片段可清理；其他目录与文件不动。
6. **UM/CM 隔离**：现有 Binance PublicData URL 只有 `data/futures/um`。只有 USDT-M 可以用此 ZIP 读取高精度数据；COIN-M 请求在本地稀疏数据未命中时明确拒绝，不借用 UM 归档。
7. **统计口径**：`SecondCacheHits` / `TradeCacheHits` 继续表示 MySQL 稀疏行命中；`ArchiveCacheHits` 表示跨回测文件/现有客户端 ZIP 命中；`ArchiveDownloads` / `DownloadBytes` 只统计下载。数据来源及 SHA256 继续进入原 `ResolutionEvidence`。

## 二、文件

| 文件 | 内容 |
|---|---|
| `service/historicalmarket/public_data.go` | 持久化 ZIP、从磁盘恢复、原子校验元数据、缓存访问时间与结束清理 |
| `service/historicalmarket/archive_cache.go`（新增） | SHA256 磁盘复核、缓存 URL 定位、按字节上限淘汰与同进程活跃引用保护 |
| `service/historicalmarket/resolution_provider.go` | 默认文件模式、旧行读取兼容、按模式控制新增写库、UM-only |
| `service/historicalmarket/bt3_cache_test.go`（新增） | 冷热复用、损坏重取、数据/证据一致、无新增 SQL 行、交易回退、容量淘汰、并发引用保护、来源失效和 CM 隔离 |

## 三、本地验证

| Gate | 结果 |
|---|---|
| `go test ./service/historicalmarket -run '^TestBT3' -count=5 -p=2` | 通过 |
| `go test ./service/historicalmarket ./service/backtest -count=1` | 通过 |
| `go test -race ./service/historicalmarket ./service/backtest -count=1` | 通过（macOS linker 有非致命 warning） |
| `go vet ./service/historicalmarket ./service/backtest` | 通过 |
| `go build ./service/...` 和单独编译项目主程序 | 通过 |
| `go test ./service/... -run '^$' -count=1` | 生产服务包编译通过 |
| `git diff --check` | 通过 |
| 整库 `go build ./...` | **未通过**：仓库其他 `strategy_templates/research` 目录存在重复 `main`；`research_specs` 内也有冻结工具缺符号/重复定义，均非 BT-3 变更；未修改这些研究目录 |

**本地可量化证据**：固定 2 根 1s + 2 笔 Trades 的合成数据，文件模式新增数据库行 **0**，重新创建 Provider 后归档请求 **0**，冷热 `Rows` 与 `EvidenceHash` 一致；原写库模式及 Trades 推导秒线也做过数据/证据一致性对照。

## 四、未验证与审查重点

- 上述是离线测试，**没有**对真实 MySQL 8 数据量、日 ZIP 大小、完整历史区间或生产实盘做空间/速度验收；不能据此宣称端到端空间已经降低特定百分比。
- 持久缓存仍占磁盘，2 GiB 为回测结束后清理的目标预算；正在使用的文件可能延后回收。其他进程同时共享同一目录的边界尚未做跨进程锁验证，个人单服务实例为主要部署场景。
- 新客户端首次磁盘命中会扫描 ZIP 校验 SHA256（额外顺序磁盘 IO），与重复下载/大量 MySQL 行写入之间的净收益应在正式历史样本上测量。
- 独立 Code Review 重点：文件模式与原写库模式的交易/事件/权益结果保持一致（缓存命中统计可不同），归档完整性与复用安全、旧数据读取兼容、缓存清理范围及异常恢复。
- 保持 BT-1/BT-2 的 ROI、成交和指标算法不变；Mark Price、完整实盘调度/网络延迟不属于 BT-3。

**状态：BT-3 本地代码及离线 Gate 完成，待独立 Code Review 与真实历史样本的空间/性能对照。**
