# Gate USDT Futures Insurance-Fund Stress — Feasibility

Mechanism: large insurance-fund drawdowns can mark liquidation stress and forced deleveraging. The proposed use was Gate USDT futures insurance-fund balance as an external systemic-risk signal for Binance USD-M.

Gate public API endpoint: GET /api/v4/futures/usdt/insurance.

On 2026-09-29, limit=100 and limit=1000 both returned only 31 daily observations, covering 2026-08-30 through 2026-09-29. The endpoint exposes no historical from/to parameters.

Decision: historical-data blocked before any return inspection. Keep only as a possible future forward-monitoring source; do not backfill from unofficial scrapes.
