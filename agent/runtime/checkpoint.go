package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go_binance_futures/agent/permission"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/tools"
)

func (runner *DefaultRunner) saveCheckpoint(ctx context.Context, item *task.Task, state *RunState, stepID string, nextRound int) error {
	if state == nil || item == nil || !state.ResumeSafe {
		return nil
	}
	if nextRound > 0 {
		state.NextRound = nextRound
	}
	state.markCheckpoint(stepID)
	state.syncTask(item)
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode runtime checkpoint: %w", err)
	}
	item.CheckpointJSON = string(encoded)
	checkpointStore, ok := runner.cfg.Tasks.(task.CheckpointStore)
	if !ok {
		return nil
	}
	if err := checkpointStore.SaveCheckpoint(ctx, item.ID, item.CheckpointJSON); err != nil {
		return fmt.Errorf("save runtime checkpoint: %w", err)
	}
	return runner.cfg.Tasks.Save(context.Background(), item)
}

func (runner *DefaultRunner) clearCheckpoint(ctx context.Context, item *task.Task, state *RunState) error {
	if state != nil {
		state.ResumeSafe = false
		state.syncTask(item)
	}
	if item != nil {
		item.CheckpointJSON = ""
	}
	checkpointStore, ok := runner.cfg.Tasks.(task.CheckpointStore)
	if ok && item != nil {
		if err := checkpointStore.ClearCheckpoint(ctx, item.ID); err != nil {
			return err
		}
	}
	if item != nil {
		return runner.cfg.Tasks.Save(context.Background(), item)
	}
	return nil
}

func (runner *DefaultRunner) loadCheckpoint(ctx context.Context, taskID string) (*RunState, error) {
	checkpointStore, ok := runner.cfg.Tasks.(task.CheckpointStore)
	if !ok {
		return nil, fmt.Errorf("task store does not support runtime checkpoints")
	}
	raw, err := checkpointStore.LoadCheckpoint(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return nil, err
	}
	var state RunState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, fmt.Errorf("decode runtime checkpoint: %w", err)
	}
	if state.Version != runStateVersion || !state.ResumeSafe {
		return nil, fmt.Errorf("task checkpoint is not safely resumable")
	}
	if state.RequiredTools == nil {
		state.RequiredTools = map[string]bool{}
	}
	if state.SuccessfulTools == nil {
		state.SuccessfulTools = map[string]bool{}
	}
	if state.ToolResults == nil {
		state.ToolResults = map[string]json.RawMessage{}
	}
	return &state, nil
}

func (runner *DefaultRunner) restoreToolResults(state *RunState) (map[string]any, error) {
	results := make(map[string]any, len(state.ToolResults))
	for name, raw := range state.ToolResults {
		selectedTool, ok := runner.cfg.Tools.Get(name)
		if !ok {
			return nil, fmt.Errorf("checkpoint tool %q is no longer registered", name)
		}
		codec, ok := selectedTool.(tools.CheckpointCodec)
		if !ok {
			return nil, fmt.Errorf("checkpoint tool %q cannot restore its result", name)
		}
		value, err := codec.RestoreCheckpoint(raw)
		if err != nil {
			return nil, fmt.Errorf("restore checkpoint tool %q: %w", name, err)
		}
		results[name] = value
	}
	return results, nil
}

func toolCreatesSafeCheckpoint(selectedTool tools.Tool) bool {
	metadata := selectedTool.Metadata()
	return metadata.Idempotent && selectedTool.Risk() == permission.RiskRead
}
