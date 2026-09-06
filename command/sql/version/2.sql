-- Additive schema is created by orm.RunSyncdb before this version script.
-- Initialize the new per-Skill Chat visibility switch without changing runtime enablement.
UPDATE agent_skills
SET chat_enabled = 0
WHERE type = 'native'
  AND chat_enabled NOT IN (0, 1)
  AND name IN ('alert_analysis', 'market_regime', 'strategy_builder', 'alert_triage', 'strategy_experiment_propose', 'strategy_experiment_summary');

UPDATE agent_skills
SET chat_enabled = 1
WHERE chat_enabled NOT IN (0, 1)
  AND (type = 'portable' OR (type = 'native' AND name NOT IN ('alert_analysis', 'market_regime', 'strategy_builder', 'alert_triage', 'strategy_experiment_propose', 'strategy_experiment_summary')));
