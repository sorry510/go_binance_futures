# Binance Funding Cap/Floor Saturation — Feasibility

Proposed mechanism: trade true final funding-cap/floor saturation as an extreme crowding/risk-control state, distinct from ordinary funding extremes and the v154 flat-zone transition.

Binance's funding process applies a final cap/floor tied to contract risk parameters. However, Binance Vision historical fundingRate files expose only:
- calc_time
- funding_interval_hours
- last_funding_rate

They do not include the point-in-time cap/floor or historical maintenance-margin bracket used to derive it. Current leverage-bracket data cannot safely be projected backward because risk tiers changed historically.

Therefore a historical event cannot be labeled as "truly cap-saturated" without reconstructing point-in-time risk parameters from an incomplete announcement series.

Decision: **point-in-time data blocked / freeze feasibility**. Do not infer caps from today's brackets or from repeated funding-rate values. No returns read.
