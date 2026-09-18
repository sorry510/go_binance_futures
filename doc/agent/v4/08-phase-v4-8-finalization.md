# Phase V4-8：Finalization

## 目标

完成 V4 收尾，确保新增验证链路适合长期个人使用，并保持现有系统简单。

## 收尾范围

- 清理 V4 开发过程中确认无用的重复代码和文案。
- README 与多语言说明同步。
- 系统看板只增加必要的 V4 任务/Shadow/Testnet 异常摘要；不增加大型统计中心。
- 检查 Scheduler 长期运行时不会重复创建相同 Shadow/Testnet 任务。
- 检查数据库索引和查询计划，避免 V4 聚合拖慢行情/交易主循环。
- 确认新数据清理策略，不允许日志清理误删 Backtest/Shadow/Testnet/Live 交易结果。
- 确认 ARM 环境兼容，不引入只支持 amd64 的依赖。

## 最终验证

后端：

```bash
go test ./...
go test -race ./service/historicalmarket ./service/backtest <V4相关包>
go build ./...
git diff --check
```

前端：

```bash
pnpm typecheck
pnpm build
```

随后按既有约定将前端 `dist` 同步到后端 `static`。

数据库 Schema 如有升级，只通过：

```bash
./go_binance_futures sync db
```

## 运行约束

- 不修改 `app.conf`。
- 测试结束后不留下运行中的测试进程。
- 不回滚工作区中其它未提交修改。
- 不新增长期占用大量磁盘的重复历史缓存。

## V4 Definition of Done

- 一个新策略可以一次测试多个常用币。
- 系统能发现最常见的过拟合迹象：交易频率下降、跨币种不稳定、跨时间不稳定、收益集中。
- 候选策略可以通过 Shadow 和 Testnet 做前向验证。
- AI 可以基于这些真实结果辅助提出下一轮少量改进假设。
- 用户能够在一个复盘入口分别比较 Backtest、Shadow、Testnet 和 Live。
- 最终是否用于真实交易始终由用户决定。
- V4 没有演变成复杂量化平台或多 Agent 自主交易系统。
