# BT-6：1m 预取月度 ZIP 并发与缓存覆盖检查优化

日期：2026-11-20

## 本轮目标

1. 币安公开数据的 **月度 ZIP 最多 3 路并发下载**。
2. ZIP 解析、指标外数据导入和 MySQL 更新 **仍按月份顺序串行**，避免并行数据库写冲突。
3. 本地 **已完整覆盖** 的 1m 压缩月份通过单次元数据 SQL 快速跳过，避免逐月加载压缩 payload、重复 COUNT 查询。
4. 保留 SHA-256 完整性校验、REST 补缺、预取阶段进度、原有归档缓存清理及故障回退语义。
5. 不修改数据库结构、应用配置、生产数据库，前端接口无变更。

## 修改文件

- `service/backtest/prefetch_public_data.go`
  - `prefetchPublicMonthly1m`：开始时批量取得已有压缩月份覆盖状态。
  - 对未被压缩 Chunk 元数据覆盖的月份，使用原 `KlineRangeComplete` 做精确校验（包含旧 1m 行表）；将校验结果缓存供下游阶段复用。
  - 最多提前发起 3 个缺失月份 ZIP 下载；每消费一个归档才启动下一个，保证同时下载/待导入的归档数量有界。
  - 主 goroutine 顺序等待、解析 CSV、导入、校验；后台 goroutine **只执行 FetchArchive**。
  - 取消/错误退出时取消尚未完成的请求并等待下载 goroutine 退出。
- `service/historicalmarket/kline_1m_coverage.go`
  - `CompleteKline1mChunkMonths`：用 **一次 SQL** 查询指定币种各月 `month_start/start_time/end_time/point_count/encoding/compression`，不传输 `payload`。
  - 对时间范围、每分钟序号及数量进行元数据一致性验证，只有完全覆盖请求区间才允许快速跳过。
  - 缺失、部分覆盖或元数据异常，交给现有精确检查；不直接判定为空、不跳过 REST。
- `service/backtest/prefetch_parallel_test.go`
  - 真实本地 HTTP ZIP 下载与 SHA256 测试：6 个月中 1 个月已缓存，要求只有 5 次下载、最大并发 2–3、顺序入库完整、第二次预取无额外下载。
  - 校验损坏必须终止导入，不能静默回退 REST。
- `service/historicalmarket/kline_1m_coverage_test.go`
  - 部分月度压缩 Chunk 覆盖匹配、缺失分钟不可误认为完整、点数异常不可走快速跳过，以及空时间范围测试。

## 本地验证

- `go test ./service/backtest ./service/historicalmarket -count=3 -p=2`：通过。
- `go vet ./service/backtest ./service/historicalmarket`：通过。
- `go test -race ./service/backtest ./service/historicalmarket -run 'TestPrefetchMonthly|TestPrefetchPublicMonthly1m|TestCompleteKline1mChunkMonths' -count=1`：通过（macOS linker 有已知 `LC_DYSYMTAB` warning，但测试退出码为 0）。
- `go build -o /tmp/go_binance_futures_bt6_buildcheck .`：通过。
- `git diff --check`：通过。

## 性能预期和范围

- 已完成且有效的月度 Chunk：从逐月查询（含可能的 payload）优化为一次元数据查询。对于大量已缓存月份，数据库 I/O 会明显减少。
- 首次大量缺失：原本逐月下载，现在最多三个同时进行，可重叠网络等待和前一个月的 ZIP 解析/入库。下载上限固定 3，**不能保证整体耗时必然缩短 3 倍**：网络带宽、限速、ZIP 解析和数据库导入也会影响总时长。
- 已完成月份不再重复进行第二轮远端获取；未完整月份仍按原有 REST 逻辑补缺。
- 实际用户首次预取的分阶段耗时尚未采样；测试证明逻辑和上限，没有推断生产网络提速百分比。

## 部署说明

本轮仅修改本地代码，**未重启服务、未替换根目录正在运行的可执行文件、未操作数据库数据**。重新编译并重启本地后端后生效；不需要前端重新构建、MySQL schema migration 或修改 `app.conf`。
