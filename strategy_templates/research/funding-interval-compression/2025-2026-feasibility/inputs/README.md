# Inputs

The feasibility audit reads local research tables only:
- market_funding_rates
- market_klines_1h

No raw market archive is duplicated in this bundle.

gap_audit.go reports observed settlement-gap frequencies after 2025-05-02.
replay.go searches for repeated 8h -> 4h transitions and applies the >=730-day history gate.
