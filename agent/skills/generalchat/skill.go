package generalchat

import (
	"context"
	"strings"

	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/validator"
	"go_binance_futures/llm"
)

const Name = "general_chat"

const systemPrompt = `You are the general conversational assistant inside this application.
Answer the user's latest message directly and naturally. Use the conversation history supplied in context so follow-up questions remain coherent.
Do not invent tool results, market data, account data, or other facts that are not present in the conversation context.
No business Skill is selected for this request, so do not assume a specialized workflow or execute tools.
Return the answer as plain text or Markdown. Do not wrap the answer in an agent action JSON envelope.`

type Definition struct{}

var _ skill.ChatAdapter = (*Definition)(nil)

func New() *Definition { return &Definition{} }

func (*Definition) Name() string         { return Name }
func (*Definition) SystemPrompt() string { return systemPrompt }
func (*Definition) Tools() []string      { return nil }
func (*Definition) MaxRounds() int       { return 1 }
func (*Definition) ChatEnabled() bool    { return true }
func (*Definition) BuildChatInput(_ context.Context, content string) (string, error) {
	return strings.TrimSpace(content), nil
}
func (*Definition) BuildInput(_ context.Context, req skill.Request) ([]llm.Message, error) {
	return []llm.Message{{Role: llm.RoleUser, Content: strings.TrimSpace(req.Input)}}, nil
}
func (*Definition) Validator() validator.FinalValidator { return validator.Passthrough{} }
func (*Definition) PlainTextFinalAllowed() bool         { return true }

func (*Definition) DirectTextFinalAllowed() bool { return true }
