package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type TimeoutMiddleware struct {
	*BaseMiddleware
	maxAttempts int
	delay       time.Duration
}

func NewTimeoutMiddleware(delay time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		delay: delay,
	}
}

func (m *TimeoutMiddleware) Process(
	ctx context.Context,
	action actions.Action,
	next ActionHandler,
) (actions.Action, []actions.Action, error) {
	var results []actions.Action
	var err error

	logger := actions.GetLogger(ctx)
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		action, results, err = next(ctx, action)
	}()

	select {
	case <-done:
	case <-timeoutCtx.Done():
		err = fmt.Errorf("action processing timeout %s %s %w",
			action.ID, action.Type, err)
		logger.Error("Action processing timeout", "action", action.ID)

	}

	return action, results, err
}
