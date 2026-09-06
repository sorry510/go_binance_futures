package skill

import "context"

// ChatAdapter exposes a Skill to the human Chat entrypoint without changing
// its existing Runtime input contract or security policy.
type ChatAdapter interface {
	ChatEnabled() bool
	BuildChatInput(context.Context, string) (string, error)
}

// ChatContextAdapter optionally lets a chat-capable Skill derive deterministic
// input from its own previously succeeded Task inputs. It must not use another
// LLM or bypass the Skill's existing input validator.
type ChatContextAdapter interface {
	BuildChatInputWithContext(context.Context, string, []string) (string, error)
}

// ChatInputOptions contains explicit user selections supplied by the Chat UI.
// These values are deterministic context and must take precedence over guesses
// extracted from free-form message text.
type ChatInputOptions struct {
	Symbol string
}

// ChatOptionsAdapter lets a Chat-capable Skill consume explicit UI selections
// while preserving its existing Runtime input contract.
type ChatOptionsAdapter interface {
	BuildChatInputWithOptions(context.Context, string, []string, ChatInputOptions) (string, error)
}

// PlainTextFinalAdapter allows a Skill adapter to accept a plain-text LLM reply
// as its final result when no package resource or external Tool is required.
// Structured Native Skills should not implement this interface.
type PlainTextFinalAdapter interface {
	PlainTextFinalAllowed() bool
}
