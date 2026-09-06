-- V2-12 controlled trade execution defaults.
-- RunSyncdb creates the additive columns/tables before this migration.
-- Real AI execution stays disabled after upgrade and has no allowed symbols.
UPDATE config
SET agent_trade_execution_enable = 0,
    agent_trade_allowed_symbols = '',
    agent_trade_max_risk_usdt = 5,
    agent_trade_max_notional_usdt = 50,
    agent_trade_max_total_exposure_usdt = 200,
    agent_trade_max_leverage = 3,
    agent_trade_price_freshness_sec = 10,
    agent_trade_max_slippage_bps = 30,
    agent_trade_cooldown_sec = 900,
    agent_trade_proposal_ttl_min = 15
WHERE version < 3;
