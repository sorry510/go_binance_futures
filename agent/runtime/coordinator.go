package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/agent/validator"
)

type runSession struct {
	req            Request
	resumed        bool
	selectedSkill  skill.Skill
	skillRequest   skill.Request
	currentTask    *task.Task
	state          *RunState
	allowedTools   map[string]bool
	toolResults    map[string]any
	finalValidator validator.FinalValidator
}

type coordinator struct {
	runner *DefaultRunner
}

func (coordinator *coordinator) run(ctx context.Context, req Request) (*Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, coordinator.runner.cfg.Timeout)
	defer cancel()
	session, err := coordinator.prepareNew(runCtx, req)
	if err != nil {
		return nil, err
	}
	return coordinator.execute(runCtx, session)
}

func (coordinator *coordinator) resume(ctx context.Context, taskID string) (*Result, error) {
	runCtx, cancel := context.WithTimeout(ctx, coordinator.runner.cfg.Timeout)
	defer cancel()
	session, err := coordinator.prepareResume(runCtx, taskID)
	if err != nil {
		return nil, err
	}
	return coordinator.execute(runCtx, session)
}

func (coordinator *coordinator) validateResume(ctx context.Context, taskID string) error {
	_, err := coordinator.prepareResumeState(ctx, taskID, false)
	return err
}

func (coordinator *coordinator) prepareNew(ctx context.Context, req Request) (*runSession, error) {
	if strings.TrimSpace(req.Skill) == "" {
		return nil, fmt.Errorf("agent skill is required")
	}
	selectedSkill, exists := coordinator.runner.cfg.Skills.Get(req.Skill)
	if !exists {
		return nil, fmt.Errorf("skill %q is not registered", req.Skill)
	}
	mode, err := resolveExecutionMode(selectedSkill)
	if err != nil {
		return nil, err
	}
	snapshot := req.ExecutionSnapshot
	if snapshot == nil {
		frozen := coordinator.runner.FreezeExecution(selectedSkill)
		snapshot = &frozen
	} else {
		copy := *snapshot
		currentIdentity := coordinator.runner.FreezeExecution(selectedSkill)
		if copy.Version.SkillPackageHash == "" {
			copy.Version.SkillPackageHash = currentIdentity.Version.SkillPackageHash
		}
		if copy.Version.ToolCatalogHash == "" {
			copy.Version.ToolCatalogHash = currentIdentity.Version.ToolCatalogHash
		}
		snapshot = &copy
	}
	maxRounds := selectedSkill.MaxRounds()
	if maxRounds <= 0 {
		maxRounds = coordinator.runner.cfg.DefaultMaxRounds
	}
	maxToolCalls, maxTotalTokens := coordinator.runner.budgetFor(selectedSkill.Name())
	now := time.Now().UTC()
	taskID := strings.TrimSpace(req.TaskID)
	if taskID == "" {
		taskID = newTaskID()
	}
	currentTask := &task.Task{
		ID: taskID, Skill: selectedSkill.Name(), ConversationID: strings.TrimSpace(req.ConversationID),
		Status: task.StatusQueued, Stage: "queued", Input: req.Input, MaxRounds: maxRounds,
		Provider: string(coordinator.runner.cfg.Client.Provider()), ExecutionMode: string(mode), CreatedAt: now, UpdatedAt: now,
	}
	currentTask.ApplyVersionMetadata(snapshot.Version)
	state := newRunState(taskID, selectedSkill.Name(), mode, *snapshot, maxRounds, maxToolCalls, maxTotalTokens)
	state.ResumeMetadata = resumableMetadata(req.Metadata)
	state.syncTask(currentTask)
	session := &runSession{req: req, selectedSkill: selectedSkill, currentTask: currentTask, state: state}
	coordinator.runner.record(currentTask, task.StatusQueued, "queued", 0, "agent task created")
	startedAt := time.Now().UTC()
	currentTask.StartedAt = &startedAt
	session.skillRequest = skill.Request{Input: req.Input, ConversationID: req.ConversationID, Metadata: req.Metadata}
	coordinator.initializeSkillRuntime(session)
	return session, nil
}

func (coordinator *coordinator) buildContext(ctx context.Context, session *runSession) error {
	state, currentTask := session.state, session.currentTask
	stepID := state.startStep(StepBuildContext, 1, "build skill input")
	coordinator.runner.recordStep(currentTask, state, stepID, task.StatusRunning, "building_input", 3, "building skill input", "", false)
	messages, err := session.selectedSkill.BuildInput(ctx, session.skillRequest)
	if err != nil {
		state.finishStep(stepID, StepFailed, "", "build_input_failed", err)
		state.syncTask(currentTask)
		return coordinator.runner.fail(currentTask, "build_input_failed", err)
	}
	state.Messages = messages
	state.ContextBlocks = contextengine.InitialMessageBlocks(messages)
	resourceCount := 0
	if provider, ok := session.selectedSkill.(skill.ContextResourceProvider); ok {
		resources, loadErr := coordinator.runner.cfg.ContextEngine.LoadResources(ctx, provider.ContextResources(session.skillRequest), requestedContextResourceIDs(session.req.Metadata))
		if loadErr != nil {
			state.finishStep(stepID, StepFailed, "", "context_resource_failed", loadErr)
			state.syncTask(currentTask)
			return coordinator.runner.fail(currentTask, "build_input_failed", loadErr)
		}
		for _, block := range resources {
			state.appendContextBlock(block)
		}
		resourceCount = len(resources)
	}
	state.finishStep(stepID, StepSucceeded, fmt.Sprintf("%d messages, %d resources", len(messages), resourceCount), "", nil)
	if err := coordinator.runner.saveCheckpoint(ctx, currentTask, state, stepID, 1); err != nil {
		return coordinator.runner.fail(currentTask, "checkpoint_failed", err)
	}
	coordinator.runner.recordStep(currentTask, state, stepID, task.StatusRunning, "input_built", 4, "skill input built", "", true)
	return nil
}

func requestedContextResourceIDs(metadata map[string]any) []string {
	if metadata == nil {
		return nil
	}
	value, exists := metadata["context_resource_ids"]
	if !exists {
		return nil
	}
	result := []string{}
	switch typed := value.(type) {
	case []string:
		result = append(result, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	case string:
		result = strings.Split(typed, ",")
	}
	clean := result[:0]
	for _, item := range result {
		if item = strings.TrimSpace(item); item != "" {
			clean = append(clean, item)
		}
	}
	return clean
}

func (coordinator *coordinator) prepareResume(ctx context.Context, taskID string) (*runSession, error) {
	return coordinator.prepareResumeState(ctx, taskID, true)
}

func (coordinator *coordinator) prepareResumeState(ctx context.Context, taskID string, mutate bool) (*runSession, error) {
	item, err := coordinator.runner.cfg.Tasks.Get(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return nil, err
	}
	if !isResumableTask(item) {
		return nil, fmt.Errorf("task %q is not in a resumable state (%s/%s)", item.ID, item.Status, item.Stage)
	}
	state, err := coordinator.runner.loadCheckpoint(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	selectedSkill, exists := coordinator.runner.cfg.Skills.Get(item.Skill)
	if !exists {
		return nil, fmt.Errorf("skill %q is not registered", item.Skill)
	}
	if err := validateResumeIdentity(item, state, coordinator.runner.FreezeExecution(selectedSkill)); err != nil {
		return nil, err
	}
	toolResults, err := coordinator.runner.restoreToolResults(state)
	if err != nil {
		return nil, err
	}
	req := Request{TaskID: item.ID, Skill: item.Skill, Input: item.Input, ConversationID: item.ConversationID, Metadata: restoreResumeMetadata(state.ResumeMetadata), ExecutionSnapshot: &state.Snapshot}
	session := &runSession{
		req: req, resumed: true, selectedSkill: selectedSkill,
		skillRequest: skill.Request{Input: item.Input, ConversationID: item.ConversationID},
		currentTask:  item, state: state, toolResults: toolResults,
	}
	coordinator.initializeSkillRuntime(session)
	if !mutate {
		return session, nil
	}
	item.ResumeCount++
	item.Error = ""
	item.CompletedAt = nil
	state.syncTask(item)
	coordinator.runner.record(item, task.StatusRunning, "resuming", item.Progress, "resuming agent from safe checkpoint")
	return session, nil
}

func (coordinator *coordinator) initializeSkillRuntime(session *runSession) {
	session.allowedTools = make(map[string]bool, len(session.selectedSkill.Tools()))
	for _, name := range session.selectedSkill.Tools() {
		if name = strings.TrimSpace(name); name != "" {
			session.allowedTools[name] = true
		}
	}
	if len(session.state.RequiredTools) == 0 {
		if provider, ok := session.selectedSkill.(skill.ToolRequirementProvider); ok {
			for _, name := range provider.RequiredTools(session.skillRequest) {
				if name = strings.TrimSpace(name); name != "" {
					session.state.RequiredTools[name] = true
				}
			}
		}
	}
	if session.toolResults == nil {
		session.toolResults = map[string]any{}
	}
	session.finalValidator = session.selectedSkill.Validator()
	if provider, ok := session.selectedSkill.(skill.RequestValidatorProvider); ok {
		session.finalValidator = provider.ValidatorFor(session.skillRequest)
	}
}

func (coordinator *coordinator) execute(ctx context.Context, session *runSession) (*Result, error) {
	runStarted := time.Now()
	coordinator.runner.observe(Observation{
		Type: "task_started", TaskID: session.currentTask.ID, ConversationID: session.currentTask.ConversationID,
		Skill: session.currentTask.Skill, Provider: session.currentTask.Provider, Status: string(session.currentTask.Status),
	})
	defer func() {
		errorType := ""
		if session.currentTask.Status != task.StatusSucceeded {
			errorType = session.currentTask.Stage
		}
		coordinator.runner.observe(Observation{
			Type: "task_finished", TaskID: session.currentTask.ID, ConversationID: session.currentTask.ConversationID,
			Skill: session.currentTask.Skill, Provider: session.currentTask.Provider, Model: session.currentTask.Model,
			Status: string(session.currentTask.Status), ErrorType: errorType, Error: session.currentTask.Error,
			Round: session.currentTask.Round, DurationMs: elapsedMilliseconds(runStarted), Usage: session.currentTask.Usage,
		})
	}()
	if !session.resumed {
		if err := coordinator.buildContext(ctx, session); err != nil {
			return nil, err
		}
	}
	switch session.state.Mode {
	case ExecutionModeReact:
		return (&reactExecutor{runner: coordinator.runner}).execute(ctx, session)
	case ExecutionModeWorkflow:
		return nil, coordinator.runner.fail(session.currentTask, "execution_mode_unavailable", fmt.Errorf("workflow execution mode is not configured in V2-1"))
	case ExecutionModePlanExecute:
		return (&planExecuteExecutor{runner: coordinator.runner}).execute(ctx, session)
	default:
		return nil, coordinator.runner.fail(session.currentTask, "execution_mode_invalid", fmt.Errorf("unsupported execution mode %q", session.state.Mode))
	}
}

func (runner *DefaultRunner) budgetFor(skillName string) (int, int) {
	maxToolCalls, maxTotalTokens := runner.cfg.MaxToolCalls, runner.cfg.MaxTotalTokens
	if runner.cfg.BudgetProvider != nil {
		budget := runner.cfg.BudgetProvider(skillName)
		if budget.MaxToolCalls > 0 {
			maxToolCalls = budget.MaxToolCalls
		}
		if budget.MaxTotalTokens > 0 {
			maxTotalTokens = budget.MaxTotalTokens
		}
	}
	return maxToolCalls, maxTotalTokens
}

func resolveExecutionMode(selectedSkill skill.Skill) (ExecutionMode, error) {
	mode := ExecutionModeReact
	if provider, ok := selectedSkill.(skill.ExecutionModeProvider); ok {
		if value := strings.TrimSpace(provider.ExecutionMode()); value != "" {
			mode = ExecutionMode(value)
		}
	}
	switch mode {
	case ExecutionModeReact, ExecutionModeWorkflow, ExecutionModePlanExecute:
		return mode, nil
	default:
		return "", fmt.Errorf("skill %q declares unsupported execution mode %q", selectedSkill.Name(), mode)
	}
}

func isResumableTask(item *task.Task) bool {
	if item == nil || strings.TrimSpace(item.CheckpointJSON) == "" {
		return false
	}
	if item.Status == task.StatusInterrupted || item.Status == task.StatusCancelled {
		return true
	}
	return item.Status == task.StatusFailed && item.Stage == "timeout"
}

func validateResumeIdentity(item *task.Task, state *RunState, current ExecutionSnapshot) error {
	if state.TaskID != item.ID || state.Skill != item.Skill {
		return fmt.Errorf("runtime checkpoint identity does not match task")
	}
	if item.RuntimeVersion != CurrentVersion || state.Snapshot.Version.RuntimeVersion != CurrentVersion {
		return fmt.Errorf("task runtime version %q cannot be resumed by runtime %q", item.RuntimeVersion, CurrentVersion)
	}
	if item.ModelConfigID > 0 && current.Version.ModelConfigID != item.ModelConfigID {
		return fmt.Errorf("resume requires frozen model config id %d, got %d", item.ModelConfigID, current.Version.ModelConfigID)
	}
	stored := state.Snapshot.Version
	actual := current.Version
	if stored.SkillVersion != actual.SkillVersion || stored.InputContractVersion != actual.InputContractVersion ||
		stored.OutputContractVersion != actual.OutputContractVersion || stored.SkillSource != actual.SkillSource ||
		stored.SkillSourceVersion != actual.SkillSourceVersion || stored.SkillPackageHash != actual.SkillPackageHash ||
		stored.ToolCatalogHash != actual.ToolCatalogHash {
		return fmt.Errorf("skill implementation changed since checkpoint; refusing unsafe resume")
	}
	return nil
}

func resumableMetadata(metadata map[string]any) map[string]string {
	result := map[string]string{}
	if metadata == nil {
		return result
	}
	// Only persist Runtime-owned scalar metadata that is required to preserve
	// completion semantics across process restarts. Arbitrary Skill metadata
	// may be large, sensitive or non-serializable and is intentionally excluded.
	if value, ok := metadata["scheduler_job"].(string); ok && strings.TrimSpace(value) != "" {
		result["scheduler_job"] = strings.TrimSpace(value)
	}
	return result
}

func restoreResumeMetadata(metadata map[string]string) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	result := make(map[string]any, len(metadata))
	for key, value := range metadata {
		result[key] = value
	}
	return result
}
