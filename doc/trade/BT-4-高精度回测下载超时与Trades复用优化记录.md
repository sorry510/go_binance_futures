# BT-4｜Adaptive 回测下载超时分析与 Trades 复用优化记录

日期：2026-11-20
状态：本地代码已修改，待用户用相同数据窗口复测；不代表已达到某一提速倍数。

## 1. 最新失败任务

- 回测任务：`bt_33e9c080bed2979cbe7aa22e0e0ca471`
- 策略：`S510-20261118-750 quality_frequency_highfreq_parentlong_compression_short`
- BTCUSDT，`adaptive_intrabar_v3` / `backtest_engine_v11`，约 2025-01-01～2026-11-19
- 运行约 38 分 37 秒后标记为 `failed`。
- 本地 DB 原始错误：
  ```text
  1s archive unavailable (binance public data archive not found:
  https://data.binance.vision/data/futures/um/daily/klines/BTCUSDT/1s/BTCUSDT-1s-2025-01-18.zip)
  and trades fallback failed:
  download https://data.binance.vision/data/futures/um/daily/trades/BTCUSDT/BTCUSDT-trades-2025-01-18.zip
  failed after retries: context deadline exceeded
  ```
- **直接故障原因**：日级 Trades 下载网络请求超时；不是策略运算出错，也不是客户端自行设置了约 40 分钟的整个回测截止时间。1s 不可用时走 Trades 回退属于既有预期路径，不应跳过。
- 原 `PublicDataClient` 默认 `http.Client.Timeout=5m`；重试仍受单次请求超时影响。未核实当时具体发生的是代理阻塞、传输带宽不足还是上游响应慢，不作超出证据的归因。

## 2. 性能根因与证据

- 同一 BTC 2026-01～2026-11 Adaptive 任务：冷缓存约 60.56 分钟，下载 68 份 / 2.28GB；热缓存分别约 33.15 分钟和 59.35 分钟，均下载 5 份 / 195MB。
- 这些任务均有约 1,759 个下钻分钟及 85,085 个 Trades 下钻秒，因此不能把整体时间差直接等同于下载耗时。
- 当前的 `SecondBars` 需要从日级 Trades 合成 1s 时，已解压并解析一个候选分钟的成交；随后同一分钟每秒再次 `Trades`，造成 SQL 查表与 ZIP 扫描重复。
- 离线扫描本地 ZIP（不下载、不查数据库）：71MB BTC 日包在 18:34 位置解析整分钟约 984ms，紧接着请求该分钟第一秒重扫约 2,249ms。证明原方案中存在可观的非网络重复开销。

## 3. 已完成的本地代码变更

### 3.1 P0：分钟级 Trades 复用

位置：`service/historicalmarket/resolution_provider.go`

- 只在独立 Adaptive Provider 实例内保留**一个候选分钟**的已验证、按 `TradeTime/TradeID` 排好序的成交。
- 启用条件：当分钟完整对齐、文件经过原有 SHA 校验、`writeSparse=false`、该分钟 MySQL 稀疏 Trades **一行都没有**。若存在历史稀疏数据，保持原查询优先级，不擅自绕过数据库。
- 次级逐秒请求用二分定位范围、复制子切片，不重复读 SQL、不重新打开 ZIP、不重解压 CSV；空秒区间返回正常的空数组/原有证据。
- 对每次逐秒请求重新生成与原实现相同的 `source_ref`（含范围）、`ResolutionEvidence` 与 `tradeEvidenceHash`，继续逐笔执行原规则，不跳过同秒歧义、资金费或交易判定。
- 进入下一候选分钟或关闭 Provider 时丢弃上一个分钟帧，不常驻全年的历史 Trades。
- 原 `trade_cache_hits` 纳入分钟内存命中，同时增加更清晰的 `minute_trade_reuse_hits`。

### 3.2 下载超时配置

位置：`service/historicalmarket/resolution_provider.go`、`conf/app.conf`、`conf/app.conf.example`

```ini
[binance]
public_data_download_timeout_minutes = 10
```

- Adaptive 高精度归档网络请求默认从原先 5 分钟放宽到 **10 分钟**，配置正数支持覆盖，最大限定 60 分钟。
- **保留原来的重试机制、SHA256 检验、临时文件清理及失败即停止**；这只提高慢速请求容忍度，不保证网络长时间不可用时回测必然成功。
- 没有改动数据库配置。当前本地 `public_data_cache_max_gb=40` 保持原样，不覆盖用户的设置。

### 3.3 P2：细分统计（无需新表）

`resolution_stats_json` 后续新增：

| 字段 | 含义 |
|---|---|
| `minute_trade_reuse_hits` | 逐秒 Trades 直接使用已解析分钟帧次数 |
| `archive_fetch_ms` | 成功获取归档的累计耗时（命中缓存 + 网络，包含校验等） |
| `archive_download_ms` | 其中实际下载并校验成功的归档累计耗时；为上个字段的子集，**不能再加一遍** |
| `archive_parse_ms` | 日级 ZIP/CSV 读取、解析的累计耗时 |
| `sparse_query_ms` | 1s/Trades 稀疏 SQL 查询累计耗时 |
| `zip_scanner_opens` | ZIP/CSV 扫描器打开次数 |
| `zip_scanner_rewinds` | 相同 ZIP 因请求时间倒退而重新从头扫描的次数 |

统计是回测成功时随原有 `resolution_stats_json` 写入的，不创建数据库表。HTTP 失败的尝试不计入成功下载耗时，完整计算剩余部分仍需结合 `timing.engine_ms`，不能把某个残差武断称为纯策略 CPU 时间。

## 4. 回归验证

- 新增 `service/historicalmarket/bt4_minute_cache_test.go`，验证：分钟 1s 生成、逐秒及空秒范围、同秒多笔及排序、逐秒证据哈希、仅下载一次、跨分钟释放、返回数据不可污染缓存、取消操作。
- 增加 Adaptive 下载超时参数测试（缺省 / 自定义 / 边界）。
- `go test ./service/historicalmarket ./service/backtest -count=3`：通过。
- `go vet ./service/historicalmarket ./service/backtest`：通过。
- `git diff --check`：通过。
- `go test -race ./service/historicalmarket -run 'TestBT4MinuteTradesReuse|TestBT3Verified|TestAdaptiveArchiveDownloadTimeout' -count=1`：通过（macOS 链接器提示一次非致命 LC_DYSYMTAB warning）。
- `go build -o /tmp/go_binance_futures_bt4_buildcheck .`、`go build ./service/historicalmarket ./service/backtest`：通过，正式可执行入口编译成功，没有覆盖正在运行的二进制。
- `go build ./...`：被现存 `research_specs/20261119-web-robust-short-009-012/tools` 目录中混用 `backtest` / `historicalmarket` package 阻断；与本次代码无关，未修改研究目录。

## 5. 下一步验收

1. 编译并重启本地后端后，新逻辑才生效；不可热修改当前运行中的旧二进制。本轮未主动重启服务、没有改生产库。
2. 用**完全相同**的 BTCUSDT 策略、2026-01～2026-11 窗口跑一次 Adaptive，对比原 61 笔交易、事件、权益、净收益、手续费、资金费、DataHash 与精度证据。
3. 分别记录无缓存和热缓存条件下下载次数/字节、`archive_download_ms`、`archive_parse_ms`、`sparse_query_ms`、`zip_scanner_rewinds`、`engine_ms`。
4. 若真实收益、逐笔成交或 DataHash 不一致，应定位差异后再继续优化，不能用“总收益接近”作为验收。
5. 第二阶段按上述计数决定是否要增加**小并发的按需预下载**；不立即引入新的 DB、Parquet、DuckDB，也不全量预下载一年 Trades。

备注：此文中的 71MB ZIP 微基准是旧路径的局部样本，不代表新版本完整回测已经实测获得相同幅度加速。
