package middleware

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type ActionMiddleware interface {
	Process(
		ctx context.Context,
		action actions.Action,
		next ActionHandler,
	) (actions.Action, []actions.Action, error)
}

type ActionHandler func(context.Context, actions.Action) (actions.Action, []actions.Action, error)
