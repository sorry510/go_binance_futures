package replay

import "go_binance_futures/agent/task"

type VersionDifference struct {
	Field string `json:"field"`
	From  any    `json:"from"`
	To    any    `json:"to"`
}

func CompareVersions(from, to task.VersionMetadata) []VersionDifference {
	result := make([]VersionDifference, 0)
	compare := func(field string, left, right any) {
		if left != right {
			result = append(result, VersionDifference{Field: field, From: left, To: right})
		}
	}
	compare("runtime_version", from.RuntimeVersion, to.RuntimeVersion)
	compare("skill_version", from.SkillVersion, to.SkillVersion)
	compare("prompt_version", from.PromptVersion, to.PromptVersion)
	compare("prompt_hash", from.PromptHash, to.PromptHash)
	compare("model_config_id", from.ModelConfigID, to.ModelConfigID)
	compare("input_contract_version", from.InputContractVersion, to.InputContractVersion)
	compare("output_contract_version", from.OutputContractVersion, to.OutputContractVersion)
	compare("skill_source", from.SkillSource, to.SkillSource)
	compare("skill_source_version", from.SkillSourceVersion, to.SkillSourceVersion)
	compare("tool_catalog_hash", from.ToolCatalogHash, to.ToolCatalogHash)
	compare("skill_package_hash", from.SkillPackageHash, to.SkillPackageHash)
	return result
}
