# CODE_REVIEW_CM-4-rereview · 币本位周期汇总 / inverse PnL / 钱包只读复审

> 日期：2026-11-18（北京时间）
>
> 分支：`feat/m-coin-futures`；HEAD `e6162bf feat: cm-3`（父 `1d5be90 feat: cm-2`）；master `0b3fa34 (tag v2.2.3)`。
>
> 本报告是**新增复审**，不覆盖 `doc/trade/CODE_REVIEW_CM-4.md`。审查对象为当前工作区未提交的 CM-4 改动：3 个跟踪文件修改、9 个新增非静态文件、静态资源重建；前端仓库 11 项状态变更。
>
> 审计期间未修改业务源码、未提交代码、未修改 `conf/app.conf`、未访问真实 Binance、未执行 `sync db`、未重启服务、未触碰真实仓位。探针只写在隔离副本中。

---

## 一、复审结论

| 项目 | 结论 |
| --- | --- |
| 独立 COIN-M Position/Order/Fill 只读汇总 | ✅ 基本实现 |
| inverse PnL 计算公式与精度 | ✅ 通过 |
| DAPI 公开 mark 价格、8 秒超时、120 秒新鲜度 | ✅ 离线/Mock 通过 |
| 按 `marginAsset` 分组、筛选后全量汇总、参考 USD 估值 | ✅ 通过 |
| 手续费按 `commission_asset` 分组、资金费不伪造 | ✅ 通过 |
| 签名 DAPI 钱包只读、无凭证 fail closed | ✅ 离线/Mock 通过 |
| 前端账户页、受控交易只读页、i18n、静态产物 | ✅ 通过，但有阶段文案与非 CM-4 夹带问题 |
| 后端离线 Gate | ✅ 通过 |
| **Fill 证据的 symbol / positionSide 一致性** | ❌ **不通过：F1，P1 阻塞** |
| 真实 DAPI / 签名 User WS / 真实成交 / MySQL 8 / 生产发布 | ⛔ 全部 BLOCKED |
| 阶段意见 | **❌ 不认可 CM-4 完成；F1 修复并复审前不得进入 CM-5** |

**核心判定：** CM-4 的实现范围和离线测试覆盖已经接近完成标准，但隔离探针证实的 F1 直接破坏「PnL 正确 / Fill 唯一 / 证据一致」这一 CM-4 核心验收项。当前主方案和实施方自评中「CM-4 已完成」「可以进入 CM-5」的结论**不成立**。

F1 不是边角展示问题：它把跨合约、跨 positionSide 的 Fill 当作当前 Managed 周期的已验证证据，并把由此计算的 PnL 标成 `verified`，同时输出非空总额。该行为与 CM-4 文档中「Fill 的 order_id、position_id、symbol、positionSide、BUY/SELL 及 OPEN/CLOSE 意图需一致」的自身承诺直接冲突。

---

## 二、基线与隔离副本

### 2.1 基线

```text
branch        feat/m-coin-futures
HEAD          e6162bf feat: cm-3
parent        1d5be90 feat: cm-2
master        0b3fa34 tag v2.2.3
audit date    2026-11-18
worktree      171 项状态变更（含 static 重建）
frontend      11 项状态变更
```

CM-4 核心改动：

- 修改：`controllers/coin_m_quant_read.go`、`routers/router.go`、CM 主方案状态；
- 新增服务：`service/coinmquant/pnl.go`、`ledger_summary.go` 及测试；
- 新增行情/钱包：`feature/api/binance/coin_m_mark_price.go`、`coin_m_wallet_cm4_test.go`、`controllers/coin_m_wallet_cm4_test.go`；
- 新增文档：`doc/trade/CODE_REVIEW_CM-4.md`；
- 新增路由：`/coin-m/wallet`；
- 前端：账户页增加钱包、资产汇总、证据状态、USD 参考估值；受控交易页保持只读。

### 2.2 等价校验

后端隔离副本使用 `git archive HEAD` + 同步 M/?? + 删除 D，`conf/app.conf` 用 `app.conf.example` 兜底：

```text
copy          /tmp/phase_audit_cm4
SYNCED_CHANGES=171
FILES_CHECKED=1122
GO_FILES=737
STATIC_FILES=120
SHA256_MATCHED=1122
SHA256_DIFF=0
SHA256_MISSING=0
DELETED_CONFIRMED=79
APP_CONF_PRESENT=True
```

前端隔离副本：

```text
copy          /tmp/phase_audit_cm4_ui
SYNCED_CHANGES=11
FILES_CHECKED=295
SHA256_MATCHED=295
SHA256_DIFF=0
SHA256_MISSING=0
```

未使用全仓 `os.walk` / `grep -r` 扫描历史 research 目录。

---

## 三、方案契约逐项对照

| CM-4 要求 | 实现证据 | 判断 |
| --- | --- | --- |
| 独立持仓周期、订单、真实 Fill 只读汇总 | `ledger_summary.go` 只查 `coin_m_managed_*` / `coin_m_trade_fills`；`Detail` 返回 orders/fills | ✅ 通过 |
| LONG / SHORT inverse PnL | `pnl.go:43-72` 用 `math/big.Rat`，公式 N×C×(1/Pe−1/Px) / 反向 | ✅ 通过 |
| 已实现盈亏来自真实 Fill | `ledger_summary.go:172` 汇总 `realized_pnl`；按 BTC/ETH 结算资产分离 | ⚠️ 正常路径通过，F1 下失真 |
| 未实现盈亏使用 DAPI mark，不用 ticker close 兜底 | `coin_m_mark_price.go` 固定 `/dapi/v1/premiumIndex`；`CoinMMarkTimeValid` 120 秒内且未来容差 30 秒；缺 mark 则 `unpriced` | ✅ 通过 |
| USD 参考换算 | 已实现按每笔 Fill 的 `price`，未实现按 mark；默认关闭显示 | ✅ 通过 |
| 筛选结果全量汇总、分页前计算、按资产分组 | `PositionSummaries` 先全量聚合再切片；`AssetTotal` 按 `margin_asset` | ✅ 通过 |
| 手续费 / 资金费边界 | 手续费按 `commission_asset`；`FundingIncluded=false`，不伪造 Income | ✅ 通过 |
| 异常证据状态 | `verified / pending_fills / unverified_fills / unpriced / invalid`；异常时总额为 `null` | ⚠️ 状态机基本通过，F1 会误判为 `verified` |
| DAPI 钱包只读 | `GetWallet` JWT 后调用 `GetDeliveryBalanceContext`；8 秒超时；失败 503，不假零余额、不回退主网 | ✅ 离线/Mock 通过 |
| 前端账户与受控交易页 | 钱包、周期/明细、资产汇总、USD 开关、证据标签；受控页无交易按钮 | ✅ 通过，见 O1/O2 |
| 不新增 Schema | 未改 v27；Fill 三元组唯一约束沿用 CM-1 | ✅ 通过 |
| 不提前解锁真实交易 | `CM_LIVE_ORDER_TOKENS=0`，`CM_CROSS_PRODUCT_API_IMPORTS=0`，受控页 `CONTROLLED_TRADE_ACTIONS=0` | ✅ 通过 |

---

## 四、F1 · P1 · 跨合约 / 跨 positionSide Fill 被当作已验证证据

### 4.1 代码位置

`service/coinmquant/ledger_summary.go`：

- `97-112`：`loadFillEvidence` 的 JOIN 只比较 `f` 与 `o` 的 `position_id / symbol / position_side / side`，**没有比较 `f.symbol = p.symbol` 或 `f.position_side = p.position_side`**；
- `20-31`：`fillEvidence` 读取了 `Side`、`PositionSide`，但后续 `113-182` 的校验只使用 `f.Side` 与 `pos.PositionSide` 推导 BUY/SELL 是否一致；
- `175-177`：`settledAssetForSymbol(p.Symbol, p.MarginAsset)` 使用 Managed 周期的 symbol，不会发现 Fill 实际属于另一个合约；
- `316-327`：只要 open/close 张数与周期字段对得上，就进入 `remaining>0` 的 mark 估值或 `remaining==0` 的已完成分支，并可能置为 `verified`。

这与实施方自评 `CODE_REVIEW_CM-4.md` §2.2 的表述矛盾：

> Fill 的 order_id、position_id、symbol、positionSide、BUY/SELL 及 OPEN/CLOSE 意图需一致。

### 4.2 隔离探针

探针：`/tmp/phase_audit_cm4/service/coinmquant/review_cm4_probe_test.go`

构造：

- Managed 周期：`BTCUSD_PERP / LONG / margin_asset=BTC / opened=10 / remaining=5`；
- 注入两笔内部自洽但明显错误的 order/fill：
  - `symbol=ETHUSD_PERP`、`position_side=SHORT`、intent `OPEN`、side `BUY`、10 张；
  - `symbol=ETHUSD_PERP`、`position_side=SHORT`、intent `CLOSE`、side `SELL`、5 张、`realized_pnl=0.000909090909090909`、`pnl_asset=BTC`；
- 提供有效的 `BTCUSD_PERP` mark 价格 55000；
- 断言这种周期应被标记为 `invalid`，且不得输出未实现/总 PnL。

实际结果：

```text
--- FAIL: TestReviewCM4CrossSymbolAndPositionSideFillMustBeRejected
cross-symbol/cross-positionSide Fill was accepted:
status=verified
open=10 close=5 realized=0.000909090909090909
unreal=<non-nil> total=<non-nil>
assetTotal={BTC 1 1 0 0 0 0.000909090909090909 ...}
FAIL go_binance_futures/service/coinmquant
```

该失败是审计探针按预期抓到的代码缺陷，不是环境、夹具或命令错误。

### 4.3 影响

1. 跨产品污染：ETH / SHORT Fill 可以进入 BTC / LONG 周期，破坏 COIN-M 资产隔离。
2. 虚假已验证证据：错误 Fill 不会触发 `invalid` / `unverified_fills`，反而输出 `verified`。
3. 错误 PnL：已实现、未实现、资产总计、USD 参考估值均可能被污染。
4. 与 CM-4 完成标准「PnL 正确、Fill 唯一」以及自评一致性承诺直接冲突。

### 4.4 修复要求

修复前不得宣称 CM-4 完成，也不得进入 CM-5：

1. 在 `loadFillEvidence` JOIN 中显式加入：
   - `p.symbol = f.symbol`
   - `p.position_side = f.position_side`
   - `p.symbol = o.symbol`
   - `p.position_side = o.position_side`
2. 在逐 Fill 校验中拒绝 `f.Symbol != pos.Symbol` 或 `f.PositionSide != pos.PositionSide`，设置 `agg.bad` 并使周期为 `invalid`。
3. 将本次隔离探针转为永久回归测试，并覆盖：
   - 跨 symbol；
   - 跨 positionSide；
   - order 与 fill 内部一致但与 Managed 周期不一致；
   - 资产总计保持 `null`。
4. 修复后必须重跑本轮全部离线 Gate 和探针，并另起 `CODE_REVIEW_CM-4-rereview-2.md` 或同等复审报告。

---

## 五、其他观察项

### O1 · P2 · 前端 diff 仍夹带非 CM-4 改动

前端工作区 11 项变更中，除 CM-4 钱包/账本展示外，仍包含此前 CM-3 已记录的非 CM 改动：

- `src/api/leadTrading.ts`
- `src/views/futures/leadTrading.vue`
- `src/router/modules/futures.ts`
- `src/views/login/index.vue`
- `src/views/dashboard/configShow.vue` 中的非 CM 配置文案
- `src/views/futures/symbol.vue`
- `locales/zh-CN.yaml` / `locales/en.yaml` 中大量 lead trading / 429 文案
- `spotQuantTrading` 文案从「现货量化交易」改为「现货交易」

这些改动不改变 CM-4 的 PnL 判定，但会污染阶段边界、回滚范围和发布说明。建议拆 commit 或在提交说明逐项解释。

### O2 · P2 · 阶段文案不一致

`locales/zh-CN.yaml` / `en.yaml` 中：

- `coinMPage.account.readOnly` 已写 CM-4；
- `coinMPage.controlled.notice` 仍写「CM-3 is read-only / CM-3 只读」。

受控交易页实际仍是只读且真实操作留 CM-5，但阶段名称应统一为 CM-4，避免 UI 与状态表矛盾。

### O3 · P3 · 主方案提前写「CM-4 已完成」

工作区主方案状态已更新为「CM-0～CM-4 已完成」，但 F1 尚未修复，复审结论为不通过。建议将状态改为「CM-4 待 F1 修复并复审」，待闭环后再更新。

---

## 六、离线 Gate

执行时清空 `BINANCE_TESTNET_*`、`SPOT_QUANT_TESTNET_*`、`SPOT_QUANT_VALIDATE_*`、`SPOT_LEDGER_MYSQL_DSN`、`SPOT_STRATEGY_LIVE_TRADING_APPROVAL`、`BINANCE_COIN_M_ENVIRONMENT`、`BINANCE_COIN_M_DEMO_*` 等 env，确保不联网、不下单。

### 6.1 后端

```text
git diff --check（工作区）                                      PASS
go vet ./...                                                   PASS
go build -o /tmp/go_binance_futures_cm4_audit .                PASS
go test -count=1 -p=2 ./...                                    PASS
    OK=74 NO_TEST=13 FAIL=0 TOTAL=87
go test -count=5 -p=2 -run '^TestCM4' ./service/coinmquant \
    ./feature/api/binance ./controllers                        PASS
go test -count=5 -p=2 ./service/coinmquant \
    ./feature/api/binance ./controllers                        PASS
go test -race -count=2 -p=2 ./feature/api/binance \
    ./service/coinmquant ./controllers                         PASS
```

macOS linker `LC_DYSYMTAB` warning 非致命，退出码为 0。

### 6.2 前端

```text
pnpm typecheck    PASS
pnpm build        PASS
```

Browserslist / `caniuse-lite` 过期提示非致命。

### 6.3 静态产物

```text
frontend/dist      vs backend/static      DIFF_EXIT=0
rebuilt dist       vs frontend/dist       DIFF_EXIT=0
rebuilt dist       vs backend/static      DIFF_EXIT=0
files per tree     120
```

### 6.4 静态边界扫描

```text
CM_LIVE_ORDER_TOKENS=0
CM_CROSS_PRODUCT_API_IMPORTS=0
CM_ROUTES=13
CONTROLLED_TRADE_ACTIONS=0
CONFIG_COIN_M_SWITCH_TEXT_REFS=5（实际 el-switch 仅 1 个）
```

前端 `src/router/modules/coinM.ts` 中唯一匹配是注释：`never their APIs`，未发现 FAPI / Spot / U 本位订单调用。

### 6.5 既有测试与本次探针

PASS：

- `TestCM4InversePnLExamples` / `TestCM4InversePnLRejectInvalidInputs`；
- `TestCM4FilteredAssetTotalsAndCycleReconciliation`；
- `TestCM4MissingMarkAndFillIntegrityFailClosed`；
- `TestCM4PartialOrMissingEvidenceNeverPretendsVerified`；
- `TestCM4PendingOpeningIsNotFakePnL`；
- `TestCM4PublicMarkPriceOnlyUsesSelectedDAPINetwork` / `TestCM4MarkPriceFreshness`；
- `TestCM4WalletSignedReadStaysOnChosenDAPINetwork`；
- `TestCM4WalletRequiresJWTBeforeAnyDAPIRequest`。

FAIL：

- 本次新增 `TestReviewCM4CrossSymbolAndPositionSideFillMustBeRejected`：跨 symbol / positionSide Fill 被接受为 `verified`（见 F1）。

---

## 七、未通过 / 仍 BLOCKED 的 Gate

1. F1 修复与永久回归测试缺失；
2. 修复后完整 Gate 与专项复审未执行；
3. 真实 MySQL 8 v27 升级、备份、索引、P95 与回滚演练；
4. 真实签名 DAPI 钱包、账户、持仓模式读取；
5. 签名 User Data WS、真实 `userTrades`、Fill 写入与对账；
6. COIN-M Testnet / Demo 真实开平仓、部分成交、重试、撤单、Algo、429/418；
7. Mainnet 小额实盘；
8. ARM Linux + 静态资源 + schema 成套发布、灰度、回滚、持久化授权；
9. CM-5 真实 LONG/SHORT/CLOSE、失败恢复、仓位归属与异常人工处理；
10. CM-6 调度、两级开关、出场优先与节流；CM-7 端到端发布验收。

---

## 八、最终判定

**CM-4 不通过。**

理由：

- 离线 Gate 和大部分功能实现确实达到较高完成度；
- 但 F1 已被隔离探针实证：跨合约、跨 positionSide 的 Fill 可以被计入当前周期并输出 `verified` PnL；
- 该问题直接破坏 CM-4 的核心金融口径，且与自评的一致性承诺冲突；
- 实施方已将主方案状态写成「CM-4 已完成」，与复审事实不符。

**下一步动作：**

1. 修复 F1；
2. 将探针转为永久回归测试；
3. 重跑本轮全部 Gate；
4. 另起第二轮复审报告；
5. 在 F1 闭环前，不认可 CM-4 完成，不进入 CM-5，不做任何真实 DAPI 交易或生产发布。

---

## 九、后续复审追踪（2026-11-18）

本报告 F1/P1 原始失败证据保留不变。后续实施方已修复 Fill/Order/Managed 三方归属校验，新增 11 组永久回归探针，并重新通过 87 个非历史研究包的离线 Gate。**请以 [CODE_REVIEW_CM-4-rereview-2.md](CODE_REVIEW_CM-4-rereview-2.md) 的后续状态为准**。这仅解除 CM-4 代码审查阻断，不构成生产环境或真实下单验收。
