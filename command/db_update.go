package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

func InitData(version int64) error {
	createConfig(version)
	createStrategyTemplates()
	return nil
}

func createConfig(version int64) error {
	_, err := orm.NewOrm().Raw("INSERT INTO config (version,future_enable,future_buy_timeout,future_exclude_symbols,future_max_count,future_order_type,future_allow_long,future_allow_short,future_strategy_trade,future_strategy_coin,future_new_enable,spot_new_enable,notice_coin_enable,listen_coin_enable,listen_funding_rate_enable,future_test,future_test_notice_limit_min,spot_enable,delivery_enable,ws_futures_enable,ws_spot_enable,ws_delivery_enable,futures_position_convert_enable) VALUES (?, '0','300','BTCUSDT','10','MARKET','1','1','line3','coin6','0','0','0','0','1',0,65,0,0,1,0,0,0);", version).Exec()
	if err != nil {
		logs.Error("init config table error:", err)
	}
	return err
}

func createStrategyTemplates() error {
	filepath := "./command/sql/strategy_templates.sql"
	err := readAndExecuteSQLFile(filepath)
	if err != nil {
		logs.Error("init strategy_templates table error:", err)
	}
	return err
}

func readAndExecuteSQLFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("can't open SQL file: %v", err)
	}
	defer file.Close()

	// 创建一个新的 ORM 实例
	o := orm.NewOrm()

	// 读取文件内容并逐行执行 SQL 语句
	scanner := bufio.NewScanner(file)
	var sqlStatements []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "--") {
			continue // 跳过空行和注释
		}
		sqlStatements = append(sqlStatements, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SQL file error: %v", err)
	}

	// 将多行 SQL 语句合并为一个完整的 SQL 语句
	sqlQuery := strings.Join(sqlStatements, " ")

	// 执行 SQL 语句
	_, err = o.Raw(sqlQuery).Exec()
	if err != nil {
		return fmt.Errorf("exec SQL error: %v", err)
	}

	return nil
}

func UpdateDatabase(oldVersion int64, newVersion int64) error {
	o := orm.NewOrm()
    to, err := o.Begin()
	if err != nil {
		logs.Error("begin transaction error:", err)
		return err
	}
	version := oldVersion + 1
	for ;version <= newVersion; version++ {
		// 逐个执行数据库更新脚本
		filepath := fmt.Sprintf("./command/sql/version/%d.sql", version)
		readAndExecuteSQLFile(filepath)
	}
	
	err = to.Commit()
	if err != nil {
		logs.Error("commit transaction error:", err)
		return err
	}
	return nil
}