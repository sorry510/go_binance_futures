package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/coinmquant"
)

func (p *Processor) configForEntry(symbol, side string) (coinmquant.SymbolView, error) {
	var enabled int64
	err := p.DB.Raw("SELECT coin_m_quant_enable FROM config LIMIT 1").QueryRow(&enabled)
	if err != nil {
		return coinmquant.SymbolView{}, fmt.Errorf("COIN-M global switch unavailable: %w", err)
	}
	if enabled != 1 {
		return coinmquant.SymbolView{}, errors.New("COIN-M global new-entry switch is OFF")
	}
	var id int64
	err = p.DB.Raw("SELECT id FROM delivery_symbols WHERE symbol=? AND contract_type='PERPETUAL'", symbol).QueryRow(&id)
	if err != nil {
		return coinmquant.SymbolView{}, err
	}
	cfg, err := coinmquant.Symbol(p.DB, id)
	if err != nil {
		return cfg, err
	}
	if cfg.Enable != 1 {
		return cfg, errors.New("COIN-M symbol new-entry switch is OFF")
	}
	if cfg.ContractSize <= 0 || cfg.MarginAsset == "" || cfg.ContractType != "PERPETUAL" || cfg.Leverage < 1 {
		return cfg, errors.New("COIN-M contract metadata or leverage missing")
	}
	if cfg.MarginType != "ISOLATED" && cfg.MarginType != "CROSSED" {
		return cfg, errors.New("invalid COIN-M configured margin mode")
	}
	if _, err = positive(cfg.BudgetUSD); err != nil {
		return cfg, errors.New("COIN-M USD margin budget is not positive")
	}
	rules, err := coinmquant.Rules(cfg.Strategy)
	if err != nil {
		return cfg, err
	}
	needed := "long"
	if side == "SHORT" {
		needed = "short"
	}
	hasEntry := false
	for _, r := range rules {
		if r.Enable && r.Type == needed {
			hasEntry = true
			break
		}
	}
	if !hasEntry {
		return cfg, errors.New("COIN-M entry strategy for requested direction is not enabled")
	}
	return cfg, nil
}
func (p *Processor) preflight(ctx context.Context, symbol, side, contracts, typ, price string, spec feature.CoinMPerpetualSpec) error {
	if spec.Symbol != symbol || spec.ContractType != "PERPETUAL" {
		return errors.New("COIN-M fresh perpetual contract metadata differs")
	}
	if typ != "MARKET" && typ != "LIMIT" {
		return errors.New("COIN-M only LIMIT/MARKET orders; protection uses Algo")
	}
	if err := feature.ValidateCoinMOrderFilters(spec, contracts, price, typ == "MARKET"); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.requireReady(ctx); err != nil {
		return err
	}
	return nil
}

// Open never starts automatically: CM-6 owns the scheduling/strategy signal
// integration. This method is a manually invoked service primitive, not a
// public HTTP route, and still checks both persisted entry switches.
func (p *Processor) Open(ctx context.Context, req OpenRequest) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unconfigured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := validClientID(req.ClientOrderID); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if req.PositionSide != "LONG" && req.PositionSide != "SHORT" {
		return models.CoinMManagedOrder{}, errors.New("entry must be LONG or SHORT")
	}
	if req.Type != "MARKET" && req.Type != "LIMIT" {
		return models.CoinMManagedOrder{}, errors.New("invalid entry order type")
	}
	existing, ok, err := localOrder(p.DB, req.ClientOrderID)
	if err != nil {
		return existing, err
	}
	if ok {
		if existing.Intent != "OPEN" || existing.Symbol != req.Symbol || existing.PositionSide != req.PositionSide {
			return existing, errors.New("COIN-M client ID already belongs to a different order")
		}
		// An UNKNOWN previous POST may have executed. Never blind POST again.
		return existing, nil
	}
	if err = p.requireReady(ctx); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	cfg, err := p.configForEntry(req.Symbol, req.PositionSide)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	spec, err := p.API.Spec(ctx, req.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if spec.ContractSize != cfg.ContractSize || spec.MarginAsset != cfg.MarginAsset {
		return models.CoinMManagedOrder{}, errors.New("COIN-M local face value or margin asset differs from fresh exchangeInfo")
	}
	step := spec.MarketStepSize
	if req.Type == "LIMIT" {
		step = spec.StepSize
	}
	contracts, err := contractsForBudget(cfg.BudgetUSD, cfg.Leverage, spec.ContractSize, step)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = p.preflight(ctx, req.Symbol, req.PositionSide, contracts, req.Type, req.Price, spec); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = p.API.TradeSettings(ctx, req.Symbol, req.PositionSide, cfg.Leverage, cfg.MarginType); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	pending, err := p.API.HasOpenOrders(ctx, req.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if pending {
		return models.CoinMManagedOrder{}, errors.New("COIN-M has exchange open orders, refusing duplicate or unmanaged entry")
	}
	remoteAlgos, err := p.API.OpenAlgos(ctx, req.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: cannot check exchange protective orders: %v", ErrUncertain, err)
	}
	if len(remoteAlgos) > 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M exchange has unresolved protective Algo; refusing new entry")
	}
	for _, s := range []string{"LONG", "SHORT"} {
		n, e := p.API.Position(ctx, req.Symbol, s)
		if e != nil {
			return models.CoinMManagedOrder{}, e
		}
		size, e := decimal(n)
		if e != nil || size.Sign() != 0 {
			return models.CoinMManagedOrder{}, errors.New("COIN-M remote account already holds symbol position")
		}
	}
	mark, err := p.API.Mark(ctx, req.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	markN, err := positive(mark)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	avail, err := p.API.Available(ctx, cfg.MarginAsset)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	free, err := decimal(avail)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	budget, err := positive(cfg.BudgetUSD)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	needed := new(big.Rat).Mul(budget, big.NewRat(102, 100))
	if new(big.Rat).Mul(free, markN).Cmp(needed) < 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M settlement asset available balance below USD budget with 2% reserve")
	}
	// Never reopen while any old protection remains unresolved; otherwise
	// a lingering closePosition Algo could affect a future cycle.
	var orphanProtection int64
	if err = p.DB.Raw("SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='PROTECT' AND status NOT IN ('CANCELED','EXPIRED','REJECTED','FILLED')").QueryRow(&orphanProtection); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if orphanProtection > 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M unresolved protective Algo prevents re-entry")
	}
	// A single personal CM cycle at once (unrelated USD-M/Spot unaffected).
	var active int64
	if err = p.DB.Raw("SELECT COUNT(*) FROM coin_m_managed_positions WHERE status NOT IN ('CLOSED','CANCELED')").QueryRow(&active); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if active > 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M already has an active Managed cycle")
	}
	// Check remote prior to first submission. NOT_FOUND is the only acceptable
	// negative response; timeouts and 429 never authorize a POST.
	remote, err := p.API.Query(ctx, req.Symbol, req.ClientOrderID)
	if err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: cannot confirm client ID absent remotely: %v", ErrUncertain, err)
	}
	if remote != nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M client ID exists remotely but is not owned by local ledger")
	}
	now := time.Now().UnixMilli()
	tx, err := p.DB.Begin()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	defer tx.Rollback()
	if err = tx.Raw("SELECT COUNT(*) FROM coin_m_managed_positions WHERE status NOT IN ('CLOSED','CANCELED')").QueryRow(&active); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if active > 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M concurrent active cycle found")
	}
	result, err := tx.Raw("INSERT INTO coin_m_managed_positions (symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)",
		req.Symbol, req.PositionSide, spec.MarginAsset, spec.ContractSize, "0", "0", "0", "0", "PENDING_OPEN", now, now, 0).Exec()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	posID, err := result.LastInsertId()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	side, _ := orderSide(req.PositionSide, "OPEN")
	sent := binance.CoinMTradeOrder{Symbol: req.Symbol, PositionSide: req.PositionSide,
		Side: side, Type: req.Type, Price: req.Price, Contracts: contracts, ClientOrderID: req.ClientOrderID}
	if err = insertOrder(tx, posID, sent, "OPEN"); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	// No network writes until local ownership/idempotency record is durable.
	if err = ctx.Err(); err != nil {
		_ = p.updateAck(err, req.ClientOrderID, nil)
		local, _ := loadOrder(p.DB, req.ClientOrderID)
		return local, fmt.Errorf("%w: local COIN-M intent persisted but context cancelled: %v", ErrUncertain, err)
	}
	ack, postErr := p.API.Create(ctx, sent)
	if postErr != nil {
		_ = p.updateAck(postErr, req.ClientOrderID, nil)
		local, _ := loadOrder(p.DB, req.ClientOrderID)
		return local, fmt.Errorf("%w: POST failed, query original clientID: %v", ErrUncertain, postErr)
	}
	if err = p.updateAck(nil, req.ClientOrderID, ack); err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: %v", ErrUncertain, err)
	}
	return loadOrder(p.DB, req.ClientOrderID)
}

func (p *Processor) Close(ctx context.Context, req CloseRequest) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unconfigured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := validClientID(req.ClientOrderID); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if req.PositionID <= 0 {
		return models.CoinMManagedOrder{}, errors.New("missing owned position ID")
	}
	existing, ok, err := localOrder(p.DB, req.ClientOrderID)
	if err != nil {
		return existing, err
	}
	if ok {
		if existing.Intent != "CLOSE" || existing.PositionID != req.PositionID {
			return existing, errors.New("client ID belongs to another order")
		}
		return existing, nil
	}
	if err = p.requireReady(ctx); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	pos, err := loadPosition(p.DB, req.PositionID)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = validatePosition(pos); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if pos.Status != "OPEN" {
		return models.CoinMManagedOrder{}, errors.New("COIN-M position is not open and verified")
	}
	remaining, err := positive(pos.RemainingContracts)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	var unresolved int64
	err = p.DB.Raw("SELECT COUNT(*) FROM coin_m_managed_orders WHERE position_id=? AND status IN ('PENDING_SUBMIT','UNKNOWN','NEW','PARTIALLY_FILLED','ACTIVE','AWAITING_FILLS')", pos.ID).QueryRow(&unresolved)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if unresolved != 0 {
		return models.CoinMManagedOrder{}, errors.New("unsettled COIN-M order blocks duplicate close")
	}
	spec, err := p.API.Spec(ctx, pos.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if spec.ContractSize <= 0 || !checkFaceValue(pos.ContractSize, spec.ContractSize) || spec.MarginAsset != pos.MarginAsset {
		return models.CoinMManagedOrder{}, errors.New("COIN-M position contract details differ from fresh exchange metadata")
	}
	if err = p.preflight(ctx, pos.Symbol, pos.PositionSide, str(remaining), req.Type, req.Price, spec); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	// Exiting ignores the two new-entry switches. It must not blindly close
	// someone else's manual holding or a different hedge side.
	remote, err := p.API.Position(ctx, pos.Symbol, pos.PositionSide)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	held, err := decimal(remote)
	if err != nil || held.Cmp(remaining) != 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M exchange quantity differs from managed remaining contracts; refusing unmanaged close")
	}
	existingRemote, err := p.API.Query(ctx, pos.Symbol, req.ClientOrderID)
	if err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: cannot check prior close ID: %v", ErrUncertain, err)
	}
	if existingRemote != nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M close client ID exists remotely, not locally owned")
	}
	side, _ := orderSide(pos.PositionSide, "CLOSE")
	sent := binance.CoinMTradeOrder{Symbol: pos.Symbol, PositionSide: pos.PositionSide, Side: side,
		Type: req.Type, Price: req.Price, Contracts: str(remaining), ClientOrderID: req.ClientOrderID}
	tx, err := p.DB.Begin()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	defer tx.Rollback()
	if err = insertOrder(tx, pos.ID, sent, "CLOSE"); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if _, err = tx.Raw("UPDATE coin_m_managed_positions SET status='CLOSING',updated_at=? WHERE id=? AND status='OPEN'",
		time.Now().UnixMilli(), pos.ID).Exec(); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = ctx.Err(); err != nil {
		_ = p.updateAck(err, req.ClientOrderID, nil)
		local, _ := loadOrder(p.DB, req.ClientOrderID)
		return local, fmt.Errorf("%w: local COIN-M intent persisted but context cancelled: %v", ErrUncertain, err)
	}
	ack, postErr := p.API.Create(ctx, sent)
	if postErr != nil {
		_ = p.updateAck(postErr, req.ClientOrderID, nil)
		local, _ := loadOrder(p.DB, req.ClientOrderID)
		return local, fmt.Errorf("%w: close may have been accepted: %v", ErrUncertain, postErr)
	}
	if err = p.updateAck(nil, req.ClientOrderID, ack); err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: %v", ErrUncertain, err)
	}
	return loadOrder(p.DB, req.ClientOrderID)
}
