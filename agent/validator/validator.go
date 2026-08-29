package validator

import (
	"context"
	"encoding/json"
)

type FinalValidator interface {
	Validate(ctx context.Context, raw json.RawMessage) (any, error)
}

type Func func(context.Context, json.RawMessage) (any, error)

func (fn Func) Validate(ctx context.Context, raw json.RawMessage) (any, error) {
	return fn(ctx, raw)
}

type Passthrough struct{}

func (Passthrough) Validate(_ context.Context, raw json.RawMessage) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}
