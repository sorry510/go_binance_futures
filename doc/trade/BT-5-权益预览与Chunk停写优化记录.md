# BT-5：权益预览限额与关键点采样优化

日期：2026-11-20

## 目标与边界

- 新回测不再向 `agent_backtest_equity_chunks` 追加完整分钟级权益；只存 `agent_backtest_equity_preview`，最多 20,000 点。
- 回测策略执行、信号、平仓、资金费、权益的逐分钟生成和 `calculateMetrics` 等全部按原完整数据执行。只在准备保存结果时做抽样。
- 不删表、不做 SQL 数据迁移、不删除已有历史运行记录。既有 Chunk 读取、旧分钟权益表读取与按任务删除功能全部保留。
- 无需升级 DB，不调整前端接口和图表展示协议。

## 核心算法

新文件：`service/backtest/equity_sampling.go`，`sampleImportantEquityPoints`。

1. 保留第一点、最后一点、权益全局最高最低点、最大回撤的谷值与其此前峰值。
2. 对每笔已完成交易的 EntryTime 和 ExitTime，按 BarTime 二分查询并保留时间相邻的两个分钟权益点；候选非常多时，交易附近点最多占总限额四分之一，按时间均匀选取，避免挤掉整段曲线。
3. 将完整区间划分成约 `limit/3` 个连续子区间，分别取每个区间权益最低点与最高点，预算约三分之二，优先保护短暂局部波动。
4. 用等间距采样补足剩余预算，确保整体曲线均匀覆盖，保证时间与序号严格递增、结果确定性、最多 20,000 点。
5. 回测点数不超过上限时完整保留。小限额支持 1 点、2 点等边界。

抽样仅决定**展示快照**：不能把新预览用于重算 Sharpe、Sortino、真实最大回撤或其他绩效指标。只保留预览会失去任意分钟完整的历史权益还原能力。

## 数据流调整

- `service/backtest/equity_store.go`：`buildEquityStorage` 改为生成单份 Preview；`persistPreparedEquityStorage` 只写入 Preview；保留旧 Chunk 解码、完整性校验、按序分块读取。
- `service/backtest/store.go`：向采样器传入 `result.Trades`，保存交易、事件、预览仍在一个事务中，失败全部回滚。
- `service/backtest/equity_codec.go`：已有稀疏预览若需要进一步缩小点数，改为按预览**下标**抽样，而不是按原 `sequence` 范围，否则原始序号出现跳号会导致返回点数不足。
- `models/agent_backtest.go`：注释明确 Chunk 为历史兼容用途。
- 不变更表结构、不变更 API 返回字段、保持 `binary_v1_base64/gzip` 格式和 CRC 校验。

## 验收结果

- `TestImportantEquitySamplingPreservesKeyEventsAndExtrema`：覆盖 10 万点、突发尖峰、回撤谷值、局部低点、开平仓前后时点、排序、确切点数、确定性，以及稀疏预览再次采样。
- `TestImportantEquitySamplingLimitsAndTinyRuns`：小规模与限额边界。
- `TestExistingFullEquityChunksStillReadableWithoutPreview`：旧 Chunk 无预览时依旧可读取。
- `TestEquityStorageUsesOnlyPreview`、`TestSaveResultWithProgressPersistsAllBatches`：新保存不会写 Chunk。
- `TestSaveResultTransactionRollsBackTradesAndEventsOnPreviewFailure`：预览插入冲突时，交易/事件回滚。
- `go test ./service/backtest ./service/historicalmarket -count=3 -p=2`：通过。
- `go vet ./service/backtest ./service/historicalmarket`：通过。
- `go build -o /tmp/go_binance_futures_equity_preview_buildcheck .`：通过（未覆盖运行中的二进制）。
- `git diff --check`：通过。

## 性能基准

`BenchmarkImportantEquitySamplingFourYears`，Apple M1 Pro，模拟四年 `4*365*24*60 = 2,102,400` 个分钟权益点、350 笔交易，保留 20,000 点：

- 2 次基准测量，约 **12.22 ms/op**、**3,403,776 B/op**、**3 allocs/op**。
- 仅采样函数；不包含策略执行、权益生成、数据库传输、Gzip 等，不能直接视为回测整体提速。
- 新版本后续不再生成完整 Chunk，主要价值在避免数据库持续膨胀；之前约 7.4 GiB 的历史 Chunk 空间**不会自动回收**。

## 部署和清理注意

- 代码修改只在本地工作树；需要重新编译和启动本地后端才能生效。
- **本轮未修改数据库现有数据、未重启服务、未清理旧 Chunk，也未修改生产库。**
- 后续若确认旧数据完全无保留需求，应单独制定按任务或全表删除及回收 InnoDB 空间的执行计划。不能把删除历史权益与本次代码优化混为一项自动操作。
