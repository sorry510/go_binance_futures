package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type armSnapshotTemplate struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	TechnologyID string `json:"technology_semantic_sha256"`
	StrategyID   string `json:"strategy_semantic_sha256"`
}

type armSnapshotDB struct {
	Database         string                `json:"database"`
	AsOfMS           int64                 `json:"as_of_ms"`
	TemplateCount    int64                 `json:"strategy_template_count"`
	TemplateIDMin    int64                 `json:"strategy_template_id_min"`
	TemplateIDMax    int64                 `json:"strategy_template_id_max"`
	ResultCount      int64                 `json:"test_strategy_result_count"`
	ResultIDMin      int64                 `json:"test_strategy_result_id_min"`
	ResultIDMax      int64                 `json:"test_strategy_result_id_max"`
	VersionTemplates []armSnapshotTemplate `json:"version_templates"`
	Tables           *armTableCapture      `json:"-"`
}

type armTableColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
}

type armTableCapture struct {
	AsOfMS    int64                       `json:"as_of_ms"`
	RowCount  int64                       `json:"row_count"`
	IDMin     int64                       `json:"id_min"`
	IDMax     int64                       `json:"id_max"`
	Schemas   map[string][]armTableColumn `json:"schemas"`
	Rows      []map[string]any            `json:"rows"`
	Templates []map[string]any            `json:"templates"`
}

func main() {
	if err := snapshotARM(); err != nil {
		fmt.Fprintln(os.Stderr, "ARM snapshot error:", err)
		os.Exit(1)
	}
}

func snapshotARM() error {
	confPath := flag.String("conf", "conf/app.conf", "configuration with the commented ARM block")
	outputPath := flag.String("output", "", "optional JSON output path")
	v29ExportPath := flag.String("v29-export", "", "optional portable v29 export from go_binance")
	tableExportPath := flag.String("table-export", "", "optional complete read-only table capture for all three databases")
	diagnoseMinuteSymbol := flag.String("diagnose-minute-symbol", "", "inspect a read-only ARM minute-query plan and active state for one symbol")
	flag.Parse()
	cfg, err := parseArmResearchConfig(*confPath)
	if err != nil {
		return err
	}
	if *diagnoseMinuteSymbol != "" {
		return diagnoseARMMinuteQuery(cfg, *diagnoseMinuteSymbol)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	items := make([]armSnapshotDB, 0, 3)
	captures := make(map[string]*armTableCapture)
	for _, database := range []string{"go_binance", "go_bn_oracle1", "go_bn_oracle2"} {
		item, err := snapshotARMDatabase(ctx, cfg, database, *tableExportPath != "")
		if err != nil {
			return fmt.Errorf("%s: %w", database, err)
		}
		items = append(items, item)
		if item.Tables != nil {
			captures[database] = item.Tables
		}
		fmt.Printf("%s as_of=%d templates=%d results=%d versions=%d\n", database, item.AsOfMS, item.TemplateCount, item.ResultCount, len(item.VersionTemplates))
	}
	if *tableExportPath != "" {
		content, err := json.MarshalIndent(struct {
			Source    string                      `json:"source"`
			ReadOnly  bool                        `json:"read_only"`
			Databases map[string]*armTableCapture `json:"databases"`
		}{Source: "commented # arm in conf/app.conf; separate repeatable-read snapshots per database", ReadOnly: true, Databases: captures}, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(*tableExportPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(*tableExportPath, append(content, '\n'), 0o600); err != nil {
			return err
		}
		fmt.Println("complete table capture saved:", *tableExportPath)
	}
	if *v29ExportPath != "" {
		if len(items[0].VersionTemplates) != 1 {
			return fmt.Errorf("go_binance has %d v29 templates, expected one", len(items[0].VersionTemplates))
		}
		if err := exportARMV29(ctx, cfg, items[0].VersionTemplates[0].ID, *v29ExportPath); err != nil {
			return err
		}
	}
	content, err := json.MarshalIndent(struct {
		Source    string          `json:"source"`
		ReadOnly  bool            `json:"read_only"`
		Databases []armSnapshotDB `json:"databases"`
	}{Source: "commented # arm in conf/app.conf", ReadOnly: true, Databases: items}, "", "  ")
	if err != nil {
		return err
	}
	if *outputPath == "" {
		fmt.Println(string(content))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(*outputPath, append(content, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Println("snapshot saved:", *outputPath)
	return nil
}

func diagnoseARMMinuteQuery(cfg armResearchConfig, symbol string) error {
	if strings.TrimSpace(symbol) == "" || strings.ContainsAny(symbol, " %_") {
		return fmt.Errorf("invalid minute diagnostic symbol")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := openArmResearchDB(cfg, "go_binance")
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx,
		"SELECT ID,TIME,COALESCE(STATE,''),COALESCE(INFO,'') FROM information_schema.PROCESSLIST WHERE DB=? AND INFO LIKE '%market_klines_1m%' AND INFO NOT LIKE '%information_schema.PROCESSLIST%' ORDER BY TIME DESC LIMIT 5",
		"go_binance")
	if err != nil {
		return fmt.Errorf("inspect ARM process list: %w", err)
	}
	for rows.Next() {
		var id, seconds int64
		var state, query string
		if err := rows.Scan(&id, &seconds, &state, &query); err != nil {
			rows.Close()
			return err
		}
		fmt.Printf("minute_query id=%d seconds=%d state=%q query_bytes=%d\n", id, seconds, state, len(query))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	var chunks, points, first, last, largestBytes, totalBytes int64
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*),COALESCE(SUM(point_count),0),COALESCE(MIN(start_time),0),COALESCE(MAX(end_time),0),COALESCE(MAX(OCTET_LENGTH(payload)),0),COALESCE(SUM(OCTET_LENGTH(payload)),0) FROM market_klines_1m_chunks WHERE market=? AND symbol=?",
		researchMarket, symbol).Scan(&chunks, &points, &first, &last, &largestBytes, &totalBytes); err != nil {
		return fmt.Errorf("inspect ARM minute chunks: %w", err)
	}
	fmt.Printf("minute_chunks symbol=%s chunks=%d points=%d first=%d last=%d largest_bytes=%d total_bytes=%d\n", symbol, chunks, points, first, last, largestBytes, totalBytes)
	const explain = "EXPLAIN FORMAT=JSON SELECT open_time,close_time,open_price,high_price,low_price,close_price,volume,quote_volume,trade_count,taker_buy_quote_volume FROM market_klines_1m FORCE INDEX (market) WHERE market=? AND symbol=? AND open_time>=? AND open_time<=? ORDER BY open_time"
	var plan string
	if err := db.QueryRowContext(ctx, explain, researchMarket, symbol,
		time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
		time.Date(2026, 8, 31, 23, 59, 0, 0, time.UTC).UnixMilli()).Scan(&plan); err != nil {
		return fmt.Errorf("explain ARM minute query: %w", err)
	}
	fmt.Println(plan)
	return nil
}

func exportARMV29(ctx context.Context, cfg armResearchConfig, id int64, path string) error {
	db, err := openArmResearchDB(cfg, "go_binance")
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name, technology, strategy string
	if err := tx.QueryRowContext(ctx, "SELECT name,technology,strategy FROM strategy_templates WHERE id=?", id).Scan(&name, &technology, &strategy); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	var techJSON, strategyJSON json.RawMessage
	if err := json.Unmarshal([]byte(technology), &techJSON); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(strategy), &strategyJSON); err != nil {
		return err
	}
	content, err := json.Marshal(struct {
		Name       string          `json:"name"`
		Technology json.RawMessage `json:"technology"`
		Strategy   json.RawMessage `json:"strategy"`
	}{Name: name, Technology: techJSON, Strategy: strategyJSON})
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if prior, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(prior, content) {
			return fmt.Errorf("v29 export %s already exists with different content", path)
		}
		fmt.Println("v29 export already matches:", path)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return err
	}
	fmt.Println("v29 exported:", path)
	return nil
}

func snapshotARMDatabase(ctx context.Context, cfg armResearchConfig, database string, captureTables bool) (armSnapshotDB, error) {
	db, err := openArmResearchDB(cfg, database)
	if err != nil {
		return armSnapshotDB{}, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return armSnapshotDB{}, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return armSnapshotDB{}, err
	}
	defer tx.Rollback()
	item := armSnapshotDB{Database: database, VersionTemplates: []armSnapshotTemplate{}}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(MIN(id),0),COALESCE(MAX(id),0) FROM strategy_templates").Scan(&item.TemplateCount, &item.TemplateIDMin, &item.TemplateIDMax); err != nil {
		return armSnapshotDB{}, err
	}
	// The first InnoDB table read establishes the consistent snapshot before its cutoff.
	if err := tx.QueryRowContext(ctx, "SELECT CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3))*1000 AS UNSIGNED)").Scan(&item.AsOfMS); err != nil {
		return armSnapshotDB{}, err
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(MIN(id),0),COALESCE(MAX(id),0) FROM test_strategy_results").Scan(&item.ResultCount, &item.ResultIDMin, &item.ResultIDMax); err != nil {
		return armSnapshotDB{}, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,name,technology,strategy FROM strategy_templates ORDER BY id")
	if err != nil {
		return armSnapshotDB{}, err
	}
	for rows.Next() {
		var id int64
		var name, technology, strategy string
		if err := rows.Scan(&id, &name, &technology, &strategy); err != nil {
			rows.Close()
			return armSnapshotDB{}, err
		}
		if !strings.Contains(strings.ToLower(name), "v29") {
			continue
		}
		techHash, err := semanticJSONHash(technology)
		if err != nil {
			rows.Close()
			return armSnapshotDB{}, fmt.Errorf("template %d technology: %w", id, err)
		}
		strategyHash, err := semanticJSONHash(strategy)
		if err != nil {
			rows.Close()
			return armSnapshotDB{}, fmt.Errorf("template %d strategy: %w", id, err)
		}
		item.VersionTemplates = append(item.VersionTemplates, armSnapshotTemplate{ID: id, Name: name, TechnologyID: techHash, StrategyID: strategyHash})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return armSnapshotDB{}, err
	}
	if err := rows.Close(); err != nil {
		return armSnapshotDB{}, err
	}
	if captureTables {
		templates, templateSchema, err := captureARMTable(ctx, tx, "strategy_templates", item.TemplateCount)
		if err != nil {
			return armSnapshotDB{}, err
		}
		results, resultSchema, err := captureARMTable(ctx, tx, "test_strategy_results", item.ResultCount)
		if err != nil {
			return armSnapshotDB{}, err
		}
		item.Tables = &armTableCapture{AsOfMS: item.AsOfMS, RowCount: item.ResultCount,
			IDMin: item.ResultIDMin, IDMax: item.ResultIDMax, Rows: results, Templates: templates,
			Schemas: map[string][]armTableColumn{"strategy_templates": templateSchema, "test_strategy_results": resultSchema}}
	}
	if err := tx.Commit(); err != nil {
		return armSnapshotDB{}, err
	}
	return item, nil
}

func captureARMTable(ctx context.Context, tx *sql.Tx, table string, expected int64) ([]map[string]any, []armTableColumn, error) {
	if table != "strategy_templates" && table != "test_strategy_results" {
		return nil, nil, fmt.Errorf("table capture target is not whitelisted")
	}
	if expected > 20000 {
		return nil, nil, fmt.Errorf("%s has %d rows, exceeding bounded capture; select a narrower cohort", table, expected)
	}
	rows, err := tx.QueryContext(ctx, "SELECT * FROM `"+table+"` ORDER BY id")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	schema := make([]armTableColumn, len(columns))
	values, pointers := make([]sql.RawBytes, len(columns)), make([]any, len(columns))
	for i, column := range columns {
		schema[i] = armTableColumn{Name: column.Name(), DataType: column.DatabaseTypeName()}
		pointers[i] = &values[i]
	}
	result := make([]map[string]any, 0, int(expected))
	seen := make(map[string]bool)
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			if values[i] == nil {
				row[column.Name()] = nil
			} else {
				row[column.Name()] = string(values[i])
			}
		}
		id, ok := row["id"].(string)
		parsed, idErr := strconv.ParseInt(id, 10, 64)
		if !ok || idErr != nil || parsed <= 0 || seen[id] {
			return nil, nil, fmt.Errorf("%s contains invalid or duplicate id", table)
		}
		seen[id] = true
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if int64(len(result)) != expected {
		return nil, nil, fmt.Errorf("%s captured %d rows, snapshot count %d", table, len(result), expected)
	}
	return result, schema, nil
}

func semanticJSONHash(raw string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}
