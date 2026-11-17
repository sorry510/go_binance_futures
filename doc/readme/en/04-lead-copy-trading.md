> **Update (2026-11-17):** Lead entries now have a default-OFF manual switch. Save dedicated credentials and risk limits first. Turning ON may submit real orders without requiring a separate read-only verification. Turning OFF preserves managed exits. Matching Main Futures keys are rejected; distinct keys can still belong to the same Binance account, so confirm that yourself. See doc/trade/CODE_REVIEW_Lead-04-ManualSwitch.md.

# Lead Copy Trading

[中文](../zh-CN/04-合约带单.md) · [Back to English README](../../../README.EN.md)

## Lead Copy Trading (Lead Account 4)

> **Current status: a default-OFF manual switch now enables the Lead trading code path. Live Portfolio identity, the deployed MySQL schema, private WS, and funded execution have not been independently verified. Turning the switch ON can place real orders; local tests are not a live acceptance test.** This feature is separate from AI Controlled Trading and cannot guarantee any follower fills.

Open **Futures Trading** and click the **Copy Trading** button next to **Funding Rate** (`/futures/lead`). The standalone sidebar entry is hidden. The page has six Cards:

| Card | What is available today / data boundary |
| --- | --- |
| Account Status | Disabled/blocking reasons, masked Lead key, Portfolio *label*, latest read-only verification and private WS status. The label is not verified Portfolio identity. |
| Dedicated Credentials | Encrypted AES-GCM storage of a separate Lead API Key/Secret and user-triggered read-only verification. Secrets are never returned; rotating credentials revokes the old result and observer. |
| Account Risk | Lead-01 now exposes only max order/total notional and an optional daily loss cap. Retired drawdown/position-count inputs are no longer shown; strategy and symbol selection will reuse Main. |
| Automated Trading | One default-OFF, manually operable **Allow automatic new entries** switch and an emergency-pause control are provided. Enable via configuration save; the legacy `/start` endpoint still returns HTTP 423. |
| Managed Positions | **Local Lead Ownership** including closed historical records; this is not proof of Binance live positions, manual positions, or real-time PnL. |
| Orders & Diagnostics | Local Lead managed orders without a separate audit-history tab. The `/reconcile` write-back operation remains locked without authoritative live verification. |

The **System Dashboard** now includes a read-only Lead summary, private WS health, five-minute REST request count/weight, and a link to the Lead page. Alert drops display **`—` when no queue is running, rather than a misleading zero**. The Lead page refreshes local status approximately every 15 seconds; viewing it does not enable live orders.

#### Create a dedicated Futures Lead API key

This application needs an **API Key / API Secret created for a Futures Lead Trading Portfolio**, not an ordinary Main-account key or a Spot Copy Trading key.

1. On Binance, open **Futures → Copy Trading → My Lead** and create or select your lead portfolio.
2. Within that portfolio, select **API → Create Copy Trading API**.
3. In **Edit restrictions**, whitelist your server's public IPv4 address and enable **Enable Futures** trading permission.
4. Paste the resulting API Key / Secret into **Lead Copy Trading → Dedicated credentials** and save them encrypted. The **Portfolio label** is optional local display text only and is not used to identify the account.

Each Futures lead portfolio supports at most two keys, and Futures lead APIs support USDT only. Do not enable withdrawals. The application rejects exact matches to known Main API keys, but **does not verify whether two distinct keys belong to the same Binance account**.

Binance official guide: [Create an API Key for a Futures Lead Trading Portfolio](https://www.binance.com/en/support/faq/detail/2bec848b904b422197ce121d0925f20b).

#### Lead risk settings and dedicated credentials

**Lead risk limits and the automatic-entry switch are stored in the database table `lead_trade_settings`** (schema v26). The application no longer reads `BINANCE_LEAD_CONFIG_FILE` or `risk.json`. The single Lead row (`account_id=lead`) stores `allow_new_opens`, the per-order notional cap, total notional cap, daily loss limit and last update timestamp. A missing row means **automatic entries OFF**; database failures and invalid records fail closed. No data is automatically imported from old JSON files, and the app does not delete those files.

Lead **API Key / Secret** still use the existing encrypted credential file (never plain-text DB columns). Configure these securely in the running process environment; these are placeholders:

```text
BINANCE_LEAD_CREDENTIAL_KEY=<standard-base64-of-32-random-bytes>
BINANCE_LEAD_CREDENTIAL_FILE=/private/lead/credentials.enc
# BINANCE_LEAD_AUDIT_FILE=/private/lead/operations.jsonl  (optional)
```

- The encryption master key must be standard Base64 for 32 random bytes. Losing it prevents decryption of existing credentials. Never commit secrets or put them in `conf/app.conf`.
- Keep encrypted credentials outside the repository in private, non-symlinked storage. The audit file is optional and does not block ordinary risk-setting updates.
- **Back up and inspect the target database, then run `./go_binance_futures sync db` manually to upgrade schema v25 to v26** before starting the updated binary. Do not modify a running production database without a planned maintenance window.
- v26 creates only the new `lead_trade_settings` table. It does not alter Main account settings, existing orders, or positions. Risk settings are saved to the DB from the UI. Re-enter any old `risk.json` values manually if needed; there is no implicit migration.
- Passing offline SQLite tests does not prove the actual MySQL 8 schema or live Binance account is ready.

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

#### Historical Lead-01 configuration semantics (not current switch behavior)

- Only three user-managed risk fields are exposed: `max_order_notional_usdt` and `max_total_notional_usdt` (both positive; the single-order cap must not exceed the total) and optional `max_daily_realized_loss_usdt` (`0` disables the additional daily loss cap).
- **Unfinished old Lead JSON formats are not supported or migrated.** Only the new config is accepted; old `enabled`, drawdown and position-count fields are strictly rejected. Explicitly save a new config if necessary; this never enables live trading.
- `GET /futures/lead/config` now reports `risk_configured` to indicate whether the two required notional caps are complete. Invalid form submissions fail explicitly.
- Authorized risk and encrypted-credential writes no longer depend on a configured audit file; optional events are recorded when available. `/start`, `/reconcile`, the live Broker and `LeadDurableOpenGate` remain locked.
- Only one disabled new-entry switch is displayed. The existing Pause operation remains a **one-way in-process safety latch** with a confirmation. Lead-01 does not add a live trading runner.
- Lead/Main account isolation, read-only WS observer, exchange identity checks, Order/Fill/Ownership and API weight controls are preserved. The internal duplicated gates and risk adapter will be consolidated with the existing Main execution model in Lead-02/03 rather than removed prematurely.

[Lead-01 code review](../../trade/CODE_REVIEW_Lead-01.md)

### Orders and WS records (shared with Main futures)

Lead uses the existing `futuresownership.Executor`. A normal Binance response confirms **order acceptance**; `ORDER_TRADE_UPDATE` updates order status and actual fills. Main and Lead use the same account-scoped local `futures_orders` and `futures_trade_fills` writer (with `account_id=lead` for Lead). Normal accepted orders and protections no longer trigger a global `pendingReconcile` lock. Unknown submissions are looked up by the original `clientOrderId` rather than blindly resent. The Lead user stream still requires explicit `read_only` opt-in, and live automatic trading remains disabled.
