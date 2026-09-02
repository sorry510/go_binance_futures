package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go_binance_futures/agent/task"
)

type planExecuteExecutor struct {
	runner *DefaultRunner
}

func (executor *planExecuteExecutor) execute(ctx context.Context, session *runSession) (*Result, error) {
	state, item := session.state, session.currentTask
	plannerStepID := ""
	if state.Plan == nil {
		if executor.runner.cfg.Planner == nil {
			return nil, executor.runner.fail(item, "execution_mode_unavailable", fmt.Errorf("plan_execute requires a configured Planner"))
		}
		plannerStepID = state.startStep(StepPlan, 1, "build constrained execution plan")
		executor.runner.recordStep(item, state, plannerStepID, task.StatusRunning, "planning", 5, "building constrained execution plan", "", false)
		allowedTools := make([]string, 0, len(session.allowedTools))
		for name := range session.allowedTools {
			allowedTools = append(allowedTools, name)
		}
		sort.Strings(allowedTools)
		plan, err := executor.runner.cfg.Planner.Plan(ctx, PlanRequest{
			TaskID: item.ID, Skill: item.Skill, Input: item.Input,
			AllowedTools: allowedTools, MaxToolCalls: state.MaxToolCalls - state.ToolCalls,
		})
		if err != nil {
			state.finishStep(plannerStepID, StepFailed, "", classifyError(err), err)
			state.syncTask(item)
			if ctx.Err() != nil {
				return nil, executor.runner.finishContextError(item, ctx.Err())
			}
			return nil, executor.runner.fail(item, "planning_failed", err)
		}
		if err := validatePlan(plan, session.allowedTools, state.MaxToolCalls-state.ToolCalls); err != nil {
			state.finishStep(plannerStepID, StepFailed, "", "invalid_plan", err)
			state.syncTask(item)
			return nil, executor.runner.fail(item, "invalid_plan", err)
		}
		state.Plan = &plan
		state.PlanCursor = 0
		state.finishStep(plannerStepID, StepSucceeded, fmt.Sprintf("%d planned tool steps", len(plan.Steps)), "", nil)
		if err := executor.runner.saveCheckpoint(ctx, item, state, plannerStepID, 1); err != nil {
			return nil, executor.runner.fail(item, "checkpoint_failed", err)
		}
		executor.runner.recordStep(item, state, plannerStepID, task.StatusRunning, "plan_ready", 7, "execution plan ready", "", true)
	} else {
		for _, step := range state.Steps {
			if step.Type == StepPlan {
				plannerStepID = step.StepID
				break
			}
		}
	}

	react := &reactExecutor{runner: executor.runner}
	messages := state.Messages
	for index := state.PlanCursor; index < len(state.Plan.Steps); index++ {
		planned := state.Plan.Steps[index]
		arguments := planned.Arguments
		if len(arguments) == 0 || string(arguments) == "null" {
			arguments = json.RawMessage(`{}`)
		}
		progress := 8
		if len(state.Plan.Steps) > 0 {
			progress += index * 20 / len(state.Plan.Steps)
		}
		decision := decision{Action: "tool", Tool: planned.Tool, Arguments: arguments}
		if err := react.executeTool(ctx, session, &messages, decision, progress, 0, plannerStepID, func() {
			state.PlanCursor = index + 1
		}); err != nil {
			return nil, err
		}
		state.PlanCursor = index + 1
		state.Messages = messages
	}
	state.NextRound = 1
	state.Messages = messages
	return react.execute(ctx, session)
}

func validatePlan(plan Plan, allowedTools map[string]bool, remainingToolCalls int) error {
	if remainingToolCalls < 0 {
		remainingToolCalls = 0
	}
	if len(plan.Steps) > remainingToolCalls {
		return fmt.Errorf("plan requires %d tool calls but budget allows %d", len(plan.Steps), remainingToolCalls)
	}
	seen := make(map[string]bool, len(plan.Steps))
	for index := range plan.Steps {
		step := &plan.Steps[index]
		step.StepID = strings.TrimSpace(step.StepID)
		if step.StepID == "" {
			step.StepID = fmt.Sprintf("plan-%03d", index+1)
		}
		if seen[step.StepID] {
			return fmt.Errorf("duplicate planned step id %q", step.StepID)
		}
		seen[step.StepID] = true
		if step.Type == "" {
			step.Type = StepTool
		}
		if step.Type != StepTool {
			return fmt.Errorf("planned step %q type %q is not supported in V2-1", step.StepID, step.Type)
		}
		step.Tool = strings.TrimSpace(step.Tool)
		if step.Tool == "" {
			return fmt.Errorf("planned step %q requires tool", step.StepID)
		}
		if !allowedTools[step.Tool] {
			return fmt.Errorf("planned step %q uses tool %q outside the skill allowlist", step.StepID, step.Tool)
		}
		for _, dependency := range step.DependsOn {
			dependency = strings.TrimSpace(dependency)
			if dependency == "" {
				continue
			}
			if dependency == step.StepID {
				return fmt.Errorf("planned step %q cannot depend on itself", step.StepID)
			}
			if !seen[dependency] {
				return fmt.Errorf("planned step %q has unresolved dependency %q", step.StepID, dependency)
			}
		}
	}
	return nil
}
