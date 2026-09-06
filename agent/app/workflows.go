package app

import (
	workflowservice "go_binance_futures/service/workflow"
	"sync"
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
		service := workflowservice.Service{Manager: manager, Store: workflowservice.Store{}}
		defaultWorkflow = &service
	})
	return defaultWorkflow, defaultWorkflowErr
}
