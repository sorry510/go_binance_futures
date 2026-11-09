package tradefills

// SchemaDDL is the single SQL source of truth for both production migration
// and every SQLite ledger test fixture. Never register the ledger in Beego ORM:
// the string DTO would alter its financial DECIMAL columns to VARCHAR.
const SchemaDDL = `CREATE TABLE IF NOT EXISTS futures_trade_fills (
    market VARCHAR(16) NOT NULL,
    account_id VARCHAR(16) NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    order_id VARCHAR(32) NOT NULL,
    trade_id VARCHAR(32) NOT NULL,
    position_id BIGINT NULL,
    client_order_id VARCHAR(64) NULL,
    intent VARCHAR(24) NULL,
    position_side VARCHAR(8) NULL,
    qty DECIMAL(36,18) NOT NULL,
    price DECIMAL(36,18) NOT NULL,
    commission DECIMAL(36,18) NOT NULL,
    commission_asset VARCHAR(16) NOT NULL,
    realized_pnl DECIMAL(36,18) NOT NULL,
    pnl_asset VARCHAR(16) NOT NULL,
    trade_time BIGINT NOT NULL,
    rest_verified TINYINT NOT NULL DEFAULT 0,
    PRIMARY KEY (market, account_id, symbol, order_id, trade_id)
)`
