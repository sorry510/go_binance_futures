# Futures Trading and Orders

[中文](../zh-CN/02-合约交易与订单.md) · [Back to English README](../../../README.EN.md)

## Futures Trading

The **Futures Trading → Futures Trading** page allows per-symbol configurations for technical indicators, strategy methods and templates, margin mode, USDT collateral, leverage, take-profit and stop-loss percentages, and enabled status. Every Futures symbol uses its own custom strategy; there is no strategy-type selector.

### Global Futures Switch Semantics (UF-1)

The **Configuration Center → Futures Trading → Enable Futures Trading** switch controls **NEW Main auto_strategy entries only**. When OFF:

- Main no longer submits new \`auto_strategy\` OPEN orders.
- Existing Main \`auto_strategy\` Managed positions still follow their normal strategy/TP/SL exit rules.
- Outstanding owned orders continue order-status reconciliation, expiration handling and verified-Fill recovery.
- Turning OFF **does not force close a position or cancel all protective orders**. CLOSE orders require their original strategy exit conditions.
- When no Main auto_strategy work is outstanding, the system avoids additional authenticated account polling from this trading loop.

The Main switch does not change independently scoped Lead, COIN-M or Spot entry permissions.

### Why Did My Strategy Not Open a Position? (UF-4)

Under **Futures Trading → Futures Trading**, expand **Why no entry?** to inspect only the real decisions from the Main strategy runner's latest evaluation. Filter by symbol and LONG/SHORT. Results distinguish the global/symbol switches, selector exclusions, existing positions/orders, risk limits, unmatched signals, unavailable order-book depth, failed margin/leverage preflight and order submission errors.

- The panel is collapsed by default and polls its **in-process snapshot every 15 seconds only when expanded**. It never reruns a strategy, calls Binance or writes to the database.
- **Last accepted order submission** is an order acknowledgement, not an authenticated Fill; inspect real Futures Orders for verified execution.
- Restarting the backend clears the snapshot. The UI shows the observation time and warns after 30 seconds; an unselected symbol is not automatically classified as low volume or stale market data.
- This covers Main U-margined entries only. Existing Managed exits still execute when new entries are disabled.

### Enabling Futures Trading

1. In the **Configuration Center → Futures Trading** Card, turn on **Enable Futures Trading**.
2. Enable **Allow Long** and/or **Allow Short** as needed; configure the symbol-selection strategy, position limits, and order type.
![Futures configuration](../../../img/ui/config/funture-config.png)
3. Under **Futures Trading → Futures Trading**, verify the target symbol's custom strategy, trade parameters, and **Enabled** status.
![Futures symbol list](../../../img/ui/futures/futures-list.png)

#### Important Notes

1. A symbol's close strategy is evaluated only after the configured **Take Profit (%)** or **Stop Loss (%)** threshold has been exceeded.
2. Each Futures symbol uses its own custom strategy configuration.
3. The symbol's USDT amount represents the Futures margin.

### Configuration Options

#### Fast Price Movement (Optional)

**Configuration Center → Fast Price Movement** controls whether fast-move monitoring is enabled, its threshold, recovery threshold, cooldown, and monitoring windows.

#### Liquidations (Optional)

**Configuration Center → Liquidations** controls liquidation-order collection, BTC liquidation signal aggregation windows, notional threshold, and cooldown.

#### Position PnL Sign-Change Notifications (Optional)

This switch is below **Order Type** in **Configuration Center → Futures Trading**. It sends a notification when a position's PnL changes from positive to negative or vice versa. Large numbers of positions or enabled symbols may increase API requests, so enable it with care.

#### Allow Long and Allow Short (At Least One Required for Trading)

These are direction-wide switches. When a direction is disabled, a matching strategy signal cannot automatically open a position in that direction.

#### Symbol-Selection Strategy (Required)

Determines the pool of candidates for automatic trading. It does not alter the custom trading strategy executed by an individual symbol.

#### Maximum Positions (Required)

When the limit is reached, no further positions are opened automatically even if a strategy signal is valid.

#### Maximum Losing Positions (Required)

No new automatic position is opened after this limit is reached. **Automatically Scale Maximum Losing Positions** can adjust the limit based on consecutive wins/losses.

#### Market Condition (Optional)

**Configuration Center → Market Condition** allows manual market-condition selection or automatic updates. The Card title shows an **Auto / Manual** status tag, and the Card displays progress and recent AI analysis.

#### Automatic Order Type

`LIMIT` creates limit orders; `MARKET` creates market orders. The UI displays the applicable options according to the selected type.

### Quantitative Trading Flow

1. With the global Futures switch ON, the Main runner evaluates new entries using each symbol's strategy. Existing Main auto_strategy Managed positions may still exit by the original rules even when the global switch is OFF.
2. With `LIMIT` entry orders, a pending order is placed. It may not fill, and the system cancels it after the configured timeout.
3. After an entry fills successfully, its records are available under **Futures Orders** and **Futures Positions**.

## Verified Live Strategy Performance (UF-5)

On **Futures Trading → Futures Orders**, expand **Verified live strategy performance** to inspect the previous **7 or 30 days**. Filter by exact symbol, LONG/SHORT, or original 64-character rule hash (use `unknown` for legacy records without a reliable rule version). The period is based on **`closed_at`**, not the history table's opening date or current page.

- Only **Main / auto_strategy** cycles formally marked `settlement_status=settled`, with fully authenticated matching opening/closing Binance REST trade Fills, are included. Legacy, incomplete, missing-Fill, unconvertible-fee, or pre-approved-verification-cutoff (`BINANCE_ORDER_HISTORY_T0_MAIN_MS`) cycles are listed separately under exclusion reasons.
- **Net PnL = realized exchange PnL minus actual exchange commissions**. Profit factor is positive net PnL total divided by the absolute negative net PnL total; with no losing trades it is undefined, not infinity. Break-even trades are counted separately. Commissions in unrelated assets (e.g. BNB) with no verified conversion cause exclusion.
- **USDT and USDC amounts are not added together**. The reported *cumulative settled net PnL drawdown* is a peak-to-trough **currency amount** calculated from closed, certified net PnL only, **not an account equity maximum drawdown**. Unrealized mark-to-market gains and funding fees are not part of the metric.
- Presentation-only manual history edits do not authenticate Fills or change the verified performance calculation. This panel is strictly read-only: it does not place orders, reconcile exchange balances, or modify settlement markers.

## Faster Futures Order Pages and Full-Filter PnL (UF-6)

On **Futures Trading → Futures Orders**, the first query reads the current page and the entire filtered PnL as separate requests. Page 1→2→3 and changing the number of records per page **only retrieve the requested page**, without re-scanning every matching cycle. Changing account, Owner, symbol, side, status, settlement, or opening-date filters and pressing Search recomputes the full-filter PnL.

- Explicit **Refresh**, or a successful bulk history change/deletion, forces the list and the full-filter summary to recalculate. The reused aggregate is an in-page browser snapshot, **not a live exchange account balance**.
- The UI shows when the summary was **calculated** and when the newest underlying local **price snapshot** was last updated. Additional Fills or market-price updates may be reflected only after refreshing. A failed summary is shown as an error, never a fabricated zero.
- Existing `GET /orders` consumers keep their complete legacy response by default. The new screen opts into `GET /orders?include_summary=false` for page-only reads and `GET /orders/summary` for the complete aggregate.
- All values preserve the previous order-history calculation: realized price-spread PnL plus unrealized estimates based on local `symbols.close`, no commissions or funding, separate USDT/USDC totals and an explicit count of unpriceable cycles. This is **not the same metric** as UF-5's formally settled REST-certified net PnL.
