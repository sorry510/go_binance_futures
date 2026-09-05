package llm

import (
	"context"
	"fmt"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const routerSettingsID int64 = 1

func DefaultRouterSettings() RouterSettings {
	return RouterSettings{Enabled: 0, FallbackEnabled: 1, FailureThreshold: 3, CooldownSeconds: 60, HealthWindow: 20}
}

func (store Store) RouterSettings(ctx context.Context) (RouterSettings, error) {
	if err := ctx.Err(); err != nil {
		return RouterSettings{}, err
	}
	row := models.LLMRouterSetting{ID: routerSettingsID}
	err := store.ormer().Read(&row)
	if err == orm.ErrNoRows {
		return DefaultRouterSettings(), nil
	}
	if err != nil {
		return RouterSettings{}, err
	}
	return normalizeRouterSettings(RouterSettings{Enabled: row.Enabled, FallbackEnabled: row.FallbackEnabled, FailureThreshold: row.FailureThreshold, CooldownSeconds: row.CooldownSeconds, HealthWindow: row.HealthWindow}), nil
}

func (store Store) UpdateRouterSettings(ctx context.Context, input RouterSettings) (RouterSettings, error) {
	if err := ctx.Err(); err != nil {
		return RouterSettings{}, err
	}
	value, err := validateRouterSettings(input)
	if err != nil {
		return RouterSettings{}, err
	}
	row := models.LLMRouterSetting{
		ID: routerSettingsID, Enabled: value.Enabled, FallbackEnabled: value.FallbackEnabled,
		FailureThreshold: value.FailureThreshold, CooldownSeconds: value.CooldownSeconds,
		HealthWindow: value.HealthWindow, UpdatedAt: time.Now().UnixMilli(),
	}
	o := store.ormer()
	if err := o.Read(&models.LLMRouterSetting{ID: routerSettingsID}); err == orm.ErrNoRows {
		if _, err := o.Insert(&row); err != nil {
			return RouterSettings{}, err
		}
	} else if err != nil {
		return RouterSettings{}, err
	} else if _, err := o.Update(&row); err != nil {
		return RouterSettings{}, err
	}
	return value, nil
}

func validateRouterSettings(input RouterSettings) (RouterSettings, error) {
	if input.Enabled != 0 && input.Enabled != 1 {
		return input, fmt.Errorf("router enabled must be 0 or 1")
	}
	if input.FallbackEnabled != 0 && input.FallbackEnabled != 1 {
		return input, fmt.Errorf("fallback_enabled must be 0 or 1")
	}
	if input.FailureThreshold < 1 || input.FailureThreshold > 20 {
		return input, fmt.Errorf("failure_threshold must be between 1 and 20")
	}
	if input.CooldownSeconds < 5 || input.CooldownSeconds > 3600 {
		return input, fmt.Errorf("cooldown_seconds must be between 5 and 3600")
	}
	if input.HealthWindow < 5 || input.HealthWindow > 100 {
		return input, fmt.Errorf("health_window must be between 5 and 100")
	}
	return input, nil
}

func normalizeRouterSettings(input RouterSettings) RouterSettings {
	defaults := DefaultRouterSettings()
	if input.FailureThreshold <= 0 {
		input.FailureThreshold = defaults.FailureThreshold
	}
	if input.CooldownSeconds <= 0 {
		input.CooldownSeconds = defaults.CooldownSeconds
	}
	if input.HealthWindow <= 0 {
		input.HealthWindow = defaults.HealthWindow
	}
	if input.FallbackEnabled != 0 && input.FallbackEnabled != 1 {
		input.FallbackEnabled = defaults.FallbackEnabled
	}
	if input.Enabled != 0 && input.Enabled != 1 {
		input.Enabled = defaults.Enabled
	}
	return input
}

func (store Store) RoutingConfigs(ctx context.Context) ([]RoutingConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.LLMConfig
	if _, err := store.ormer().QueryTable(new(models.LLMConfig)).Filter("Deleted", 0).OrderBy("-enabled", "id").All(&rows); err != nil {
		return nil, err
	}
	result := make([]RoutingConfig, 0, len(rows))
	for _, row := range rows {
		if row.Enabled != 1 && row.RouterCandidate != 1 {
			continue
		}
		cfg, err := configFromModel(row)
		if err != nil {
			return nil, fmt.Errorf("load routing config %d: %w", row.ID, err)
		}
		result = append(result, RoutingConfig{
			Config: cfg, Name: row.Name, Primary: row.Enabled == 1,
			Profile: ModelProfile{StructuredOutput: row.StructuredOutput == 1, NativeToolCalling: row.NativeToolCalling == 1,
				Reasoning: row.Reasoning == 1, LongContext: row.LongContext == 1, JSONReliability: row.JSONReliability,
				MaxContextTokens: row.MaxContextTokens, CostClass: normalizeClass(row.CostClass), LatencyClass: normalizeClass(row.LatencyClass)},
		})
	}
	return result, nil
}
