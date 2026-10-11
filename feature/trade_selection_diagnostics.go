package feature

import (
	"github.com/beego/beego/v2/core/logs"
	"go_binance_futures/models"
	"strings"
	"sync"
	"time"
)

// Emit at most one warning per five minutes. StartTrade normally runs every
// two seconds; a log every tick would bury operational diagnostics.
var emptySmartSelectionWarning struct {
	sync.Mutex
	last   time.Time
	reason string
}

func describeEmptySmartSelection(coins []*models.Symbols) (string, bool) {
	enabled := 0
	enabledUSDT := 0
	for _, sym := range coins {
		if sym == nil || sym.Enable != 1 {
			continue
		}
		enabled++
		if strings.TrimSpace(sym.Type) == "USDT" && strings.HasSuffix(strings.ToUpper(strings.TrimSpace(sym.Symbol)), "USDT") {
			enabledUSDT++
		}
	}
	if enabled == 0 {
		return "smart_local_v2 no candidates: all futures symbols have enable=0; enable intended symbols under Futures > Symbols (global futures switch is not a per-symbol switch)", true
	}
	if enabledUSDT == 0 {
		return "smart_local_v2 no candidates: no enabled USDT futures symbols; check per-symbol enable flags", true
	}
	return "smart_local_v2 no candidates: check market data freshness (30s), minimum 24h quote volume (5M USDT), and recent close cooldown", true
}

func logMainEmptySmartSelection(all []*models.Symbols, selected int) {
	emptySmartSelectionWarning.Lock()
	defer emptySmartSelectionWarning.Unlock()
	if selected > 0 {
		emptySmartSelectionWarning.last = time.Time{}
		emptySmartSelectionWarning.reason = ""
		return
	}
	message, ok := describeEmptySmartSelection(all)
	if !ok {
		return
	}
	now := time.Now()
	if message == emptySmartSelectionWarning.reason && now.Sub(emptySmartSelectionWarning.last) < 5*time.Minute {
		return
	}
	emptySmartSelectionWarning.last = now
	emptySmartSelectionWarning.reason = message
	logs.Warning(message)
}
