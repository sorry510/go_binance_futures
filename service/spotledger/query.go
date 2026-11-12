package spotledger

import (
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/models"
)

type PositionFilter struct {
	Symbol     string // exact match by default: execution/recovery contract
	SymbolLike bool   // explicit UI-only substring search
	Status     string
	Page       int
	Limit      int
}

type PositionPage struct {
	Total int64                        `json:"total"`
	List  []models.SpotManagedPosition `json:"list"`
}

// ListPositions is the 06-3 backend read model consumed by the 06-5 Spot
// local account page. It never joins the Spot wallet or any Futures table.
func (l *Ledger) ListPositions(f PositionFilter) (PositionPage, error) {
	var page PositionPage
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
	params := []interface{}{}
	if f.Symbol != "" {
		f.Symbol = strings.ToUpper(strings.TrimSpace(f.Symbol))
		if !assetPattern.MatchString(f.Symbol) {
			return page, fmt.Errorf("invalid Spot symbol filter")
		}
		// Execution/recovery depends on exact symbol identity. Fuzzy
		// matching is explicitly opt-in for the read-only UI.
		if f.SymbolLike {
			where = append(where, "symbol LIKE ?")
			params = append(params, "%"+f.Symbol+"%")
		} else {
			where = append(where, "symbol=?")
			params = append(params, f.Symbol)
		}
	}
	if f.Status != "" {
		if f.Status != "PENDING_BUY" && f.Status != "OPEN" && f.Status != "CLOSED" {
			return page, fmt.Errorf("invalid Spot position status filter")
		}
		where = append(where, "status=?")
		params = append(params, f.Status)
	}
	filter := ""
	if len(where) > 0 {
		filter = " WHERE " + strings.Join(where, " AND ")
	}
	if err := l.db.Raw("SELECT COUNT(*) FROM spot_managed_positions"+filter, params...).QueryRow(&page.Total); err != nil {
		return page, err
	}
	page.List = make([]models.SpotManagedPosition, 0)
	query := "SELECT id,symbol,base_asset,quote_asset,bought_qty,remaining_qty,avg_cost_quote,realized_pnl,status,created_at,updated_at,closed_at FROM spot_managed_positions" + filter + " ORDER BY id DESC LIMIT " + strconv.Itoa(f.Limit) + " OFFSET " + strconv.Itoa((f.Page-1)*f.Limit)
	_, err := l.db.Raw(query, params...).QueryRows(&page.List)
	return page, err
}
