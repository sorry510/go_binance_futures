package command

import "fmt"

type migrationColumn struct {
	table, name, definition string
}

var v22Columns = []migrationColumn{
	{"futures_managed_positions", "settlement_status", "VARCHAR(16) NOT NULL DEFAULT 'legacy'"},
	{"futures_managed_positions", "settlement_processed_at", "BIGINT NOT NULL DEFAULT 0"},
	{"futures_managed_positions", "settlement_pnl_sign", "INT NOT NULL DEFAULT 0"},
	{"config", "settlement_scale_balance", "INT NOT NULL DEFAULT 0"},
	{"config", "settlement_consecutive_losses", "INT NOT NULL DEFAULT 0"},
}

// The exact names are a fixed allowlist, never caller-supplied SQL fragments.
func columnExists(o rawExecutor, table, column string) (bool, error) {
	var count int64
	var err error
	if defaultDatabaseIsMySQL() {
		err = o.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, column).QueryRow(&count)
	} else {
		err = o.Raw(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?", table), column).QueryRow(&count)
	}
	return count > 0, err
}
func ensureV22SettlementColumns(o rawExecutor) error {
	for _, col := range v22Columns {
		exists, err := columnExists(o, col.table, col.name)
		if err != nil {
			return fmt.Errorf("inspect %s.%s: %w", col.table, col.name, err)
		}
		if !exists {
			sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", col.table, col.name, col.definition)
			if _, err = o.Raw(sql).Exec(); err != nil {
				return fmt.Errorf("add %s.%s: %w", col.table, col.name, err)
			}
		}
	}
	// Old rows are explicitly excluded from settlement; never mark them open.
	for _, sql := range []string{
		"UPDATE futures_managed_positions SET settlement_status='legacy' WHERE settlement_status IS NULL OR settlement_status=''",
		"UPDATE futures_managed_positions SET settlement_processed_at=0 WHERE settlement_processed_at IS NULL",
		"UPDATE futures_managed_positions SET settlement_pnl_sign=0 WHERE settlement_pnl_sign IS NULL",
		"UPDATE config SET settlement_scale_balance=0 WHERE settlement_scale_balance IS NULL",
		"UPDATE config SET settlement_consecutive_losses=0 WHERE settlement_consecutive_losses IS NULL",
	} {
		if _, err := o.Raw(sql).Exec(); err != nil {
			return fmt.Errorf("normalize v22 settlement defaults: %w", err)
		}
	}
	return ensureV22SettlementIndexes(o)
}
func ensureV22SettlementIndexes(o rawExecutor) error {
	// Explicitly create indexes even when the automatic ORM schema upgrade
	// fails to add its indexes on MySQL 8. The lookup is idempotent.
	for _, entry := range []struct{ table, index, column string }{
		{"futures_managed_positions", "idx_fmp_settlement_status", "settlement_status"},
		{"futures_managed_positions", "idx_fmp_settlement_processed", "settlement_processed_at"},
	} {
		var count int64
		if defaultDatabaseIsMySQL() {
			err := o.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?",
				entry.table, entry.index).QueryRow(&count)
			if err != nil {
				return fmt.Errorf("inspect settlement index %s: %w", entry.index, err)
			}
		} else {
			err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", entry.index).QueryRow(&count)
			if err != nil {
				return err
			}
		}
		if count == 0 {
			ddl := "CREATE INDEX "
			if !defaultDatabaseIsMySQL() {
				ddl += "IF NOT EXISTS "
			}
			ddl += entry.index + " ON " + entry.table + "(" + entry.column + ")"
			if _, err := o.Raw(ddl).Exec(); err != nil {
				return fmt.Errorf("create settlement index %s: %w", entry.index, err)
			}
		}
	}
	return nil
}
func ensureV23RESTVerificationColumn(o rawExecutor) error {
	exists, err := columnExists(o, "futures_trade_fills", "rest_verified")
	if err != nil {
		return fmt.Errorf("inspect trade fill REST verification column: %w", err)
	}
	if exists {
		return nil
	}
	_, err = o.Raw("ALTER TABLE futures_trade_fills ADD COLUMN rest_verified TINYINT NOT NULL DEFAULT 0").Exec()
	if err != nil {
		return fmt.Errorf("add REST fill confirmation column: %w", err)
	}
	return nil
}
