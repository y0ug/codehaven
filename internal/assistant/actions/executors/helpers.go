package executors

import (
	"context"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

func logActionStart(ctx context.Context, msg string) {
	logger := actions.GetLogger(ctx)
	logger.Debug(msg)
}

func logActionEnd(ctx context.Context, msg string, err error) {
	logger := actions.GetLogger(ctx)
	if err != nil {
		logger.Error(msg, "error", err)
	} else {
		logger.Debug(msg)
	}
}
