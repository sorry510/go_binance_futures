package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"go_binance_futures/feature"
	"go_binance_futures/models"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const marketConditionTaskRetention = 30 * time.Minute

type MarketConditionUpdateTask struct {
	TaskID      string                         `json:"taskId"`
	Status      string                         `json:"status"`
	Progress    int                            `json:"progress"`
	Stage       string                         `json:"stage"`
	Result      *feature.MarketConditionResult `json:"result,omitempty"`
	Error       string                         `json:"error,omitempty"`
	CreatedAt   time.Time                      `json:"createdAt"`
	UpdatedAt   time.Time                      `json:"updatedAt"`
	CompletedAt *time.Time                     `json:"completedAt,omitempty"`
}

var marketConditionTaskStore = struct {
	sync.RWMutex
	tasks        map[string]*MarketConditionUpdateTask
	activeTaskID string
}{
	tasks: make(map[string]*MarketConditionUpdateTask),
}

func startMarketConditionUpdateTask(systemConfig models.Config) MarketConditionUpdateTask {
	now := time.Now().UTC()
	marketConditionTaskStore.Lock()
	cleanupMarketConditionTasksLocked(now)
	if activeTask := marketConditionTaskStore.tasks[marketConditionTaskStore.activeTaskID]; activeTask != nil && isMarketConditionTaskRunning(activeTask.Status) {
		task := cloneMarketConditionUpdateTask(activeTask)
		marketConditionTaskStore.Unlock()
		return task
	}

	taskID := newMarketConditionTaskID()
	task := &MarketConditionUpdateTask{
		TaskID:    taskID,
		Status:    "queued",
		Progress:  0,
		Stage:     "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	marketConditionTaskStore.tasks[taskID] = task
	marketConditionTaskStore.activeTaskID = taskID
	result := cloneMarketConditionUpdateTask(task)
	marketConditionTaskStore.Unlock()

	go runMarketConditionUpdateTask(taskID, systemConfig)
	return result
}

func getMarketConditionUpdateTask(taskID string) (MarketConditionUpdateTask, bool) {
	marketConditionTaskStore.RLock()
	defer marketConditionTaskStore.RUnlock()
	task, exists := marketConditionTaskStore.tasks[taskID]
	if !exists {
		return MarketConditionUpdateTask{}, false
	}
	return cloneMarketConditionUpdateTask(task), true
}

func runMarketConditionUpdateTask(taskID string, systemConfig models.Config) {
	updateMarketConditionTaskProgress(taskID, feature.MarketConditionProgress{Progress: 1, Stage: "starting"})
	result, err := feature.UpdateMarketConditionWithProgress(&systemConfig, func(progress feature.MarketConditionProgress) {
		updateMarketConditionTaskProgress(taskID, progress)
	})
	if err != nil {
		logs.Error("UpdateMarketCondition task %s: %s", taskID, err.Error())
		finishMarketConditionTask(taskID, nil, "行情分析失败，请查看服务日志")
		return
	}
	finishMarketConditionTask(taskID, &result, "")
}

func updateMarketConditionTaskProgress(taskID string, progress feature.MarketConditionProgress) {
	marketConditionTaskStore.Lock()
	defer marketConditionTaskStore.Unlock()
	task := marketConditionTaskStore.tasks[taskID]
	if task == nil || !isMarketConditionTaskRunning(task.Status) {
		return
	}
	task.Status = "running"
	task.Progress = progress.Progress
	task.Stage = progress.Stage
	task.UpdatedAt = time.Now().UTC()
}

func finishMarketConditionTask(taskID string, result *feature.MarketConditionResult, errorMessage string) {
	marketConditionTaskStore.Lock()
	defer marketConditionTaskStore.Unlock()
	task := marketConditionTaskStore.tasks[taskID]
	if task == nil {
		return
	}
	now := time.Now().UTC()
	task.Progress = 100
	task.UpdatedAt = now
	task.CompletedAt = &now
	if errorMessage != "" {
		task.Status = "failed"
		task.Stage = "failed"
		task.Error = errorMessage
	} else {
		task.Status = "succeeded"
		task.Stage = "completed"
		task.Result = result
	}
	if marketConditionTaskStore.activeTaskID == taskID {
		marketConditionTaskStore.activeTaskID = ""
	}
}

func cleanupMarketConditionTasksLocked(now time.Time) {
	for taskID, task := range marketConditionTaskStore.tasks {
		if isMarketConditionTaskRunning(task.Status) || now.Sub(task.UpdatedAt) <= marketConditionTaskRetention {
			continue
		}
		delete(marketConditionTaskStore.tasks, taskID)
	}
}

func isMarketConditionTaskRunning(status string) bool {
	return status == "queued" || status == "running"
}

func cloneMarketConditionUpdateTask(task *MarketConditionUpdateTask) MarketConditionUpdateTask {
	cloned := *task
	if task.Result != nil {
		result := *task.Result
		cloned.Result = &result
	}
	return cloned
}

func newMarketConditionTaskID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err == nil {
		return hex.EncodeToString(value)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
