package controllers

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var controllerTestAliasSequence atomic.Uint64
var controllerSymbolModelsOnce sync.Once
var controllerStrategyModelsOnce sync.Once

func newControllerTestAlias(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, controllerTestAliasSequence.Add(1))
}
