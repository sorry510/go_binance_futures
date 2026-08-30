package manager

import (
	"fmt"
	"strings"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/llm"
)

func TestAdmissionRunsBeforeLLMInitialization(t *testing.T) {
	skills := skill.NewRegistry()
	_ = skills.Register(skill.Definition{SkillName: "disabled", Rounds: 1})
	clientCalls := 0
	manager, err := New(Config{
		Skills: skills,
		Admission: func(string) error { return fmt.Errorf("skill disabled") },
		NewClient: func() (llm.Client, error) {
			clientCalls++
			return &fakeClient{}, nil
		},
		RuntimeConfig: agentruntime.Config{Timeout: time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Start(agentruntime.Request{Skill: "disabled", Input: `{}`})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected admission rejection, got %v", err)
	}
	if clientCalls != 0 {
		t.Fatalf("LLM client initialized %d times after admission rejection", clientCalls)
	}
}
