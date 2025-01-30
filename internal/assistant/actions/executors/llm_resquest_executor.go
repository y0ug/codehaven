package executors

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type (
	Processor[T any]   func(context.Context, T) ([]T, error)
	LLMRequestExecutor struct {
		cb Processor[actions.Action]
	}
)

func NewLLMRequestExecutor(
	cb Processor[actions.Action],
) *LLMRequestExecutor {
	return &LLMRequestExecutor{
		cb: cb,
	}
}

func (e *LLMRequestExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.LLMRequestAction)
	return ok
}

func (e *LLMRequestExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	// logger := actions.GetLogger(ctx)

	_ = action.Payload.(actions.LLMRequestAction)
	// resp := val.Resp

	results, err := e.cb(ctx, action)
	return action, results, err
}
