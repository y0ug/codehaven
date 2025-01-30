package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/codehaven/internal/assistant/llm/metrics"
	"github.com/y0ug/codehaven/internal/assistant/settings"
	"github.com/y0ug/llmhaven/chat"
)

type Completion func(ctx context.Context, messages []*chat.ChatMessage, tools []chat.Tool) (*chat.ChatResponse, error)

type Client struct {
	client   chat.Provider
	settings *settings.CoderSettings
	logger   *slog.Logger
	metrics  *metrics.MetricsBasic
}

func New(
	client chat.Provider,
	settings *settings.CoderSettings,
	logger *slog.Logger,
) ChatCompleter {
	c := &Client{
		client:   client,
		settings: settings,
		logger:   logger,
		metrics:  metrics.NewMetricsTracker(logger, *settings.MainModel()),
	}

	return WithMiddleware(c, MetricsMiddleware(c.metrics))
}

func (c *Client) GetMetrics() *metrics.MetricsBasic {
	return c.metrics
}

func (c *Client) SendMessages(
	ctx context.Context,
	messages []*chat.ChatMessage,
	tools []chat.Tool,
	opts ...func(*Params),
) (*chat.ChatResponse, error) {
	chatOpts := WithChatParams(
		chat.WithMaxTokens(c.settings.GetMaxOutputToken()),
		chat.WithModel(c.settings.GetModelName()),
		chat.WithMessages(messages...),
		chat.WithTools(tools...))

	// Opts can overide the default one
	params := NewParams(append([]func(*Params){
		chatOpts,
		WithStream(false),
	}, opts...)...)

	fn := c.handleNonStreamingResponse

	if params.Stream && params.StreamProcessor != nil && params.StreamProcessor.HasWriter() {
		fn = c.handleStreamingResponse
	}

	resp, err := fn(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (c *Client) handleNonStreamingResponse(
	ctx context.Context,
	params *Params,
) (*chat.ChatResponse, error) {
	resp, err := c.client.Send(ctx, *params.ChatParams)
	if err != nil {
		return nil, fmt.Errorf("error chatting: %w", err)
	}

	return c.processResponse(resp)
}

func (c *Client) handleStreamingResponse(
	ctx context.Context,
	params *Params,
) (*chat.ChatResponse, error) {
	respChan, err := c.client.Stream(ctx, *params.ChatParams)
	if err != nil {
		return nil, fmt.Errorf("error streaming response: %w", err)
	}

	resp, err := params.StreamProcessor.ProcessStream(ctx, respChan)
	if err != nil {
		return nil, fmt.Errorf("error processing stream: %w", err)
	}

	return c.processResponse(resp)
}

func (c *Client) processResponse(resp *chat.ChatResponse) (*chat.ChatResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("error processing response, nil response")
	}
	msgParams := resp.ToMessageParams()
	if msgParams == nil {
		return nil, fmt.Errorf("error converting response to message params, no choice")
	}

	return resp, nil
}
