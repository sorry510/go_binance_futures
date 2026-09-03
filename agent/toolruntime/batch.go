package toolruntime

import (
	"context"
	"sync"

	"go_binance_futures/agent/permission"
)

type BatchRequest struct {
	Request   ExecuteRequest
	DependsOn []string
}

type BatchResult struct {
	Index  int
	Result ExecuteResult
	Err    error
}

// ExecuteBatch executes a batch in parallel only when every item has no
// dependency and the registered system descriptor is read + idempotent.
// Otherwise the complete batch is executed in input order.
func (runtime *Runtime) ExecuteBatch(ctx context.Context, requests []BatchRequest, maxParallel int) []BatchResult {
	results := make([]BatchResult, len(requests))
	if len(requests) == 0 {
		return results
	}
	parallel := len(requests) > 1
	for _, item := range requests {
		descriptor, ok := runtime.Descriptor(item.Request.ToolName)
		if len(item.DependsOn) > 0 || !ok || descriptor.Risk != permission.RiskRead || !descriptor.Idempotent {
			parallel = false
			break
		}
	}
	if !parallel {
		for index, item := range requests {
			result, err := runtime.Execute(ctx, item.Request)
			results[index] = BatchResult{Index: index, Result: result, Err: err}
		}
		return results
	}
	if maxParallel <= 0 || maxParallel > len(requests) {
		maxParallel = len(requests)
	}
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	for index, item := range requests {
		wg.Add(1)
		go func(index int, item BatchRequest) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[index] = BatchResult{Index: index, Err: ctx.Err()}
				return
			}
			defer func() { <-sem }()
			result, err := runtime.Execute(ctx, item.Request)
			results[index] = BatchResult{Index: index, Result: result, Err: err}
		}(index, item)
	}
	wg.Wait()
	return results
}
