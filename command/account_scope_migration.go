package command

import (
	"fmt"
	"strings"
)

// accountScopeMigration runs only after ORM has added account_id columns.
// It backfills legacy data as main and adds safe composite lookup indexes.
// No legacy rows are reinterpreted as managed ownership.
func accountScopeMigration(executor rawExecutor) error {
	tables := []string{"futures_managed_positions", "futures_managed_orders", "futures_positions", "futures_orders", "order"}
	for _, table := range tables {
		query := fmt.Sprintf("UPDATE `%s` SET account_id = 'main' WHERE account_id IS NULL OR account_id = ''", table)
		if _, err := executor.Raw(query).Exec(); err != nil {
			return fmt.Errorf("backfill %s account_id: %w", table, err)
		}
		var bad int64
		if err := executor.Raw(fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE account_id NOT IN ('main','lead') OR account_id IS NULL", table)).QueryRow(&bad); err != nil {
			return err
		}
		if bad != 0 {
			return fmt.Errorf("refuse unknown account_id in %s (%d rows)", table, bad)
		}
	}
	return nil
}

// MySQL 5.6 InnoDB commonly uses the 767-byte index-key limit. Pre-Stage-2
// ORM models created several identity columns as VARCHAR(255) under utf8mb4.
// A full composite index would exceed the limit. For Binance identity fields,
// use short prefix indexes on MySQL: equality of full valid IDs always implies
// equality of these prefixes. Oversized legacy values are rejected rather than
// silently coalesced; the application only writes Binance-compliant IDs.
//
// This leaves existing column definitions/data untouched, and also works with
// a partially applied v19 DDL migration (all CREATE INDEX operations are
// independently checked for existence before attempting them).
var accountIndexPrefixLength = map[string]int{
	"account_id":    16,
	"symbol":        32,
	"side":          8,
	"position_side": 8,
	"order_id":      32,
	"owner":         32,
	"status":        32,
}

func accountIndexColumns(columns string, isMySQL bool) (string, error) {
	parts := strings.Split(columns, ",")
	quoted := make([]string, 0, len(parts))
	bytes := 0
	for _, raw := range parts {
		column := strings.TrimSpace(raw)
		if column == "" {
			return "", fmt.Errorf("empty account index column")
		}
		prefix, allowed := accountIndexPrefixLength[column]
		if !allowed {
			return "", fmt.Errorf("unregistered account index column %q", column)
		}
		item := "`" + column + "`"
		if isMySQL {
			item += fmt.Sprintf("(%d)", prefix)
			bytes += prefix * 4 // conservative utf8mb4 estimate
		}
		quoted = append(quoted, item)
	}
	if isMySQL && bytes > 767 {
		return "", fmt.Errorf("MySQL 5.6 index too large: %d bytes", bytes)
	}
	return strings.Join(quoted, ","), nil
}

// Both full identity strings and the prefixed indexes are safe only if all
// historic values are within Binance's supported identity sizes. Prevent a
// rebuild from quietly conflating any oversized legacy identifiers.
func validateAccountIndexData(executor rawExecutor, isMySQL bool) error {
	if !isMySQL {
		return nil
	}
	checks := [][3]string{
		{"futures_positions", "symbol", "32"},
		{"futures_positions", "side", "8"},
		{"futures_orders", "order_id", "32"},
		{"futures_managed_positions", "symbol", "32"},
		{"futures_managed_positions", "position_side", "8"},
		{"futures_managed_orders", "owner", "32"},
		{"futures_managed_orders", "status", "32"},
		{"order", "symbol", "32"},
		{"order", "side", "8"},
	}
	for _, check := range checks {
		table, col, max := check[0], check[1], check[2]
		var invalid int64
		sql := "SELECT COUNT(*) FROM `" + table + "` WHERE CHAR_LENGTH(`" + col + "`) > " + max
		if err := executor.Raw(sql).QueryRow(&invalid); err != nil {
			return fmt.Errorf("validate %s.%s: %w", table, col, err)
		}
		if invalid > 0 {
			return fmt.Errorf("refuse v19 index: %d %s.%s values exceed %s characters; back up and inspect before retry", invalid, table, col, max)
		}
	}
	return nil
}

// In MySQL the existing indexes and historical order_id uniqueness are not
// altered in place. Composite secondary indexes are additive, not unique.
func ensureAccountScopeIndexes(executor rawExecutor) error {
	if err := validateAccountIndexData(executor, defaultDatabaseIsMySQL()); err != nil {
		return err
	}
	indexes := [][3]string{
		{"futures_managed_positions", "idx_fmp_account_symbol_side", "account_id,symbol,position_side"},
		{"futures_managed_orders", "idx_fmo_account_owner_status", "account_id,owner,status"},
		{"futures_orders", "idx_fo_account_order", "account_id,order_id"},
		{"futures_positions", "idx_fp_account_symbol_side", "account_id,symbol,side"},
		{"order", "idx_hist_account_side_symbol", "account_id,side,symbol"},
	}
	for _, index := range indexes {
		table, name, columns := index[0], index[1], index[2]
		if defaultDatabaseIsMySQL() {
			var count int64
			err := executor.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", table, name).QueryRow(&count)
			if err != nil {
				return err
			}
			if count != 0 {
				continue
			}
		}
		columnSQL, err := accountIndexColumns(columns, defaultDatabaseIsMySQL())
		if err != nil {
			return fmt.Errorf("build account index %s: %w", name, err)
		}
		sql := "CREATE INDEX "
		if !defaultDatabaseIsMySQL() {
			sql += "IF NOT EXISTS "
		}
		sql += "`" + name + "` ON `" + table + "` (" + columnSQL + ")"
		if _, err := executor.Raw(sql).Exec(); err != nil {
			return fmt.Errorf("create account index %s: %w", name, err)
		}
	}
	return nil
}

// Unique keys are installed after the legacy backfill. If the old mirror has
// duplicate logical rows, abort rather than silently picking a row or deleting
// user data. Operators can inspect/repair the affected table before re-sync.
func ensureAccountMirrorUniqueIndexes(executor rawExecutor) error {
	for _, index := range [][4]string{
		{"futures_positions", "uq_fp_account_symbol_side", "account_id,symbol,side", "symbol,side"},
		{"futures_orders", "uq_fo_account_order", "account_id,order_id", "order_id"},
	} {
		table, name, columns, _ := index[0], index[1], index[2], index[3]
		var duplicates int64
		sql := "SELECT COUNT(*) FROM (SELECT " + columns + " FROM `" + table + "` GROUP BY " + columns + " HAVING COUNT(*)>1) duplicate_slots"
		if err := executor.Raw(sql).QueryRow(&duplicates); err != nil {
			return fmt.Errorf("check %s unique rows: %w", table, err)
		}
		if duplicates > 0 {
			return fmt.Errorf("refuse unique mirror index %s: %d duplicate account-scoped slots exist; inspect and resolve before retry", table, duplicates)
		}
		if defaultDatabaseIsMySQL() {
			var found int64
			err := executor.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?", table, name).QueryRow(&found)
			if err != nil {
				return err
			}
			if found != 0 {
				continue
			}
		}
		columnSQL, err := accountIndexColumns(columns, defaultDatabaseIsMySQL())
		if err != nil {
			return fmt.Errorf("build unique account index %s: %w", name, err)
		}
		statement := "CREATE UNIQUE INDEX "
		if !defaultDatabaseIsMySQL() {
			statement += "IF NOT EXISTS "
		}
		statement += "`" + name + "` ON `" + table + "` (" + columnSQL + ")"
		if _, err := executor.Raw(statement).Exec(); err != nil {
			return fmt.Errorf("create %s: %w", name, err)
		}
	}
	return nil
}
