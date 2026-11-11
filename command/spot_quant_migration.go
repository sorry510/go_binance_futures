package command

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// Spot Quant tables are intentionally not ORM-managed. Decimal fields must
// remain SQL DECIMAL(36,18), not become VARCHAR or floating-point columns.
// This migration creates only the four independent Spot Quant business tables.
func ensureV25SpotQuantTables(o orm.Ormer) error {
	idColumn := "INTEGER PRIMARY KEY AUTOINCREMENT"
	if defaultDatabaseIsMySQL() {
		idColumn = "BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"
	} else if spotQuantIsPostgres() {
		idColumn = "BIGSERIAL PRIMARY KEY"
	}

	tables := []struct {
		name, ddl string
		columns   []string
	}{
		{
			name: "spot_managed_positions",
			ddl: fmt.Sprintf(`CREATE TABLE IF NOT EXISTS spot_managed_positions (
				id %s,
				symbol VARCHAR(32) NOT NULL,
				base_asset VARCHAR(32) NOT NULL,
				quote_asset VARCHAR(32) NOT NULL,
				bought_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
				remaining_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
				avg_cost_quote DECIMAL(36,18) NOT NULL DEFAULT 0,
				realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0,
				status VARCHAR(32) NOT NULL,
				created_at BIGINT NOT NULL,
				updated_at BIGINT NOT NULL,
				closed_at BIGINT NOT NULL DEFAULT 0
			)`, idColumn),
			columns: []string{"id", "symbol", "base_asset", "quote_asset", "bought_qty", "remaining_qty", "avg_cost_quote", "realized_pnl", "status", "created_at", "updated_at", "closed_at"},
		},
		{
			name: "spot_managed_orders",
			ddl: fmt.Sprintf(`CREATE TABLE IF NOT EXISTS spot_managed_orders (
				id %s,
				position_id BIGINT NOT NULL,
				symbol VARCHAR(32) NOT NULL,
				side VARCHAR(4) NOT NULL,
				order_type VARCHAR(24) NOT NULL,
				client_order_id VARCHAR(64) NOT NULL,
				exchange_order_id VARCHAR(64) NULL,
				requested_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
				filled_qty DECIMAL(36,18) NOT NULL DEFAULT 0,
				status VARCHAR(32) NOT NULL,
				created_at BIGINT NOT NULL,
				updated_at BIGINT NOT NULL,
				CONSTRAINT uq_spot_orders_client UNIQUE (client_order_id),
				CONSTRAINT uq_spot_orders_exchange UNIQUE (symbol, exchange_order_id)
			)`, idColumn),
			columns: []string{"id", "position_id", "symbol", "side", "order_type", "client_order_id", "exchange_order_id", "requested_qty", "filled_qty", "status", "created_at", "updated_at"},
		},
		{
			name: "spot_trade_fills",
			ddl: fmt.Sprintf(`CREATE TABLE IF NOT EXISTS spot_trade_fills (
				id %s,
				position_id BIGINT NOT NULL,
				order_id BIGINT NOT NULL,
				exchange_order_id VARCHAR(64) NOT NULL,
				trade_id VARCHAR(64) NOT NULL,
				symbol VARCHAR(32) NOT NULL,
				side VARCHAR(4) NOT NULL,
				qty DECIMAL(36,18) NOT NULL,
				price DECIMAL(36,18) NOT NULL,
				quote_qty DECIMAL(36,18) NOT NULL,
				commission DECIMAL(36,18) NOT NULL,
				commission_asset VARCHAR(32) NOT NULL,
				trade_time BIGINT NOT NULL,
				CONSTRAINT uq_spot_fill_exchange UNIQUE (symbol, exchange_order_id, trade_id)
			)`, idColumn),
			columns: []string{"id", "position_id", "order_id", "exchange_order_id", "trade_id", "symbol", "side", "qty", "price", "quote_qty", "commission", "commission_asset", "trade_time"},
		},
		{
			name: "spot_strategy_templates",
			ddl: fmt.Sprintf(`CREATE TABLE IF NOT EXISTS spot_strategy_templates (
				id %s,
				name VARCHAR(128) NOT NULL,
				technology TEXT NOT NULL,
				strategy TEXT NOT NULL,
				create_time BIGINT NOT NULL,
				update_time BIGINT NOT NULL
			)`, idColumn),
			columns: []string{"id", "name", "technology", "strategy", "create_time", "update_time"},
		},
	}

	for _, table := range tables {
		if _, err := o.Raw(table.ddl).Exec(); err != nil {
			return fmt.Errorf("create %s: %w", table.name, err)
		}
		// IF NOT EXISTS is not enough: do not silently reuse an unrelated
		// table from an obsolete implementation.
		for _, column := range table.columns {
			exists, err := spotQuantColumnExists(o, table.name, column)
			if err != nil {
				return fmt.Errorf("inspect %s.%s: %w", table.name, column, err)
			}
			if !exists {
				return fmt.Errorf("incompatible existing %s: missing %s", table.name, column)
			}
		}
	}

	indexes := []struct{ name, table, columns string }{
		{"idx_spot_positions_symbol_status", "spot_managed_positions", "symbol, status"},
		{"idx_spot_orders_position", "spot_managed_orders", "position_id"},
		{"idx_spot_fills_position", "spot_trade_fills", "position_id"},
		{"idx_spot_fills_order", "spot_trade_fills", "order_id"},
	}
	for _, index := range indexes {
		if defaultDatabaseIsMySQL() {
			var count int64
			if err := o.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=?", index.table, index.name).QueryRow(&count); err != nil {
				return fmt.Errorf("inspect index %s: %w", index.name, err)
			}
			if count > 0 {
				continue
			}
		}
		ddl := "CREATE INDEX "
		if !defaultDatabaseIsMySQL() {
			ddl += "IF NOT EXISTS "
		}
		ddl += index.name + " ON " + index.table + " (" + index.columns + ")"
		if _, err := o.Raw(ddl).Exec(); err != nil {
			return fmt.Errorf("create index %s: %w", index.name, err)
		}
	}
	return nil
}

func spotQuantIsPostgres() bool {
	db, err := orm.GetDB("default")
	if err != nil {
		return false
	}
	name := strings.ToLower(fmt.Sprintf("%T", db.Driver()))
	return strings.Contains(name, "pq") || strings.Contains(name, "postgres")
}

// Inputs are exclusively fixed, developer-defined table/column identifiers.
func spotQuantColumnExists(o orm.Ormer, table, column string) (bool, error) {
	if spotQuantIsPostgres() {
		var count int64
		err := o.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=? AND column_name=?", table, column).QueryRow(&count)
		return count > 0, err
	}
	return columnExists(o, table, column)
}

// ensureV25SpotQuantFlagTypes corrects Beego's generic INT mapping for Go
// integer flag fields. It touches only the two v25 opt-in columns on MySQL.
func ensureV25SpotQuantFlagTypes(o orm.Ormer) error {
	if !defaultDatabaseIsMySQL() {
		return nil
	}
	for _, field := range []struct{ table, column string }{
		{"config", "spot_quant_enable"},
		{"spot_symbols", "spot_quant_enabled"},
	} {
		var dataType string
		err := o.Raw(
			"SELECT DATA_TYPE FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?",
			field.table, field.column,
		).QueryRow(&dataType)
		if err != nil {
			return fmt.Errorf("inspect spot quant flag %s.%s: %w", field.table, field.column, err)
		}
		if strings.EqualFold(dataType, "tinyint") {
			continue
		}
		if !strings.EqualFold(dataType, "int") {
			return fmt.Errorf("refuse unexpected spot quant flag type %s.%s: %s", field.table, field.column, dataType)
		}
		var invalid int64
		err = o.Raw(fmt.Sprintf(
			"SELECT COUNT(*) FROM %s WHERE %s IS NULL OR %s NOT IN (0,1)",
			field.table, field.column, field.column,
		)).QueryRow(&invalid)
		if err != nil {
			return fmt.Errorf("validate spot quant flag %s.%s: %w", field.table, field.column, err)
		}
		if invalid != 0 {
			return fmt.Errorf("refuse to narrow %s.%s: %d non-boolean values", field.table, field.column, invalid)
		}
		ddl := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s TINYINT NOT NULL DEFAULT 0", field.table, field.column)
		if _, err := o.Raw(ddl).Exec(); err != nil {
			return fmt.Errorf("narrow spot quant flag %s.%s: %w", field.table, field.column, err)
		}
	}
	return nil
}

// VerifyV25SpotQuantSchema is read-only. It is also called at application
// startup: matching config.version alone is not sufficient to start a v25
// binary, since the retired Spot branch used the same version number.
func VerifyV25SpotQuantSchema(o orm.Ormer) error {
	required := []struct {
		table   string
		columns []string
	}{
		{"config", []string{"spot_quant_enable"}},
		{"spot_symbols", []string{"spot_quant_enabled"}},
		{"spot_managed_positions", []string{"id", "symbol", "base_asset", "quote_asset", "bought_qty", "remaining_qty", "avg_cost_quote", "realized_pnl", "status", "created_at", "updated_at", "closed_at"}},
		{"spot_managed_orders", []string{"id", "position_id", "symbol", "side", "order_type", "client_order_id", "exchange_order_id", "requested_qty", "filled_qty", "status", "created_at", "updated_at"}},
		{"spot_trade_fills", []string{"id", "position_id", "order_id", "exchange_order_id", "trade_id", "symbol", "side", "qty", "price", "quote_qty", "commission", "commission_asset", "trade_time"}},
		{"spot_strategy_templates", []string{"id", "name", "technology", "strategy", "create_time", "update_time"}},
	}
	for _, item := range required {
		for _, column := range item.columns {
			exists, err := spotQuantColumnExists(o, item.table, column)
			if err != nil {
				return fmt.Errorf("read %s.%s: %w", item.table, column, err)
			}
			if !exists {
				return fmt.Errorf("missing %s.%s (v25 Spot Quant schema incomplete)", item.table, column)
			}
		}
	}

	// These fields are SQL DECIMAL, never float or ORM-generated VARCHAR.
	for _, item := range []struct {
		table   string
		columns []string
	}{
		{"spot_managed_positions", []string{"bought_qty", "remaining_qty", "avg_cost_quote", "realized_pnl"}},
		{"spot_managed_orders", []string{"requested_qty", "filled_qty"}},
		{"spot_trade_fills", []string{"qty", "price", "quote_qty", "commission"}},
	} {
		for _, column := range item.columns {
			var actual string
			switch {
			case defaultDatabaseIsMySQL():
				if err := o.Raw("SELECT COLUMN_TYPE FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", item.table, column).QueryRow(&actual); err != nil {
					return fmt.Errorf("read numeric type %s.%s: %w", item.table, column, err)
				}
			case spotQuantIsPostgres():
				var precision, scale int
				if err := o.Raw("SELECT numeric_precision, numeric_scale FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=? AND column_name=?", item.table, column).QueryRow(&precision, &scale); err != nil {
					return fmt.Errorf("read numeric type %s.%s: %w", item.table, column, err)
				}
				actual = fmt.Sprintf("decimal(%d,%d)", precision, scale)
			default:
				if err := o.Raw(fmt.Sprintf("SELECT type FROM pragma_table_info('%s') WHERE name=?", item.table), column).QueryRow(&actual); err != nil {
					return fmt.Errorf("read numeric type %s.%s: %w", item.table, column, err)
				}
			}
			if !strings.EqualFold(actual, "DECIMAL(36,18)") {
				return fmt.Errorf("unexpected type %s.%s: %s; want DECIMAL(36,18)", item.table, column, actual)
			}
		}
	}
	return nil
}
