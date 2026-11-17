package command

import (
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// v27 adds an independent inverse-futures ledger. The four tables are raw
// SQL, not Beego models: exact DECIMAL columns must never be converted to
// VARCHAR/float. Never migrate or reassign existing USDT-M, Lead or Spot data.
func ensureV27CoinMSchema(o orm.Ormer) error {
	// budget_usd is the only new delivery_symbols field not managed by ORM.
	// Its default zero prevents legacy usdt=10 from authorizing CM orders.
	exists, err := spotQuantColumnExists(o, "delivery_symbols", "budget_usd")
	if err != nil {
		return fmt.Errorf("inspect COIN-M budget column: %w", err)
	}
	if !exists {
		if _, err := o.Raw("ALTER TABLE delivery_symbols ADD COLUMN budget_usd DECIMAL(36,18) NOT NULL DEFAULT 0").Exec(); err != nil {
			return fmt.Errorf("add COIN-M budget column: %w", err)
		}
	}

	id := "INTEGER PRIMARY KEY AUTOINCREMENT"
	if defaultDatabaseIsMySQL() {
		id = "BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY"
	} else if spotQuantIsPostgres() {
		id = "BIGSERIAL PRIMARY KEY"
	}
	tables := []struct {
		name string
		sql  string
	}{
		{"coin_m_managed_positions", fmt.Sprintf("CREATE TABLE IF NOT EXISTS coin_m_managed_positions ("+
			"id %s, symbol VARCHAR(32) NOT NULL, position_side VARCHAR(8) NOT NULL, "+
			"margin_asset VARCHAR(32) NOT NULL, contract_size DECIMAL(36,18) NOT NULL DEFAULT 0, "+
			"entry_price DECIMAL(36,18) NOT NULL DEFAULT 0, "+
			"opened_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, "+
			"remaining_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, "+
			"realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0, status VARCHAR(32) NOT NULL, "+
			"created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL, closed_at BIGINT NOT NULL DEFAULT 0)", id)},
		{"coin_m_managed_orders", fmt.Sprintf("CREATE TABLE IF NOT EXISTS coin_m_managed_orders ("+
			"id %s, position_id BIGINT NOT NULL, symbol VARCHAR(32) NOT NULL, "+
			"position_side VARCHAR(8) NOT NULL, intent VARCHAR(16) NOT NULL, "+
			"side VARCHAR(4) NOT NULL, order_type VARCHAR(32) NOT NULL, "+
			"client_order_id VARCHAR(64) NOT NULL, exchange_order_id VARCHAR(64) NULL, "+
			"requested_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, "+
			"filled_contracts DECIMAL(36,18) NOT NULL DEFAULT 0, status VARCHAR(32) NOT NULL, "+
			"created_at BIGINT NOT NULL, updated_at BIGINT NOT NULL)", id)},
		{"coin_m_trade_fills", fmt.Sprintf("CREATE TABLE IF NOT EXISTS coin_m_trade_fills ("+
			"id %s, position_id BIGINT NOT NULL, order_id BIGINT NOT NULL, "+
			"symbol VARCHAR(32) NOT NULL, position_side VARCHAR(8) NOT NULL, side VARCHAR(4) NOT NULL, "+
			"exchange_order_id VARCHAR(64) NOT NULL, trade_id VARCHAR(64) NOT NULL, "+
			"contracts DECIMAL(36,18) NOT NULL, price DECIMAL(36,18) NOT NULL, "+
			"realized_pnl DECIMAL(36,18) NOT NULL DEFAULT 0, pnl_asset VARCHAR(32) NOT NULL, "+
			"commission DECIMAL(36,18) NOT NULL DEFAULT 0, commission_asset VARCHAR(32) NOT NULL, "+
			"trade_time BIGINT NOT NULL)", id)},
		{"coin_m_strategy_templates", fmt.Sprintf("CREATE TABLE IF NOT EXISTS coin_m_strategy_templates ("+
			"id %s, name VARCHAR(128) NOT NULL, technology TEXT NOT NULL, "+
			"strategy TEXT NOT NULL, create_time BIGINT NOT NULL, update_time BIGINT NOT NULL)", id)},
	}
	for _, table := range tables {
		if _, err := o.Raw(table.sql).Exec(); err != nil {
			return fmt.Errorf("create %s: %w", table.name, err)
		}
	}
	for _, index := range []struct {
		name, table, columns string
	}{
		{"uq_coin_m_order_client", "coin_m_managed_orders", "client_order_id"},
		{"uq_coin_m_order_exchange", "coin_m_managed_orders", "symbol, exchange_order_id"},
		{"uq_coin_m_fill_trade", "coin_m_trade_fills", "symbol, exchange_order_id, trade_id"},
		{"idx_coin_m_position_symbol_status", "coin_m_managed_positions", "symbol, position_side, status"},
		{"idx_coin_m_order_position", "coin_m_managed_orders", "position_id"},
		{"idx_coin_m_fill_position", "coin_m_trade_fills", "position_id"},
		{"idx_coin_m_fill_order", "coin_m_trade_fills", "order_id"},
	} {
		present, err := coinMIndexExists(o, index.name)
		if err != nil {
			return fmt.Errorf("check CM index %s: %w", index.name, err)
		}
		if present {
			continue
		}
		prefix := "CREATE INDEX"
		if strings.HasPrefix(index.name, "uq_") {
			prefix = "CREATE UNIQUE INDEX"
		}
		ddl := fmt.Sprintf("%s %s ON %s (%s)", prefix, index.name, index.table, index.columns)
		if _, err := o.Raw(ddl).Exec(); err != nil {
			return fmt.Errorf("create CM index %s: %w", index.name, err)
		}
	}
	return VerifyV27CoinMSchema(o)
}

func coinMIndexExists(o orm.Ormer, name string) (bool, error) {
	var count int64
	var err error
	switch {
	case defaultDatabaseIsMySQL():
		err = o.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND index_name=?", name).QueryRow(&count)
	case spotQuantIsPostgres():
		err = o.Raw("SELECT COUNT(*) FROM pg_indexes WHERE schemaname=current_schema() AND indexname=?", name).QueryRow(&count)
	default:
		err = o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", name).QueryRow(&count)
	}
	return count > 0, err
}

// coinMUniqueIndexExists verifies that essential order/Fill deduplication keys
// are UNIQUE, not just indexes with the expected name.
func coinMUniqueIndexExists(o orm.Ormer, table, name string) (bool, error) {
	var count int64
	switch {
	case defaultDatabaseIsMySQL():
		err := o.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name=? AND index_name=? AND non_unique=0", table, name).QueryRow(&count)
		return count > 0, err
	case spotQuantIsPostgres():
		err := o.Raw("SELECT COUNT(*) FROM pg_indexes WHERE schemaname=current_schema() AND tablename=? AND indexname=? AND indexdef LIKE 'CREATE UNIQUE INDEX%'", table, name).QueryRow(&count)
		return count > 0, err
	default:
		var rows []struct {
			Name   string `orm:"column(name)"`
			Unique int    `orm:"column(unique)"`
		}
		_, err := o.Raw("PRAGMA index_list('" + table + "')").QueryRows(&rows)
		if err != nil {
			return false, err
		}
		for _, item := range rows {
			if item.Name == name {
				return item.Unique == 1, nil
			}
		}
		return false, nil
	}
}

func coinMDecimalType(o orm.Ormer, table, column string) (string, error) {
	var typ string
	switch {
	case defaultDatabaseIsMySQL():
		err := o.Raw("SELECT COLUMN_TYPE FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", table, column).QueryRow(&typ)
		return typ, err
	case spotQuantIsPostgres():
		var p, s int
		err := o.Raw("SELECT numeric_precision, numeric_scale FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=? AND column_name=?", table, column).QueryRow(&p, &s)
		return fmt.Sprintf("DECIMAL(%d,%d)", p, s), err
	default:
		err := o.Raw(fmt.Sprintf("SELECT type FROM pragma_table_info('%s') WHERE name=?", table), column).QueryRow(&typ)
		return typ, err
	}
}

// ensureV27CoinMFlagType mirrors the existing Spot Quant MySQL handling.
// Beego maps int8 to INT; the actual opt-in flag should be TINYINT.
func ensureV27CoinMFlagType(o orm.Ormer) error {
	if !defaultDatabaseIsMySQL() {
		return nil
	}
	var typ string
	if err := o.Raw("SELECT DATA_TYPE FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?", "config", "coin_m_quant_enable").QueryRow(&typ); err != nil {
		return fmt.Errorf("inspect COIN-M flag type: %w", err)
	}
	if strings.EqualFold(typ, "tinyint") {
		return nil
	}
	if !strings.EqualFold(typ, "int") {
		return fmt.Errorf("unexpected COIN-M flag column type: %s", typ)
	}
	var invalid int64
	if err := o.Raw("SELECT COUNT(*) FROM config WHERE coin_m_quant_enable NOT IN (0,1) OR coin_m_quant_enable IS NULL").QueryRow(&invalid); err != nil {
		return fmt.Errorf("validate COIN-M flag before narrowing: %w", err)
	}
	if invalid != 0 {
		return fmt.Errorf("refuse to narrow COIN-M flag: %d invalid values", invalid)
	}
	if _, err := o.Raw("ALTER TABLE config MODIFY COLUMN coin_m_quant_enable TINYINT NOT NULL DEFAULT 0").Exec(); err != nil {
		return fmt.Errorf("narrow COIN-M flag to TINYINT: %w", err)
	}
	return nil
}

// VerifyV27CoinMSchema is read-only. A database already claiming v27 with
// missing tables, columns, exact decimals, or unique keys MUST fail sync.
// It must never silently regenerate a lost Managed ledger.
func VerifyV27CoinMSchema(o orm.Ormer) error {
	required := []struct {
		table   string
		columns []string
	}{
		{"config", []string{"coin_m_quant_enable"}},
		{"delivery_symbols", []string{"enable", "contract_size", "margin_asset", "contract_type", "budget_usd"}},
		{"coin_m_managed_positions", []string{"id", "symbol", "position_side", "margin_asset", "contract_size", "entry_price", "opened_contracts", "remaining_contracts", "realized_pnl", "status", "created_at", "updated_at", "closed_at"}},
		{"coin_m_managed_orders", []string{"id", "position_id", "symbol", "position_side", "intent", "side", "order_type", "client_order_id", "exchange_order_id", "requested_contracts", "filled_contracts", "status", "created_at", "updated_at"}},
		{"coin_m_trade_fills", []string{"id", "position_id", "order_id", "symbol", "position_side", "side", "exchange_order_id", "trade_id", "contracts", "price", "realized_pnl", "pnl_asset", "commission", "commission_asset", "trade_time"}},
		{"coin_m_strategy_templates", []string{"id", "name", "technology", "strategy", "create_time", "update_time"}},
	}
	for _, item := range required {
		for _, column := range item.columns {
			exists, err := spotQuantColumnExists(o, item.table, column)
			if err != nil {
				return fmt.Errorf("inspect v27 %s.%s: %w", item.table, column, err)
			}
			if !exists {
				return fmt.Errorf("missing v27 %s.%s", item.table, column)
			}
		}
	}
	for _, item := range []struct {
		table   string
		columns []string
	}{
		{"delivery_symbols", []string{"budget_usd"}},
		{"coin_m_managed_positions", []string{"contract_size", "entry_price", "opened_contracts", "remaining_contracts", "realized_pnl"}},
		{"coin_m_managed_orders", []string{"requested_contracts", "filled_contracts"}},
		{"coin_m_trade_fills", []string{"contracts", "price", "realized_pnl", "commission"}},
	} {
		for _, column := range item.columns {
			typ, err := coinMDecimalType(o, item.table, column)
			if err != nil {
				return fmt.Errorf("read v27 %s.%s numeric type: %w", item.table, column, err)
			}
			if !strings.EqualFold(typ, "DECIMAL(36,18)") {
				return fmt.Errorf("invalid v27 %s.%s: got %s, require DECIMAL(36,18)", item.table, column, typ)
			}
		}
	}
	for _, idx := range []string{
		"uq_coin_m_order_client", "uq_coin_m_order_exchange", "uq_coin_m_fill_trade",
		"idx_coin_m_position_symbol_status", "idx_coin_m_order_position",
		"idx_coin_m_fill_position", "idx_coin_m_fill_order",
	} {
		present, err := coinMIndexExists(o, idx)
		if err != nil {
			return fmt.Errorf("inspect v27 index %s: %w", idx, err)
		}
		if !present {
			return fmt.Errorf("missing v27 index %s", idx)
		}
	}
	for _, idx := range []struct{ table, name string }{
		{"coin_m_managed_orders", "uq_coin_m_order_client"},
		{"coin_m_managed_orders", "uq_coin_m_order_exchange"},
		{"coin_m_trade_fills", "uq_coin_m_fill_trade"},
	} {
		unique, err := coinMUniqueIndexExists(o, idx.table, idx.name)
		if err != nil {
			return fmt.Errorf("verify v27 unique key %s: %w", idx.name, err)
		}
		if !unique {
			return fmt.Errorf("v27 index %s is not UNIQUE", idx.name)
		}
	}
	var badCount int64
	if err := o.Raw("SELECT COUNT(*) FROM config WHERE coin_m_quant_enable NOT IN (0,1) OR coin_m_quant_enable IS NULL").QueryRow(&badCount); err != nil {
		return fmt.Errorf("validate CM toggle: %w", err)
	}
	if badCount > 0 {
		return fmt.Errorf("invalid CM global switch values: %d", badCount)
	}
	return nil
}
