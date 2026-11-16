# AI and Agents

[中文](../zh-CN/07-AI-Agent.md) · [Back to English README](../../../README.EN.md)

## AI features

### Chat

**AI → Chat** is the unified human entry point for the Agent platform. Select any Skill that is allowed in chat. For symbol analysis, the composer includes a searchable selector backed by the application's real futures contract list; the selected Symbol is sent explicitly and takes precedence over text parsing, so Chinese names, aliases, or free-form wording cannot accidentally select another market. Without a Symbol, the Skill can still handle ordinary conversation, and only explicit follow-up wording reuses the previous Symbol.

### Symbol Analysis

Open **AI → Market Analysis → Symbol Analysis**.

Select a real USDT perpetual Symbol and optionally add an analysis focus. The application runs the `symbol_analysis` Skill with market data, current market regime, and the latest successful comparison context, then returns a structured `TradingPlan`. History shows status, direction, confidence, market regime, analysis-time price, current price, later movement, and summary.

### Opportunities

**AI → Market Analysis → Opportunities** is the V3-6 Opportunity Watch entry point. The system can discover candidates from FastMove, liquidation Signals, MarketEvent/Binance Announcement, and scheduled `market_scan`, then reuse the existing `symbol_analysis` Skill after deterministic deduplication and cooldown checks to produce a complete `TradingPlanV1`. Opportunities expire after 60 minutes by default, and the same Symbol + source type is analyzed at most once per 30-minute cooldown window.

- Opportunity Watch is disabled by default. Enable it under **AI → AI Settings → AI Configuration**, the first/default tab. Minimum confidence accepts **1%–100%** with one decimal place, saves automatically, and verifies the server-read value; the API stores a 0–1 fractional value. The backend rejects empty or malformed patches and confidence values outside 1%–100%, rather than reporting a successful save.
- `neutral`, `partial`, `data_missing`, `failed`, and expired opportunities are view-only and cannot create a real-trading Proposal.
- Eligible opportunities still require the existing controlled-trading Risk Engine, human approval, and V3-5 Ownership-safe Execution. The Futures Positions page no longer provides Proposal creation, approval, or execution controls. An Opportunity has no direct Binance submission path.
- The detail view reads the original `symbol_analysis` Agent Task for the full plan and can jump back to Symbol Analysis for a fresh run; full AI JSON is not duplicated into the Opportunity record.

### Model Configuration

Open **AI → AI Settings → Models**.

LLM configurations are database-backed. The page supports provider templates and custom providers, configuration name, API endpoint, API key, model, request timeout, Temperature, connection testing, and switching the active model. New Agent tasks use the active configuration without restarting the service. API keys are never returned in plaintext.

### Skill Management

Skill registration and governance settings are stored in the database. Native and Portable Skills are shown in separate tabs with search and pagination. The page supports create/edit, enable/disable, deletion, and a separate **Chat Enabled** switch. `enabled` controls whether a Skill can run; `chat_enabled` only controls whether it is exposed to chat. Portable `allowed-tools` entries are permission requests and cannot self-grant high-risk or trading access.

### MCP Management

The application can act as an MCP Client for third-party standard HTTP MCP Servers and bring remote Tools, Resources, and Prompts into the same Runtime, Permission, Trace, and Context system. The UI supports connection tests, Catalog refresh, authentication/OAuth, and per-Skill grants. External capabilities identified as real trading actions are forced into the high-risk trade class and kept disabled so they cannot bypass controlled execution.

### Memory Management

Long-term Memory is persisted separately from Conversations and Tasks. The page exposes content, source, Scope, TTL, status, and update time. Memory contributes context only; it does not gain Tool or trading privileges.

### Business Workflows

Business Workflows reuse the existing Agent Runtime instead of creating a second Agent stack:

- `market_scan`: a deterministic Scanner narrows the market first, then the Agent analyzes a small candidate set and returns opportunities.
- `strategy_review`: reviews a strategy template using existing test results, post-fee performance, and the current market regime; it proposes changes but never edits the formal template directly.
- `strategy_experiment` (**Strategy Draft Validation**): the Agent proposes a candidate, deterministic expression validation and fixed-scenario tests run, and the Agent summarizes the result. This is **not a historical single-position backtest**, and it never overwrites a formal strategy.
- `alert_triage`: when AI is enabled, related same-symbol Signals arriving close together can be grouped into an Incident to reduce duplicate notifications; AI-off/failure paths keep deterministic fallback behavior.
- `daily_market_brief`: aggregates market regime, Scanner results, and important Signals into a fixed-schema brief; its Scheduler is disabled by default.

### Controlled Trading

**UI change:** Futures Positions now only shows the position/order ownership card, manual claims and reconciliation. Trade Proposal creation/approval/execution controls were removed from this page. The backend process below remains documented but is not exposed by this page.

Controlled trading does **not** expose a direct trading Tool to the LLM. Real execution follows this fixed path:

`successful symbol_analysis → Trade Proposal → deterministic Risk Engine → human approval → risk re-check → Execution Service → Binance → Audit`.

- Real AI execution is disabled by default and the allowed-symbol list starts empty.
- A Proposal can only originate from a successful structured symbol analysis; the LLM does not choose the final quantity.
- The Risk Engine checks the allowlist, direction, current market regime and regime drift, price freshness, Entry Zone, Stop Loss, estimated slippage, existing positions/open orders, duplicates, cooldown, leverage, per-trade risk, notional size, total exposure, and the Kill Switch.
- Final quantity is computed deterministically from maximum risk, stop distance, maximum notional, and the exchange StepSize.
- Risk is run again before real execution, and the Kill Switch is read again immediately before broker submission.
- A unique `client_order_id` makes submission idempotent. Ambiguous network results are never automatically resubmitted; they can only be reconciled by that ID.
- Proposal, Risk, approval/rejection, execution, and reconciliation actions are audited.
- After a controlled-trade entry fills, the V3-5 Ownership Lifecycle creates the required `STOP_MARKET` protection for the managed position and, when the TradingPlan has a valid target, an optional `TAKE_PROFIT_MARKET`; after a manual partial reduction it reconciles and repairs protection to the remaining managed quantity.

### AI Configuration

All AI runtime configuration is centralized under **AI → AI Settings → AI Configuration**, the first/default tab, including:

- Alert Pipeline / AI analysis switches, minimum severity, cooldown, concurrency, and per-minute limits.
- Opportunity Watch master switch, Market Scan interval, and minimum confidence (1%–100%, saved and verified via API read-back).
- Market-regime update and daily-market-brief Schedulers.
- Global Agent admission limits plus Token and Tool-call budgets.
- Controlled-trading Kill Switch, Symbol allowlist, max per-trade risk/notional, total exposure, leverage, price freshness, slippage, cooldown, and Proposal TTL.

The remaining AI Settings tabs are Models, Skills, MCP, and Memory. These runtime settings have been moved out of **Configuration Center**, which now contains only non-AI trading, market-monitoring, alert, and system runtime settings.

### Task Center, System Dashboard, and Alert History

- **Task Center:** governance state, direct Runtime trade permission vs controlled execution state, Scheduler jobs, Task history, model identity, tokens, and task details.
- **System Dashboard:** the former **AI → Observability** page is now the first top-level menu. It shows six health checks: Database, Binance REST, consolidated market-data WS, LLM, Agent Runtime, and Trade Safety. Expanding WS details lists USD-M Futures (symbols), Spot (spot_symbols), and COIN-M Futures (delivery_symbols) freshness separately; non-started COIN-M WS remains disabled rather than an error. Announcement, market intelligence, MCP and scheduler details are collapsed by default. Agent task counts, success, tokens, LLM errors, traces and Skill/model/tool metrics remain. The REST budget and 429/418 counters remain visible while detailed rankings are on demand. The local REST usage snapshot refreshes silently every second (no overlapping polls or loading indicators; failed polls retain the previous snapshot) and system health every 60 seconds; changing Agent time ranges does not re-query REST/health. Lead status remains on its dedicated page, and change history was removed from the dashboard.
- **Alert Pipeline History:** the end-to-end Signal path from Event Bus and Signal Engine through AI analysis/triage to Notification or deterministic fallback.
