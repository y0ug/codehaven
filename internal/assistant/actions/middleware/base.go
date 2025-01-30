package middleware

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type BaseMiddleware struct {
	next ActionHandler
}

func (m *BaseMiddleware) Process(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	return m.next(ctx, action)
}

func NewBaseMiddleware(next ActionHandler) *BaseMiddleware {
	return &BaseMiddleware{next: next}
}
