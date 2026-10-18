package app

import (
	"sync"

	"go_binance_futures/agent/task"
	"go_binance_futures/service/opportunity"
	workflowservice "go_binance_futures/service/workflow"
)

var defaultWorkflowOnce sync.Once
var defaultWorkflow *workflowservice.Service
var defaultWorkflowErr error

func DefaultWorkflowService() (*workflowservice.Service, error) {
	defaultWorkflowOnce.Do(func() {
		manager, err := DefaultManager()
		if err != nil {
			defaultWorkflowErr = err
			return
		}
		service := workflowservice.Service{
			Manager: manager,
			Store:   workflowservice.Store{},
			OnMarketScanComplete: func(item *task.Task) {
				opportunity.DefaultEmitManualMarketScanTask(item)
			},
		}
		defaultWorkflow = &service
	})
	return defaultWorkflow, defaultWorkflowErr
}
