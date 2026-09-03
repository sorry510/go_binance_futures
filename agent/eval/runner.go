package eval

import (
	"context"
	"fmt"

	"go_binance_futures/agent/replay"
	"go_binance_futures/agent/skill"
)

func Run(ctx context.Context, item Case, definition skill.Skill) (Report, error) {
	fixture, err := replay.Load(item.FixturePath())
	if err != nil {
		return Report{}, err
	}
	return Evaluate(ctx, item, fixture, definition), nil
}

func Evaluate(ctx context.Context, item Case, fixture replay.Fixture, definition skill.Skill) Report {
	started := now()
	if definition == nil || definition.Name() != item.Skill || fixture.Skill != item.Skill {
		return Report{CaseName: item.Name, Skill: item.Skill, Error: fmt.Sprintf("case/fixture/definition skill mismatch: %s/%s", item.Skill, fixture.Skill)}
	}
	out := replay.Run(ctx, fixture, definition)
	duration := now().Sub(started)
	return score(item, out, duration)
}
