package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type RetryMiddleware struct {
	*BaseMiddleware
	maxAttempts int
	delay       time.Duration
}

func NewRetryMiddleware(maxAttempts int, delay time.Duration) *RetryMiddleware {
	return &RetryMiddleware{
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

func (m *RetryMiddleware) Process(
	ctx context.Context,
	action actions.Action,
	next ActionHandler,
) (actions.Action, []actions.Action, error) {
	var lastErr error
	for i := 0; i < m.maxAttempts; i++ {
		action, results, err := next(ctx, action)
		if err == nil {
			return action, results, nil
		}
		lastErr = err
		time.Sleep(m.delay)
	}
	return action, nil, fmt.Errorf("after %d attempts, last error: %w", m.maxAttempts, lastErr)
}
