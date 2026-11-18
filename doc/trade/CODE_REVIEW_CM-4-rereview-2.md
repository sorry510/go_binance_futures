# CODE_REVIEW_CM-4-rereview-2 · F1 修复后的第二轮复审

> 日期：2026-11-18。分支 `feat/m-coin-futures`，基线 `e6162bf feat: cm-3`。
>
> 原问题来自 [第一次独立复审](CODE_REVIEW_CM-4-rereview.md)。该报告的 F1 失败记录作为历史证据原样保留，本报告记录源代码修复与实际 Gate 结果。CM-4 只处理现有 COIN-M 独立账本的离线/只读计算；没有进行生产迁移、交易或真实钱包读取。

## 一、最终结果

| Finding | 原级别 | 本次处理 | 状态 |
| --- | --- | --- | --- |
| **F1：跨 symbol / positionSide Fill 被计入另一 Managed 周期** | **P1** | 修改 `service/coinmquant/ledger_summary.go` 中 `fillEvidence` 投影与 `loadFillEvidence` 校验；新增永久回归探针 `service/coinmquant/ledger_summary_rereview_test.go` | **已修复：11 个恶意/损坏证据场景均通过，重复 5 次** |
| O1：前端混入其他功能的未提交改动 | P2 | 不改动 Lead、登录或 U 本位并行开发文件；CM-4 的源码提交应仅选择相关文件。重新构建 `dist` 会包含当前整个前端工作区，这是发布前需单独处理的文件归属问题 | **未改业务逻辑；提交隔离要求保留** |
| O2：受控交易页文案仍称 CM-3 | P2 | 仅修改 `locales/zh-CN.yaml`、`locales/en.yaml` 的 `coinMPage.controlled.notice`，统一写为 CM-4，只读操作边界保持不变 | **已修复** |
| O3：主开发方案提前标记 CM-4 完成 | P3 | 第一次复审当时状态确实应视为阻断；F1 修复并完成本轮全部代码 Gate 后，允许恢复「CM-4 代码/离线范围完成」的状态，不能表述成实盘验收 | **经本轮复审恢复通过** |

## 二、F1 根因与代码修复

原 SQL 只要求 `f` 与 `o` 的 `position_id / symbol / position_side / side` 对应，而没有进一步要求二者与目标 Managed 周期 `p` 对应。即便同一个 `position_id` 被错误赋给其他合约的订单和 Fill，只要张数凑齐，算法仍可能返回 `verified` 和非空总收益。

修复后，查询同时投影：

- Fill 的 `symbol`、`position_side`、`side`、`position_id`；
- Order 的 `position_id`、`symbol`、`position_side`、`side`、`intent`；
- Managed 周期的 `id`、`symbol`、`position_side`。

逐笔校验要求 **`f.symbol = p.symbol = o.symbol`，`f.position_side = p.position_side = o.position_side`，`f.position_id = p.id = o.position_id` 且 `f.side = o.side`**；随后仍验证 LONG/SHORT 与 OPEN/CLOSE 的 BUY/SELL 方向、Fill 结算资产、合约张数和价格。

**为什么没有直接把这四个等式全部写入 SQL JOIN 条件？** 若将错误数据在 JOIN 时直接过滤，坏 Fill 可能悄然消失，无法确定错误属于缺失成交还是篡改归属。当前使用 `LEFT JOIN` 保留关联订单不存在或不匹配的证据，在 Go 层显式校验，发现问题立即设置 `agg.bad`。因此不是只修正 JOIN 的表面匹配，而是能把错误证据准确标记为 `invalid`。

对于 F1 场景，最终 `PositionSummary.PnLStatus=invalid`，未实现和完整总收益均为 `null`，对应 `AssetTotal.Invalid=1` 且汇总总收益为 `null`；不能用部分看起来正常的 Fill 产生 `verified` 盈亏。

## 三、永久回归探针（隔离 SQLite）

新增 `TestCM4CrossSymbolAndPositionSideFillMustBeRejected`，每组先构造 BTCUSD_PERP / LONG、开仓 10 张、剩余 5 张的**有效基线**并确认 `verified`，再注入以下数据异常：

| 组别 | 注入故障 | 断言 |
| --- | --- | --- |
| 1 | 关联 Order 和 Fill 均改成 ETHUSD_PERP | `invalid`，无总收益 |
| 2 | Order 和 Fill 均改成 SHORT | `invalid`，无总收益 |
| 3 | Order 和 Fill 同时改成 ETHUSD_PERP / SHORT | `invalid`，无总收益 |
| 4 | ETHUSD_PERP / SHORT 的 OPEN SELL、CLOSE BUY **内部完全自洽**，但错误指向 BTC/LONG 周期 | `invalid`，无总收益 |
| 5 | 只有 Fill 的 symbol 错误 | `invalid`，无总收益 |
| 6 | 只有 Order 的 symbol 错误 | `invalid`，无总收益 |
| 7 | 只有 Fill 的 positionSide 错误 | `invalid`，无总收益 |
| 8 | 只有 Order 的 positionSide 错误 | `invalid`，无总收益 |
| 9 | Order 与 Fill 的 BUY/SELL 不一致 | `invalid`，无总收益 |
| 10 | Fill 指向已不存在的 Order | `invalid`，无总收益 |
| 11 | Order 的 position_id 被重写为另一个 Managed 周期 | `invalid`，无总收益 |

除每行状态外，还明确断言资产层 `Verified=0`、`Invalid=1`、`TotalPnL/TotalUSD/RealizedUSD/UnrealizedPnL = null`，防止页面仍汇总错误利润。

## 四、本机实际 Gate

| Gate | 实测 |
| --- | --- |
| 永久 F1 探针 `go test -run '^TestCM4CrossSymbolAndPositionSideFillMustBeRejected$' -count=5` | **PASS** |
| `go test -count=5 -p=2 ./service/coinmquant ./feature/api/binance ./controllers` | **PASS** |
| `go test -count=1 -p=2`（排除历史 `strategy_templates/research` 的全部 87 个 Go 包） | **PASS** |
| 同一 87 包 `go vet` | **PASS** |
| `go test -race -count=2 -p=2 ./service/coinmquant ./feature/api/binance ./controllers` | **PASS**；macOS `LC_DYSYMTAB` linker 警告非致命 |
| `go build -o /tmp/go_binance_futures_cm4_rereview_v27 .` | **PASS**；构建在 `/tmp`，不覆盖当前运行程序 |
| 前端 `pnpm typecheck` | **PASS** |
| 前端 `pnpm build` | **PASS** |
| `rsync dist/ → 后端 static/` 与 `diff -qr` | **PASS**，静态目录一致 |
| 前后端 `git diff --check` | **PASS** |

注意：当前后端工作区原有大量 `static/` 哈希命名文件替换，来自前端打包，不代表这些文件已经提交。前端还存在 Lead、登录等**并行修改**；本轮没有修改、清理或提交这些独立工作。

## 五、继续阻断的真实部署/交易范围

本轮只证明业务源码、离线 SQLite 和 Mock 环境的正确性，并不表示完成：

- 实际 MySQL 8 上 v27 迁移、备份及唯一索引校验；
- 真实 DAPI 签名钱包/账户和私有 User Data WS；
- `/dapi/v1/userTrades` 真实 Fill 采集、双向持仓和退出对账；
- COIN-M Demo/Testnet 真实开平仓、撤单、Algo 条件单、失败恢复；
- Mainnet 实盘与 ARM Linux 发布验收。

这些 Gate 在 CM-5～CM-7 按原计划验证，**不得以本轮 Gate 通过作为直接执行真实订单的许可**；没有新增复杂审批系统或第三个交易开关。

## 六、复审裁定

**F1/P1 证据归属漏洞已修复，原第一次复审的阻塞在代码和离线验证范围内关闭。CM-4 可以重新判定为「代码与离线测试范围通过」，具备进入 CM-5 开发的条件。**

这不是生产迁移通过、真实 Fill 已接入或允许币本位自动交易的声明。

本次没有执行 `sync db`、没有修改 `app.conf`、没有操作生产数据库、没有发送任何 Binance 订单，也没有 Git 提交。
