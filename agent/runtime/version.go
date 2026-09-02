package runtime

import (
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

const CurrentVersion = "2.1.0"

type ExecutionSnapshot struct {
	SystemPrompt string               `json:"system_prompt"`
	Version      task.VersionMetadata `json:"version"`
}

func FreezeExecution(selectedSkill skill.Skill, client llm.Client) ExecutionSnapshot {
	prompt := selectedSkill.SystemPrompt()
	info := skill.ResolveVersionInfo(selectedSkill, prompt)
	return ExecutionSnapshot{
		SystemPrompt: prompt,
		Version: task.VersionMetadata{
			RuntimeVersion:        CurrentVersion,
			SkillVersion:          info.SkillVersion,
			PromptVersion:         info.PromptVersion,
			PromptHash:            info.PromptHash,
			ModelConfigID:         llm.ConfigID(client),
			InputContractVersion:  info.InputContractVersion,
			OutputContractVersion: info.OutputContractVersion,
			SkillSource:           info.Source,
			SkillSourceVersion:    info.SourceVersion,
		},
	}
}
