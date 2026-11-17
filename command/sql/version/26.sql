-- Lead account risk and default-OFF live entry authorization; no key material.
CREATE TABLE IF NOT EXISTS lead_trade_settings (
    account_id VARCHAR(16) NOT NULL PRIMARY KEY,
    allow_new_opens SMALLINT NOT NULL DEFAULT 0,
    max_order_notional_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_total_notional_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    max_daily_realized_loss_usdt DECIMAL(20,8) NOT NULL DEFAULT 0,
    updated_at BIGINT NOT NULL DEFAULT 0
);
