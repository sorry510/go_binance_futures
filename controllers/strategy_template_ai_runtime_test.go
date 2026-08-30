package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	conversationstore "go_binance_futures/agent/conversation"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

type strategyBuilderFakeClient struct {
	mu        sync.Mutex
	responses []string
	requests  []llm.Request
}

func (*strategyBuilderFakeClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *strategyBuilderFakeClient) Generate(_ context.Context, request llm.Request) (*llm.Response, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.requests = append(client.requests, request)
	index := len(client.requests) - 1
	if index >= len(client.responses) {
		return nil, fmt.Errorf("unexpected LLM call %d", index+1)
	}
	return &llm.Response{Model: "fake", Content: client.responses[index]}, nil
}

func TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation(t *testing.T) {
	resetStrategyTemplateAITaskStoreForRuntimeTest()
	client := &strategyBuilderFakeClient{responses: []string{
		validStrategyBuilderEnvelope("runtime-first"),
		validStrategyBuilderEnvelope("runtime-second"),
	}}
	original := newStrategyBuilderLLMClient
	originalAdmission := admitStrategyBuilderSkill
	originalBudget := strategyBuilderBudgetProvider
	newStrategyBuilderLLMClient = func() (llm.Client, error) { return client, nil }
	admitStrategyBuilderSkill = func(string) error { return nil }
	strategyBuilderBudgetProvider = func(string) agentruntime.Budget {
		return agentruntime.Budget{MaxToolCalls: maxStrategyTemplateAIRounds, MaxTotalTokens: 120000}
	}
	defer func() {
		newStrategyBuilderLLMClient = original
		admitStrategyBuilderSkill = originalAdmission
		strategyBuilderBudgetProvider = originalBudget
	}()

	first, err := startStrategyTemplateAITask(strategyTemplateAIGenerationRequest{Prompt: "生成一个简单策略"})
	if err != nil {
		t.Fatal(err)
	}
	firstDone := waitStrategyTemplateAITask(t, first.TaskID)
	if firstDone.Status != "succeeded" || !strings.Contains(firstDone.JSON, "runtime-first") {
		t.Fatalf("unexpected first task: %+v", firstDone)
	}

	second, err := startStrategyTemplateAITask(strategyTemplateAIGenerationRequest{Prompt: "基于上一版修改名称", ConversationID: first.ConversationID})
	if err != nil {
		t.Fatal(err)
	}
	if second.ConversationID != first.ConversationID {
		t.Fatalf("conversation id changed: %s -> %s", first.ConversationID, second.ConversationID)
	}
	if second.TaskID == first.TaskID {
		t.Fatalf("each conversation turn must create a new task: %s", second.TaskID)
	}
	secondDone := waitStrategyTemplateAITask(t, second.TaskID)
	if secondDone.Status != "succeeded" || !strings.Contains(secondDone.JSON, "runtime-second") {
		t.Fatalf("unexpected second task: %+v", secondDone)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.requests) != 2 {
		t.Fatalf("LLM calls = %d, want 2", len(client.requests))
	}
	secondMessages := client.requests[1].Messages
	joined := ""
	for _, message := range secondMessages {
		joined += "\n" + message.Content
	}
	if !strings.Contains(joined, "runtime-first") || !strings.Contains(joined, "基于上一版修改名称") {
		t.Fatalf("conversation history missing from second run: %s", joined)
	}
}

func waitStrategyTemplateAITask(t *testing.T, taskID string) strategyTemplateAIGenerationTask {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		item, ok := getStrategyTemplateAITask(taskID)
		if ok && item.Status != "queued" && item.Status != "running" {
			return item
		}
		time.Sleep(10 * time.Millisecond)
	}
	item, _ := getStrategyTemplateAITask(taskID)
	t.Fatalf("task did not finish: %+v", item)
	return item
}

func resetStrategyTemplateAITaskStoreForRuntimeTest() {
	strategyTemplateAITaskStore.Lock()
	strategyTemplateAITaskStore.tasks = make(map[string]*strategyTemplateAIGenerationTask)
	strategyTemplateAITaskStore.Unlock()
	strategyTemplateConversationStore = conversationstore.NewMemoryStore()
	strategyTemplatePersistentTaskStore = task.NewMemoryStore()
}

func validStrategyBuilderEnvelope(name string) string {
	return fmt.Sprintf(`{"action":"final","summary":"done","result":{"name":%q,"technology":{"ma":[],"ema":[],"macd":[],"adx":[],"mfi":[],"obv":[],"cci":[],"roc":[],"kdj":[],"rsi":[],"kc":[],"boll":[],"donchian":[],"atr":[],"supertrend":[]},"strategy":[{"name":"long","type":"long","code":"NowPrice > 0","fullScreen":false,"enable":true}]}}`, name)
}

func TestStrategyTemplateValidationAcceptsMarketConditionStringComparison(t *testing.T) {
	candidate := []byte(`{"name":"market-condition","technology":{"ma":[],"ema":[],"macd":[],"adx":[],"mfi":[],"obv":[],"cci":[],"roc":[],"kdj":[],"rsi":[],"kc":[],"boll":[],"donchian":[],"atr":[],"supertrend":[]},"strategy":[{"name":"regime","type":"long","code":"MarketCondition == \"1\" && NowPrice > 0","fullScreen":false,"enable":true}]}`)
	if err := validateGeneratedStrategyTemplateJSON(candidate); err != nil {
		t.Fatalf("MarketCondition string comparison should compile: %v", err)
	}
}

func TestStrategyTemplateValidationAcceptsMultilineLetExpression(t *testing.T) {
	code := "let warmup_ok = (NowTime - SystemStartTime) / 1000 > 600;\nlet strong_bull = MarketCondition == \"1\" && BasicTrend >= 0.5;\nwarmup_ok && strong_bull"
	technology := map[string]any{"ma": []any{}, "ema": []any{}, "macd": []any{}, "adx": []any{}, "mfi": []any{}, "obv": []any{}, "cci": []any{}, "roc": []any{}, "kdj": []any{}, "rsi": []any{}, "kc": []any{}, "boll": []any{}, "donchian": []any{}, "atr": []any{}, "supertrend": []any{}}
	candidate, err := json.Marshal(map[string]any{"name": "readable-let", "technology": technology, "strategy": []map[string]any{{"name": "long", "type": "long", "code": code, "fullScreen": false, "enable": true}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGeneratedStrategyTemplateJSON(candidate); err != nil {
		t.Fatalf("multiline let expression should compile and run: %v", err)
	}
}
