package spotexecution

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

// DBGate reads both independently controlled switches on EACH attempt,
// including immediately before the exchange POST; no in-memory cache.
type DBGate struct{ DB orm.Ormer }

func (g DBGate) Enabled(ctx context.Context, symbol string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if g.DB == nil {
		return false, fmt.Errorf("nil Spot quant gate DB")
	}
	var global int64
	if err := g.DB.Raw("SELECT spot_quant_enable FROM config WHERE id=1").QueryRow(&global); err != nil {
		return false, err
	}
	if global != 1 {
		return false, nil
	}
	var symbolFlag int64
	if err := g.DB.Raw("SELECT spot_quant_enabled FROM spot_symbols WHERE symbol=?", symbol).QueryRow(&symbolFlag); err != nil {
		return false, err
	}
	return symbolFlag == 1, nil
}
