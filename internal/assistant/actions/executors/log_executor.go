package executors

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type LogExecutor struct{}

func NewLogExecutor() *LogExecutor {
	return &LogExecutor{}
}

func (e *LogExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.LogAction)
	return ok
}

func (e *LogExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	logger := actions.GetLogger(ctx)
	log := action.Payload.(actions.LogAction)
	logger.Info("Log", "Message", log.Message)
	return action, nil, nil
}
