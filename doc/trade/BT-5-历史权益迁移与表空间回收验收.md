# BT-5｜历史权益 Preview 迁移与 Chunk 表空间回收验收

执行时间：2026-11-20 22:25–22:39（北京时间）  
数据库：仅本地 Mac Docker 容器 `db`（MySQL 5.6.51，hostname `be26c483fa43`），`go_bn_test`。  
**未连接、未修改生产服务器数据库，未修改应用数据库连接配置。**

## 操作前状态

| 项目 | 操作前 |
|---|---:|
| `agent_backtest_runs` 已完成 | 407 |
| `agent_backtest_equity_chunks` 实际行数 | 24,126 |
| Chunk 完整权益点数 | 782,461,080 |
| `agent_backtest_equity_preview` | 407 |
| 预览总点数 | 8,140,000 |
| Chunk InnoDB `DATA_LENGTH` | 7,578.55 MiB（统计） |
| Chunk 独立 `.ibd` 文件 | 8.5 GiB（文件实际） |
| MySQL `innodb_file_per_table` | 1 |
| 引用 Chunk 的外键 | 0 |
| 活跃回测 | 0 |

所有历史任务均为 `succeeded`，全部 407 条旧预览均存在，每条 20,000 点。

## 备份

清理前已通过本机 Docker MySQL `mysqldump --single-transaction --quick` 完整备份以下两张表（含结构及原始数据）：

- `agent_backtest_equity_chunks`
- `agent_backtest_equity_preview`

备份路径：

`/Users/zhz/Backups/go_bn_test_equity_20261120.sql.gz`（压缩文件约 6.0 GiB，实际 6,478,224,361 字节）

校验文件：

`/Users/zhz/Backups/go_bn_test_equity_20261120.sql.gz.sha256`

备份完整性：`mysqldump` 成功退出、`gzip -t` 通过、生成 SHA-256，清理前又执行 `shasum -a 256 -c` 返回 `OK`。

**保留备份文件，不自动删除。** 若将来需要恢复完整分钟权益，必须先安排服务维护窗口、评估备份后的数据库变更，然后执行受控恢复；不可直接覆盖正在使用的库。

## 旧数据迁移

使用与 BT-5 新回测完全相同的 `sampleImportantEquityPoints` 函数：

1. 按 run_id 和 chunk_index 读取全部旧 Chunk；逐块验证索引、时间连续性、`point_count`、CRC、解码数据与元数据相符。
2. 读取对应回测的开/平仓成交时间，采样首尾、全局极值、最大回撤峰谷、交易附近的权益、分区极值，并均匀补足，最多 20,000 点。
3. 对新预览重新编码并解码检查字节级/结构级一致性；在单个事务中覆盖对应旧预览。
4. 对新数据执行事务内数据库读回校验；全部完成才允许回收旧 Chunk。

执行结果：

`BT5_MIGRATION_COMPLETE runs=407 source_chunks=24126 source_equity_points=782461080 duration=3m41s`

所有 407 个回测均成功迁移；没有跳过、失败、丢弃任务，也没有修改任何交易事件、回测指标、数据哈希和原始交易。

迁移日志：`/Users/zhz/Backups/bt5_equity_migration_20261120.log`

## 表空间回收

完整迁移成功并通过备份 SHA-256 核查后，二次检查本地库身份、活跃任务为零、成功运行数 407、20,000 点预览数 407、Chunk 总行数 24,126、无外键引用，然后执行：

```sql
TRUNCATE TABLE go_bn_test.agent_backtest_equity_chunks;
```

这利用 `innodb_file_per_table=1` 重新创建空表文件，比逐行 `DELETE` 后执行 `OPTIMIZE TABLE` 更直接。

| 项目 | 操作后 |
|---|---:|
| `agent_backtest_equity_chunks` | **0 行** |
| Chunk 独立 `.ibd` 文件 | **112 KiB** |
| Chunk `DATA_LENGTH` | **0.02 MiB** |
| `agent_backtest_equity_preview` | **407 行** |
| Preview `DATA_LENGTH` | **105.52 MiB** |
| 宿主机可用磁盘 | 操作前约 51 GiB，操作后约 61 GiB（近似读数，非单操作精确归因） |

清理日志：`/Users/zhz/Backups/bt5_equity_shrink_20261120.log`

## 重要部署注意

操作时运行的本地后端进程 PID `25045` 仍在使用旧可执行文件（项目根目录 `go_binance_futures` 文件修改时间 18:44，早于 BT-5 代码修改）。**未主动中断或重启该服务**。

**务必在下一次新回测前重新编译并重启本地后端，使新的「只写 Preview、不写 Chunk」逻辑生效，否则旧进程仍可能向空 Chunk 表写入新数据。**

已验证新版后端可编译，构建验证产物：
`/tmp/go_binance_futures_equity_preview_buildcheck`。
正式部署前可重新编译并确认业务流程。本轮没有改变启动管理方式或触碰正式交易。

迁移操作仅一次性执行，临时迁移程序位于 `/tmp/s510_bt5_migrate_local_test.go`，日常构建不需要该程序。
