# Binance Futures Testnet E2E

用于验证 V3-5 Ownership 改造后的真实 Binance USD-M Futures 下单语义。

## 安全边界

- 测试代码强制设置 `futures.UseTestnet = true`，不会请求主网 Futures REST。
- 测试默认跳过，只有 `BINANCE_TESTNET_E2E=1` 时才会提交测试网订单。
- 测试网凭据只从以下位置读取：
  - `BINANCE_TESTNET_API_KEY` / `BINANCE_TESTNET_API_SECRET`；或
  - `conf/app.conf` 的 `[binance] testnet_api_key/testnet_api_secret`。
- **不会**回退读取主网 `api_key/api_secret`。
- 默认使用 `BTCUSDT`；可通过 `BINANCE_TESTNET_SYMBOL` 修改。
- 开始前要求目标 Symbol 没有已有仓位和挂单，否则直接拒绝执行，避免碰到其它测试仓位。
- 测试结束会尽最大努力撤销 clientOrderId 前缀为 `v35e2e_` 的测试挂单，并清理本测试创建的目标 Symbol 仓位。

## 覆盖场景

1. 开多 → 平多：`BUY + LONG` 开仓，`SELL + LONG` 平仓，Quantity 始终为正数。
2. 开空 → 平空：`SELL + SHORT` 开仓，`BUY + SHORT` 平仓，Quantity 始终为正数。
3. TP / SL：在真实测试网 LONG 仓位上创建 `TAKE_PROFIT_MARKET` 与 `STOP_MARKET`，确认 Binance 接受且 Ownership 记录为活动保护单，然后安全撤销。
4. 人工部分减仓：先由 Ownership 开 LONG，再直接调用测试网 API 模拟人工 `SELL + LONG` 减仓；随后执行 Reconcile，确认 `managed_qty` 只缩小、不补回；最后 Ownership 只平剩余 managed quantity。
5. 测试使用真实 Binance Testnet exchange order / position 状态，不使用 Fake Broker。

TP/SL E2E 验证的是“真实测试网接受保护单 + 方向/数量/Ownership 状态正确”。不依赖市场自然波动等待触发；保护单成交后的 sibling cleanup 由 `service/futuresownership` 的 Reconcile 单元测试覆盖。

## 配置

`conf/app.conf`：

```ini
[binance]
testnet_api_key = "YOUR_TESTNET_KEY"
testnet_api_secret = "YOUR_TESTNET_SECRET"
```

也可以只在当前 shell 提供环境变量；环境变量优先级高于 `conf/app.conf`。

若测试网账户不是 Hedge Mode，默认拒绝执行。仅在专用测试网账户上确认允许自动切换时，增加：

```bash
export BINANCE_TESTNET_ALLOW_HEDGE_MODE_CHANGE=1
```

代理默认复用 `[binance] proxy_url`，也可以单独设置 `BINANCE_TESTNET_PROXY_URL`。

## 执行

```bash
BINANCE_TESTNET_E2E=1 \
  go test ./service/futuresownership \
  -run TestBinanceFuturesTestnetE2E \
  -v -count=1
```

普通 `go test ./...` 不设置 `BINANCE_TESTNET_E2E=1` 时只会 Skip，不会连接测试网账户或提交订单。

## 2026-09-12 实际验证结果

使用 Binance USD-M Futures Testnet 实际执行完整 E2E，结果全部通过：

- 开多 → 平多：PASS。
- 开空 → 平空：PASS。
- `TAKE_PROFIT_MARKET` / `STOP_MARKET` 创建、查询、撤销：PASS。
- 人工部分减仓 → Reconcile → 仅平剩余 managed quantity：PASS。
- 最终清理断言：BTCUSDT 无仓位、无普通挂单、无 Algo 挂单。

真实测试过程中发现 Binance 已将 STOP/TP 条件单迁移到 Algo Order API。Ownership Broker 现在按 OrderType 分流：`MARKET/LIMIT` 走普通 Order API，STOP/TP/Trailing Stop 系列走 Algo Order API；查询、Reconcile、Cancel 也使用对应接口。

同时修复订单浮点序列化问题，避免 `0.0024` 被发送为 `0.0024000000000000002` 或 trigger price 出现二进制尾数并触发 Binance `-1111 Precision is over the maximum defined for this asset`。
