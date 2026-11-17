# Lead Copy Trading

[中文](../zh-CN/04-合约带单.md) · [Back to English README](../../../README.EN.md)

## Lead Copy Trading (Lead Account 4)

> **Current status: the management UI and account-isolation infrastructure are implemented and verified with Mock/Race tests. Live Lead Portfolio identity, MySQL 8 physical schema v25, private WS, Algo orders, and funded execution have NOT passed their live gates. Live Lead order execution remains locked.** This feature is separate from AI Controlled Trading and cannot guarantee any follower fills.

Open **Futures Trade → Lead Copy Trading** (`/futures/lead`). The page has six Cards:

| Card | What is available today / data boundary |
| --- | --- |
| Account Status | Disabled/blocking reasons, masked Lead key, Portfolio *label*, latest read-only verification and private WS status. The label is not verified Portfolio identity. |
| Dedicated Credentials | Encrypted AES-GCM storage of a separate Lead API Key/Secret and user-triggered read-only verification. Secrets are never returned; rotating credentials revokes the old result and observer. |
| Account Risk | Lead-01 now exposes only max order/total notional and an optional daily loss cap. Retired drawdown/position-count inputs are no longer shown; strategy and symbol selection will reuse Main. |
| Automated Trading | Only one disabled **Allow automatic new entries** switch and a safety-pause control remain. The backend `/start` still returns HTTP 423; there is no live order execution. |
| Managed Positions | **Local Lead Ownership** including closed historical records; this is not proof of Binance live positions, manual positions, or real-time PnL. |
| Orders & Diagnostics | Local Lead managed orders without a separate audit-history tab. The `/reconcile` write-back operation remains locked without authoritative live verification. |

The **System Dashboard** now includes a read-only Lead summary, private WS health, five-minute REST request count/weight, and a link to the Lead page. Alert drops display **`—` when no queue is running, rather than a misleading zero**. The Lead page refreshes local status approximately every 15 seconds; viewing it does not enable live orders.

#### Separate and secure Lead configuration

Lead credentials and risk settings are **not stored in** `conf/app.conf` or frontend code. The optional audit file is no longer required for ordinary configuration saves and read-only verification. Supply the following to the application process using protected environment/secrets management (these are placeholders; **do not put real secrets in the README, Git, logs, or shell history**):

```text
BINANCE_LEAD_CREDENTIAL_KEY=<standard-base64-of-32-random-bytes>
BINANCE_LEAD_CREDENTIAL_FILE=/private/lead/credentials.enc
BINANCE_LEAD_CONFIG_FILE=/private/lead/risk.json
# BINANCE_LEAD_AUDIT_FILE=/private/lead/operations.jsonl  (optional)
# Do not set BINANCE_LEAD_OBSERVER_MODE by default
```

- `BINANCE_LEAD_CREDENTIAL_KEY` must decode from **standard Base64 to exactly 32 random bytes**. Generate and persist it securely: losing/rotating this master key makes the existing encrypted credentials unreadable.
- All configured private file paths must be **absolute and outside the repository**, with an operator-owned private parent directory (recommended `0700`) and private files (`0600`), not symlinks. The app does not silently create private directories or generate a master key.
- After setting these values and deploying the updated backend, use the UI to save a dedicated Lead credential, request read-only verification, and save independent risk limits. Missing secure storage causes fail-closed errors; **do not work around this by putting secrets into `app.conf`**.
- Only after backing up the target DB, checking the actual version/data and taking the service down should an operator explicitly run `./go_binance_futures sync db`. The current binary requires **schema v25** (see `appversion.DatabaseSchemaVersion`; verify the production physical v25 schema read-only before deployment). Passing SQLite/Mock migration tests does **not** confirm your real MySQL 8 schema has been migrated.

#### Private read-only observer: explicit opt-in performs real Binance I/O

Leave `BINANCE_LEAD_OBSERVER_MODE` **unset by default**: no Lead credentials are loaded and no Lead private WS is opened. Only when the separate key, DB prerequisites and **authorized live read-only testing** are ready, set the following in the protected process environment:

```text
BINANCE_LEAD_OBSERVER_MODE=read_only
```

This is **not** an offline simulation. On startup the new process really **creates/renews/closes a Lead listenKey, opens a private User Data WebSocket, and reads live balances/positions/open-order snapshots**. It requires a valid Lead key/permissions and working network/proxy. The observer runs SELECT-only Ownership inspection and updates an in-memory mirror; it **does not submit or cancel orders, apply fills to the managed ledger, or unlock trading**. Futures Testnet success must not be presented as live Lead Portfolio authorization.

To turn it off, remove the environment variable from the deployment configuration and restart/stop the process. Credential rotation stops the old observer. There is no UI on/off switch and no implicit proof of Portfolio identity. Authenticated `GET /futures/lead/status` and `GET /futures/lead/metrics` expose `read_only_observer`, WS generation, mirror health, inspection status and observed REST metrics.

#### Manual safety checks and remaining live gates

1. **Verify the deployed binary and database**: syncing `dist` into backend `static` updates only web assets. New `/futures/lead/*` APIs and the Stage 7 observer require the updated backend binary. A listening old process is not proof of deployment.
2. **Start with UI and access controls**: open all six Cards, switch languages and refresh. Missing JWT must deny Lead APIs; an authenticated session must show `allow_new_opens=false` and `live_trade_authorized=false`. Verify risk settings persist and invalid/cross-account/enable requests are denied.
3. **Test Pause last**: `POST /futures/lead/pause` first asserts a **one-way, process-local emergency stop**, then attempts durable config/audit writes. When persistence succeeds it returns HTTP 200; otherwise HTTP 202 with `pause_persisted=false`. A process-local stop cannot guarantee safety after a restart or in another process. There is no UI resume switch.
4. **Test the observer separately**: only under explicit permission, set `read_only` and restart the updated backend. Check WS/snapshot health, outstanding-order inspection, account ownership, and compare live figures with the official Binance Portfolio UI. Read-only status must always keep `live_trade_authorized=false`.
5. **Live orders remain off-limits**: `/start`, unverified `/reconcile`, and unsafe credential deletion return HTTP 423 (or fail earlier with a safe storage/audit error when not configured). STOP/TP Algo support, dynamic price filters, partial fills/manual reductions, multi-process restart recovery, real dual-account two-second API load, and minimum-size funded trades require separate Stage 7-5/7-6 gates and explicit human authorization.

See the Lead Copy Trading staged implementation plan (referenced document not present in checkout: `doc/trade/币安合约自动带单-Stage实施方案.md`) and the implementation/review logs in `doc/trade/`. **This is a personal research/operations feature, not a declaration that live copy trading is production-ready.**

#### Lead-01 simplified configuration semantics

- Only three user-managed risk fields are exposed: `max_order_notional_usdt` and `max_total_notional_usdt` (both positive; the single-order cap must not exceed the total) and optional `max_daily_realized_loss_usdt` (`0` disables the additional daily loss cap).
- **Unfinished old Lead JSON formats are not supported or migrated.** Only the new config is accepted; old `enabled`, drawdown and position-count fields are strictly rejected. Explicitly save a new config if necessary; this never enables live trading.
- `GET /futures/lead/config` now reports `risk_configured` to indicate whether the two required notional caps are complete. Invalid form submissions fail explicitly.
- Authorized risk and encrypted-credential writes no longer depend on a configured audit file; optional events are recorded when available. `/start`, `/reconcile`, the live Broker and `LeadDurableOpenGate` remain locked.
- Only one disabled new-entry switch is displayed. The existing Pause operation remains a **one-way in-process safety latch** with a confirmation. Lead-01 does not add a live trading runner.
- Lead/Main account isolation, read-only WS observer, exchange identity checks, Order/Fill/Ownership and API weight controls are preserved. The internal duplicated gates and risk adapter will be consolidated with the existing Main execution model in Lead-02/03 rather than removed prematurely.

[Lead-01 code review](../../trade/CODE_REVIEW_Lead-01.md)
