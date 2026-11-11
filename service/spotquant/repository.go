package spotquant

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

type SymbolFilters struct {
	Symbol  string
	Quote   string
	Enabled string
	Pin     bool
	Sort    string
	Page    int
	Limit   int
}

type SymbolPage struct {
	Total int64                `json:"total"`
	List  []models.SpotSymbols `json:"list"`
}

func Paginate(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func ListSymbols(o orm.Ormer, f SymbolFilters) (SymbolPage, error) {
	f.Page, f.Limit = Paginate(f.Page, f.Limit)
	query := o.QueryTable(new(models.SpotSymbols))
	if f.Quote != "" && f.Quote != "FAVORITE" {
		if f.Quote != "USDT" && f.Quote != "USDC" && f.Quote != "FDUSD" {
			return SymbolPage{}, fmt.Errorf("unsupported quote asset")
		}
		query = query.Filter("Type", f.Quote)
	}
	if f.Symbol != "" {
		symbol := strings.ToUpper(strings.TrimSpace(f.Symbol))
		if len(symbol) > 32 || strings.ContainsAny(symbol, "%_") {
			return SymbolPage{}, fmt.Errorf("invalid symbol search")
		}
		query = query.Filter("Symbol__icontains", symbol)
	}
	if f.Enabled != "" {
		if f.Enabled != "0" && f.Enabled != "1" {
			return SymbolPage{}, fmt.Errorf("enabled must be 0 or 1")
		}
		query = query.Filter("SpotQuantEnabled", f.Enabled)
	}
	if f.Pin || f.Quote == "FAVORITE" {
		query = query.Filter("Pin", 1)
	}
	total, err := query.Count()
	if err != nil {
		return SymbolPage{}, err
	}
	order := []string{"-Pin", "ID"}
	switch f.Sort {
	case "symbol+":
		order = []string{"Symbol", "ID"}
	case "symbol-":
		order = []string{"-Symbol", "ID"}
	case "percent_change+":
		order = []string{"-PercentChange", "ID"}
	case "percent_change-":
		order = []string{"PercentChange", "ID"}
	case "quoteVolume+":
		order = []string{"-QuoteVolume", "ID"}
	case "quoteVolume-":
		order = []string{"QuoteVolume", "ID"}
	case "", "id": // default
	default:
		return SymbolPage{}, fmt.Errorf("unsupported sort")
	}
	list := make([]models.SpotSymbols, 0)
	if _, err = query.OrderBy(order...).Limit(f.Limit, (f.Page-1)*f.Limit).All(&list); err != nil {
		return SymbolPage{}, err
	}
	return SymbolPage{Total: total, List: list}, nil
}

// GetSymbol only reads the existing Spot ticker, never Futures symbols.
func GetSymbol(o orm.QueryExecutor, id int64) (models.SpotSymbols, error) {
	var row models.SpotSymbols
	if id <= 0 {
		return row, fmt.Errorf("invalid symbol id")
	}
	err := o.QueryTable(new(models.SpotSymbols)).Filter("ID", id).One(&row)
	return row, err
}

func CreateSymbol(o orm.Ormer, name string) (models.SpotSymbols, error) {
	name = strings.ToUpper(strings.TrimSpace(name))
	if len(name) > 32 || len(name) < 5 {
		return models.SpotSymbols{}, fmt.Errorf("invalid spot symbol")
	}
	for _, char := range name {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
			return models.SpotSymbols{}, fmt.Errorf("symbol may only contain letters and digits")
		}
	}
	quote := SpotQuoteAsset(name)
	if quote == "" {
		return models.SpotSymbols{}, fmt.Errorf("symbol requires USDT/USDC/FDUSD quote asset")
	}
	exists := o.QueryTable(new(models.SpotSymbols)).Filter("Symbol", name).Exist()
	if exists {
		return models.SpotSymbols{}, fmt.Errorf("symbol already exists")
	}
	item := models.SpotSymbols{
		Symbol: name, Type: quote, Enable: 0, SpotQuantEnabled: 0,
		Close: "0", Open: "0", High: "0", Low: "0", LastClose: "0",
		Usdt: "10", TickSize: "0.1", StepSize: "0.1", Profit: "100", Loss: "100",
		UpdateTime: time.Now().UnixMilli(),
	}
	id, err := o.Insert(&item)
	item.ID = id
	return item, err
}

// ApplySymbolPatch whitelists only the fields managed by Spot Quant.
// It intentionally never rewrites legacy enable, market prices, or trade
// precision columns which are managed by the existing Spot listener.
func ApplySymbolPatch(o orm.QueryExecutor, id int64, payload []byte) (models.SpotSymbols, error) {
	current, err := GetSymbol(o, id)
	if err != nil {
		return current, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return current, fmt.Errorf("invalid symbol JSON: %w", err)
	}
	if len(fields) == 0 {
		return current, fmt.Errorf("empty symbol patch")
	}
	params := orm.Params{}
	for key, raw := range fields {
		if string(raw) == "null" {
			return current, fmt.Errorf("%s must not be null", key)
		}
		switch key {
		case "spot_quant_enabled":
			var on int
			if err := json.Unmarshal(raw, &on); err != nil || on < 0 || on > 1 {
				return current, fmt.Errorf("spot_quant_enabled must be 0 or 1")
			}
			current.SpotQuantEnabled = int8(on)
			params["SpotQuantEnabled"] = current.SpotQuantEnabled
		case "pin":
			var pin int
			if err := json.Unmarshal(raw, &pin); err != nil || pin < 0 || pin > 1 {
				return current, fmt.Errorf("pin must be 0 or 1")
			}
			current.Pin = int64(pin)
			params["Pin"] = current.Pin
		case "usdt", "profit", "loss":
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				text = strings.TrimSpace(string(raw))
			}
			if err := ValidateBudget(text); err != nil {
				return current, fmt.Errorf("%s: %w", key, err)
			}
			switch key {
			case "usdt":
				current.Usdt = text
				params["Usdt"] = text
			case "profit":
				current.Profit = text
				params["Profit"] = text
			case "loss":
				current.Loss = text
				params["Loss"] = text
			}
		case "technology":
			technology, err := ParseJSONField(raw, true)
			if err != nil {
				return current, err
			}
			current.Technology = technology
			params["Technology"] = technology
		case "strategy":
			strategy, err := ParseJSONField(raw, false)
			if err != nil {
				return current, err
			}
			current.Strategy = strategy
			params["Strategy"] = strategy
		default:
			return current, fmt.Errorf("field %q is not editable by Spot Quant", key)
		}
	}
	if current.SpotQuantEnabled == 1 {
		if err := ValidateActiveStrategy(current.Strategy); err != nil {
			return current, err
		}
		if err := ValidateTechnology(current.Technology); err != nil {
			return current, err
		}
	}
	_, err = o.QueryTable(new(models.SpotSymbols)).Filter("ID", id).Update(params)
	return current, err
}

func UpdateSymbolsEnabled(o orm.Ormer, flag int) (int64, error) {
	if flag != 0 && flag != 1 {
		return 0, fmt.Errorf("enabled must be 0 or 1")
	}
	if flag == 0 {
		return o.QueryTable(new(models.SpotSymbols)).Update(orm.Params{"SpotQuantEnabled": 0})
	}
	// Never activate unconfigured tickers. Only configured entries are
	// switched on. This operation never touches legacy spot_symbols.enable.
	all := make([]models.SpotSymbols, 0)
	_, err := o.QueryTable(new(models.SpotSymbols)).All(&all)
	if err != nil {
		return 0, err
	}
	tx, err := o.Begin()
	if err != nil {
		return 0, err
	}
	var total int64
	for _, row := range all {
		if row.SpotQuantEnabled == 1 || ValidateActiveStrategy(row.Strategy) != nil || ValidateTechnology(row.Technology) != nil {
			continue
		}
		count, err := tx.QueryTable(new(models.SpotSymbols)).Filter("ID", row.ID).Update(orm.Params{"SpotQuantEnabled": 1})
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		total += count
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return total, nil
}

type TemplatePage struct {
	Total int64                         `json:"total"`
	List  []models.SpotStrategyTemplate `json:"list"`
}

func ListTemplates(o orm.Ormer, name string, page, limit int) (TemplatePage, error) {
	page, limit = Paginate(page, limit)
	like := strings.TrimSpace(name)
	if len(like) > 128 || strings.ContainsAny(like, "%_") {
		return TemplatePage{}, fmt.Errorf("invalid template name search")
	}
	where := ""
	args := []interface{}{}
	if like != "" {
		where = " WHERE name LIKE ?"
		args = append(args, "%"+like+"%")
	}
	var total int64
	if err := o.Raw("SELECT COUNT(*) FROM spot_strategy_templates"+where, args...).QueryRow(&total); err != nil {
		return TemplatePage{}, err
	}
	list := make([]models.SpotStrategyTemplate, 0)
	query := "SELECT id,name,technology,strategy,create_time,update_time FROM spot_strategy_templates" + where + " ORDER BY id DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa((page-1)*limit)
	if _, err := o.Raw(query, args...).QueryRows(&list); err != nil {
		return TemplatePage{}, err
	}
	return TemplatePage{Total: total, List: list}, nil
}

func GetTemplate(o orm.Ormer, id int64) (models.SpotStrategyTemplate, error) {
	var rows []models.SpotStrategyTemplate
	if id <= 0 {
		return models.SpotStrategyTemplate{}, fmt.Errorf("invalid template id")
	}
	_, err := o.Raw("SELECT id,name,technology,strategy,create_time,update_time FROM spot_strategy_templates WHERE id=?", id).QueryRows(&rows)
	if err != nil {
		return models.SpotStrategyTemplate{}, err
	}
	if len(rows) == 0 {
		return models.SpotStrategyTemplate{}, fmt.Errorf("spot strategy template not found")
	}
	return rows[0], nil
}

func ParseTemplateFields(payload []byte, existing *models.SpotStrategyTemplate) (models.SpotStrategyTemplate, error) {
	var current models.SpotStrategyTemplate
	if existing != nil {
		current = *existing
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return current, fmt.Errorf("invalid template JSON: %w", err)
	}
	if len(fields) == 0 {
		return current, fmt.Errorf("empty template patch")
	}
	for key, raw := range fields {
		switch key {
		case "name":
			if err := json.Unmarshal(raw, &current.Name); err != nil {
				return current, fmt.Errorf("name must be text")
			}
			current.Name = strings.TrimSpace(current.Name)
		case "technology":
			value, err := ParseJSONField(raw, true)
			if err != nil {
				return current, err
			}
			current.Technology = value
		case "strategy":
			value, err := ParseJSONField(raw, false)
			if err != nil {
				return current, err
			}
			current.Strategy = value
		case "id", "create_time", "update_time":
			if existing == nil {
				return current, fmt.Errorf("import metadata cannot overwrite template identity")
			}
		default:
			return current, fmt.Errorf("unknown template field %q", key)
		}
	}
	if current.Name == "" || len([]rune(current.Name)) > 128 {
		return current, fmt.Errorf("template name must be 1-128 characters")
	}
	if err := ValidateTechnology(current.Technology); err != nil {
		return current, err
	}
	if _, err := ParseRules(current.Strategy); err != nil {
		return current, err
	}
	return current, nil
}

func CreateTemplate(o orm.Ormer, payload []byte) (models.SpotStrategyTemplate, error) {
	item, err := ParseTemplateFields(payload, nil)
	if err != nil {
		return item, err
	}
	now := time.Now().UnixMilli()
	result, err := o.Raw("INSERT INTO spot_strategy_templates(name,technology,strategy,create_time,update_time) VALUES(?,?,?,?,?)", item.Name, item.Technology, item.Strategy, now, now).Exec()
	if err != nil {
		return item, err
	}
	item.ID, err = result.LastInsertId()
	item.CreateTime = now
	item.UpdateTime = now
	return item, err
}

func EditTemplate(o orm.Ormer, id int64, payload []byte) (models.SpotStrategyTemplate, error) {
	current, err := GetTemplate(o, id)
	if err != nil {
		return current, err
	}
	item, err := ParseTemplateFields(payload, &current)
	if err != nil {
		return current, err
	}
	item.UpdateTime = time.Now().UnixMilli()
	_, err = o.Raw("UPDATE spot_strategy_templates SET name=?,technology=?,strategy=?,update_time=? WHERE id=?", item.Name, item.Technology, item.Strategy, item.UpdateTime, id).Exec()
	return item, err
}

func DeleteTemplate(o orm.Ormer, id int64) error {
	if _, err := GetTemplate(o, id); err != nil {
		return err
	}
	_, err := o.Raw("DELETE FROM spot_strategy_templates WHERE id=?", id).Exec()
	return err
}

// ImportTemplate accepts a single JSON object exported from Spot Quant.
// On a same-name match, it updates the existing Spot template rather than
// duplicating it. It never accesses Futures strategy_templates.
func ImportTemplate(o orm.Ormer, payload []byte) (models.SpotStrategyTemplate, string, error) {
	item, err := ParseTemplateFields(payload, nil)
	if err != nil {
		return item, "", err
	}
	if item.Technology == "" || item.Strategy == "" {
		return item, "", fmt.Errorf("import requires technology and strategy")
	}
	// Importing a draft is allowed. A valid LONG entry is required only
	// when a symbol is enabled, not when saving a template for editing.
	var id int64
	err = o.Raw("SELECT id FROM spot_strategy_templates WHERE name=? ORDER BY id DESC LIMIT 1", item.Name).QueryRow(&id)
	if err != nil && err != orm.ErrNoRows {
		return item, "", err
	}
	if err == nil {
		updated, err := EditTemplate(o, id, payload)
		return updated, "updated", err
	}
	added, err := CreateTemplate(o, payload)
	return added, "created", err
}

// BatchSymbolPatch is deliberately scoped to explicit ids. A paginated UI
// cannot silently modify the entire legacy spot_symbols table.
func BatchSymbolPatch(o orm.Ormer, ids []int64, patch []byte) (int, error) {
	if len(ids) == 0 || len(ids) > 500 {
		return 0, fmt.Errorf("select 1-500 Spot symbols for batch editing")
	}
	seen := make(map[int64]struct{}, len(ids))
	tx, err := o.Begin()
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if id <= 0 {
			_ = tx.Rollback()
			return 0, fmt.Errorf("invalid symbol id")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if _, err = ApplySymbolPatch(tx, id, patch); err != nil {
			_ = tx.Rollback()
			return 0, fmt.Errorf("symbol %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(seen), nil
}
