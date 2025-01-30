package middleware

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type MiddlewareChain struct {
	middlewares []ActionMiddleware
	handler     ActionHandler
}

func NewMiddlewareChain(handler ActionHandler, middlewares ...ActionMiddleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
		handler:     handler,
	}
}

func (c *MiddlewareChain) Process(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	// Build the chain in reverse order
	var next ActionHandler = c.handler
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		middleware := c.middlewares[i]
		next = func(current ActionHandler, m ActionMiddleware) ActionHandler {
			return func(ctx context.Context, a actions.Action) (actions.Action, []actions.Action, error) {
				return m.Process(ctx, a, current)
			}
		}(next, middleware)
	}

	return next(ctx, action)
}
