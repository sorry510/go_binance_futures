package command

import (
	"fmt"
	"go_binance_futures/service/tradefills"
)

// ensureFuturesTradeFillsTable creates the one new immutable USDM fill ledger.
// The numeric values are SQL DECIMAL, not ORM float64 fields. The composite
// exchange identity is the primary key to reject duplicate WS/REST events.
func ensureFuturesTradeFillsTable(executor rawExecutor) error {
	statement := tradefills.SchemaDDL
	if _, err := executor.Raw(statement).Exec(); err != nil {
		return fmt.Errorf("create futures_trade_fills: %w", err)
	}

	// A partial MySQL DDL migration is retryable. SQLite supports IF NOT EXISTS.
	const indexName = "idx_ftf_position_id"
	if defaultDatabaseIsMySQL() {
		var exists int64
		if err := executor.Raw(
			"SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='futures_trade_fills' AND index_name=?",
			indexName,
		).QueryRow(&exists); err != nil {
			return fmt.Errorf("check futures fills position index: %w", err)
		}
		if exists != 0 {
			return nil
		}
	}
	indexDDL := "CREATE INDEX "
	if !defaultDatabaseIsMySQL() {
		indexDDL += "IF NOT EXISTS "
	}
	indexDDL += indexName + " ON futures_trade_fills (position_id)"
	if _, err := executor.Raw(indexDDL).Exec(); err != nil {
		return fmt.Errorf("create futures fills position index: %w", err)
	}
	return nil
}
