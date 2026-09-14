package outcomereview

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	backtestservice "go_binance_futures/service/backtest"
	"go_binance_futures/service/strategy"

	"github.com/beego/beego/v2/client/orm"
)

type Service struct{}

type Filter struct {
	StartTime          int64
	EndTime            int64
	StrategyTemplateID int64
	Symbol             string
	Side               string
	MarketCondition    int
}

type Group struct {
	Key              string  `json:"key"`
	Label            string  `json:"label"`
	TradeCount       int     `json:"trade_count"`
	Wins             int     `json:"wins"`
	NetPnL           float64 `json:"net_pnl"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`
	Fees             float64 `json:"fees"`
	Funding          float64 `json:"funding"`
	AverageHoldingMs int64   `json:"average_holding_ms"`
}

type BacktestSummary struct {
	RunCount             int     `json:"run_count"`
	TradeCount           int     `json:"trade_count"`
	NetPnL               float64 `json:"net_pnl"`
	ReturnPct            float64 `json:"return_pct"`
	MaxDrawdownPct       float64 `json:"max_drawdown_pct"`
	MaxDrawdownAvailable bool    `json:"max_drawdown_available"`
	WinRate              float64 `json:"win_rate"`
	ProfitFactor         float64 `json:"profit_factor"`
	Fees                 float64 `json:"fees"`
	Funding              float64 `json:"funding"`
	AverageHoldingMs     int64   `json:"average_holding_ms"`
	BySymbol             []Group `json:"by_symbol"`
	BySide               []Group `json:"by_side"`
	ByMarketCondition    []Group `json:"by_market_condition"`
}

type LiveProposalSummary struct {
	ProposalID      string `json:"proposal_id" orm:"column(proposal_id)"`
	SourceTaskID    string `json:"source_task_id" orm:"column(source_task_id)"`
	Symbol          string `json:"symbol" orm:"column(symbol)"`
	Side            string `json:"side" orm:"column(side)"`
	Status          string `json:"status" orm:"column(status)"`
	RiskStatus      string `json:"risk_status" orm:"column(risk_status)"`
	MarketCondition int    `json:"market_condition" orm:"column(market_condition)"`
	CreatedAt       int64  `json:"created_at" orm:"column(created_at)"`
	ExecutedAt      int64  `json:"executed_at" orm:"column(executed_at)"`
}

type LiveSummary struct {
	Proposals       int64                 `json:"proposals"`
	Executed        int64                 `json:"executed"`
	OpenPositions   int64                 `json:"open_positions"`
	ClosedPositions int64                 `json:"closed_positions"`
	ManagedOrders   int64                 `json:"managed_orders"`
	PnLAvailable    bool                  `json:"pnl_available"`
	RecentProposals []LiveProposalSummary `json:"recent_proposals"`
}

type runMetricRow struct {
	InitialEquity float64 `orm:"column(initial_equity)"`
	MetricsJSON   string  `orm:"column(metrics_json)"`
}

type aggregateRow struct {
	TradeCount       int64   `orm:"column(trade_count)"`
	Wins             int64   `orm:"column(wins)"`
	NetPnL           float64 `orm:"column(net_pnl)"`
	WinPnL           float64 `orm:"column(win_pnl)"`
	LossAbs          float64 `orm:"column(loss_abs)"`
	Fees             float64 `orm:"column(fees)"`
	Funding          float64 `orm:"column(funding)"`
	AverageHoldingMs float64 `orm:"column(average_holding_ms)"`
}

type groupAggregateRow struct {
	Key              string  `orm:"column(group_key)"`
	TradeCount       int64   `orm:"column(trade_count)"`
	Wins             int64   `orm:"column(wins)"`
	NetPnL           float64 `orm:"column(net_pnl)"`
	WinPnL           float64 `orm:"column(win_pnl)"`
	LossAbs          float64 `orm:"column(loss_abs)"`
	Fees             float64 `orm:"column(fees)"`
	Funding          float64 `orm:"column(funding)"`
	AverageHoldingMs float64 `orm:"column(average_holding_ms)"`
}

const backtestAggregateColumns = `COUNT(*) AS trade_count,
	COALESCE(SUM(CASE WHEN t.net_pnl > 0 THEN 1 ELSE 0 END), 0) AS wins,
	COALESCE(SUM(t.net_pnl), 0) AS net_pnl,
	COALESCE(SUM(CASE WHEN t.net_pnl > 0 THEN t.net_pnl ELSE 0 END), 0) AS win_pnl,
	COALESCE(SUM(CASE WHEN t.net_pnl < 0 THEN -t.net_pnl ELSE 0 END), 0) AS loss_abs,
	COALESCE(SUM(t.fees), 0) AS fees,
	COALESCE(SUM(t.funding_pnl), 0) AS funding,
	COALESCE(AVG(t.holding_ms), 0) AS average_holding_ms`

func (Service) Backtest(ctx context.Context, f Filter) (BacktestSummary, error) {
	if err := validateFilter(f); err != nil {
		return BacktestSummary{}, err
	}
	where, args := backtestWhere(f, "r", "t")
	o := orm.NewOrm()
	baseSQL := ` FROM agent_backtest_trades t JOIN agent_backtest_runs r ON r.run_id=t.run_id WHERE r.status='succeeded'` + where
	var total aggregateRow
	if err := o.Raw(`SELECT `+backtestAggregateColumns+baseSQL, args...).QueryRow(&total); err != nil {
		return BacktestSummary{}, fmt.Errorf("query backtest outcome summary: %w", err)
	}
	result := summaryFromAggregate(total)
	var err error
	if result.BySymbol, err = queryBacktestGroups(o, baseSQL, args, "t.symbol"); err != nil {
		return BacktestSummary{}, fmt.Errorf("query backtest outcomes by symbol: %w", err)
	}
	if result.BySide, err = queryBacktestGroups(o, baseSQL, args, "t.side"); err != nil {
		return BacktestSummary{}, fmt.Errorf("query backtest outcomes by side: %w", err)
	}
	if result.ByMarketCondition, err = queryBacktestGroups(o, baseSQL, args, "t.market_condition"); err != nil {
		return BacktestSummary{}, fmt.Errorf("query backtest outcomes by market condition: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return BacktestSummary{}, err
	}

	var runRows []runMetricRow
	runWhere, runArgs := backtestRunWhere(f, "r")
	if _, err := o.Raw(`SELECT r.initial_equity, r.metrics_json FROM agent_backtest_runs r WHERE r.status='succeeded'`+runWhere, runArgs...).QueryRows(&runRows); err != nil {
		return BacktestSummary{}, fmt.Errorf("query backtest runs: %w", err)
	}
	result.RunCount = len(runRows)
	result.MaxDrawdownAvailable = strings.TrimSpace(f.Side) == "" && f.MarketCondition <= 0
	initial := 0.0
	for _, row := range runRows {
		initial += row.InitialEquity
		if result.MaxDrawdownAvailable {
			var m backtestservice.Metrics
			if json.Unmarshal([]byte(row.MetricsJSON), &m) == nil && m.MaxDrawdownPct > result.MaxDrawdownPct {
				result.MaxDrawdownPct = m.MaxDrawdownPct
			}
		}
	}
	if initial > 0 {
		result.ReturnPct = result.NetPnL / initial * 100
	}
	return result, nil
}

func (Service) Paper(ctx context.Context, f Filter) (strategy.TestResultReviewStats, error) {
	if err := validateFilter(f); err != nil {
		return strategy.TestResultReviewStats{}, err
	}
	if f.MarketCondition > 0 {
		return strategy.TestResultReviewStats{}, fmt.Errorf("market_condition is not available for paper outcomes")
	}
	opts := strategy.TestResultsOptions{Symbol: f.Symbol, PositionSide: f.Side, StrategyTemplateID: f.StrategyTemplateID, Page: 1, Limit: 1, DefaultLimit: 1, MaxLimit: 1}
	if f.StartTime > 0 {
		opts.StartTime = strconv.FormatInt(f.StartTime, 10)
	}
	if f.EndTime > 0 {
		opts.EndTime = strconv.FormatInt(f.EndTime, 10)
	}
	result, err := (strategy.Service{}).ListTestResults(ctx, opts)
	if err != nil {
		return strategy.TestResultReviewStats{}, err
	}
	return result.Stats, nil
}

func (Service) Live(ctx context.Context, f Filter) (LiveSummary, error) {
	if err := validateFilter(f); err != nil {
		return LiveSummary{}, err
	}
	if f.StrategyTemplateID > 0 {
		return LiveSummary{}, fmt.Errorf("strategy_template_id is not available for live outcomes")
	}
	o := orm.NewOrm()
	proposalWhere, proposalArgs := liveProposalWhere(f)
	var result LiveSummary
	if err := o.Raw(`SELECT COUNT(*) FROM agent_trade_proposals WHERE 1=1`+proposalWhere, proposalArgs...).QueryRow(&result.Proposals); err != nil {
		return result, err
	}
	if err := o.Raw(`SELECT COUNT(*) FROM agent_trade_proposals WHERE executed_at>0`+proposalWhere, proposalArgs...).QueryRow(&result.Executed); err != nil {
		return result, err
	}
	positionWhere, positionArgs := liveManagedWhere(f, "mp")
	if err := o.Raw(`SELECT COUNT(*) FROM futures_managed_positions mp WHERE mp.owner='agent_trade' AND mp.status!='closed' AND mp.managed_qty>0`+positionWhere, positionArgs...).QueryRow(&result.OpenPositions); err != nil {
		return result, err
	}
	if err := o.Raw(`SELECT COUNT(*) FROM futures_managed_positions mp WHERE mp.owner='agent_trade' AND mp.status='closed'`+positionWhere, positionArgs...).QueryRow(&result.ClosedPositions); err != nil {
		return result, err
	}
	orderWhere, orderArgs := liveManagedWhere(f, "mo")
	if err := o.Raw(`SELECT COUNT(*) FROM futures_managed_orders mo WHERE mo.owner='agent_trade'`+orderWhere, orderArgs...).QueryRow(&result.ManagedOrders); err != nil {
		return result, err
	}
	result.RecentProposals = make([]LiveProposalSummary, 0)
	if _, err := o.Raw(`SELECT proposal_id, source_task_id, symbol, side, status, risk_status, market_condition, created_at, executed_at FROM agent_trade_proposals WHERE 1=1`+proposalWhere+` ORDER BY created_at DESC LIMIT 20`, proposalArgs...).QueryRows(&result.RecentProposals); err != nil {
		return result, err
	}
	result.PnLAvailable = false
	return result, ctx.Err()
}

func summaryFromAggregate(row aggregateRow) BacktestSummary {
	result := BacktestSummary{
		TradeCount: int(row.TradeCount), NetPnL: row.NetPnL, Fees: row.Fees, Funding: row.Funding,
		AverageHoldingMs: int64(row.AverageHoldingMs),
	}
	if row.TradeCount > 0 {
		result.WinRate = float64(row.Wins) / float64(row.TradeCount)
	}
	if row.LossAbs > 0 {
		result.ProfitFactor = row.WinPnL / row.LossAbs
	} else if row.WinPnL > 0 {
		result.ProfitFactor = row.WinPnL
	}
	return result
}

func queryBacktestGroups(o orm.Ormer, baseSQL string, args []interface{}, keyExpr string) ([]Group, error) {
	var rows []groupAggregateRow
	query := `SELECT ` + keyExpr + ` AS group_key, ` + backtestAggregateColumns + baseSQL + ` GROUP BY ` + keyExpr + ` ORDER BY trade_count DESC, group_key ASC`
	if _, err := o.Raw(query, args...).QueryRows(&rows); err != nil {
		return nil, err
	}
	result := make([]Group, 0, len(rows))
	for _, row := range rows {
		summary := summaryFromAggregate(aggregateRow{
			TradeCount: row.TradeCount, Wins: row.Wins, NetPnL: row.NetPnL, WinPnL: row.WinPnL,
			LossAbs: row.LossAbs, Fees: row.Fees, Funding: row.Funding, AverageHoldingMs: row.AverageHoldingMs,
		})
		result = append(result, Group{
			Key: row.Key, Label: row.Key, TradeCount: summary.TradeCount, Wins: int(row.Wins),
			NetPnL: summary.NetPnL, WinRate: summary.WinRate, ProfitFactor: summary.ProfitFactor,
			Fees: summary.Fees, Funding: summary.Funding, AverageHoldingMs: summary.AverageHoldingMs,
		})
	}
	return result, nil
}

func validateFilter(f Filter) error {
	if f.StartTime > 0 && f.EndTime > 0 && f.StartTime > f.EndTime {
		return fmt.Errorf("start_time must not exceed end_time")
	}
	side := strings.ToUpper(strings.TrimSpace(f.Side))
	if side != "" && side != "LONG" && side != "SHORT" {
		return fmt.Errorf("invalid side")
	}
	return nil
}
func backtestWhere(f Filter, r, t string) (string, []interface{}) {
	parts := []string{}
	args := []interface{}{}
	if f.StartTime > 0 {
		parts = append(parts, r+".completed_at>=?")
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		parts = append(parts, r+".completed_at<=?")
		args = append(args, f.EndTime)
	}
	if f.StrategyTemplateID > 0 {
		parts = append(parts, r+".strategy_template_id=?")
		args = append(args, f.StrategyTemplateID)
	}
	if s := strings.TrimSpace(f.Symbol); s != "" {
		parts = append(parts, t+".symbol=?")
		args = append(args, strings.ToUpper(s))
	}
	if s := strings.TrimSpace(f.Side); s != "" {
		parts = append(parts, t+".side=?")
		args = append(args, strings.ToUpper(s))
	}
	if f.MarketCondition > 0 {
		parts = append(parts, t+".market_condition=?")
		args = append(args, f.MarketCondition)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " AND " + strings.Join(parts, " AND "), args
}
func backtestRunWhere(f Filter, r string) (string, []interface{}) {
	parts := []string{}
	args := []interface{}{}
	if f.StartTime > 0 {
		parts = append(parts, r+".completed_at>=?")
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		parts = append(parts, r+".completed_at<=?")
		args = append(args, f.EndTime)
	}
	if f.StrategyTemplateID > 0 {
		parts = append(parts, r+".strategy_template_id=?")
		args = append(args, f.StrategyTemplateID)
	}
	if s := strings.TrimSpace(f.Symbol); s != "" {
		parts = append(parts, r+".symbol=?")
		args = append(args, strings.ToUpper(s))
	}
	tradeParts := []string{}
	if s := strings.TrimSpace(f.Side); s != "" {
		tradeParts = append(tradeParts, "tf.side=?")
		args = append(args, strings.ToUpper(s))
	}
	if f.MarketCondition > 0 {
		tradeParts = append(tradeParts, "tf.market_condition=?")
		args = append(args, f.MarketCondition)
	}
	if len(tradeParts) > 0 {
		parts = append(parts, "EXISTS (SELECT 1 FROM agent_backtest_trades tf WHERE tf.run_id="+r+".run_id AND "+strings.Join(tradeParts, " AND ")+")")
	}
	if len(parts) == 0 {
		return "", args
	}
	return " AND " + strings.Join(parts, " AND "), args
}
func liveProposalWhere(f Filter) (string, []interface{}) {
	parts := []string{}
	args := []interface{}{}
	if f.StartTime > 0 {
		parts = append(parts, "created_at>=?")
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		parts = append(parts, "created_at<=?")
		args = append(args, f.EndTime)
	}
	if s := strings.TrimSpace(f.Symbol); s != "" {
		parts = append(parts, "symbol=?")
		args = append(args, strings.ToUpper(s))
	}
	if s := strings.TrimSpace(f.Side); s != "" {
		parts = append(parts, "side=?")
		args = append(args, strings.ToUpper(s))
	}
	if f.MarketCondition > 0 {
		parts = append(parts, "market_condition=?")
		args = append(args, f.MarketCondition)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " AND " + strings.Join(parts, " AND "), args
}
func liveManagedWhere(f Filter, managedAlias string) (string, []interface{}) {
	parts := []string{}
	args := []interface{}{}
	if f.StartTime > 0 {
		parts = append(parts, "p.created_at>=?")
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		parts = append(parts, "p.created_at<=?")
		args = append(args, f.EndTime)
	}
	if symbol := strings.TrimSpace(f.Symbol); symbol != "" {
		parts = append(parts, "p.symbol=?")
		args = append(args, strings.ToUpper(symbol))
	}
	if side := strings.TrimSpace(f.Side); side != "" {
		parts = append(parts, "p.side=?")
		args = append(args, strings.ToUpper(side))
	}
	if f.MarketCondition > 0 {
		parts = append(parts, "p.market_condition=?")
		args = append(args, f.MarketCondition)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " AND " + managedAlias + ".source_ref IN (SELECT p.proposal_id FROM agent_trade_proposals p WHERE " + strings.Join(parts, " AND ") + ")", args
}
