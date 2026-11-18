# 10 · COIN-M Futures Trading

[English README](../../../README.EN.md) · [中文](../zh-CN/10-币本位合约交易.md) · [Implementation plan](../../trade/开发方案-CM-币本位合约交易-复刻现货精简版.md)

## Product scope and release status

- **Product**: Binance COIN-M USD-margined inverse perpetual contracts via DAPI (`contractType=PERPETUAL`). Quarterly delivery contracts and rollover are excluded.
- **Implemented in source**: CM-0 through CM-7: independent contract and strategy configuration, Managed position/order/Fill ledgers, inverse PnL, DAPI gateway, private User Data WS, STOP/TP Algo management, reuse of the existing two-second scheduler, and four dedicated UI pages.
- **Validated**: source compilation plus isolated Fake/Mock/SQLite Go and frontend tests. **Live MySQL 8 v27 migration, Demo/Testnet open/close orders, private WS, triggered Algo children, and mainnet operation have NOT been verified.**
- **Trading risk**: once the new build is deployed, enabling BOTH global and per-symbol entry switches can allow the automatic scheduler to **submit real Binance orders** when account and strategy prerequisites pass. No manual-trade button does **not** mean the automatic executor is inert.

## Pages and switches

| Location | Actual capability |
| --- | --- |
| Configuration Center → COIN-M Futures Trading | The only new global entry switch, `coin_m_quant_enable` (database default `0`) |
| COIN-M Futures Trading → Symbols | PERPETUAL symbol search, per-symbol `delivery_symbols.enable`, USD-equivalent budget, leverage, margin mode, STOP/TP and LONG/SHORT/CLOSE rules |
| COIN-M Futures Trading → Local Account | Read-only Managed cycles, orders, verified exchange Fills, PnL grouped by settlement asset, optional reference USD estimates and separate signed wallet reads |
| COIN-M Futures Trading → Strategy Templates | Isolated templates, import/export, four directions, technical indicators; no cross-product template IDs |
| COIN-M Futures Trading → Positions & Controlled Trading | Read-only local runtime state, Managed cycles, unresolved orders and evidence; **no manual submit/cancel/close/claim endpoints** |

Either switch being OFF stops **new entries only**. Owned positions, protective orders and uncertain orders still require maintenance and exits. Switching OFF is not a force close or protective-order cancellation.

## DAPI environment and credentials

| Environment | Required selection | Credentials |
| --- | --- | --- |
| mainnet | `binance.testnet=false` and optional `BINANCE_COIN_M_ENVIRONMENT=mainnet` (default) | Existing mainnet API key and secret |
| testnet | `binance.testnet=true` and optional `BINANCE_COIN_M_ENVIRONMENT=testnet` (default) | Separate Testnet key and secret in `app.conf` |
| demo | `binance.testnet=true` and `BINANCE_COIN_M_ENVIRONMENT=demo` | `BINANCE_COIN_M_DEMO_API_KEY` and `BINANCE_COIN_M_DEMO_API_SECRET` |

- COIN-M DAPI endpoints, wallets and private WebSocket are isolated from USD-M FAPI and Spot. Wrong endpoint, missing credentials or unverified private stream blocks new trading; no silent mainnet fallback.
- COIN-M and USD-M may share Binance hedge-mode settings. The application **reads and validates** existing position mode and never changes it automatically.
- Quantity is a **contract count**, not a BTC or ETH coin amount: USD budget × leverage ÷ USD face value per contract, rounded to exchange filters. BTCUSD_PERP requires BTC margin, ETHUSD_PERP requires ETH. No automatic collateral conversion or transfer.
- `/coin-m/runtime-status` reads local database state only. A switch, enabled-symbol count or pending-order count **does not prove** private WS connectivity, exchange wallet readiness or live position ownership.

## Verified Fills, inverse PnL and recovery

- LONG settlement-asset PnL = contracts × contract face value in USD × (1/entry price − 1/mark or closing price).
- SHORT settlement-asset PnL = contracts × USD face value × (1/mark or closing price − 1/entry price).
- Realized PnL comes from identified exchange trade Fills; unrealized PnL needs a fresh DAPI mark. Missing Fills, inconsistent symbols/sides/assets or stale marks do not produce made-up total PnL.
- Commissions remain grouped by `commission_asset`; funding is omitted without matching authoritative income records. Never directly sum BTC and ETH settlement PnL. USD conversion is **indicative only**, not the actual settlement.
- OPEN/CLOSE use stable clientOrderIds, Managed `position_id`, and Fill uniqueness `(symbol,exchange_order_id,trade_id)`. Unknown POST/429/timeouts reconcile by original ID rather than submit duplicates.
- The automatic runner requires executable STOP/TP configuration **before entry**. It also rejects stale entry signals when the strategy, indicator config, `Profit/Loss/TickSize/Leverage` changes during signal evaluation. Owned exits are prioritized over new entries.

## Pre-release checklist (NOT completed)

- [ ] Validate v26→v27 and repeated migrations against an **isolated** database; obtain a production DB backup and maintenance approval before manually running `./go_binance_futures sync db`.
- [ ] Verify the chosen Demo/Testnet REST/WS host, dedicated credentials, private listenKey, keepalive, reconnection, and permissions without reusing production keys.
- [ ] Verify small controlled Demo/Testnet LONG, SHORT, partial Fills, STOP/TP triggered children, cancellation, disconnection recovery, and exits with both entry switches OFF.
- [ ] Check MySQL 8 unique indexes, exact `budget_usd` decimals, actual Fill reconciliation, and unchanged USD-M/Spot/Lead behavior.
- [ ] Review all concurrent frontend changes before building and copying `dist/` to backend `static/`. Run `pnpm typecheck` and `pnpm build` first.
- [ ] Keep `coin_m_quant_enable=0` and individual symbol entry switches OFF until an explicit production go-live decision. Do not rely on the absence of UI manual-order actions.

## Offline developer checks

These tests use Fake/Mock/SQLite, without real DAPI order submissions:

```bash
go test -count=5 -p=2 -run '^TestCM7' ./service/coinmexecution ./controllers
go test -count=5 -p=2 ./service/coinmexecution ./service/coinmquant ./feature/api/binance ./controllers
go vet ./service/coinmexecution ./service/coinmquant ./feature/api/binance ./controllers
```

In the frontend project, run `pnpm typecheck` and `pnpm build`. Sync the build to backend `static/` only after all working-tree changes are approved.

**Passing offline source tests is not authorization to deploy or trade on live Binance.**
