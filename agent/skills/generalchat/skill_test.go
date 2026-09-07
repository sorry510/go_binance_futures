package generalchat

import (
	"context"
	"testing"

	"go_binance_futures/agent/skill"
	"go_binance_futures/llm"
)

func TestGeneralChatIsPlainTextNoToolSkill(t *testing.T) {
	definition := New()
	if definition.Name() != Name || len(definition.Tools()) != 0 || definition.MaxRounds() != 1 || !definition.PlainTextFinalAllowed() || !definition.DirectTextFinalAllowed() {
		t.Fatalf("unexpected general chat definition")
	}
	if _, ok := any(definition).(skill.ChatAdapter); ok {
		t.Fatal("general chat must stay hidden from the explicit Skill selector")
	}
	messages, err := definition.BuildInput(context.Background(), skill.Request{Input: "  hello  "})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Role != llm.RoleUser || messages[0].Content != "hello" {
		t.Fatalf("unexpected messages: %+v", messages)
	}
}
