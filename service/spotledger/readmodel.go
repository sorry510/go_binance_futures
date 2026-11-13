package spotledger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
	"go_binance_futures/models"
)

// PositionView derives its accounting from immutable Spot trade fills.
// The close price is local Spot-symbol ticker data, never Futures markPrice.
type PositionView struct {
	Position      models.SpotManagedPosition `json:"position"`
	MarkPrice     string                     `json:"mark_price,omitempty"`
	PriceTime     int64                      `json:"price_time,omitempty"`
	HasMarkPrice  bool                       `json:"has_mark_price"`
	RealizedPnL   string                     `json:"realized_pnl"`
	UnrealizedPnL string                     `json:"unrealized_pnl,omitempty"`
	TotalPnL      string                     `json:"total_pnl,omitempty"`
	FullyPriced   bool                       `json:"fully_priced"`
	UnpricedFees  map[string]string          `json:"unpriced_fees"`
	NeedsReview   bool                       `json:"needs_review"`
	ReviewReason  string                     `json:"review_reason,omitempty"`
}
type QuoteSummary struct {
	QuoteAsset        string `json:"quote_asset"`
	Positions         int64  `json:"positions"`
	RealizedPnL       string `json:"realized_pnl"`
	TotalPnL          string `json:"total_pnl,omitempty"`
	FullyPriced       bool   `json:"fully_priced"`
	MissingPrices     int64  `json:"missing_prices"`
	UnpricedPositions int64  `json:"unpriced_positions"`
	ReviewPositions   int64  `json:"review_positions"`
}

// ErrSummaryTooLarge means the request would require too many immutable
// Fill projections to guarantee a complete cross-page sum.
var ErrSummaryTooLarge = errors.New("Spot filtered summary exceeds safe limit")

type PositionViews struct {
	List    []PositionView `json:"list"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	Limit   int            `json:"limit"`
	Summary []QuoteSummary `json:"summary"`
}
type SpotReadService struct {
	ledger *Ledger
	db     orm.Ormer
	clock  func() time.Time
}

func NewSpotReadService(db orm.Ormer, dialect string) (*SpotReadService, error) {
	l, err := New(db, dialect)
	if err != nil {
		return nil, err
	}
	return &SpotReadService{ledger: l, db: db, clock: time.Now}, nil
}
func (s *SpotReadService) Ledger() *Ledger { return s.ledger }
func (s *SpotReadService) spotPrice(symbol string) (string, int64, error) {
	var price string
	var stamp int64
	err := s.db.Raw("SELECT `close`,`updateTime` FROM spot_symbols WHERE symbol=?", symbol).QueryRow(&price, &stamp)
	if err == orm.ErrNoRows {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	if _, err = parseDBDecimal(price, true); err != nil {
		return "", stamp, nil
	}
	millis := stamp
	if millis > 0 && millis < 100000000000 {
		millis *= 1000
	}
	age := s.clock().UnixMilli() - millis
	// Do not present PnL as current if local Spot WS price is stale.
	if millis <= 0 || age < 0 || age > 5*60*1000 {
		return "", stamp, nil
	}
	return price, stamp, nil
}
func (s *SpotReadService) Position(id int64) (PositionView, error) {
	var view PositionView
	if id <= 0 {
		return view, fmt.Errorf("invalid spot position id")
	}
	stored, err := s.ledger.GetPosition(id)
	if err != nil {
		return view, err
	}
	price, stamp, err := s.spotPrice(stored.Symbol)
	if err != nil {
		return view, err
	}
	view.PriceTime = stamp
	accounting, err := s.ledger.Accounting(id, price)
	if err != nil {
		// A malformed order must be VISIBLE for an operator to reconcile.
		// Never return its mutable cached PnL as a trustworthy number.
		if errors.Is(err, ErrNeedsReview) {
			// Keep identity/status so the operator can find the record,
			// but never serialize mutable cached monetary amounts.
			stored.BoughtQty = ""
			stored.RemainingQty = ""
			stored.AvgCostQuote = ""
			stored.RealizedPnL = ""
			view.Position = stored
			view.NeedsReview = true
			view.ReviewReason = err.Error()
			view.UnpricedFees = map[string]string{}
			return view, nil
		}
		return view, err
	}
	view.Position = accounting.Position
	view.MarkPrice = accounting.MarkPrice
	view.HasMarkPrice = accounting.HasMarkPrice
	view.RealizedPnL = accounting.RealizedPnL
	view.UnrealizedPnL = accounting.UnrealizedPnL
	view.FullyPriced = accounting.FullyPriced
	if view.FullyPriced {
		view.TotalPnL = accounting.TotalPnL
	}
	view.UnpricedFees = accounting.UnpricedFees
	// A CLOSED position has no unrealized PnL even without a quote.
	if view.Position.Status == "CLOSED" && !view.HasMarkPrice {
		view.UnrealizedPnL = "0.000000000000000000"
		if view.FullyPriced {
			view.TotalPnL = view.RealizedPnL
		}
	}
	return view, nil
}
func (s *SpotReadService) Positions(f PositionFilter) (PositionViews, error) {
	var output PositionViews
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	output.Page = f.Page
	output.Limit = f.Limit
	page, err := s.ledger.ListPositions(f)
	if err != nil {
		return output, err
	}
	output.Total = page.Total
	// Exact Fill replay is intentionally not replaced by mutable cached
	// PnL. Bound the work so a large history cannot exhaust an HTTP
	// request. Never return a deceptively partial "total" instead.
	const maxSummaryPositions = 1000
	if output.Total > maxSummaryPositions {
		return PositionViews{}, fmt.Errorf("%w: %d matching Spot positions exceed the exact summary limit of %d; narrow filters", ErrSummaryTooLarge, output.Total, maxSummaryPositions)
	}
	output.List = make([]PositionView, 0, len(page.List))
	viewByID := make(map[int64]PositionView, len(page.List))
	for _, row := range page.List {
		v, err := s.Position(row.ID)
		if err != nil {
			return output, err
		}
		viewByID[row.ID] = v
		output.List = append(output.List, v)
	}
	// Summary deliberately spans the ENTIRE filtered result, not page 1.
	// Multiple quote assets are never added together as if all were USDT.
	quotes := map[string]*QuoteSummary{}
	realizedTotals := map[string]decimal.Decimal{}
	totalTotals := map[string]decimal.Decimal{}
	filter := f
	for offsetPage := 1; ; offsetPage++ {
		filter.Page = offsetPage
		filter.Limit = 100
		all, err := s.ledger.ListPositions(filter)
		if err != nil {
			return output, err
		}
		for _, p := range all.List {
			entry, ok := quotes[p.QuoteAsset]
			if !ok {
				entry = &QuoteSummary{QuoteAsset: p.QuoteAsset, FullyPriced: true}
				quotes[p.QuoteAsset] = entry
			}
			entry.Positions++
			v, known := viewByID[p.ID]
			if !known {
				v, err = s.Position(p.ID)
				if err != nil {
					return output, err
				}
			}
			if v.NeedsReview {
				entry.ReviewPositions++
				entry.FullyPriced = false
				continue
			}
			realized, err := decimal.NewFromString(v.RealizedPnL)
			if err != nil {
				return output, err
			}
			realizedTotals[p.QuoteAsset] = realizedTotals[p.QuoteAsset].Add(realized)
			if !v.FullyPriced {
				entry.UnpricedPositions++
				entry.FullyPriced = false
			}
			if v.TotalPnL == "" {
				entry.MissingPrices++
				entry.FullyPriced = false
				continue
			}
			total, err := decimal.NewFromString(v.TotalPnL)
			if err != nil {
				return output, err
			}
			totalTotals[p.QuoteAsset] = totalTotals[p.QuoteAsset].Add(total)
		}
		if offsetPage*100 >= int(all.Total) {
			break
		}
	}
	for _, q := range []string{"USDT", "USDC", "FDUSD"} {
		entry, ok := quotes[q]
		if !ok {
			continue
		}
		entry.RealizedPnL = realizedTotals[q].StringFixed(18)
		if entry.FullyPriced {
			entry.TotalPnL = totalTotals[q].StringFixed(18)
		}
		output.Summary = append(output.Summary, *entry)
	}
	return output, nil
}

type OrderFilter struct {
	PositionID int64
	Symbol     string
	Status     string
	Page       int
	Limit      int
}
type OrderPage struct {
	Total int64                     `json:"total"`
	List  []models.SpotManagedOrder `json:"list"`
}

func (s *SpotReadService) Orders(f OrderFilter) (OrderPage, error) {
	var out OrderPage
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	where := []string{}
	args := []interface{}{}
	if f.PositionID > 0 {
		where = append(where, "position_id=?")
		args = append(args, f.PositionID)
	}
	if f.Symbol != "" {
		symbol := strings.ToUpper(strings.TrimSpace(f.Symbol))
		if !assetPattern.MatchString(symbol) {
			return out, fmt.Errorf("invalid spot symbol filter")
		}
		where = append(where, "symbol LIKE ?")
		args = append(args, "%"+symbol+"%")
	}
	if f.Status != "" {
		if !orderStatuses[f.Status] {
			return out, fmt.Errorf("invalid spot order status")
		}
		where = append(where, "status=?")
		args = append(args, f.Status)
	}
	condition := ""
	if len(where) > 0 {
		condition = " WHERE " + strings.Join(where, " AND ")
	}
	if err := s.db.Raw("SELECT COUNT(*) FROM spot_managed_orders"+condition, args...).QueryRow(&out.Total); err != nil {
		return out, err
	}
	out.List = make([]models.SpotManagedOrder, 0)
	sql := "SELECT id,position_id,symbol,side,order_type,client_order_id,exchange_order_id,requested_qty,filled_qty,status,created_at,updated_at FROM spot_managed_orders" + condition +
		" ORDER BY id DESC LIMIT ? OFFSET ?"
	_, err := s.db.Raw(sql, append(args, f.Limit, (f.Page-1)*f.Limit)...).QueryRows(&out.List)
	return out, err
}
