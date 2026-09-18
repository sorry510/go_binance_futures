# Phase V4-0：Baseline

## 目标

冻结当前 V3 + 回测优化后的稳定状态，给 V4 后续改造建立回归基线。

## 范围

- 记录当前数据库 Schema Version、Backtest Engine Version 和关键行为。
- 固化现有 Historical Backtest 的确定性 Fixture。
- 固化 1m chunk 读取、Listing Boundary、Funding Boundary 和 Adaptive Resolution 的关键测试。
- 固化 Controlled Trade / Ownership / Testnet 的现有安全行为。
- 保存一组固定 Strategy Template + Symbol + 时间范围的基准回测结果。
- 记录批量回测改造前的单次回测耗时和内存占用，供后续比较。

## 验收 Gate

- `go test ./...` 通过。
- 相关 Backtest/Historical Market race test 通过。
- `go build ./...` 通过。
- `git diff --check` 通过。
- 前端 `pnpm typecheck`、`pnpm build` 通过。
- 不修改 `app.conf`。
- 数据库升级仍只允许通过 `./go_binance_futures sync db`。

## 本阶段不做

- 不新增业务功能。
- 不修改策略执行语义。
- 不调整真实交易逻辑。
- 不为 Benchmark 新建长期运行服务。
