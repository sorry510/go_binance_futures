package binance

import (
	"context"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/futures"
	"golang.org/x/sync/singleflight"
)

const (
	accountPositionSnapshotTTL   = 2 * time.Second
	accountOpenOrdersSnapshotTTL = 5 * time.Second
)

type positionSnapshotEntry struct {
	rows      []*futures.PositionRisk
	expiresAt time.Time
	valid     bool
}

type openOrdersSnapshotEntry struct {
	rows      []*futures.Order
	expiresAt time.Time
	valid     bool
}

var (
	accountReadCacheMu         sync.RWMutex
	accountReadCacheGeneration uint64
	positionSnapshot           positionSnapshotEntry
	openOrdersSnapshot         openOrdersSnapshotEntry
	positionSnapshotSF         singleflight.Group
	openOrdersSnapshotSF       singleflight.Group
)

func clonePositionRisks(rows []*futures.PositionRisk) []*futures.PositionRisk {
	if rows == nil {
		return nil
	}
	out := make([]*futures.PositionRisk, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			out = append(out, nil)
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out
}

func cloneFuturesOrders(rows []*futures.Order) []*futures.Order {
	if rows == nil {
		return nil
	}
	out := make([]*futures.Order, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			out = append(out, nil)
			continue
		}
		copied := *row
		out = append(out, &copied)
	}
	return out
}

func loadPositionSnapshot() ([]*futures.PositionRisk, bool) {
	accountReadCacheMu.RLock()
	entry := positionSnapshot
	accountReadCacheMu.RUnlock()
	if !entry.valid || !entry.expiresAt.After(time.Now()) {
		return nil, false
	}
	return clonePositionRisks(entry.rows), true
}

func accountReadGeneration() uint64 {
	accountReadCacheMu.RLock()
	defer accountReadCacheMu.RUnlock()
	return accountReadCacheGeneration
}

func storePositionSnapshot(rows []*futures.PositionRisk, generation uint64) {
	accountReadCacheMu.Lock()
	defer accountReadCacheMu.Unlock()
	if generation != accountReadCacheGeneration {
		return
	}
	positionSnapshot = positionSnapshotEntry{
		rows:      clonePositionRisks(rows),
		expiresAt: time.Now().Add(accountPositionSnapshotTTL),
		valid:     true,
	}
}

func loadOpenOrdersSnapshot() ([]*futures.Order, bool) {
	accountReadCacheMu.RLock()
	entry := openOrdersSnapshot
	accountReadCacheMu.RUnlock()
	if !entry.valid || !entry.expiresAt.After(time.Now()) {
		return nil, false
	}
	return cloneFuturesOrders(entry.rows), true
}

func storeOpenOrdersSnapshot(rows []*futures.Order, generation uint64) {
	accountReadCacheMu.Lock()
	defer accountReadCacheMu.Unlock()
	if generation != accountReadCacheGeneration {
		return
	}
	openOrdersSnapshot = openOrdersSnapshotEntry{
		rows:      cloneFuturesOrders(rows),
		expiresAt: time.Now().Add(accountOpenOrdersSnapshotTTL),
		valid:     true,
	}
}

func InvalidateAccountReadCache() {
	accountReadCacheMu.Lock()
	accountReadCacheGeneration++
	positionSnapshot = positionSnapshotEntry{}
	openOrdersSnapshot = openOrdersSnapshotEntry{}
	accountReadCacheMu.Unlock()
	// Future callers must not join a request that started before the invalidation.
	// Generation checks below also prevent such an older request from writing its
	// result back after the mutation/fresh refresh.
	positionSnapshotSF.Forget("all_positions")
	openOrdersSnapshotSF.Forget("all_open_orders")
}

func getAllPositionsCached(ctx context.Context, loader func(context.Context) ([]*futures.PositionRisk, error)) ([]*futures.PositionRisk, error) {
	source := binanceapiusage.SourceFromContext(ctx)
	if rows, ok := loadPositionSnapshot(); ok {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return rows, nil
	}

	value, err, shared := positionSnapshotSF.Do("all_positions", func() (interface{}, error) {
		if rows, ok := loadPositionSnapshot(); ok {
			return rows, nil
		}
		generation := accountReadGeneration()
		rows, loadErr := loader(ctx)
		if loadErr == nil {
			storePositionSnapshot(rows, generation)
		}
		return rows, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	rows, _ := value.([]*futures.PositionRisk)
	return clonePositionRisks(rows), nil
}

func getAllOpenOrdersCached(ctx context.Context, loader func(context.Context) ([]*futures.Order, error)) ([]*futures.Order, error) {
	source := binanceapiusage.SourceFromContext(ctx)
	if rows, ok := loadOpenOrdersSnapshot(); ok {
		binanceapiusage.RecordOptimization(source, "cache_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return rows, nil
	}

	value, err, shared := openOrdersSnapshotSF.Do("all_open_orders", func() (interface{}, error) {
		if rows, ok := loadOpenOrdersSnapshot(); ok {
			return rows, nil
		}
		generation := accountReadGeneration()
		rows, loadErr := loader(ctx)
		if loadErr == nil {
			storeOpenOrdersSnapshot(rows, generation)
		}
		return rows, loadErr
	})
	if shared {
		binanceapiusage.RecordOptimization(source, "coalesced", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
	}
	if err != nil {
		return nil, err
	}
	rows, _ := value.([]*futures.Order)
	return cloneFuturesOrders(rows), nil
}

func isAllAccountPositionRequest(params PositionParams) bool {
	return strings.TrimSpace(params.Symbol) == ""
}

func getAllPositionsFresh(ctx context.Context, loader func(context.Context) ([]*futures.PositionRisk, error)) ([]*futures.PositionRisk, error) {
	InvalidateAccountReadCache()
	generation := accountReadGeneration()
	rows, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	storePositionSnapshot(rows, generation)
	return clonePositionRisks(rows), nil
}

func getAllOpenOrdersFresh(ctx context.Context, loader func(context.Context) ([]*futures.Order, error)) ([]*futures.Order, error) {
	InvalidateAccountReadCache()
	generation := accountReadGeneration()
	rows, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	storeOpenOrdersSnapshot(rows, generation)
	return cloneFuturesOrders(rows), nil
}
