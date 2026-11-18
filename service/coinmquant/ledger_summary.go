package coinmquant

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

type PositionPage struct {
	Total       int64             `json:"total"`
	List        []PositionSummary `json:"list"`
	Totals      []AssetTotal      `json:"totals"`
	PriceStatus string            `json:"price_status"`
}
type fillEvidence struct {
	PositionID        int64  `orm:"column(position_id)"`
	FillSymbol        string `orm:"column(fill_symbol)"`
	FillPositionSide  string `orm:"column(fill_position_side)"`
	FillSide          string `orm:"column(fill_side)"`
	OrderPositionID   int64  `orm:"column(order_position_id)"`
	OrderSymbol       string `orm:"column(order_symbol)"`
	OrderPositionSide string `orm:"column(order_position_side)"`
	OrderSide         string `orm:"column(order_side)"`
	PnLAsset          string `orm:"column(pnl_asset)"`
	RealizedPnL       string `orm:"column(realized_pnl)"`
	CommissionAsset   string `orm:"column(commission_asset)"`
	Commission        string `orm:"column(commission)"`
	Contracts         string `orm:"column(contracts)"`
	Price             string `orm:"column(price)"`
	Intent            string `orm:"column(intent)"`
}
type fillAmounts struct {
	open, close, realized, realizedUSD *big.Rat
	fees                               map[string]*big.Rat
	count                              int
	bad                                string
}

func newFillAmounts() *fillAmounts {
	return &fillAmounts{
		open: decimalZero(), close: decimalZero(), realized: decimalZero(), realizedUSD: decimalZero(), fees: map[string]*big.Rat{},
	}
}
func positiveOrZero(s string) (*big.Rat, error) {
	n, err := exact(s, false)
	if err != nil {
		return nil, err
	}
	if n.Sign() < 0 {
		return nil, fmt.Errorf("negative contract count")
	}
	return n, nil
}
func loadPositionsFiltered(o orm.Ormer, f LedgerFilter) ([]models.CoinMManagedPosition, int64, string, []interface{}, error) {
	where, args, err := ledgerWhere(f)
	if err != nil {
		return nil, 0, "", nil, err
	}
	var count int64
	if err = o.Raw("SELECT COUNT(*) FROM coin_m_managed_positions"+where, args...).QueryRow(&count); err != nil {
		return nil, 0, "", nil, err
	}
	rows := make([]models.CoinMManagedPosition, 0)
	if count == 0 {
		return rows, 0, where, args, nil
	}
	query := "SELECT id,symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at FROM coin_m_managed_positions" + where + " ORDER BY id DESC"
	if _, err = o.Raw(query, args...).QueryRows(&rows); err != nil {
		return nil, 0, "", nil, err
	}
	return rows, count, where, args, nil
}
func NeedMarks(o orm.Ormer, f LedgerFilter) (bool, error) {
	where, args, err := ledgerWhere(f)
	if err != nil {
		return false, err
	}
	var n int64
	sql := "SELECT COUNT(*) FROM coin_m_managed_positions" + where
	if where == "" {
		sql += " WHERE remaining_contracts>0"
	} else {
		sql += " AND remaining_contracts>0"
	}
	err = o.Raw(sql, args...).QueryRow(&n)
	return n > 0, err
}

func loadFillEvidence(o orm.Ormer, where string, args []interface{}, positions map[int64]models.CoinMManagedPosition) (map[int64]*fillAmounts, error) {
	sums := map[int64]*fillAmounts{}
	if len(positions) == 0 {
		return sums, nil
	}
	for id := range positions {
		sums[id] = newFillAmounts()
	}
	// Join orders by ID only so corrupt/mismatched order+Fill evidence is
	// still visible and can invalidate the managed cycle. Filtering by
	// symbol/side inside the JOIN would silently discard hostile evidence,
	// possibly allowing a different set of Fills to look verified.
	scoped := strings.ReplaceAll(strings.ReplaceAll(where, "symbol=", "p.symbol="), "status=", "p.status=")
	query := "SELECT f.position_id AS position_id, f.symbol AS fill_symbol, " +
		"f.position_side AS fill_position_side, f.side AS fill_side, " +
		"COALESCE(o.position_id,0) AS order_position_id, " +
		"COALESCE(o.symbol,'') AS order_symbol, " +
		"COALESCE(o.position_side,'') AS order_position_side, " +
		"COALESCE(o.side,'') AS order_side, " +
		"COALESCE(o.intent,'') AS intent, " +
		"f.pnl_asset AS pnl_asset, f.realized_pnl AS realized_pnl, " +
		"f.commission_asset AS commission_asset, f.commission AS commission, " +
		"f.contracts AS contracts, f.price AS price " +
		"FROM coin_m_trade_fills f " +
		"JOIN coin_m_managed_positions p ON p.id=f.position_id " +
		"LEFT JOIN coin_m_managed_orders o ON o.id=f.order_id"
	if scoped != "" {
		query += " " + scoped
	}
	rows := make([]fillEvidence, 0)
	if _, err := o.Raw(query, args...).QueryRows(&rows); err != nil {
		return nil, err
	}
	for _, f := range rows {
		pos, found := positions[f.PositionID]
		if !found {
			continue
		}
		agg := sums[f.PositionID]
		agg.count++
		if agg.bad != "" {
			continue
		}
		// Validate BOTH relationships, not merely the order-to-Fill
		// relationship. A self-consistent ETH/SHORT order and Fill must
		// never be counted as BTC/LONG managed position evidence.
		if f.FillSymbol != pos.Symbol ||
			f.FillPositionSide != pos.PositionSide ||
			f.OrderPositionID != pos.ID ||
			f.OrderSymbol != pos.Symbol ||
			f.OrderPositionSide != pos.PositionSide ||
			f.OrderSymbol != f.FillSymbol ||
			f.OrderPositionSide != f.FillPositionSide ||
			f.FillSide != f.OrderSide {
			agg.bad = "Fill/order symbol, positionSide, side or ownership differs from managed cycle"
			continue
		}
		if f.PnLAsset != pos.MarginAsset && f.PnLAsset != "" {
			agg.bad = "Fill PnL asset differs from settlement asset"
			continue
		}
		amount, err := positiveOrZero(f.Contracts)
		if err != nil || amount.Sign() == 0 {
			agg.bad = "invalid Fill contract count"
			continue
		}
		realized, err := exact(f.RealizedPnL, false)
		if err != nil {
			agg.bad = "invalid Fill realized PnL"
			continue
		}
		fillPrice, err := exact(f.Price, true)
		if err != nil {
			agg.bad = "invalid verified Fill price"
			continue
		}
		fee, err := exact(f.Commission, false)
		if err != nil {
			agg.bad = "invalid Fill commission"
			continue
		}
		if f.CommissionAsset == "" {
			agg.bad = "missing Fill commission asset"
			continue
		}
		switch f.Intent {
		case "OPEN":
			if (pos.PositionSide == "LONG" && f.FillSide != "BUY") || (pos.PositionSide == "SHORT" && f.FillSide != "SELL") {
				agg.bad = "Fill opening side conflicts with position"
				continue
			}
			agg.open.Add(agg.open, amount)
		case "CLOSE", "PROTECT":
			if (pos.PositionSide == "LONG" && f.FillSide != "SELL") || (pos.PositionSide == "SHORT" && f.FillSide != "BUY") {
				agg.bad = "Fill closing side conflicts with position"
				continue
			}
			agg.close.Add(agg.close, amount)
		default:
			agg.bad = "Fill order intent is neither OPEN nor CLOSE"
			continue
		}
		if f.PnLAsset == "" && realized.Sign() != 0 {
			agg.bad = "nonzero PnL missing settlement asset"
			continue
		}
		agg.realized.Add(agg.realized, realized)
		// Reference USD conversion uses each Fill's own execution price,
		// never today's mark for a historical realized trade.
		if settledAssetForSymbol(pos.Symbol, pos.MarginAsset) {
			agg.realizedUSD.Add(agg.realizedUSD, usdEstimate(realized, fillPrice))
		}
		if _, ok := agg.fees[f.CommissionAsset]; !ok {
			agg.fees[f.CommissionAsset] = decimalZero()
		}
		agg.fees[f.CommissionAsset].Add(agg.fees[f.CommissionAsset], fee)
	}
	return sums, nil
}
func sortedFees(fees map[string]*big.Rat) []AssetFee {
	keys := make([]string, 0, len(fees))
	for k := range fees {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]AssetFee, 0, len(keys))
	for _, k := range keys {
		result = append(result, AssetFee{Asset: k, Amount: decimalString(fees[k])})
	}
	return result
}

type assetAccum struct {
	total                                            AssetTotal
	realized, unrealized, realizedUSD, unrealizedUSD *big.Rat
	fees                                             map[string]*big.Rat
	usdComplete                                      bool
	allPriced                                        bool
}

func accumulateAsset(m map[string]*assetAccum, summary PositionSummary) {
	asset := summary.MarginAsset
	a, ok := m[asset]
	if !ok {
		a = &assetAccum{total: AssetTotal{Asset: asset, FundingIncluded: false},
			realized: decimalZero(), unrealized: decimalZero(), realizedUSD: decimalZero(),
			unrealizedUSD: decimalZero(), fees: map[string]*big.Rat{}, usdComplete: true, allPriced: true}
		m[asset] = a
	}
	a.total.Positions++
	if summary.PnLStatus == "pending_fills" {
		a.total.Pending++
		a.allPriced = false
		a.usdComplete = false
		return
	}
	if summary.PnLStatus == "invalid" || summary.PnLStatus == "unverified_fills" {
		a.total.Invalid++
		a.allPriced = false
		a.usdComplete = false
		return
	}
	a.total.Verified++
	realized, _ := exact(summary.RealizedPnL, false)
	a.realized.Add(a.realized, realized)
	for _, fee := range summary.FeeByAsset {
		n, _ := exact(fee.Amount, false)
		if _, ok := a.fees[fee.Asset]; !ok {
			a.fees[fee.Asset] = decimalZero()
		}
		a.fees[fee.Asset].Add(a.fees[fee.Asset], n)
	}
	if summary.UnrealizedPnL == nil {
		a.total.Unpriced++
		a.allPriced = false
	} else {
		n, _ := exact(*summary.UnrealizedPnL, false)
		a.unrealized.Add(a.unrealized, n)
	}
	if summary.RealizedUSD == nil {
		a.usdComplete = false
	} else {
		n, _ := exact(*summary.RealizedUSD, false)
		a.realizedUSD.Add(a.realizedUSD, n)
	}
	if summary.UnrealizedUSD == nil {
		a.usdComplete = false
	} else {
		n, _ := exact(*summary.UnrealizedUSD, false)
		a.unrealizedUSD.Add(a.unrealizedUSD, n)
	}
}

// PositionSummaries builds a full FILTERED ledger snapshot before paging.
// Consequently totals are scoped to the filter, not just the visible page.
// Funds never move; it only reads the dedicated COIN-M tables.
func PositionSummaries(o orm.Ormer, f LedgerFilter, marks map[string]Mark) (PositionPage, error) {
	positions, count, where, args, err := loadPositionsFiltered(o, f)
	if err != nil {
		return PositionPage{}, err
	}
	result := PositionPage{Total: count, List: []PositionSummary{}, Totals: []AssetTotal{}}
	if count == 0 {
		return result, nil
	}
	byID := make(map[int64]models.CoinMManagedPosition, len(positions))
	for _, p := range positions {
		byID[p.ID] = p
	}
	evidence, err := loadFillEvidence(o, where, args, byID)
	if err != nil {
		return PositionPage{}, err
	}
	totals := map[string]*assetAccum{}
	page, limit := pageSize(f.Page, f.Limit)
	start := (page - 1) * limit
	end := start + limit
	for i, p := range positions {
		s := PositionSummary{CoinMManagedPosition: p, RealizedPnL: "0", PnLStatus: "verified",
			FeeByAsset: []AssetFee{}, FundingIncluded: false, OpenFilled: "0", CloseFilled: "0"}
		agg := evidence[p.ID]
		if agg == nil {
			agg = newFillAmounts()
		}
		s.OpenFilled = decimalString(agg.open)
		s.CloseFilled = decimalString(agg.close)
		s.RealizedPnL = decimalString(agg.realized)
		s.FeeByAsset = sortedFees(agg.fees)
		if settledAssetForSymbol(p.Symbol, p.MarginAsset) && agg.bad == "" {
			s.RealizedUSD = stringPtr(decimalString(agg.realizedUSD))
		}
		opened, er1 := positiveOrZero(p.OpenedContracts)
		remaining, er2 := positiveOrZero(p.RemainingContracts)
		contract, er3 := exact(p.ContractSize, true)
		entry, er4 := exact(p.EntryPrice, true)
		if agg.count == 0 && er1 == nil && er2 == nil && opened.Sign() == 0 && remaining.Sign() == 0 &&
			(p.Status == "PENDING_OPEN" || p.Status == "OPENING") {
			s.PnLStatus = "pending_fills"
			s.Warning = "position awaits the first verified opening Fill"
		} else if agg.bad != "" || er1 != nil || er2 != nil || er3 != nil || er4 != nil ||
			(opened != nil && remaining != nil && remaining.Cmp(opened) > 0) ||
			(p.PositionSide != "LONG" && p.PositionSide != "SHORT") ||
			(strings.EqualFold(p.Status, "CLOSED") && remaining != nil && remaining.Sign() != 0) {
			s.PnLStatus = "invalid"
			s.Warning = "invalid position or Fill evidence"
			if agg.bad != "" {
				s.Warning = agg.bad
			}
		} else {
			expectedClose := new(big.Rat).Sub(opened, remaining)
			if agg.open.Cmp(opened) != 0 || agg.close.Cmp(expectedClose) != 0 {
				s.PnLStatus = "unverified_fills"
				s.Warning = "Fill contract counts do not reconcile with managed cycle"
			} else if remaining.Sign() == 0 {
				if s.RealizedUSD != nil {
					zeroUSD := "0"
					s.UnrealizedUSD = &zeroUSD
					s.TotalUSD = s.RealizedUSD
				}
				zero := "0"
				s.UnrealizedPnL = &zero
				s.TotalPnL = &s.RealizedPnL
			} else {
				mark, exists := marks[p.Symbol]
				if !exists || mark.Check(p.Symbol) != nil || mark.Time <= 0 {
					s.PnLStatus = "unpriced"
					s.Warning = "fresh COIN-M mark price unavailable"
				} else {
					un, err := InversePnL(p.PositionSide, decimalString(remaining), decimalString(contract),
						decimalString(entry), mark.Price)
					if err != nil {
						s.PnLStatus = "invalid"
						s.Warning = err.Error()
					} else {
						s.MarkPrice = &mark.Price
						s.MarkTime = mark.Time
						s.UnrealizedPnL = &un
						unreal, _ := exact(un, false)
						s.TotalPnL = stringPtr(decimalString(decimalAdd(agg.realized, unreal)))
						if settledAssetForSymbol(p.Symbol, p.MarginAsset) {
							price, _ := exact(mark.Price, true)
							s.UnrealizedUSD = stringPtr(decimalString(usdEstimate(unreal, price)))
							if s.RealizedUSD != nil {
								sum, _ := exact(*s.RealizedUSD, false)
								s.TotalUSD = stringPtr(decimalString(decimalAdd(sum, usdEstimate(unreal, price))))
							}
						}
					}
				}
			}
		}
		accumulateAsset(totals, s)
		if i >= start && i < end {
			result.List = append(result.List, s)
		}
	}
	keys := make([]string, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		a := totals[key]
		a.total.RealizedPnL = decimalString(a.realized)
		a.total.Fees = sortedFees(a.fees)
		if a.allPriced {
			unreal := decimalString(a.unrealized)
			a.total.UnrealizedPnL = &unreal
			a.total.TotalPnL = stringPtr(decimalString(decimalAdd(a.realized, a.unrealized)))
		}
		if a.usdComplete && a.total.Verified > 0 {
			a.total.RealizedUSD = stringPtr(decimalString(a.realizedUSD))
			a.total.UnrealizedUSD = stringPtr(decimalString(a.unrealizedUSD))
			a.total.TotalUSD = stringPtr(decimalString(decimalAdd(a.realizedUSD, a.unrealizedUSD)))
		}
		result.Totals = append(result.Totals, a.total)
	}
	return result, nil
}
func stringPtr(s string) *string { return &s }
func ParsePageParam(s string, defaultValue int) int {
	n, e := strconv.Atoi(s)
	if e != nil {
		return defaultValue
	}
	return n
}
