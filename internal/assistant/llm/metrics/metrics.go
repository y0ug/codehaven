package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/llm/models"
	"github.com/y0ug/llmhaven/chat"
)

type MetricsBasic struct {
	logger *slog.Logger
	model  models.Model
	mu     sync.Mutex // Protects all cumulative fields below

	// Cumulative metrics
	totalInputTokens              int
	totalInputAudioTokens         int
	totalInputCachedTokens        int
	totalInputCacheCreationTokens int
	totalOutputTokens             int
	totalOutputAudioTokens        int
	totalOutputReasoningTokens    int
	totalCost                     float64
	totalRequests                 int
	totalDuration                 time.Duration
}

func NewMetricsTracker(logger *slog.Logger, model models.Model) *MetricsBasic {
	return &MetricsBasic{
		logger: logger,
		model:  model,
	}
}

func (m *MetricsBasic) RecordRequest(
	ctx context.Context,
	resp *chat.ChatResponse,
	duration time.Duration,
) {
	if resp == nil {
		m.logger.WarnContext(ctx, "nil response received")
		return
	}

	usage := resp.Usage
	if usage == nil {
		m.logger.WarnContext(ctx, "response usage data is nil")
		return
	}

	// Calculate costs
	var inputCost, outputCost, totalCost float64
	if m.model.Info() != nil {
		inputCost = float64(usage.InputTokens) * m.model.Info().GetInputCostPerToken()
		outputCost = float64(usage.OutputTokens) * m.model.Info().GetOutputCostPerToken()
		// cacheCreationCost := float64(
		// 	usage.InputCacheCreationTokens,
		// ) * m.model.Info().
		// 	GetCacheCreationCostPerToken()
		totalCost = inputCost + outputCost
	}

	logAttrs := []slog.Attr{
		slog.Float64("total_cost", totalCost),

		slog.Int("input_tokens", usage.InputTokens),
		slog.Int("output_tokens", usage.OutputTokens),

		slog.Int("cached_tokens", usage.InputCachedTokens),

		slog.Float64("duration_seconds", duration.Seconds()),
	}

	m.logger.LogAttrs(ctx, slog.LevelInfo, "usage", logAttrs...)

	// Update cumulative metrics
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalInputTokens += usage.InputTokens
	m.totalInputAudioTokens += usage.InputAudioTokens
	m.totalInputCachedTokens += usage.InputCachedTokens
	m.totalInputCacheCreationTokens += usage.InputCacheCreationTokens
	m.totalOutputTokens += usage.OutputTokens
	m.totalOutputAudioTokens += usage.OutputAudioTokens
	m.totalOutputReasoningTokens += usage.OutputReasoningTokens
	m.totalCost += totalCost
	m.totalRequests++
	m.totalDuration += duration
}

// LogTotalMetrics logs cumulative metrics since tracker creation
func (m *MetricsBasic) LogTotalMetrics(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var avgDuration time.Duration
	if m.totalRequests > 0 {
		avgDuration = m.totalDuration / time.Duration(m.totalRequests)
	}

	logAttrs := []slog.Attr{
		slog.Int("total_requests", m.totalRequests),
		slog.Int("total_input_tokens", m.totalInputTokens),
		slog.Int("total_input_audio_tokens", m.totalInputAudioTokens),
		slog.Int("total_input_cached_tokens", m.totalInputCachedTokens),
		slog.Int("total_input_cache_creation_tokens", m.totalInputCacheCreationTokens),
		slog.Int("total_output_tokens", m.totalOutputTokens),
		slog.Int("total_output_audio_tokens", m.totalOutputAudioTokens),
		slog.Int("total_output_reasoning_tokens", m.totalOutputReasoningTokens),
		slog.Float64("total_cost", m.totalCost),
		slog.Duration("total_duration", m.totalDuration),
		slog.Duration("average_duration", avgDuration),
	}

	m.logger.LogAttrs(ctx, slog.LevelInfo, "LLM cumulative metrics", logAttrs...)
}

// LogValue implements slog.LogValuer interface for structured logging
func (m *MetricsBasic) LogValue() slog.Value {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slog.GroupValue(
		slog.Int("requests", m.totalRequests),
		slog.Int("tokens", m.totalInputTokens+m.totalOutputTokens),
		slog.Int("input_tokens", m.totalInputTokens),
		slog.Int("input_cached_tokens", m.totalInputCachedTokens),
		slog.Int("output_tokens", m.totalOutputTokens),
		slog.Int("output_reasoning_tokens", m.totalOutputReasoningTokens),
		slog.Float64("cost", m.totalCost),
		slog.Duration("duration", m.totalDuration),
	)
	// return slog.GroupValue(
	// 	slog.Int("total_requests", m.totalRequests),
	// 	slog.Int("total_input_tokens", m.totalInputTokens),
	// 	slog.Int("total_input_audio_tokens", m.totalInputAudioTokens),
	// 	slog.Int("total_input_cached_tokens", m.totalInputCachedTokens),
	// 	slog.Int("total_input_cache_creation_tokens", m.totalInputCacheCreationTokens),
	// 	slog.Int("total_output_tokens", m.totalOutputTokens),
	// 	slog.Int("total_output_audio_tokens", m.totalOutputAudioTokens),
	// 	slog.Int("total_output_reasoning_tokens", m.totalOutputReasoningTokens),
	// 	slog.Float64("total_cost", m.totalCost),
	// 	slog.Duration("total_duration", m.totalDuration),
	// )
}
