# CODE_REVIEW_06-7-4：Testnet 与交付 Gate

日期：2026-11-13
后端：`feat/trade_spot`，HEAD **`3e1a79f feat: 06-7-3`**；**06-7-4 全部改动为未提交工作区**
前端：本地 HEAD `cf319e1 feat: spot`，本轮未修改前端
主方案：[开发方案-06-7-现货量化自动策略执行链路.md](开发方案-06-7-现货量化自动策略执行链路.md)（§4/§5/§6）
阶段记录：[开发方案-06-7-4-测试网与交付Gate-实施记录.md](开发方案-06-7-4-测试网与交付Gate-实施记录.md)
新增交付物：[开发方案-06-7-4-生产只读预检.sql](开发方案-06-7-4-生产只读预检.sql)
前置报告：[CODE_REVIEW_06-7-3.md](CODE_REVIEW_06-7-3.md)、[CODE_REVIEW_06-7-3-rereview.md](CODE_REVIEW_06-7-3-rereview.md)、[CODE_REVIEW_06-ALL.md](CODE_REVIEW_06-ALL.md)（均保留原文）

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 代码与离线 Gate | **✅ PASS（合格研发基线）** |
| 06-7-3 整改前置准入 | **✅ 属实**（F1/F2/F6 已随 `3e1a79f` 进入 HEAD，F3/F4/F5/F7 保留发布边界） |
| 06-7-4「离线准备项」 | **✅ 认可**：严格 Fake Spot 环境 E2E、3d 拒单保护、只读预检 SQL、dist/static 核对均已落地 |
| **06-7-4 阶段目标（Testnet 与交付 Gate）** | **❌ 未达成**：主方案 §4 要求修复 `-2015` 并在独立测试账户验证真实买卖/WS/REST/手续费/dust，复核生产 Schema 与回滚；本轮真实 Testnet 签名账户仍 `-2015`，生产 Gate 0 项通过 |
| 06-7-4 整体验收 | **⚠️ 有条件通过（仅离线准备项）/ ❌ 不认可「06-7-4 已验收」**；新增 F1 为 P1 条件触发缺陷，须在真实执行器放行前闭环 |
| 「06-7 整体完成」/「方案 06 整体完成」 | **❌ 不认可** |
| 真实自动下单、Testnet、Mainnet、生产发布、生产 v25 预检、回滚、异常恢复运维入口 | **⛔ 全部 BLOCKED** |

一句话：**本轮把 06-7-3 遗留的「严格行情环境」补成了真正的全链路测试，并诚实保留了 Testnet/生产 Gate 的 BLOCKED 状态；但 06-7-4 的阶段定义本身就是「真实 Testnet 与交付验收」，核心 Gate 一项未过，因此不能判阶段完成。同时新引入的 3d fail-closed 门禁被错误地套在 SELL 分支上——探针实证：已有 OPEN 仓位在退出信号成立时也被拦住，仓位保持 `OPEN`、不下卖单，注入权威 3d 来源后立即恢复 SELL/CLOSED。这是「保护新开仓」误伤「平仓退出」的资金安全缺陷。**

---

## 一、审计范围与基线

### 1.1 工作区状态（实测）

```
 M doc/trade/开发方案-06-7-现货量化自动策略执行链路.md
 M doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md
 M doc/trade/开发方案-总览与实施顺序.md
 M feature/spot_quant_runner.go
 M feature/strategy/line/spot_rule_env.go
?? doc/trade/开发方案-06-7-4-测试网与交付Gate-实施记录.md
?? doc/trade/开发方案-06-7-4-生产只读预检.sql
?? feature/spot_quant_runner_0674_integration_test.go
```

`git log -8`：`3e1a79f feat: 06-7-3` → `1845572` → `5c1d4fa` → `32d203b` → `ed31061` → `35da095` → `5eb10a2` → `f2f3c9b`。`3e1a79f` 的 `git show --stat` 为 10 files / +1403 行，**包含 `CODE_REVIEW_06-7-3.md`、`CODE_REVIEW_06-7-3-rereview.md`、881 行 0673 集成测试与 v4 clientID 整改**。

`git diff --stat` 本阶段为 5 files / +94 −14（另加 220 行新测试、10003 字节实施记录、3320 字节 SQL）。**未修改 `main.go`、前端、`static/`、`conf/`、DDL/Schema、路由、控制器**；无新表、新开关、独立 worker、任务中心、模式/Owner/审计体系，符合方案 §6「明确不做」。

### 1.2 隔离副本与等价校验

`/tmp/phase_audit_0674` = `git archive HEAD` + 按 `git status --porcelain -z` 同步全部 **8 个 M/?? 路径**（`DELETED 0`）；副本 `conf/` 仅 `app.conf.example`，不含 `strategy_templates/` 研究目录。

```
WORKSPACE_GO_FILES 675 UNTRACKED_GO 1
GO_FILES_IDENTICAL 675 MISSING 0 DIFFER 0 COPY_ONLY 0
```

### 1.3 离线 Gate 实证（隔离副本，显式清空许可 env）

| Gate | 命令 / 结果 |
| --- | --- |
| 06-7-4 专项 | `go test ./feature -run '^TestStage0674' -count=1` → **PASS**；`-list` 列出 **3 个顶层测试**（StrictSpotMarketEnvironmentThroughRealProcessor / ThreeDayCandleRequiresExchangeOpenTime / ThreeDayStaleSourceFailsClosed） |
| 重复稳定 | `^(TestStage0674\|TestSpotQuant0673\|TestReview0673\|TestSpotQuant0672\|TestReview0672)` `-count=5 -p=2` → **PASS** |
| 06-7-1 整改回归 | `line` + `spotquant` 专项 `-count=1` → **PASS** |
| 10 包 test | 显式 10 包全列 `-count=1` → **10/10 ok** |
| 10 包 race | `-race` 同 10 包 → **10/10 ok，无 DATA RACE** |
| vet / build | `go vet` 10 包、`go build -o /tmp/spot0674_review_build .` → **EXIT=0** |
| diff-check | 工作区 `git diff --check` → **EXIT=0** |
| 干净副本全量 | 探针删除后 `go test ./... -count=1` → **73 ok + 13 无测试包 + 0 FAIL，EXIT=0** |
| 项目脚本 | `./scripts/test-go.sh -short` → **EXIT=0** |

> 边界：全量 EXIT=0 沿用 06-ALL-rereview 的 `tests/default_test.go` + `controllers/testdata` 兜底；**用户工作区含 `strategy_templates/research/**` 的全仓测试仍不得写成 PASS**。新实施记录 Markdown 用两个尾空格作硬换行，`git add` 后 `git diff --check` 会报 trailing whitespace（项目既有文档惯例，非功能缺陷）。

### 1.4 06-7-3 整改独立复核（前置准入）

逐项核对 `CODE_REVIEW_06-7-3-rereview.md` 与 `HEAD 3e1a79f`：

| 声明 | 独立证据 |
| --- | --- |
| F1 双时钟根治 | HEAD `runner.go` 中 `spot_quant_0673_v4` digest、`spotStoredBUYEvent` 可逆解码（34 字符 / `sq4` + 2 位周期码 + 7 位 base36 UTC 秒 + 22 位哈希）；`hasBUYInEvent` 已完全删除 `created_at` 范围条件，改为读取历史 BUY 订单比对事件区间；`oldOpen.After(now)` 与不可解析 ID 均返回 `ErrNeedsReview` |
| F2 超成交静默空转 | HEAD `query.go` 保留 `FILLED AND filled_qty <> requested_qty`（补迟到 myTrades 必需），HEAD `runner.go` 的 `reconcileUnsettled` 对 `filled > requested` / 非法数量追加 `ErrNeedsReview` 并记录币种/订单号 |
| F6 双币真实 Processor | `3e1a79f` 提交的 881 行 0673 集成测试含两币 Fake Exchange 规则/成交隔离用例 |
| F3/F4/F5/F7 | rereview 明确列为「未覆盖 / 待 Testnet / 待 MySQL8 / 保守拒绝」，未在 HEAD 中虚报闭环 |

**结论：前置准入属实**，06-7-4 可以在此基线上审查。

---

## 二、方案契约核对

### 2.1 主方案 §4「06-7-4 完成标准」

| 完成标准（原文） | 核对 |
| --- | --- |
| 修复 Spot Testnet 凭据 `-2015` | **❌ 未达成**：实施记录 §2 明写签名 `GET /api/v3/account` 返回 HTTP 401 / `-2015`，原始响应未留证 |
| 另行授权在独立测试账户验证真实买卖、WS/REST 恢复、手续费/dust | **❌ 未达成**：仅 Fake Exchange + 临时 SQLite；实施记录 §2 自述「未调用 `POST /api/v3/order`、真实 `DELETE` 取消、真实 WS 账户订阅」 |
| 复核 MySQL 8 / 生产 Schema | **⚠️ 部分**：临时 Docker MySQL 8.0.46 合成 500 周期 `sync db` / `EXPLAIN` 已声明通过（**容器已删、无原始输出留证，且明确不是 P95**）；生产库物理 Schema 未核验，06-2 检查单 §2 七项 **0 勾选** |
| 构建和回滚 | **❌ 未达成**：本地 Darwin/arm64 临时二进制（SHA 可复核），无 Linux/ARM 生产构建物、无成套发布 SHA 组合、无回滚演练 |
| 验收底线：未有真实 Testnet 证据与单独部署授权前，不解除 Mainnet 订单硬限制、不启动主网自动交易 | **✅ 遵守**：主网硬拒保持，`StartSpotQuantTrade` 仍空转 |

**按阶段定义，06-7-4 的核心交付物是「真实 Testnet 与生产验收证据」，本轮 0 项达成；实际完成的是「离线准备与诊断工具」。** 主方案状态行、§4 阶段表、实施记录综合状态与总览新增段落均如实写明 BLOCKED，未虚报实盘，这一点值得肯定；但「阶段完成」仍不能判定。

### 2.2 06-7-3 rereview 移交事项

| 移交项 | 06-7-4 处理 | 判定 |
| --- | --- | --- |
| F3 严格 Spot 环境进 E2E | `LoadVerifiedSpotRuleEnvWithSource` + 3 个 0674 顶层测试；核对代码确认与生产共用 `buildVerifiedSpotRuleEnv` + `verifySpotKlineRows`，无测试专用绕过 | **✅ 闭环（Fake 层）**；真实行情链仍 BLOCKED |
| F4 实际 3d Kline OpenTime | 公开行情实测两环境锚点不同；`spotCandleOpen("3d")` 改为返回 zero，改由 `spot3dOpen` 权威来源提供，缺源/过期 `ErrNeedsReview` | **⚠️ 部分闭环**：不再错误猜测，但生产零注入点 → 3d 自动交易实为**永久禁用**（且见 F1） |
| F5 MySQL 8 EXPLAIN/P95 | 临时合成库 4 条 `EXPLAIN ANALYZE` 数值（0.02~0.47 ms） | **⚠️ 仅兼容性证据**：记录自述「不是 P95」「不代表生产历史量」；生产 P95 仍 BLOCKED |
| F7 历史 v2/v3 clientID | Runner 对不可解析 ID fail-closed；预检 SQL 新增 `legacy_buy_ids` 统计 | **⚠️ 口径不一致**，见 F5 |

### 2.3 §5 最小测试清单

本轮新增用例覆盖 #1（严格环境失败即停）、#2/#4（BUY→SELL→CLOSED 后同 K 线不再 BUY）、#6/#10（Fake fail-closed 与状态区分）。#3（部分成交）、#5（双币隔离）、#7（Futures 不受影响）、#8（全局 OFF 不丢 Fill）沿用 06-7-3 已提交回归，本轮未新增真实 Processor + 严格环境组合用例。

---

## 三、正向结论（已实证）

1. **严格 Spot 环境确为生产同一套校验**：`LoadVerifiedSpotRuleEnvWithSource`（`spot_rule_env.go:36-54`）注入 loader 后仍逐行调用 `verifySpotKlineRows` 检查 OHLC 有效性、`CloseTime >= OpenTime`、最新一根新鲜度（窗口 = 自身周期 + 5min），再交给 `buildVerifiedSpotRuleEnv` 校验 ticker 5 分钟新鲜度、24h OHLC、`ValidateTechnologyConfig`、指标样本完整性与 `spotIndicatorMinimumBars`。**没有「测试专用 bypass」**。
2. **3 个 0674 顶层测试有效**：ticker 过期 8 分钟 / MA5 仅 3 根 Kline / 5m Kline 过期 30 分钟均不 BUY；浮盈触发 `close_long` 时 ticker 过期也不 SELL；行情恢复后真实 `spotexecution.Processor` 完成 BUY→SELL→CLOSED，后续 5 轮不重复 BUY。3d 无权威源与过期源（11-09 open vs 11-13 事件）均返回 `ErrNeedsReview` 且零 POST；Testnet/Mainnet 两个不同 3d 锚点产生不同 clientID。
3. **预检 SQL 确为纯只读**：静态解析 12 条语句，首词全部为 `SELECT` / `EXPLAIN`，`write_statements=0`、`ddl_dml=False`、无 `EXPLAIN ANALYZE`；覆盖版本/双开关/4 表/10 DECIMAL/索引/订单异常/历史 BUY ID/4 条非执行式 `EXPLAIN`。
4. **生产零接线保持**：`StartSpotQuantTrade` 函数体仍只有 `return ctx.Err()`；`main.go:449-455` 仅 2 秒回调 + `RunSpotQuantProtected`；`git grep` 显示 `NewSpotQuantRunner` 与 `spotexecution.NewProcessor` **全部调用点只在测试文件**，`spot3dOpen` 全仓仅 runner.go 定义/使用与 0674 测试赋值。打开双开关不会发真实订单。
5. **可独立复核的外部证据**：前端仓库 HEAD `cf319e1 feat: spot` 且工作区干净；`dist`↔`static` **115/115 相对路径 SHA256 全等**，`index.html` = `612b3b1884c6f6d13a1ed5995eeef09765d66b780523a4d843f38df664b6b600`（与实施记录一致）；`/tmp/spot0674_current_build` SHA256 = `c154b6873eca7f96719594f56448239e8610de46840e6e05069739d6d87ad36c`（与记录一致）；`mysql:8.0` 镜像存在且 `docker ps -a` 无 8.0 容器残留（记录「容器已删」属实）；`/tmp/spot0674_app.log` / `_clean_all.log` / `_race.log` 留有测试输出。
6. **06-ALL F4 有进展**：前端源码已从「未提交」变为已提交 `cf319e1` 且 dist/static 成套一致；但见 F7，成套发布 SHA 组合仍未闭环。

---

## 四、Findings

### F1 · P1（条件触发）· 3d fail-closed 门禁误伤 SELL：已有 OPEN 仓位无法自动平仓

`runSymbol` 的 SELL 分支（`runner.go:463-470`）与 BUY 分支共用 `resolveSpotEventOpen`。当 `spotSignalInterval(coin)` 返回 `3d`（币的 `technology` 启用 3d 指标，或规则引用 `kline_3d`），且 `r.spot3dOpen == nil` 时，函数在**退出信号已经成立之后**返回 `ErrNeedsReview`，`ExecuteSell` 不会执行。

生产构造路径 `NewSpotQuantRunner`（`:67-73`）不设置 `spot3dOpen`；`git grep` 全仓显示该字段只在 0674 测试中被赋值。因此一旦真实执行器被放行，**任何含 3d 配置的币永远无法自动平仓**；`ReportSpotQuantScheduler` 按错误文本去重日志，运维侧看到的是「静默持有」。触发路径包括：持仓期间把指标配置改为 3d、模板切换、以及配置本来就是 3d 而仓位由早期周期建仓。

隔离探针（真实 Processor + 真实 SQLite Ledger，不改写 DB 时间、不联网）：

```
PROBE0674_3D_EXIT err=BTCUSDT: spot order needs reconciliation: 3d Spot Kline OpenTime requires an authoritative market source needs_review=true places=1 position_status=OPEN remaining=0.990000000000000000
PROBE0674_3D_EXIT_CONTROL err=<nil> places=2 position_status=CLOSED
PROBE0674_PROD_WIRING runner_3d_source_nil=true start_err=<nil> exec_nil=false
PROBE0674_3D_END delta=72h0m0s   （排除 spotEventEnd 缺 3d 的误判）
```

对照实验证明阻断**完全由 3d 门禁引起**：注入合法权威来源后 SELL 立即完成并 `CLOSED`。这与 `CODE_REVIEW_06-7-3` F2「静默空转」同类，但方向更严重——**不能开仓只是错过机会，不能平仓是把已持有仓位锁死**。设计意图（rereview F4 / 实施记录 §1）是「防 3d 事件起点猜错导致重复 BUY」，SELL 的幂等身份本就由 `position_id` + clientID 保证，并不依赖 Kline OpenTime。

**要求（真实执行器注入前必须闭环）**：把 `resolveSpotEventOpen` 的 3d 校验限定在 BUY 事件识别；SELL 分支改用不依赖 3d OpenTime 的稳定事件身份（或复用已有 BUY 订单解码出的事件区间）；补「已验证 OPEN 仓位 + 3d 配置 + 退出信号成立 + 无权威源」的永久回归，断言必须发生 SELL 且 `CLOSED`。同时复核 `hasBUYInEvent` 对 3d 历史订单的区间比较（探针已证 `spotEventEnd(...,"3d") = 72h`，本身无误）。

### F2 · P2 · 阶段目标是真实 Testnet / 交付 Gate，本轮 0 项达成，不得判「06-7-4 完成」

见 §2.1。主方案 §4 把 06-7-4 定义为「修复 `-2015` 后另行授权验证真实买卖、WS/REST 恢复、手续费/dust；复核 MySQL 8/生产 Schema、构建和回滚」，实施记录 §6 自列 5 条硬 Gate 全部未满足。**本轮实际完成的是离线准备项**：严格 Fake 行情 E2E、3d 拒单、临时合成库 EXPLAIN、只读预检 SQL、静态资源核对。

文档状态行（主方案、06 主方案、总览、实施记录）均写明 `-2015` 与 BLOCKED，属诚实表述；但「06-7-4 离线预检已验」的措辞容易被链路上层读成阶段完成。**要求**：在阶段表/总览中把状态固定为「06-7-4 离线准备完成，阶段 Gate 未验收」，并把 `-2015` 修复、真实 Testnet 凭据与独立书面授权列为唯一解锁条件。

### F3 · P2 · 关键外部证据不可独立复核，摘要缺「临时/非 P95」限定语

| 声明 | 可复核性 |
| --- | --- |
| Testnet 签名 `GET /api/v3/account` → HTTP 401 / `-2015` | **不可复核**：无原始响应、状态码、时间戳、request id 或脱敏日志留证 |
| 公开 3d Kline 实测（Testnet 11-06/09/12、Mainnet 11-07/10/13） | **不可复核**：无原始 klines 响应留证；审计按纪律不主动访问真实 Binance 复现 |
| MySQL 8.0.46 `sync db` EXIT=0、10 个 `DECIMAL(36,18)`、4 条 `EXPLAIN ANALYZE` 数值 | **不可复核**：临时容器已删，无 SQL 输出、退出码日志或合成数据脚本留证（`/tmp` 仅存测试日志与二进制） |
| 生产库只读预检结果 / 检查单 | **未执行**：SQL 已给但检查单 §2 七项 0 勾选，文件本身声明「结果不是下单授权」 |
| dist/static、前端 HEAD、临时二进制 SHA、mysql:8.0 镜像、测试日志 | **✅ 已独立复核**（见 §3.5） |

实施记录 §3/§5 明确写了「不是 P95」「不能当生产历史量级验收」，但**总览新增段落与 06 主方案新增段落**把「`go test ./...`、Race、Vet、Build、MySQL8.0.46 `sync db` v25、10 个 DECIMAL、EXPLAIN 均通过」并列成一句，省略了「临时合成库 / 非 P95 / 容器已删」限定语。**要求**：把脱敏原始响应、`sync db` 退出码与日志、EXPLAIN 输出、合成数据脚本归档到 `doc/trade/` 留证；两处摘要补回限定语。

### F4 · P3 · 3d 在生产 Runner 下是「永久禁用」而非「已实现」，且每轮刷 ErrNeedsReview

`spot3dOpen` 无生产注入点，`StartSpotQuantTrade` 不构造 Runner。实施记录 §1/§6 已明确「真实行情适配器到生产 Runner 的注入仍属于未放行范围」，属诚实边界；但主方案状态行写「3d 事件拒单保护…已完成」，与「3d 自动交易在放行后仍不可用」应并列写明。启用 3d 指标的币在真实执行器下每 2 秒返回 `ErrNeedsReview`（日志被去重），且无自愈路径，只能人工改配置。建议把「3d 默认禁用」写进阶段边界与运维手册，并评估是否应在 `spotSignalInterval` 层直接拒绝 3d 配置，避免无意义轮询。

### F5 · P3 · 预检 SQL 与 Runner 的历史 clientID 判定口径不一致，且结果全靠人工判读

SQL 第 59 行用 `client_order_id NOT LIKE 'sq4%'` 统计 `legacy_buy_ids`；而 Runner 的 `spotStoredBUYEvent`（`runner.go:202-221`）要求 **34 字符 + `sq4` 前缀 + 合法周期码 + base36 秒 > 0**，任一不满足即 `ErrNeedsReview` 并禁止该币所有新 BUY。探针：

```
PROBE0674_FAKE_V4 len=34 parseable=false interval="" open=0001-01-01 00:00:00 +0000 UTC sql_would_flag=false
```

即**伪 v4 ID 能通过预检，却让该币永久 fail-closed**。此外：10 个 DECIMAL 仅靠注释「MUST return exactly 10 rows」人工数行；4 表存在性用 `GROUP BY` 返回行数，缺表不报警；`SUM(...)` 在空表返回 NULL；`config` 多行时 `ORDER BY id LIMIT 1` 可能取错。**要求**：SQL 增加显式 `PASS/FAIL` 列与完整 v4 合法性表达式（`CHAR_LENGTH`、周期码白名单、base36 段），或在文档中明确「需逐行人工判读并留存判读记录」。

### F6 · P3 · 严格环境 E2E 中评估链与执行链使用两个独立价格源

`spot0674StrictFixture`（`_0674_integration_test.go:58-63`）调用 `LoadVerifiedSpotRuleEnvWithSource(coin, time.Now(), "", klineLoader)`，`livePrice` 传空 → `validSpotTick` 回落到 `spot_symbols.close`（DB 缓存）；而真实 Processor 下单用 Fake Exchange 的 `ex.price`。测试靠手工同步二者（`ex.price="110"` + `UPDATE spot_symbols SET close='110'`）才让 SELL 条件成立。生产 `LoadVerifiedSpotRuleEnv` 用 `GetFreshSpotTickerPrice`，该函数返回空时同样回落 `coin.Close`，**两链之间无一致性校验**。这是 06-7-1 既有设计，但「严格 Spot 行情端到端」的表述应注明「评估链与执行链价格源一致性未验证」，06-7-4 真实行情接入时须补偏差/失效策略。

### F7 · P3 · 06-ALL 未决项状态更新（均未自行解除）

| 项 | 06-7-4 后状态 |
| --- | --- |
| F1 自动信号链运行时无入口 | **仍 BLOCKED**：`StartSpotQuantTrade` 空转，`NewProcessor`/`NewSpotQuantRunner` 全仓仅测试调用 |
| F4 前端源码固定发布 SHA 组合 | **有进展但仍未闭**：前端源码已提交 `cf319e1`、工作区干净、dist/static 115/115；但后端 06-7-4 仍为未提交工作区，无 Linux/ARM 生产二进制，无法形成成套 SHA 组合 |
| F5 异常订单恢复运维入口 | **仍 BLOCKED**：无 HTTP/CLI/UI；新增只读 SQL 只是诊断，不是恢复入口。F1 的仓位锁死场景恰好缺少这种入口 |
| F6 实盘 / 发布 Gate | **全部 BLOCKED** |

---

## 五、实盘 / 发布 Gate（全部 BLOCKED）

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 生产运行时自动 BUY/SELL | ⛔ | `StartSpotQuantTrade` = `return ctx.Err()`，全仓无生产执行器注入点（06-7-2 F1 / 06-ALL F1 延续，有意 Gate） |
| Spot Testnet 签名账户 | ⛔ | 实施记录 §2：HTTP 401 / `-2015`，本轮未修复；原始响应未留证 |
| 真实 Testnet 小额 BUY/SELL/Cancel/WS 恢复/myTrades/余额 | ⛔ | 仅 Fake Exchange + 临时 SQLite，无 Binance 网络 |
| 3d 真实行情 Provider 注入 | ⛔ | `spot3dOpen` 无生产注入点，自动 3d 默认停止 |
| Mainnet 订单硬限制 | ⛔ | 06-4 适配器默认只读、主网硬拒（**正确保持**） |
| 生产 MySQL 8 物理 Schema / EXPLAIN / P95 | ⛔ | 仅临时合成库兼容性证据，容器已删、无留证、明确非 P95 |
| 生产 v25 发布前 Schema 检查单 | ⛔ | `开发方案-06-发布前Schema验证检查单.md` §2 七项仍 0 勾选；新增只读 SQL 未在目标库执行 |
| 二进制 + static + schema 成套发布 / 回滚 | ⛔ | 后端代码未提交、无固定前后端 SHA 组合、无 ARM Linux 产物、无回滚演练 |
| 异常订单运维入口（HTTP / CLI / UI） | ⛔ | 06-ALL F5 未闭；F1 使其更紧迫 |
| 任何真实 Binance 请求 / 下单 / 仓位操作 | ⛔ | 审计全程未访问；探针使用 SQLite + Fake，transport 不联网 |

**06-ALL 的 F1 / F4 / F5 / F6 与 06-7-4 的 Gate 均不因本阶段离线准备通过而解除。**

---

## 六、后续准入条件

1. **F1 必须闭环**（P1）：3d 门禁不得阻断已验证仓位的 SELL；补不依赖权威 3d 源的退出身份 + 永久回归。这是任何真实执行器注入前的硬前置。
2. **F2 必须闭环或书面接受**（P2）：修复 Spot Testnet 凭据（`-2015` 涉及 Key/权限/IP 白名单，需用户在 Spot Testnet 专用环境处理），取得独立书面授权后按「测试账户真实小额 → WS/REST 恢复 → 手续费/dust/精度 → MySQL 8 生产体量 EXPLAIN/P95 → 生产 v25 检查单 → 成套发布与回滚」顺序验收。**禁止用主网密钥替换 Testnet，禁止因 Testnet 失败开启主网订单。**
3. **F3 必须补留证**（P2）：脱敏原始响应、`sync db` 日志、EXPLAIN 输出、合成数据脚本归档；两处文档摘要补「临时/非 P95」限定语。
4. **F4 / F5 / F6 建议同步处理**（P3）：3d 默认禁用的运维说明、预检 SQL 的 PASS/FAIL 断言与 v4 完整校验、评估链/执行链价格源一致性。
5. 生产默认必须保持 `exec=nil`；**06-7-4 无后续阶段编号，其验收只能由「真实 Testnet 证据 + 单独部署授权」完成，不能由更多离线测试替代**。

---

## 七、最终结论与审计边界

**06-7-4 的代码与离线 Gate 全部通过，严格 Fake Spot 行情 + 真实 Processor 的端到端、3d 拒单保护、只读预检 SQL 与静态资源核对都是扎实的离线准备，是一份合格研发基线；但阶段整体验收为 ⚠️ 有条件通过（仅离线准备项）/ ❌「06-7-4 阶段 Gate 未达成」：主方案 §4 要求的真实 Testnet 与交付验收 0 项通过，且新增 F1 为 P1 条件触发的「已有仓位无法自动平仓」缺陷。不认可「06-7 整体完成」或「方案 06 整体完成」。可以合并本地代码；不得注入真实执行器、不得宣称自动现货量化可用、不得发主网订单。所有实盘、Testnet、生产与发布 Gate 保持 BLOCKED。**

**审计边界**：本次未修改任何业务代码、未自动 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/` 与 `static/`、未对真实库执行 `sync db`、未访问真实 Binance（Testnet/Mainnet 声明仅按实施记录核对，未复现）、未重启服务、未操作生产仓位。临时探针只写在 `/tmp/phase_audit_0674` 副本内，删除后复验干净副本 `go test ./... -count=1` 为 73 ok / 0 FAIL；副本已于审计结束删除。`/tmp/spot0674_current_build` 等开发侧留证文件为只读核验，未修改或删除。本报告为新增文件，未覆盖任何历史 `CODE_REVIEW_*` 报告。
