package orderhistory

// Read-only history projection. Only Managed cycles / matching exchange
// executions may appear here; the legacy order table is not queried.
import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"
)

var ErrInvalidQuery = errors.New("invalid order history filters")
var ErrNotFound = errors.New("managed position cycle not found")
var ownerPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
var symbolPattern = regexp.MustCompile(`^[A-Z0-9]{1,32}$`)

type Filter struct {
	AccountID, Owner, Symbol, PositionSide, Status, Settlement string
	StartTime, EndTime                                         int64
	Page, Limit                                                int
}

func (q Filter) validated() (Filter, error) {
	q.AccountID = strings.ToLower(strings.TrimSpace(q.AccountID))
	if q.AccountID == "" {
		q.AccountID = "main"
	}
	if q.AccountID != "main" && q.AccountID != "lead" {
		return q, fmt.Errorf("%w: account", ErrInvalidQuery)
	}
	q.Owner = strings.ToLower(strings.TrimSpace(q.Owner))
	if q.Owner == "all" {
		q.Owner = ""
	}
	if q.Owner != "" && !ownerPattern.MatchString(q.Owner) {
		return q, fmt.Errorf("%w: owner", ErrInvalidQuery)
	}
	q.Symbol = strings.ToUpper(strings.TrimSpace(q.Symbol))
	if q.Symbol != "" && !symbolPattern.MatchString(q.Symbol) {
		return q, fmt.Errorf("%w: symbol", ErrInvalidQuery)
	}
	q.PositionSide = strings.ToUpper(strings.TrimSpace(q.PositionSide))
	if q.PositionSide == "ALL" {
		q.PositionSide = ""
	}
	if q.PositionSide != "" && q.PositionSide != "LONG" && q.PositionSide != "SHORT" {
		return q, fmt.Errorf("%w: position_side", ErrInvalidQuery)
	}
	q.Status = strings.ToLower(strings.TrimSpace(q.Status))
	if q.Status == "all" {
		q.Status = ""
	}
	if q.Status != "" && q.Status != "active" && q.Status != "suspended" && q.Status != "closed" && q.Status != "reconcile_required" {
		return q, fmt.Errorf("%w: status", ErrInvalidQuery)
	}
	q.Settlement = strings.ToLower(strings.TrimSpace(q.Settlement))
	if q.Settlement == "all" {
		q.Settlement = ""
	}
	if q.Settlement != "" && q.Settlement != "legacy" && q.Settlement != "open" && q.Settlement != "incomplete" && q.Settlement != "settled" {
		return q, fmt.Errorf("%w: settlement", ErrInvalidQuery)
	}
	if q.StartTime < 0 || q.EndTime < 0 || (q.EndTime > 0 && q.StartTime > q.EndTime) {
		return q, fmt.Errorf("%w: time range", ErrInvalidQuery)
	}
	if q.Page < 1 || q.Page > 1000000 || q.Limit < 1 || q.Limit > 100 {
		return q, fmt.Errorf("%w: pagination", ErrInvalidQuery)
	}
	return q, nil
}

type AssetFee struct {
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}
type Position struct {
	PositionID        int64      `json:"position_id"`
	AccountID         string     `json:"account_id"`
	Owner             string     `json:"owner"`
	Symbol            string     `json:"symbol"`
	PositionSide      string     `json:"position_side"`
	Status            string     `json:"status"`
	SettlementStatus  string     `json:"settlement_status"`
	Completeness      string     `json:"completeness"`
	IncompleteReason  string     `json:"incomplete_reason,omitempty"`
	OpenQty           string     `json:"open_qty"`
	CloseQty          string     `json:"close_qty"`
	RemainingQty      string     `json:"remaining_qty"`
	OpenAvgPrice      *string    `json:"open_avg_price"`
	CloseAvgPrice     *string    `json:"close_avg_price"`
	GrossPnL          *string    `json:"gross_pnl"`
	NetPnL            *string    `json:"net_pnl"`
	ReturnPct         *string    `json:"return_pct"`
	PnLAsset          string     `json:"pnl_asset,omitempty"`
	Fees              []AssetFee `json:"fees"`
	Leverage          *int       `json:"leverage"`
	CreatedAt         int64      `json:"created_at"`
	UpdatedAt         int64      `json:"updated_at"`
	ClosedAt          int64      `json:"closed_at"`
	DurationMillis    int64      `json:"duration_millis"`
	OrderCount        int        `json:"order_count"`
	FillCount         int        `json:"fill_count"`
	RESTVerifiedFills int        `json:"rest_verified_fills"`
}

// ManagedOrderView is an intentional public response allowlist. SourceRef
// and future ORM-only metadata must not be serialized automatically.
type ManagedOrderView struct {
	ID              int64   `json:"id"`
	PositionID      *int64  `json:"position_id,omitempty"`
	AccountID       string  `json:"account_id"`
	Owner           string  `json:"owner"`
	Symbol          string  `json:"symbol"`
	PositionSide    string  `json:"position_side"`
	Intent          string  `json:"intent"`
	ClientOrderID   string  `json:"client_order_id"`
	ExchangeOrderID string  `json:"exchange_order_id,omitempty"`
	ActualOrderID   *string `json:"actual_order_id,omitempty"`
	RequestedQty    float64 `json:"requested_qty"`
	FilledQty       float64 `json:"filled_qty"`
	OrderType       string  `json:"order_type"`
	Status          string  `json:"status"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
}

func managedOrderView(row models.FuturesManagedOrder) ManagedOrderView {
	return ManagedOrderView{
		ID: row.ID, PositionID: row.PositionID, AccountID: row.AccountID,
		Owner: row.Owner, Symbol: row.Symbol, PositionSide: row.PositionSide,
		Intent: row.Intent, ClientOrderID: row.ClientOrderID,
		ExchangeOrderID: row.ExchangeOrderID, ActualOrderID: row.ActualOrderID,
		RequestedQty: row.RequestedQty, FilledQty: row.FilledQty,
		OrderType: row.OrderType, Status: row.Status,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

type Order struct {
	Managed            ManagedOrderView          `json:"managed"`
	FilledQtyFromFills string                    `json:"filled_qty_from_fills"`
	AvgPrice           *string                   `json:"avg_price"`
	Completeness       string                    `json:"completeness"`
	Fills              []models.FuturesTradeFill `json:"fills"`
	Fees               []AssetFee                `json:"fees"`
	GrossPnL           *string                   `json:"gross_pnl"`
	PnLAsset           string                    `json:"pnl_asset,omitempty"`
}
type Detail struct {
	Position Position `json:"position"`
	Orders   []Order  `json:"orders"`
}
type Page struct {
	Total           int64             `json:"total"`
	Page            int               `json:"page"`
	Limit           int               `json:"limit"`
	List            []Position        `json:"list"`
	PageVerifiedNet map[string]string `json:"page_verified_net"`
}

type Store struct {
	DB  orm.Ormer
	Now func() time.Time
	// ActivatedAt injects a durable, explicitly approved UTC Unix-ms T0
	// during tests. Production reads account-scoped process configuration.
	ActivatedAt func(account string) (int64, error)
}

// ConfiguredHistoryT0 is deliberately fail-closed. T0 is never inferred
// from process start or from the creation date of an older Managed cycle.
// This only guards read-only history certification: trading/recovery/settlement
// keep operating independently for pre-activation positions.
func ConfiguredHistoryT0(account string) (int64, error) {
	var name string
	switch account {
	case "main":
		name = "BINANCE_ORDER_HISTORY_T0_MAIN_MS"
	case "lead":
		name = "BINANCE_ORDER_HISTORY_T0_LEAD_MS"
	default:
		return 0, fmt.Errorf("invalid history activation account")
	}
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, nil
	} // no approved T0: display audit rows, never certified PnL
	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || ms < 946684800000 || ms > 4102444800000 {
		return 0, fmt.Errorf("invalid UTC Unix-ms activation for %s", account)
	}
	return ms, nil
}
func (s Store) activatedAt(account string) (int64, error) {
	if s.ActivatedAt != nil {
		return s.ActivatedAt(account)
	}
	return ConfiguredHistoryT0(account)
}

func (s Store) db() orm.Ormer {
	if s.DB != nil {
		return s.DB
	}
	return orm.NewOrm()
}
func (s Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func dec(v string) (*big.Rat, error) {
	r, ok := new(big.Rat).SetString(strings.TrimSpace(v))
	if !ok {
		return nil, fmt.Errorf("invalid exchange decimal")
	}
	return r, nil
}
func val(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := r.FloatString(18)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}
func ptr(r *big.Rat) *string {
	if r == nil {
		return nil
	}
	v := val(r)
	return &v
}
func matches(actual *big.Rat, qty float64) bool {
	if actual == nil || math.IsNaN(qty) || math.IsInf(qty, 0) || qty < 0 {
		return false
	}
	expect, ok := new(big.Rat).SetString(strconv.FormatFloat(qty, 'f', 12, 64))
	if !ok {
		return false
	}
	delta := new(big.Rat).Abs(new(big.Rat).Sub(actual, expect))
	return delta.Cmp(big.NewRat(1, 1000000000)) <= 0
}

type fillRow struct {
	Market          string `orm:"column(market)"`
	AccountID       string `orm:"column(account_id)"`
	Symbol          string `orm:"column(symbol)"`
	OrderID         string `orm:"column(order_id)"`
	TradeID         string `orm:"column(trade_id)"`
	PositionID      *int64 `orm:"column(position_id)"`
	ClientOrderID   string `orm:"column(client_order_id)"`
	Intent          string `orm:"column(intent)"`
	PositionSide    string `orm:"column(position_side)"`
	Qty             string `orm:"column(qty)"`
	Price           string `orm:"column(price)"`
	Commission      string `orm:"column(commission)"`
	CommissionAsset string `orm:"column(commission_asset)"`
	RealizedPnL     string `orm:"column(realized_pnl)"`
	PnLAsset        string `orm:"column(pnl_asset)"`
	TradeTime       int64  `orm:"column(trade_time)"`
	RESTVerified    int    `orm:"column(rest_verified)"`
}

func (f fillRow) model() models.FuturesTradeFill {
	return models.FuturesTradeFill{
		Market: f.Market, AccountID: f.AccountID, Symbol: f.Symbol, OrderID: f.OrderID, TradeID: f.TradeID,
		PositionID: f.PositionID, ClientOrderID: f.ClientOrderID, Intent: f.Intent, PositionSide: f.PositionSide,
		Qty: f.Qty, Price: f.Price, Commission: f.Commission, CommissionAsset: f.CommissionAsset,
		RealizedPnL: f.RealizedPnL, PnLAsset: f.PnLAsset, TradeTime: f.TradeTime, RESTVerified: f.RESTVerified == 1,
	}
}
func realExchangeID(o models.FuturesManagedOrder) string {
	if o.ActualOrderID != nil && strings.TrimSpace(*o.ActualOrderID) != "" {
		return strings.TrimSpace(*o.ActualOrderID)
	}
	switch strings.ToUpper(o.OrderType) {
	case "STOP", "STOP_MARKET", "TAKE_PROFIT", "TAKE_PROFIT_MARKET", "TRAILING_STOP_MARKET":
		return ""
	}
	return o.ExchangeOrderID
}
func closingIntent(v string) bool { return v == "close" || v == "take_profit" || v == "stop_loss" }

func (s Store) List(ctx context.Context, q Filter) (Page, error) {
	q, err := q.validated()
	if err != nil {
		return Page{}, err
	}
	qs := s.db().QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", q.AccountID)
	if q.Owner != "" {
		qs = qs.Filter("owner", q.Owner)
	}
	if q.Symbol != "" {
		qs = qs.Filter("symbol__icontains", q.Symbol)
	}
	if q.PositionSide != "" {
		qs = qs.Filter("position_side", q.PositionSide)
	}
	if q.Status != "" {
		qs = qs.Filter("status", q.Status)
	}
	if q.Settlement != "" {
		qs = qs.Filter("settlement_status", q.Settlement)
	}
	if q.StartTime > 0 {
		qs = qs.Filter("created_at__gte", q.StartTime)
	}
	if q.EndTime > 0 {
		qs = qs.Filter("created_at__lte", q.EndTime)
	}
	total, err := qs.Count()
	if err != nil {
		return Page{}, err
	}
	page := Page{Total: total, Page: q.Page, Limit: q.Limit, List: []Position{}, PageVerifiedNet: map[string]string{}}
	if total == 0 {
		return page, nil
	}
	var positions []models.FuturesManagedPosition
	_, err = qs.OrderBy("-created_at", "-id").Limit(q.Limit, (q.Page-1)*q.Limit).All(&positions)
	if err != nil {
		return Page{}, err
	}
	if len(positions) == 0 {
		return page, nil
	}
	page.List, _, err = s.project(ctx, positions, false)
	if err != nil {
		return Page{}, err
	}
	sums := make(map[string]*big.Rat)
	for _, p := range page.List {
		if p.Completeness != "verified" || p.NetPnL == nil {
			continue
		}
		net, e := dec(*p.NetPnL)
		if e != nil {
			return Page{}, e
		}
		if sums[p.PnLAsset] == nil {
			sums[p.PnLAsset] = new(big.Rat)
		}
		sums[p.PnLAsset].Add(sums[p.PnLAsset], net)
	}
	for asset, net := range sums {
		page.PageVerifiedNet[asset] = val(net)
	}
	return page, nil
}
func (s Store) Details(ctx context.Context, account string, id int64) (Detail, error) {
	if id <= 0 {
		return Detail{}, ErrInvalidQuery
	}
	q, err := (Filter{AccountID: account, Page: 1, Limit: 1}).validated()
	if err != nil {
		return Detail{}, err
	}
	var pos models.FuturesManagedPosition
	err = s.db().QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", q.AccountID).Filter("id", id).One(&pos)
	if err == orm.ErrNoRows {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	positions, details, err := s.project(ctx, []models.FuturesManagedPosition{pos}, true)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Position: positions[0], Orders: details[id]}, nil
}
func (s Store) project(ctx context.Context, positions []models.FuturesManagedPosition, includeFills bool) ([]Position, map[int64][]Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if len(positions) == 0 {
		return []Position{}, map[int64][]Order{}, nil
	}
	account := positions[0].AccountID
	t0, err := s.activatedAt(account)
	if err != nil {
		return nil, nil, err
	}
	if t0 < 0 {
		return nil, nil, fmt.Errorf("negative history activation")
	}
	ids := make([]int64, 0, len(positions))
	for _, p := range positions {
		if p.AccountID != account {
			return nil, nil, fmt.Errorf("mixed account history")
		}
		ids = append(ids, p.ID)
	}
	var orders []models.FuturesManagedOrder
	_, err = s.db().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", account).
		Filter("position_id__in", ids).OrderBy("created_at", "id").All(&orders)
	if err != nil {
		return nil, nil, err
	}
	slots := make([]string, len(ids))
	args := []interface{}{tradefills.MarketUSDM, account}
	for i, id := range ids {
		slots[i] = "?"
		args = append(args, id)
	}
	sql := `SELECT market,account_id,symbol,order_id,trade_id,position_id,client_order_id,intent,
  position_side,qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified
  FROM futures_trade_fills WHERE market=? AND account_id=? AND position_id IN (` + strings.Join(slots, ",") + `)
  ORDER BY trade_time,order_id,trade_id`
	var fills []fillRow
	_, err = s.db().Raw(sql, args...).QueryRows(&fills)
	if err != nil {
		return nil, nil, err
	}
	ordersByID := map[int64][]models.FuturesManagedOrder{}
	fillsByID := map[int64][]fillRow{}
	for _, o := range orders {
		if o.PositionID != nil {
			ordersByID[*o.PositionID] = append(ordersByID[*o.PositionID], o)
		}
	}
	for _, f := range fills {
		if f.PositionID != nil {
			fillsByID[*f.PositionID] = append(fillsByID[*f.PositionID], f)
		}
	}
	result := make([]Position, 0, len(positions))
	details := map[int64][]Order{}
	now := s.now().UnixMilli()
	for _, p := range positions {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		summary, children := summarize(p, ordersByID[p.ID], fillsByID[p.ID], includeFills, now, t0)
		result = append(result, summary)
		details[p.ID] = children
	}
	return result, details, nil
}

func summarize(p models.FuturesManagedPosition, orders []models.FuturesManagedOrder, fills []fillRow, includeFills bool, now, t0 int64) (Position, []Order) {
	v := Position{PositionID: p.ID, AccountID: p.AccountID, Owner: p.Owner, Symbol: p.Symbol, PositionSide: p.PositionSide,
		Status: p.Status, SettlementStatus: p.SettlementStatus, Completeness: "pending",
		OpenQty: "0", CloseQty: "0", RemainingQty: strings.TrimRight(strings.TrimRight(strconv.FormatFloat(p.ManagedQty, 'f', 12, 64), "0"), "."),
		Fees: []AssetFee{}, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, ClosedAt: p.ClosedAt,
		OrderCount: len(orders), FillCount: len(fills)}
	end := now
	if p.ClosedAt > 0 {
		end = p.ClosedAt
	}
	if end >= p.CreatedAt {
		v.DurationMillis = end - p.CreatedAt
	}
	problem := ""
	mark := func(reason string) {
		if problem == "" {
			problem = reason
		}
	}
	if t0 == 0 {
		mark("history_not_activated")
	} else if p.CreatedAt < t0 {
		mark("pre_activation_position_cycle")
	}
	if p.SettlementStatus == "legacy" {
		mark("legacy_cycle_without_complete_fills")
	}
	if len(orders) == 0 {
		mark("no_linked_managed_orders")
	}
	byClient := map[string]models.FuturesManagedOrder{}
	matchOwners := map[string]string{}
	for _, o := range orders {
		if o.PositionID == nil || *o.PositionID != p.ID || o.AccountID != p.AccountID || o.Owner != p.Owner ||
			o.Symbol != p.Symbol || o.PositionSide != p.PositionSide {
			mark("managed_order_cycle_identity_mismatch")
		}
		if _, exists := byClient[o.ClientOrderID]; exists {
			mark("duplicate_client_order")
		}
		byClient[o.ClientOrderID] = o
		if o.FilledQty > 0 {
			actual := realExchangeID(o)
			if actual == "" {
				mark("filled_order_missing_real_exchange_id")
			} else if first, exists := matchOwners[actual]; exists && first != o.ClientOrderID {
				mark("duplicate_exchange_matching_order")
			} else {
				matchOwners[actual] = o.ClientOrderID
			}
		}
		if p.Status == "closed" && (o.Status == "pending" || o.Status == "submitted" || o.Status == "partially_filled" || o.Status == "reconcile_required") {
			mark("outstanding_managed_order")
		}
	}
	grouped := map[string][]fillRow{}
	opened, closed := new(big.Rat), new(big.Rat)
	openValue, closeValue := new(big.Rat), new(big.Rat)
	gross, feeTotal := new(big.Rat), new(big.Rat)
	fees := map[string]*big.Rat{}
	pnlAsset := ""
	for _, f := range fills {
		o, ok := byClient[f.ClientOrderID]
		if !ok || f.PositionID == nil || *f.PositionID != p.ID || f.Market != tradefills.MarketUSDM ||
			f.AccountID != p.AccountID || f.Symbol != p.Symbol || f.PositionSide != p.PositionSide ||
			f.Intent != o.Intent || f.OrderID != realExchangeID(o) {
			mark("fill_order_cycle_identity_mismatch")
			continue
		}
		grouped[f.ClientOrderID] = append(grouped[f.ClientOrderID], f)
		qty, e1 := dec(f.Qty)
		price, e2 := dec(f.Price)
		fee, e3 := dec(f.Commission)
		pnl, e4 := dec(f.RealizedPnL)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || qty.Sign() <= 0 || price.Sign() <= 0 {
			mark("invalid_exchange_fill_decimal")
			continue
		}
		notional := new(big.Rat).Mul(qty, price)
		if o.Intent == "open" {
			opened.Add(opened, qty)
			openValue.Add(openValue, notional)
		} else if closingIntent(o.Intent) {
			closed.Add(closed, qty)
			closeValue.Add(closeValue, notional)
		} else {
			mark("unsupported_order_intent")
		}
		if f.RESTVerified != 1 {
			mark("fee_and_pnl_not_rest_verified")
		} else {
			v.RESTVerifiedFills++
		}
		if f.PnLAsset == "" {
			if p.SettlementStatus == "settled" {
				mark("settled_fill_missing_pnl_asset")
			}
		} else if pnlAsset == "" {
			pnlAsset = f.PnLAsset
		} else if pnlAsset != f.PnLAsset {
			mark("mixed_pnl_assets")
		}
		if f.CommissionAsset == "" && fee.Sign() != 0 {
			mark("nonzero_fee_missing_asset")
		}
		if f.CommissionAsset != "" {
			if fees[f.CommissionAsset] == nil {
				fees[f.CommissionAsset] = new(big.Rat)
			}
			fees[f.CommissionAsset].Add(fees[f.CommissionAsset], fee)
		}
		gross.Add(gross, pnl)
		feeTotal.Add(feeTotal, fee)
	}
	v.OpenQty = val(opened)
	v.CloseQty = val(closed)
	if opened.Sign() > 0 {
		v.OpenAvgPrice = ptr(new(big.Rat).Quo(openValue, opened))
	}
	if closed.Sign() > 0 {
		v.CloseAvgPrice = ptr(new(big.Rat).Quo(closeValue, closed))
	}
	child := make([]Order, 0, len(orders))
	for _, o := range orders {
		rows := grouped[o.ClientOrderID]
		qty := new(big.Rat)
		money := new(big.Rat)
		entry := Order{Managed: managedOrderView(o), FilledQtyFromFills: "0", Completeness: "pending",
			Fills: []models.FuturesTradeFill{}, Fees: []AssetFee{}}
		verified := true
		orderFees := map[string]*big.Rat{}
		orderGross := new(big.Rat)
		orderPnLAsset := ""
		for _, f := range rows {
			n, _ := dec(f.Qty)
			price, _ := dec(f.Price)
			if n == nil || price == nil {
				verified = false
				continue
			}
			qty.Add(qty, n)
			money.Add(money, new(big.Rat).Mul(n, price))
			if f.RESTVerified != 1 {
				verified = false
			}
			if paid, e := dec(f.Commission); e == nil {
				if f.CommissionAsset != "" {
					if orderFees[f.CommissionAsset] == nil {
						orderFees[f.CommissionAsset] = new(big.Rat)
					}
					orderFees[f.CommissionAsset].Add(orderFees[f.CommissionAsset], paid)
				} else if paid.Sign() != 0 {
					verified = false
				}
			} else {
				verified = false
			}
			if realized, e := dec(f.RealizedPnL); e == nil {
				orderGross.Add(orderGross, realized)
			} else {
				verified = false
			}
			if f.PnLAsset != "" {
				if orderPnLAsset == "" {
					orderPnLAsset = f.PnLAsset
				} else if orderPnLAsset != f.PnLAsset {
					verified = false
				}
			}
			if includeFills {
				entry.Fills = append(entry.Fills, f.model())
			}
		}
		entry.FilledQtyFromFills = val(qty)
		if qty.Sign() > 0 {
			entry.AvgPrice = ptr(new(big.Rat).Quo(money, qty))
		}
		if !matches(qty, o.FilledQty) {
			entry.Completeness = "incomplete"
			mark("order_fill_quantity_mismatch")
		} else if verified && qty.Sign() > 0 {
			entry.Completeness = "verified"
			// Realized PnL is denominated only after the cycle's
			// exchange margin asset has been verified by settlement.
			// Do not show a currency-less gross PnL for active cycles.
			if p.SettlementStatus == "settled" &&
				(orderPnLAsset == "USDT" || orderPnLAsset == "USDC") {
				entry.GrossPnL = ptr(orderGross)
				entry.PnLAsset = orderPnLAsset
			}
			for asset, amount := range orderFees {
				entry.Fees = append(entry.Fees, AssetFee{Asset: asset, Amount: val(amount)})
			}
			sort.Slice(entry.Fees, func(i, j int) bool { return entry.Fees[i].Asset < entry.Fees[j].Asset })
		}
		if t0 == 0 || p.CreatedAt < t0 {
			// Pre-cutover or unactivated history may be inspected, but it is
			// not certified as new exchange ledger history. Keep individual
			// Fill facts visible with their REST source flags.
			entry.Completeness = "incomplete"
			entry.GrossPnL = nil
			entry.PnLAsset = ""
			entry.Fees = []AssetFee{}
		}
		child = append(child, entry)
	}
	if p.Status == "closed" && !matches(new(big.Rat).Sub(opened, closed), 0) {
		mark("opening_and_closing_fills_unbalanced")
	}
	if p.SettlementStatus == "settled" {
		if p.Status != "closed" || p.SettlementProcessedAt <= 0 {
			mark("settlement_status_conflict")
		}
		if opened.Sign() <= 0 || closed.Sign() <= 0 {
			mark("settled_cycle_missing_executions")
		}
		if pnlAsset != "USDT" && pnlAsset != "USDC" {
			mark("unknown_settlement_asset")
		}
		net := new(big.Rat).Sub(gross, feeTotal)
		if net.Sign() != p.SettlementPNLSign {
			mark("settled_pnl_sign_mismatch")
		}
		for asset := range fees {
			// Match verifiedSettlementFacts exactly: even a nonempty,
			// zero-valued different fee asset is not assumed convertible.
			if asset != pnlAsset {
				mark("mixed_commission_assets")
			}
		}
		if problem == "" {
			v.GrossPnL = ptr(gross)
			v.NetPnL = ptr(net)
			v.PnLAsset = pnlAsset
			v.Completeness = "verified"
			if openValue.Sign() > 0 {
				// Nominal return: verified net PnL / opening notional, NOT leveraged ROI.
				v.ReturnPct = ptr(new(big.Rat).Mul(new(big.Rat).Quo(net, openValue), big.NewRat(100, 1)))
			}
		}
	} else if p.Status == "closed" || p.SettlementStatus == "incomplete" {
		mark("cycle_not_settled")
	}
	for asset, x := range fees {
		v.Fees = append(v.Fees, AssetFee{Asset: asset, Amount: val(x)})
	}
	sort.Slice(v.Fees, func(i, j int) bool { return v.Fees[i].Asset < v.Fees[j].Asset })
	if problem != "" {
		v.Completeness = "incomplete"
		v.IncompleteReason = problem
		v.Fees = []AssetFee{}
		v.GrossPnL = nil
		v.NetPnL = nil
		v.ReturnPct = nil
		v.PnLAsset = ""
	}
	return v, child
}
