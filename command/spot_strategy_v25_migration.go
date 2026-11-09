package command

import (
	"fmt"

	"go_binance_futures/service/spotstrategy"
)

// ensureSpotStrategyV25 is idempotent because MySQL DDL can implicitly commit.
// It may be retried by 'sync db' after a partial migration.
func ensureSpotStrategyV25(o rawExecutor) error {
	columns := []migrationColumn{
		{"config", "spot_auto_trade_mode", "VARCHAR(16) NOT NULL DEFAULT 'legacy'"},
		{"spot_symbols", "strategy_trade_enabled", "INT NOT NULL DEFAULT 0"},
		{"spot_symbols", "strategy_budget_quote", "VARCHAR(40) NOT NULL DEFAULT '0'"},
	}
	for _, col := range columns {
		exists, err := columnExists(o, col.table, col.name)
		if err != nil {
			return fmt.Errorf("inspect spot v25 %s.%s: %w", col.table, col.name, err)
		}
		if !exists {
			if _, err := o.Raw(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", col.table, col.name, col.definition)).Exec(); err != nil {
				return fmt.Errorf("add spot v25 %s.%s: %w", col.table, col.name, err)
			}
		}
	}
	// Preserve legacy behavior for upgraded accounts. Unknown values are not
	// silently replaced: runtime gate refuses them instead.
	if _, err := o.Raw("UPDATE config SET spot_auto_trade_mode = 'legacy' WHERE spot_auto_trade_mode IS NULL OR spot_auto_trade_mode = ''").Exec(); err != nil {
		return fmt.Errorf("normalize spot mode: %w", err)
	}
	if _, err := o.Raw("UPDATE spot_symbols SET strategy_budget_quote = '0' WHERE strategy_budget_quote IS NULL OR strategy_budget_quote = ''").Exec(); err != nil {
		return fmt.Errorf("normalize spot budget: %w", err)
	}
	for _, ddl := range spotstrategy.SchemaFor(defaultDatabaseIsMySQL()) {
		if _, err := o.Raw(ddl).Exec(); err != nil {
			return fmt.Errorf("create spot strategy schema: %w", err)
		}
	}
	for _, index := range []struct{ table, name, columns string }{
		{"spot_strategy_holdings", "idx_ssh_account_symbol_status", "account_id,owner,symbol,status"},
		{"spot_strategy_orders", "idx_sso_holding_status", "holding_id,status"},
		{"spot_strategy_orders", "idx_sso_account_symbol_status", "account_id,symbol,status"},
		{"spot_strategy_fills", "idx_ssf_client_order", "client_order_id"},
	} {
		var found int64
		var query string
		if defaultDatabaseIsMySQL() {
			query = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?"
		} else {
			query = "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
		}
		var err error
		if defaultDatabaseIsMySQL() {
			err = o.Raw(query, index.table, index.name).QueryRow(&found)
		} else {
			err = o.Raw(query, index.name).QueryRow(&found)
		}
		if err != nil {
			return fmt.Errorf("inspect spot index %s: %w", index.name, err)
		}
		if found > 0 {
			continue
		}
		ddl := fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", index.name, index.table, index.columns)
		if _, err := o.Raw(ddl).Exec(); err != nil {
			return fmt.Errorf("create spot index %s: %w", index.name, err)
		}
	}
	return nil
}
