-- V4-1 Chat Workspace: ensure legacy chat conversations keep the default fallback Skill.
INSERT INTO agent_conversation_skills (conversation_id, skill_name, sort, created_at)
SELECT c.id, 'general_chat', 0, c.created_at
FROM agent_conversations c
WHERE c.skill = 'chat'
  AND NOT EXISTS (
    SELECT 1
    FROM agent_conversation_skills s
    WHERE s.conversation_id = c.id
      AND s.skill_name = 'general_chat'
  );
