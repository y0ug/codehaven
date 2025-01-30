package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware creates a middleware that logs request and response details
func TimeitMiddleware(
	logger *slog.Logger,
) func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	return func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
		start := time.Now()

		// Call the next middleware/handler
		resp, err := next(req)
		if err != nil {
			return resp, err
		}

		end := time.Now()

		logger.Debug("Request tool", "time", end.Sub(start))
		return resp, err
	}
}
