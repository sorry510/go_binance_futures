package contextengine

import (
	"context"
	"fmt"
	"strings"
)

func (engine *Engine) LoadResources(ctx context.Context, resources []Resource, requestedIDs []string) ([]ContextBlock, error) {
	requested := map[string]bool{}
	for _, id := range requestedIDs {
		if id = strings.TrimSpace(id); id != "" {
			requested[id] = true
		}
	}
	blocks := []ContextBlock{}
	for _, resource := range resources {
		if resource.Load == nil || strings.TrimSpace(resource.ID) == "" {
			continue
		}
		if resource.Disclosure == DisclosureOnDemand && !requested[resource.ID] {
			continue
		}
		content, err := resource.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("load context resource %q: %w", resource.ID, err)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		blockType := resource.Type
		if blockType == "" {
			blockType = BlockSkillInstruction
		}
		blocks = append(blocks, ContextBlock{
			ID: "resource-" + resource.ID, Type: blockType, Source: resource.Source, Priority: resource.Priority,
			Sensitive: resource.Sensitive, Freshness: FreshnessUnknown, Content: content,
		})
	}
	return blocks, nil
}
