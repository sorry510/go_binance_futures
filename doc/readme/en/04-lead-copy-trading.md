# Lead Copy Trading

[中文](../zh-CN/04-合约带单.md) · [Back to English README](../../../README.EN.md)

## Lead Copy Trading (Lead Account 4)

> **Current status: the management UI and account-isolation infrastructure are implemented and verified with Mock/Race tests. Live Lead Portfolio identity, MySQL 8 physical schema v25, private WS, Algo orders, and funded execution have NOT passed their live gates. Live Lead order execution remains locked.** This feature is separate from AI Controlled Trading and cannot guarantee any follower fills.

Open **Futures Trade → Lead Copy Trading** (`/futures/lead`). The page has six Cards:

| Card | What is available today / data boundary |
| --- | --- |
| Account Status | Disabled/blocking reasons, masked Lead key, Portfolio *label*, latest read-only verification and private WS status. The label is not verified Portfolio identity. |
| Dedicated Credentials | Encrypted AES-GCM storage of a separate Lead API Key/Secret and user-triggered read-only verification. Secrets are never returned; rotating credentials revokes the old result and observer. |
| Account Risk | Lead-only maximum order/total notional, daily loss, drawdown and position-count limits. Strategy and symbol selection continue to use Main's existing settings; there is no duplicate strategy editor. |
| Automated Trading | Trading authorization is visibly locked; **Pause New Entries** is available. The backend `/start` route still denies activation. |
| Managed Positions | **Local Lead Ownership** including closed historical records; this is not proof of Binance live positions, manual positions, or real-time PnL. |
| Orders & Diagnostics | Locally managed Lead orders and fixed-code operations audit. The `/reconcile` write-back operation remains locked without live authoritative verification. |

The **System Dashboard** now includes a read-only Lead summary, private WS health, five-minute REST request count/weight, and a link to the Lead page. Alert drops display **`—` when no queue is running, rather than a misleading zero**. The Lead page refreshes local status approximately every 15 seconds; viewing it does not enable live orders.

#### Separate and secure Lead configuration

Lead credentials, risk settings, and audit events are **not stored in** `conf/app.conf` or frontend code. Supply the following to the application process using protected environment/secrets management (these are placeholders; **do not put real secrets in the README, Git, logs, or shell history**):

```text
BINANCE_LEAD_CREDENTIAL_KEY=<standard-base64-of-32-random-bytes>
BINANCE_LEAD_CREDENTIAL_FILE=/private/lead/credentials.enc
BINANCE_LEAD_CONFIG_FILE=/private/lead/risk.json
BINANCE_LEAD_AUDIT_FILE=/private/lead/operations.jsonl
# Do not set BINANCE_LEAD_OBSERVER_MODE by default
```

- `BINANCE_LEAD_CREDENTIAL_KEY` must decode from **standard Base64 to exactly 32 random bytes**. Generate and persist it securely: losing/rotating this master key makes the existing encrypted credentials unreadable.
- All three file paths must be **absolute and outside the repository**, with an operator-owned private parent directory (recommended `0700`) and private files (`0600`), not symlinks. The app does not silently create private directories or generate a master key.
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
2. **Start with UI and access controls**: open all six Cards, switch languages and refresh. Missing JWT must deny Lead APIs; an authenticated session must still show `enabled=false` and `allow_new_opens=false`. Verify risk settings persist and invalid/cross-account/enable requests are denied.
3. **Test Pause last**: `POST /futures/lead/pause` first asserts a **one-way, process-local emergency stop**, then attempts durable config/audit writes. When persistence succeeds it returns HTTP 200; otherwise HTTP 202 with `pause_persisted=false`. A process-local stop cannot guarantee safety after a restart or in another process. There is no UI resume switch.
4. **Test the observer separately**: only under explicit permission, set `read_only` and restart the updated backend. Check WS/snapshot health, outstanding-order inspection, account ownership, and compare live figures with the official Binance Portfolio UI. Read-only status must always keep `live_trade_authorized=false`.
5. **Live orders remain off-limits**: `/start`, unverified `/reconcile`, and unsafe credential deletion return HTTP 423 (or fail earlier with a safe storage/audit error when not configured). STOP/TP Algo support, dynamic price filters, partial fills/manual reductions, multi-process restart recovery, real dual-account two-second API load, and minimum-size funded trades require separate Stage 7-5/7-6 gates and explicit human authorization.

See the Lead Copy Trading staged implementation plan (referenced document not present in checkout: `doc/trade/币安合约自动带单-Stage实施方案.md`) and the implementation/review logs in `doc/trade/`. **This is a personal research/operations feature, not a declaration that live copy trading is production-ready.**
