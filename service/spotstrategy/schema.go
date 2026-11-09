package spotstrategy

import "strings"

const (
	AccountMain  = "main"
	OwnerAuto    = "auto_strategy"
	ModeLegacy   = "legacy"
	ModeStrategy = "strategy"
	ModeOff      = "off"
)

// The financial tables are SQL-owned: Beego's string ORM fields would change
// their DECIMAL columns into VARCHAR during RunSyncdb. All indexes fit the
// MySQL 5.6 utf8mb4 767-byte limit.
var SchemaDDL = []string{
	`CREATE TABLE IF NOT EXISTS spot_strategy_holdings (
 id {{PRIMARY_ID}},
 account_id VARCHAR(16) NOT NULL,
 owner VARCHAR(32) NOT NULL,
 symbol VARCHAR(32) NOT NULL,
 base_asset VARCHAR(16) NOT NULL,
 quote_asset VARCHAR(16) NOT NULL,
 owned_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
 avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
 status VARCHAR(24) NOT NULL,
 active_slot VARCHAR(96) NULL,
 created_at BIGINT NOT NULL,
 updated_at BIGINT NOT NULL,
 closed_at BIGINT NOT NULL DEFAULT 0,
 UNIQUE (active_slot)
)`,
	`CREATE TABLE IF NOT EXISTS spot_strategy_orders (
 id {{PRIMARY_ID}},
 holding_id BIGINT NULL,
 account_id VARCHAR(16) NOT NULL,
 owner VARCHAR(32) NOT NULL,
 symbol VARCHAR(32) NOT NULL,
 side VARCHAR(8) NOT NULL,
 order_type VARCHAR(16) NOT NULL,
 intent VARCHAR(16) NOT NULL,
 client_order_id VARCHAR(36) NOT NULL,
 exchange_order_id VARCHAR(32) NULL,
 signal_key VARCHAR(64) NOT NULL,
 pending_slot VARCHAR(96) NULL,
 requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
 filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
 executed_quote_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
 reserved_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
 status VARCHAR(32) NOT NULL,
 last_error VARCHAR(255) NOT NULL DEFAULT '',
 created_at BIGINT NOT NULL,
 updated_at BIGINT NOT NULL,
 last_checked_at BIGINT NOT NULL DEFAULT 0,
 UNIQUE (client_order_id),
 UNIQUE (pending_slot),
 UNIQUE (account_id,owner,symbol,signal_key)
)`,
	`CREATE TABLE IF NOT EXISTS spot_strategy_fills (
 account_id VARCHAR(16) NOT NULL,
 symbol VARCHAR(32) NOT NULL,
 order_id VARCHAR(32) NOT NULL,
 trade_id VARCHAR(32) NOT NULL,
 holding_id BIGINT NULL,
 client_order_id VARCHAR(36) NOT NULL,
 side VARCHAR(8) NOT NULL,
 qty DECIMAL(36,18) NOT NULL,
 price DECIMAL(36,18) NOT NULL,
 quote_qty DECIMAL(36,18) NOT NULL,
 commission DECIMAL(36,18) NOT NULL,
 commission_asset VARCHAR(16) NOT NULL,
 trade_time BIGINT NOT NULL,
 rest_verified TINYINT NOT NULL DEFAULT 0,
 PRIMARY KEY (account_id,symbol,order_id,trade_id)
)`,
}

// SchemaFor provides canonical schema for both production and SQLite tests.
func SchemaFor(mysql bool) []string {
	id := "INTEGER PRIMARY KEY AUTOINCREMENT"
	if mysql {
		id = "BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"
	}
	out := make([]string, len(SchemaDDL))
	for i, ddl := range SchemaDDL {
		out[i] = strings.ReplaceAll(ddl, "{{PRIMARY_ID}}", id)
		if mysql {
			out[i] += " ENGINE=InnoDB"
		}
	}
	return out
}
