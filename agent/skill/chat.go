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

// PlainTextFinalAdapter allows a Skill adapter to accept a plain-text LLM reply
// as its final result when no package resource or external Tool is required.
// Structured Native Skills should not implement this interface.
type PlainTextFinalAdapter interface {
	PlainTextFinalAllowed() bool
}
