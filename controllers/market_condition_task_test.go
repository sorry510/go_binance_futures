package controllers

import (
	"go_binance_futures/feature"
	"testing"
	"time"
)

func TestMarketConditionTaskProgressAndCompletion(t *testing.T) {
	resetMarketConditionTaskStoreForTest()
	taskID := "task-progress"
	now := time.Now().UTC()
	marketConditionTaskStore.Lock()
	marketConditionTaskStore.tasks[taskID] = &MarketConditionUpdateTask{
		TaskID:    taskID,
		Status:    "queued",
		Stage:     "queued",
		CreatedAt: now,
		UpdatedAt: now,
	}
	marketConditionTaskStore.activeTaskID = taskID
	marketConditionTaskStore.Unlock()

	updateMarketConditionTaskProgress(taskID, feature.MarketConditionProgress{Progress: 55, Stage: "calling_llm"})
	task, exists := getMarketConditionUpdateTask(taskID)
	if !exists || task.Status != "running" || task.Progress != 55 || task.Stage != "calling_llm" {
		t.Fatalf("unexpected running task: %+v", task)
	}

	result := feature.MarketConditionResult{MarketCondition: 7, Name: "空头分化", Source: "llm", Confidence: 0.8}
	finishMarketConditionTask(taskID, &result, "")
	task, exists = getMarketConditionUpdateTask(taskID)
	if !exists || task.Status != "succeeded" || task.Progress != 100 || task.Result == nil {
		t.Fatalf("unexpected completed task: %+v", task)
	}
	if task.Result.MarketCondition != 7 || task.CompletedAt == nil {
		t.Fatalf("unexpected completed result: %+v", task)
	}

	task.Result.Name = "changed"
	storedTask, _ := getMarketConditionUpdateTask(taskID)
	if storedTask.Result.Name != "空头分化" {
		t.Fatal("returned task must not mutate the stored result")
	}
}

func TestCleanupMarketConditionTasksKeepsRunningAndRecentTasks(t *testing.T) {
	resetMarketConditionTaskStoreForTest()
	now := time.Now().UTC()
	marketConditionTaskStore.Lock()
	marketConditionTaskStore.tasks["expired"] = &MarketConditionUpdateTask{Status: "succeeded", UpdatedAt: now.Add(-marketConditionTaskRetention - time.Minute)}
	marketConditionTaskStore.tasks["recent"] = &MarketConditionUpdateTask{Status: "succeeded", UpdatedAt: now}
	marketConditionTaskStore.tasks["running"] = &MarketConditionUpdateTask{Status: "running", UpdatedAt: now.Add(-marketConditionTaskRetention - time.Minute)}
	cleanupMarketConditionTasksLocked(now)
	_, expiredExists := marketConditionTaskStore.tasks["expired"]
	_, recentExists := marketConditionTaskStore.tasks["recent"]
	_, runningExists := marketConditionTaskStore.tasks["running"]
	marketConditionTaskStore.Unlock()

	if expiredExists || !recentExists || !runningExists {
		t.Fatalf("unexpected cleanup result: expired=%t recent=%t running=%t", expiredExists, recentExists, runningExists)
	}
}

func TestNewMarketConditionTaskIDIsUnique(t *testing.T) {
	first := newMarketConditionTaskID()
	second := newMarketConditionTaskID()
	if first == "" || second == "" || first == second {
		t.Fatalf("unexpected task IDs: %q %q", first, second)
	}
}

func resetMarketConditionTaskStoreForTest() {
	marketConditionTaskStore.Lock()
	defer marketConditionTaskStore.Unlock()
	marketConditionTaskStore.tasks = make(map[string]*MarketConditionUpdateTask)
	marketConditionTaskStore.activeTaskID = ""
}
