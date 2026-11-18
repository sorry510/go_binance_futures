package coinmquant

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/expr-lang/expr/parser"
	"go_binance_futures/models"
	"go_binance_futures/service/spotquant"
)

// Four CM-only rule directions. Syntax checking never runs the expressions.
type Rule struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Code   string `json:"code"`
	Enable bool   `json:"enable"`
}
type SymbolView struct {
	models.DeliverySymbols
	BudgetUSD string `json:"budget_usd"`
}
type Page[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}
type SymbolFilters struct {
	Symbol, Enabled, Sort string
	Pin                   bool
	Page, Limit           int
}

func pageSize(p, n int) (int, int) {
	if p < 1 {
		p = 1
	}
	if n < 1 {
		n = 20
	}
	if n > 100 {
		n = 100
	}
	return p, n
}
func Budget(o orm.Ormer, id int64) (string, error) {
	var b string
	err := o.Raw("SELECT budget_usd FROM delivery_symbols WHERE id=? AND contract_type='PERPETUAL'", id).QueryRow(&b)
	return b, err
}
func Symbol(o orm.Ormer, id int64) (SymbolView, error) {
	var row models.DeliverySymbols
	if id < 1 {
		return SymbolView{}, fmt.Errorf("invalid symbol id")
	}
	if err := o.QueryTable(new(models.DeliverySymbols)).Filter("ID", id).Filter("ContractType", "PERPETUAL").One(&row); err != nil {
		return SymbolView{}, err
	}
	b, err := Budget(o, id)
	return SymbolView{DeliverySymbols: row, BudgetUSD: b}, err
}
func Symbols(o orm.Ormer, f SymbolFilters) (Page[SymbolView], error) {
	f.Page, f.Limit = pageSize(f.Page, f.Limit)
	q := o.QueryTable(new(models.DeliverySymbols)).Filter("ContractType", "PERPETUAL")
	if f.Symbol != "" {
		v := strings.ToUpper(strings.TrimSpace(f.Symbol))
		if len(v) > 32 || strings.ContainsAny(v, "%") {
			return Page[SymbolView]{}, fmt.Errorf("invalid symbol search")
		}
		if strings.Contains(v, "_") {
			// The _PERP suffix is literal, not a SQL LIKE wildcard.
			q = q.Filter("Symbol", v)
		} else {
			q = q.Filter("Symbol__icontains", v)
		}
	}
	if f.Enabled != "" {
		if f.Enabled != "0" && f.Enabled != "1" {
			return Page[SymbolView]{}, fmt.Errorf("enable must be 0 or 1")
		}
		q = q.Filter("Enable", f.Enabled)
	}
	if f.Pin {
		q = q.Filter("Pin", 1)
	}
	count, err := q.Count()
	if err != nil {
		return Page[SymbolView]{}, err
	}
	order := []string{"-Pin", "Symbol"}
	switch f.Sort {
	case "", "id":
	case "symbol+":
		order = []string{"Symbol"}
	case "symbol-":
		order = []string{"-Symbol"}
	case "percent_change+":
		order = []string{"-PercentChange"}
	case "percent_change-":
		order = []string{"PercentChange"}
	case "quoteVolume+":
		order = []string{"-QuoteVolume"}
	case "quoteVolume-":
		order = []string{"QuoteVolume"}
	default:
		return Page[SymbolView]{}, fmt.Errorf("unsupported sort")
	}
	rows := make([]models.DeliverySymbols, 0)
	if _, err = q.OrderBy(order...).Limit(f.Limit, (f.Page-1)*f.Limit).All(&rows); err != nil {
		return Page[SymbolView]{}, err
	}
	out := make([]SymbolView, 0, len(rows))
	for _, row := range rows {
		b, err := Budget(o, row.ID)
		if err != nil {
			return Page[SymbolView]{}, err
		}
		out = append(out, SymbolView{DeliverySymbols: row, BudgetUSD: b})
	}
	return Page[SymbolView]{Total: count, List: out}, nil
}

var amountPattern = regexp.MustCompile(`^[0-9]{1,18}(?:\.[0-9]{1,18})?$`)

func ValidAmount(s string) error {
	if !amountPattern.MatchString(s) {
		return fmt.Errorf("amount must be decimal with <=18 integer and fractional digits")
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() < 0 {
		return fmt.Errorf("amount must be nonnegative")
	}
	return nil
}
func decimal(raw json.RawMessage) (string, error) {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		var n json.Number
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		dec.UseNumber()
		if err := dec.Decode(&n); err != nil {
			return "", fmt.Errorf("invalid decimal")
		}
		v = n.String()
	}
	v = strings.TrimSpace(v)
	return v, ValidAmount(v)
}
func Rules(raw string) ([]Rule, error) {
	if raw == "" {
		return []Rule{}, nil
	}
	var items []Rule
	if err := json.Unmarshal([]byte(raw), &items); err != nil || items == nil {
		return nil, fmt.Errorf("strategy must be JSON array")
	}
	if len(items) > 100 {
		return nil, fmt.Errorf("too many strategy rules")
	}
	for i, r := range items {
		switch r.Type {
		case "", "long", "short", "close_long", "close_short":
		default:
			return nil, fmt.Errorf("invalid strategy[%d] direction", i)
		}
		if r.Enable && (r.Type == "" || strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.Code) == "") {
			return nil, fmt.Errorf("enabled strategy[%d] requires type, name and code", i)
		}
		if strings.TrimSpace(r.Code) != "" {
			if _, err := parser.Parse(r.Code); err != nil {
				return nil, fmt.Errorf("strategy[%d] syntax: %w", i, err)
			}
		}
	}
	return items, nil
}
func jsonField(raw json.RawMessage, tech bool) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", fmt.Errorf("JSON config cannot be null")
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		s = string(raw)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if tech {
		return s, spotquant.ValidateTechnology(s)
	}
	_, err := Rules(s)
	return s, err
}
func ParseSymbolPatch(current SymbolView, payload []byte) (orm.Params, *string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || len(fields) == 0 {
		return nil, nil, fmt.Errorf("invalid or empty symbol patch")
	}
	updates := orm.Params{}
	var budget *string
	for k, raw := range fields {
		if string(raw) == "null" {
			return nil, nil, fmt.Errorf("%s cannot be null", k)
		}
		switch k {
		case "enable", "pin":
			var n int
			if err := json.Unmarshal(raw, &n); err != nil || (n != 0 && n != 1) {
				return nil, nil, fmt.Errorf("%s must be 0 or 1", k)
			}
			if k == "enable" {
				updates["Enable"] = n
			} else {
				updates["Pin"] = n
			}
		case "budget_usd":
			v, err := decimal(raw)
			if err != nil {
				return nil, nil, err
			}
			budget = &v
		case "profit", "loss":
			v, err := decimal(raw)
			if err != nil {
				return nil, nil, err
			}
			if k == "profit" {
				updates["Profit"] = v
			} else {
				updates["Loss"] = v
			}
		case "leverage":
			var n int64
			if err := json.Unmarshal(raw, &n); err != nil || n < 1 || n > 125 {
				return nil, nil, fmt.Errorf("leverage must be integer 1-125")
			}
			updates["Leverage"] = n
		case "marginType":
			var v string
			if err := json.Unmarshal(raw, &v); err != nil || (v != "CROSSED" && v != "ISOLATED") {
				return nil, nil, fmt.Errorf("marginType must be CROSSED or ISOLATED")
			}
			updates["MarginType"] = v
		case "strategy", "technology":
			v, err := jsonField(raw, k == "technology")
			if err != nil {
				return nil, nil, err
			}
			if k == "strategy" {
				updates["Strategy"] = v
			} else {
				updates["Technology"] = v
			}
		default:
			return nil, nil, fmt.Errorf("%s cannot be edited in COIN-M settings", k)
		}
	}
	if n, ok := updates["Enable"]; ok && n.(int) == 1 {
		b := current.BudgetUSD
		if budget != nil {
			b = *budget
		}
		r, _ := new(big.Rat).SetString(b)
		if r == nil || r.Sign() <= 0 {
			return nil, nil, fmt.Errorf("set positive budget_usd before enabling")
		}
		strategy := current.Strategy
		if v, ok := updates["Strategy"]; ok {
			strategy = v.(string)
		}
		rules, err := Rules(strategy)
		if err != nil {
			return nil, nil, err
		}
		entry := false
		for _, rule := range rules {
			if rule.Enable && (rule.Type == "long" || rule.Type == "short") {
				entry = true
				break
			}
		}
		if !entry {
			return nil, nil, fmt.Errorf("configure an enabled LONG or SHORT entry strategy")
		}
		if current.ContractSize <= 0 || current.MarginAsset == "" {
			return nil, nil, fmt.Errorf("contract metadata unavailable")
		}
	}
	return updates, budget, nil
}
func PatchSymbol(o orm.Ormer, id int64, payload []byte) (SymbolView, error) {
	current, err := Symbol(o, id)
	if err != nil {
		return SymbolView{}, err
	}
	params, budget, err := ParseSymbolPatch(current, payload)
	if err != nil {
		return SymbolView{}, err
	}
	tx, err := o.Begin()
	if err != nil {
		return SymbolView{}, err
	}
	defer tx.Rollback()
	if len(params) > 0 {
		if _, err = tx.QueryTable(new(models.DeliverySymbols)).Filter("ID", id).Filter("ContractType", "PERPETUAL").Update(params); err != nil {
			return SymbolView{}, err
		}
	}
	if budget != nil {
		if _, err = tx.Raw("UPDATE delivery_symbols SET budget_usd=? WHERE id=? AND contract_type='PERPETUAL'", *budget, id).Exec(); err != nil {
			return SymbolView{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return SymbolView{}, err
	}
	return Symbol(o, id)
}
func EnableAll(o orm.Ormer, on int) (int64, error) {
	if on != 0 && on != 1 {
		return 0, fmt.Errorf("invalid enable flag")
	}
	if on == 0 {
		return o.QueryTable(new(models.DeliverySymbols)).Filter("ContractType", "PERPETUAL").Update(orm.Params{"Enable": 0})
	}
	rows := make([]models.DeliverySymbols, 0)
	if _, err := o.QueryTable(new(models.DeliverySymbols)).Filter("ContractType", "PERPETUAL").All(&rows); err != nil {
		return 0, err
	}
	var count int64
	for _, row := range rows {
		if row.Enable == 1 {
			continue
		}
		view, err := Symbol(o, row.ID)
		if err != nil {
			return count, err
		}
		if _, _, err := ParseSymbolPatch(view, []byte(`{"enable":1}`)); err != nil {
			continue
		}
		if _, err := PatchSymbol(o, row.ID, []byte(`{"enable":1}`)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
func BatchSymbols(o orm.Ormer, ids []int64, payload []byte) (int, error) {
	if len(ids) == 0 || len(ids) > 500 {
		return 0, fmt.Errorf("select 1-500 symbols")
	}
	type patch struct {
		id     int64
		fields orm.Params
		budget *string
	}
	changes := make([]patch, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id < 1 || seen[id] {
			return 0, fmt.Errorf("invalid or duplicate symbol id")
		}
		seen[id] = true
		row, err := Symbol(o, id)
		if err != nil {
			return 0, err
		}
		fields, budget, err := ParseSymbolPatch(row, payload)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", row.Symbol, err)
		}
		changes = append(changes, patch{id: id, fields: fields, budget: budget})
	}
	// One local transaction: errors must not leave half the selected symbols
	// with a different budget, leverage or strategy.
	tx, err := o.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, c := range changes {
		if len(c.fields) > 0 {
			if _, err := tx.QueryTable(new(models.DeliverySymbols)).
				Filter("ID", c.id).Filter("ContractType", "PERPETUAL").
				Update(c.fields); err != nil {
				return 0, err
			}
		}
		if c.budget != nil {
			if _, err := tx.Raw("UPDATE delivery_symbols SET budget_usd=? WHERE id=? AND contract_type='PERPETUAL'", *c.budget, c.id).Exec(); err != nil {
				return 0, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(changes), nil
}

type TemplatePage = Page[models.CoinMStrategyTemplate]

func Templates(o orm.Ormer, name string, page, limit int) (TemplatePage, error) {
	page, limit = pageSize(page, limit)
	name = strings.TrimSpace(name)
	if len(name) > 128 || strings.ContainsAny(name, "%_") {
		return TemplatePage{}, fmt.Errorf("invalid name filter")
	}
	where := ""
	args := []interface{}{}
	if name != "" {
		where = " WHERE name LIKE ?"
		args = append(args, "%"+name+"%")
	}
	var total int64
	if err := o.Raw("SELECT COUNT(*) FROM coin_m_strategy_templates"+where, args...).QueryRow(&total); err != nil {
		return TemplatePage{}, err
	}
	rows := make([]models.CoinMStrategyTemplate, 0)
	sql := "SELECT id,name,technology,strategy,create_time,update_time FROM coin_m_strategy_templates" + where + " ORDER BY id DESC LIMIT " + strconv.Itoa(limit) + " OFFSET " + strconv.Itoa((page-1)*limit)
	if _, err := o.Raw(sql, args...).QueryRows(&rows); err != nil {
		return TemplatePage{}, err
	}
	return TemplatePage{Total: total, List: rows}, nil
}
func Template(o orm.Ormer, id int64) (models.CoinMStrategyTemplate, error) {
	var rows []models.CoinMStrategyTemplate
	if id < 1 {
		return models.CoinMStrategyTemplate{}, fmt.Errorf("invalid id")
	}
	_, err := o.Raw("SELECT id,name,technology,strategy,create_time,update_time FROM coin_m_strategy_templates WHERE id=?", id).QueryRows(&rows)
	if err != nil {
		return models.CoinMStrategyTemplate{}, err
	}
	if len(rows) == 0 {
		return models.CoinMStrategyTemplate{}, fmt.Errorf("COIN-M template not found")
	}
	return rows[0], nil
}
func patchTemplate(payload []byte, item models.CoinMStrategyTemplate) (models.CoinMStrategyTemplate, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil || len(fields) == 0 {
		return item, fmt.Errorf("invalid template JSON")
	}
	for k, v := range fields {
		switch k {
		case "name":
			if err := json.Unmarshal(v, &item.Name); err != nil {
				return item, fmt.Errorf("template name must be text")
			}
			item.Name = strings.TrimSpace(item.Name)
		case "technology", "strategy":
			s, err := jsonField(v, k == "technology")
			if err != nil {
				return item, err
			}
			if k == "technology" {
				item.Technology = s
			} else {
				item.Strategy = s
			}
		default:
			return item, fmt.Errorf("unknown or immutable template field %s", k)
		}
	}
	if item.Name == "" || len([]rune(item.Name)) > 128 {
		return item, fmt.Errorf("template name must be 1-128 characters")
	}
	if err := spotquant.ValidateTechnology(item.Technology); err != nil {
		return item, err
	}
	if _, err := Rules(item.Strategy); err != nil {
		return item, err
	}
	return item, nil
}
func CreateTemplate(o orm.Ormer, payload []byte) (models.CoinMStrategyTemplate, error) {
	item, err := patchTemplate(payload, models.CoinMStrategyTemplate{})
	if err != nil {
		return item, err
	}
	now := time.Now().UnixMilli()
	res, err := o.Raw("INSERT INTO coin_m_strategy_templates(name,technology,strategy,create_time,update_time) VALUES(?,?,?,?,?)", item.Name, item.Technology, item.Strategy, now, now).Exec()
	if err != nil {
		return item, err
	}
	item.ID, err = res.LastInsertId()
	item.CreateTime = now
	item.UpdateTime = now
	return item, err
}
func EditTemplate(o orm.Ormer, id int64, payload []byte) (models.CoinMStrategyTemplate, error) {
	current, err := Template(o, id)
	if err != nil {
		return current, err
	}
	item, err := patchTemplate(payload, current)
	if err != nil {
		return current, err
	}
	item.UpdateTime = time.Now().UnixMilli()
	_, err = o.Raw("UPDATE coin_m_strategy_templates SET name=?,technology=?,strategy=?,update_time=? WHERE id=?", item.Name, item.Technology, item.Strategy, item.UpdateTime, id).Exec()
	return item, err
}
func DeleteTemplate(o orm.Ormer, id int64) error {
	if _, err := Template(o, id); err != nil {
		return err
	}
	_, err := o.Raw("DELETE FROM coin_m_strategy_templates WHERE id=?", id).Exec()
	return err
}
func ImportTemplate(o orm.Ormer, payload []byte) (models.CoinMStrategyTemplate, string, error) {
	item, err := patchTemplate(payload, models.CoinMStrategyTemplate{})
	if err != nil {
		return item, "", err
	}
	if item.Technology == "" || item.Strategy == "" {
		return item, "", fmt.Errorf("import requires technology and strategy")
	}
	var id int64
	err = o.Raw("SELECT id FROM coin_m_strategy_templates WHERE name=? ORDER BY id DESC LIMIT 1", item.Name).QueryRow(&id)
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
