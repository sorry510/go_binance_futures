# Upbit New-Market Announcement Direct-LONG — 2024 Discovery

## Hypothesis

Upbit new trading-support / market-addition announcements may create strong incremental Korean spot demand. The pre-registered trade is LONG on the corresponding Binance USD-M perpetual immediately after the official Upbit announcement.

## Event universe and eligibility

- 44 Upbit 2024 announcement articles.
- 66 token-events after complete ticker parsing.
- 61 unique tokens.
- Binance Vision production eligibility produced 12 eligible events / 12 symbols.
- Eligibility required Binance USD-M history >=730 days and prior-24h QuoteVolume >=5M USDT.

Eligible: JASMY, ARPA, EGLD, FIL, NEAR, XLM, UNI, INJ, GAL, ENS, NEO, SOL.

## Exact 1m discovery result

- 12 trades.
- 1 TP / 11 SL.
- Profit Factor: **0.0961**.
- Normalized net: **-92.89%**.
- Average: **-7.74% / event**.
- Only ENS was profitable (+9.88%).
- INJ suffered approximately -19.99% after next-minute execution, illustrating announcement gap/slippage risk.

## Decision

The direct-LONG hypothesis is decisively rejected in discovery.

**Freeze the Upbit new-market direct-LONG family.** Do not advance to 2025 OOS. Do not reverse the observed 2024 losses into a post-hoc SHORT strategy, and do not filter by market type, token, announcement timing, first-minute reaction, or later listing updates.

Canonical evidence:
- inputs/event_universe.json
- inputs/eligibility.json
- eligibility.py
- replay.py
- results/trades.json
- results/summary.json
