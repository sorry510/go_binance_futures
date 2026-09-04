package command

import (
	"bufio"
	"fmt"
	"go_binance_futures/utils"
	"os"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func InitData(version int64) error {
	if err := createConfig(version); err != nil {
		return err
	}
	return createStrategyTemplates()
}

func createConfig(version int64) error {
	_, err := orm.NewOrm().Raw("INSERT INTO config (version, future_enable, spot_enable, delivery_enable, futures_position_convert_enable, future_buy_timeout, future_exclude_symbols, future_max_count, future_order_type, future_allow_long, future_allow_short, future_strategy_trade, future_strategy_coin, future_new_enable, spot_new_enable, notice_coin_enable, listen_coin_enable, listen_funding_rate_enable, future_test, future_test_notice_limit_min, ws_futures_enable, ws_spot_enable, ws_delivery_enable, loss_max_count, loss_auto_scale, market_condition, market_condition_is_auto, future_test_auto_trade_count_limit, ws_futures_price_change_limit, ws_futures_fast_move_enable, ws_futures_fast_move_threshold, ws_futures_fast_move_recover, ws_futures_fast_move_cooldown_sec, ws_futures_fast_move_windows, ws_futures_liquidation_enable, ws_futures_liquidation_alert_window_sec, ws_futures_liquidation_alert_notional_threshold, ws_futures_liquidation_alert_cooldown_sec, future_test_fee_rate) VALUES(?, 0, 0, 0, 0, 300, 'BTCUSDT,ETHUSDT,SOLUSDT', 20, 'LIMIT', 0, 0, 'line3', 'coin5', 0, 0, 0, 0, 1, 0, 5, 1, 0, 0, 5, 0, 3, 1, 3, 0, 0, 20, 18, 900, '3m,5m,10m', 0, 60, 5000000, 300, 0.0005);", version).Exec()
	if err != nil {
		logs.Error("init config table error:", err)
	}
	return err
}

func createStrategyTemplates() error {
	o := orm.NewOrm()
	count, err := o.QueryTable("strategy_templates").Count()
	if err != nil {
		logs.Error("count strategy_templates table error:", err)
		return err
	}
	if count > 0 {
		return nil
	}

	filepath := "./command/sql/strategy_templates.sql"
	err = readAndExecuteSQLFile(o, filepath)
	if err != nil {
		logs.Error("init strategy_templates table error:", err)
	}
	return err
}

func ExecSqlFile(filepath string) error {
	// filepath := "./command/sql/strategy_templates.sql"
	err := readAndExecuteSQLFile(orm.NewOrm(), filepath)
	if err != nil {
		logs.Error("init strategy_templates table error:", err)
	}
	return err
}

type rawExecutor interface {
	Raw(query string, args ...interface{}) orm.RawSeter
}

func readAndExecuteSQLFile(executor rawExecutor, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("can't open SQL file: %v", err)
	}
	defer file.Close()

	// 读取文件内容并逐行执行 SQL 语句
	scanner := bufio.NewScanner(file)
	var sqlLines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue // 跳过空行和注释
		}
		sqlLines = append(sqlLines, utils.EscapeJSON(line))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SQL file error: %v", err)
	}

	for _, sqlQuery := range splitSQLStatements(strings.Join(sqlLines, "\n")) {
		_, err = executor.Raw(sqlQuery).Exec()
		if err != nil {
			return fmt.Errorf("exec SQL error: %v, sql: %s", err, sqlQuery)
		}
	}

	return nil
}

func splitSQLStatements(content string) []string {
	statements := make([]string, 0)
	var builder strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	escaped := false

	for _, char := range content {
		builder.WriteRune(char)

		if escaped {
			escaped = false
			continue
		}

		switch char {
		case '\\':
			if inSingleQuote || inDoubleQuote {
				escaped = true
			}
		case '\'':
			if !inDoubleQuote && !inBacktick {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote && !inBacktick {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				statement := strings.TrimSpace(builder.String())
				if statement != "" {
					statements = append(statements, statement)
				}
				builder.Reset()
			}
		}
	}

	statement := strings.TrimSpace(builder.String())
	if statement != "" {
		statements = append(statements, statement)
	}

	return statements
}

func UpdateDatabase(oldVersion int64, newVersion int64) error {
	o := orm.NewOrm()
	to, err := o.Begin()
	if err != nil {
		logs.Error("begin transaction error:", err)
		return err
	}
	version := oldVersion + 1
	for ; version <= newVersion; version++ {
		// 逐个执行数据库更新脚本
		filepath := fmt.Sprintf("./command/sql/version/%d.sql", version)
		if err := readAndExecuteSQLFile(to, filepath); err != nil {
			_ = to.Rollback()
			return fmt.Errorf("update database version %d failed: %w", version, err)
		}
	}

	err = to.Commit()
	if err != nil {
		logs.Error("commit transaction error:", err)
		return err
	}
	return nil
}

// SyncDatabase updates the registered default database to the schema and data
// version required by the current binary.
func SyncDatabase(targetVersion int64) error {
	logs.Info("database sync start, target version:", targetVersion)
	logs.Info("sync database schema...")
	if err := orm.RunSyncdb("default", false, false); err != nil {
		return fmt.Errorf("sync database schema: %w", err)
	}
	logs.Info("sync database schema success")

	systemConfig, err := utils.GetSystemConfig()
	if err != nil {
		logs.Info("system config not found, initializing database data...")
		if err := InitData(0); err != nil {
			return fmt.Errorf("initialize database data: %w", err)
		}
		systemConfig, err = utils.GetSystemConfig()
		if err != nil {
			return fmt.Errorf("load system config after initialization: %w", err)
		}
		logs.Info("initialize database data success")
	}

	if systemConfig.Version < targetVersion {
		logs.Info("update database version:", systemConfig.Version, "->", targetVersion)
		if err := UpdateDatabase(systemConfig.Version, targetVersion); err != nil {
			return err
		}
		systemConfig.Version = targetVersion
		if _, err := orm.NewOrm().Update(&systemConfig, "Version"); err != nil {
			return fmt.Errorf("save database version: %w", err)
		}
		logs.Info("update database version success:", targetVersion)
	} else if systemConfig.Version > targetVersion {
		logs.Warning("database version is newer than current binary:", systemConfig.Version, ">", targetVersion)
	} else {
		logs.Info("database version is already up to date:", systemConfig.Version)
	}

	logs.Info("database sync completed")
	return nil
}
