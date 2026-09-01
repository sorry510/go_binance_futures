package replay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go_binance_futures/llm"
)

type scriptedClient struct {
	mu            sync.Mutex
	steps         []LLMStep
	index         int
	modelConfigID int64
}

func (client *scriptedClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *scriptedClient) ConfigID() int64        { return client.modelConfigID }

func (client *scriptedClient) Generate(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	client.mu.Lock()
	if client.index >= len(client.steps) {
		client.mu.Unlock()
		return nil, fmt.Errorf("replay has no LLM step %d", client.index+1)
	}
	step := client.steps[client.index]
	client.index++
	client.mu.Unlock()
	if step.DelayMs > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(step.DelayMs) * time.Millisecond):
		}
	}
	if step.Error != "" {
		return nil, fmt.Errorf("%s", step.Error)
	}
	return &llm.Response{Model: step.Model, Content: step.Content, FinishReason: step.FinishReason, Usage: step.Usage}, nil
}
