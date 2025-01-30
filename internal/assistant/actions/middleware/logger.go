// filename: internal/assistant/actions/middleware/logger.go
package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

const (
	LoggerKey string = "logger"
)

type LoggerMiddleware struct {
	*BaseMiddleware
	baseLogger *slog.Logger
}

func NewLoggerMiddleware(baseLogger *slog.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{
		baseLogger: baseLogger,
	}
}

func (m *LoggerMiddleware) Process(
	ctx context.Context,
	action actions.Action,
	next ActionHandler,
) (actions.Action, []actions.Action, error) {
	// Create contextual logger
	logger := m.baseLogger.With(
		"action", action,
	)

	// Add to context
	ctx = actions.WithLogger(ctx, logger)

	start := time.Now()

	// Log action start
	logger.Debug("Processing action")

	// Process action
	action, results, err := next(ctx, action)

	// Log action completion
	duration := time.Since(start)
	if err != nil {
		logger.Error("Action processing failed", "error", err, "duration", duration)
	} else {
		logger.Debug("Action processing completed", "results", len(results), "duration", duration)
	}

	return action, results, err
}
