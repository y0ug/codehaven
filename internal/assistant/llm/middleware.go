package llm

import (
	"context"
	"log/slog"
	"time"

	"github.com/y0ug/llmhaven/chat"
)

type Middleware func(next ChatCompleter) ChatCompleter

type ChatCompleter interface {
	SendMessages(
		ctx context.Context,
		messages []*chat.ChatMessage,
		tools []chat.Tool,
		opts ...func(*Params),
	) (*chat.ChatResponse, error)
}

type wrappedChatCompleter struct {
	next ChatCompleter
}

func (w *wrappedChatCompleter) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	return w.next.SendMessages(ctx, messages, tools, opts...)
}

func WithMiddleware(client ChatCompleter, middlewares ...Middleware) ChatCompleter {
	for _, mw := range middlewares {
		client = mw(client)
	}
	return client
}

// DebugStore to saveRequest
type DebugStore interface {
	SaveRequest(ctx context.Context, req *chat.ChatMessage, resp *chat.ChatResponse)
}

func DebugMiddleware(store DebugStore) Middleware {
	return func(next ChatCompleter) ChatCompleter {
		return &wrappedChatCompleter{
			next: &debugChatCompleter{
				next:  next,
				store: store,
			},
		}
	}
}

type debugChatCompleter struct {
	next  ChatCompleter
	store DebugStore
}

func (d *debugChatCompleter) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	resp, err := d.next.SendMessages(ctx, messages, tools, opts...)

	if resp != nil {
		for _, msg := range messages {
			d.store.SaveRequest(ctx, msg, resp)
		}
	}

	return resp, err
}

type MetricsRecorder interface {
	RecordRequest(ctx context.Context, resp *chat.ChatResponse, duration time.Duration)
}

func MetricsMiddleware(recorder MetricsRecorder) Middleware {
	return func(next ChatCompleter) ChatCompleter {
		return &wrappedChatCompleter{
			next: &metricsChatCompleter{
				next:     next,
				recorder: recorder,
			},
		}
	}
}

type metricsChatCompleter struct {
	next     ChatCompleter
	recorder MetricsRecorder
}

func (m *metricsChatCompleter) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	start := time.Now()

	resp, err := m.next.SendMessages(ctx, messages, tools, opts...)
	if resp != nil {
		m.recorder.RecordRequest(ctx, resp, time.Since(start))
	}

	return resp, err
}

func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next ChatCompleter) ChatCompleter {
		return &wrappedChatCompleter{
			next: &loggingChatCompleter{
				next:   next,
				logger: logger,
			},
		}
	}
}

type loggingChatCompleter struct {
	next   ChatCompleter
	logger *slog.Logger
}

func (m *loggingChatCompleter) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	start := time.Now()

	m.logger.Debug("request", "messages", len(messages), "tools", len(tools))
	resp, err := m.next.SendMessages(ctx, messages, tools, opts...)

	logger := m.logger.With("duration", time.Since(start))
	logAttrs := []slog.Attr{}

	if err != nil {
		logger.Error("error", "error", err)
		return resp, err
	} else if resp == nil {
		logger.Warn("nil response received")
		return resp, err
	} else if usage := resp.Usage; usage != nil {
		// var inputCost, outputCost, totalCost float64
		// if m.model.Info() != nil {
		// 	inputCost = float64(usage.InputTokens) * m.model.Info().GetInputCostPerToken()
		// 	outputCost = float64(usage.OutputTokens) * m.model.Info().GetOutputCostPerToken()
		// 	// cacheCreationCost := float64(
		// 	// 	usage.InputCacheCreationTokens,
		// 	// ) * m.model.Info().
		// 	// 	GetCacheCreationCostPerToken()
		// 	totalCost = inputCost + outputCost
		// }

		logAttrs = []slog.Attr{
			// slog.Float64("total_cost", totalCost),

			slog.Int("input_tokens", usage.InputTokens),
			slog.Int("output_tokens", usage.OutputTokens),

			slog.Int("cached_tokens", usage.InputCachedTokens),
		}
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "response", logAttrs...)

	return resp, err
}
