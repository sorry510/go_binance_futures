# CODE_REVIEW_06-7-ALL · 整改复评（2026-11-13）

后端：`feat/trade_spot`，整改起点 HEAD `2c7c2b8 feat: 06-7-all`。本次修改为**未提交工作区改动**，没有覆盖原始 [CODE_REVIEW_06-7-ALL.md](CODE_REVIEW_06-7-ALL.md)。
范围：只处理 ALL 报告指出的 F1～F6 与相关永久回归，保持个人使用、双业务开关、`binance.testnet` 环境切换的简化设计。

## 一、复评结论

| Finding | 处理状态 | 证据与剩余限制 |
| --- | --- | --- |
| **F1 · P1** 方案与运行时契约冲突 | **文档口径已修复**；不新增发布开关 | 按用户已明确决定保留 `testnet=false` 选择正式 Spot + 原有两个业务开关；主方案 §一、§3.4、§四及“仍未注入执行器”等历史描述已统一为当前实现。**生产部署/数据库/主网真单验收仍未完成**；不会将此文档修改解释为生产已获验收。未实现原审查建议的复杂持久化授权/append-only 审计（用户明确不要）。 |
| **F2 · P2** 预检 SQL 只认 `sq4` | **修复并通过真实临时 MySQL 8** | `^(sqt\|sqm\|sq4)` + Kline interval/UTC base36 + v4 SHA-256；`sqt/sqm` 有效 BUY PASS，旧 `sq4` 输出 `legacy_sq4_needs_review` 且 Gate FAIL，格式看似正确但 hash 被篡改亦 FAIL |
| **F3 · P2** 零 Fill 终态 SELL 永久锁死 | **修复并通过真实 Processor + SQLite** | 已知 `CANCELED/EXPIRED/REJECTED`，只有远端 `Reconcile` 确认状态不变、`myTrades` 与本地 Fill 都为 0，才允许**最多一次**具有独立稳定 `sqr` ClientOrderID 的补卖；部分成交、远端状态异常、重复取消仍 `ErrNeedsReview`。没有增加表、任务或开关 |
| **F4 · P2** 主网 3d / 时序证据 | **公开数据补齐** | 主网/测试网均以公开未签名 GET 取得 HTTP 200 原始 JSON；主网最新 3d OpenTime 为 `2026-11-13T00:00:00Z`，测试网为 `2026-11-12T00:00:00Z`；旧 `-2015` 与“真实 Testnet 未验”为发生在测试网密钥更新/真实 E2E 之前的历史快照，不是最新状态 |
| **F5 · P3** 运维/发布遗留 | **部分未闭环** | 零 Fill 取消订单已能自动恢复；复杂异常/部分成交仍需人工处理，目前**没有单独 UI/CLI 运维入口**；ARM/Linux 成套部署、生产库 Schema/P95、备份/回滚均未做 |
| **F6 · P3** WS 生命周期/真实长稳 | **已知限制** | 保留现有进程级 WS 重连和 REST 只读恢复，不额外造任务中心；真实 429/418、长期 WS 重连、非零手续费 dust、Mainnet 真实订单仍未实测，不冒称闭环 |

## 二、代码修改

- `feature/spot_quant_runner.go`：只对经交易所再次核实为终态且没有任何 Fill 的历史 SELL 允许最多一次补单；ClientOrderID 固定 `sqr` + SHA-256(position identity)；与此前已有 `sqe` 不冲突，默认 SELL 正常路径不变。
- `feature/spot_quant_runner_067all_review_test.go`：新增真实 `spotexecution.Processor` + 临时 SQLite 的四组回归：`CANCELED/EXPIRED/REJECTED` 零 Fill 正确补卖并最终 CLOSED；已部分成交拒绝补卖；远端状态变化拒绝；两笔终态取消防无限重试。
- `doc/trade/开发方案-06-7-4-生产只读预检.sql`：升级新前缀和 legacy 识别，保留原 SELECT/EXPLAIN-only 范围。
- `doc/trade/开发方案-06-7-现货量化自动策略执行链路.md`：修正旧阶段描述与当前统一接线矛盾的条款；不新增制度或业务开关。
- [MySQL 8 实测证据](CODE_REVIEW_06-7-ALL-MySQL8预检回归证据.txt) 和 [主网/测试网 3d 原始公开行情](CODE_REVIEW_06-7-ALL-Spot3d公开行情证据.json) 作为独立新文件保留；历史报告与历史证据不覆盖。

## 三、MySQL 8 隔离回归实测

每次创建自动删除的 Docker MySQL 8 实例与合成库，**不是任何用户或生产数据库**，先运行项目自身 `sync db` 建立 v25 Schema，然后使用真实预检 SQL 查询。

| 场景 | `filled_over_qty_gate` | `buy_client_id_gate` | 非法 BUY | 旧 `sq4` |
| --- | --- | --- | ---: | ---: |
| 250 `sqt` + 250 `sqm` 有效 ID | PASS | **PASS** | 0 | 0 |
| 将其中一条改成合法 `sq4` 旧前缀 | PASS | **FAIL** | 0 | 1 |
| 再修改一条 `sqm` 的 SHA-256 字符 | PASS | **FAIL** | 1 | 1 |

三组均符合预期，验证命令 EXIT=0；容器和临时 Schema 在退出时销毁。原始验证记录与临时输出明确标记非生产 P95。

## 四、真实公开行情补证

只访问两次 **GET /api/v3/klines?symbol=BTCUSDT&interval=3d&limit=2**，不携带签名或账户密钥：

- Spot Mainnet：HTTP 200；OpenTime = `2026-11-10T00:00:00Z`、`2026-11-13T00:00:00Z`。
- Spot Testnet：HTTP 200；OpenTime = `2026-11-09T00:00:00Z`、`2026-11-12T00:00:00Z`。
- 两个环境 3d 时点的真实差异已证实，策略继续使用选中环境自身的 Kline，不从 Unix 时间戳自行取整 3d。

## 五、Go 回归

- `./scripts/test-go.sh -short`：**73 个 tested packages PASS、0 FAIL、13 no-test**。
- 10 个关键包 `go test -race -count=1`：**10/10 PASS**。
- 06-7 全系列专项 + 新 `TestReview067ALL*`：`-count=5 -p=2` **PASS**。
- `go vet`（10 关键包）、主程序 `go build`、`git diff --check`：**PASS**。
- 独立干净 Git 副本（HEAD + 本轮新/改 Go 文件及 SQL，不含本地 app.conf/私人研究归档）的 `go test ./... -count=1`：**73 个 tested packages PASS、0 FAIL、13 个包无测试，EXIT=0**；临时副本运行结束销毁。

## 六、不可误解的发布边界

**代码链路已接线、Testnet 曾经完成真实自动交易闭环，不代表这次已向 Mainnet 发过单或已部署生产。** 生产 MySQL 8 v25 物理检查、实际负载 P95、ARM/Linux 部署/备份/回滚和主网 signed WS / BUY / SELL 尚未完成。当前既有程序/账户状态未被本次测试改变。

本轮**没有**触碰 `conf/app.conf`、现有数据库数据、服务器运行进程、Futures 逻辑、前端 `static`，也没有任何真实 Binance 主网订单、自动 commit/push。保存所有历史证据，单独补写本复评文件。
