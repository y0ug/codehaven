package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/llmhaven/http/streaming"
)

type StreamProcessor struct {
	outputChan chan string
	logger     *slog.Logger
}

func NewStreamProcessor(
	outputChan chan string,
	logger *slog.Logger,
) *StreamProcessor {
	return &StreamProcessor{
		outputChan: outputChan,
		logger:     logger,
	}
}

func (sp *StreamProcessor) HasWriter() bool {
	return true // sp.writer != nil
}

func (sp *StreamProcessor) ProcessStream(
	ctx context.Context,
	stream streaming.Streamer[chat.EventStream],
) (*chat.ChatResponse, error) {
	eventCh := make(chan chat.EventStream)
	var chatResponse *chat.ChatResponse

	go func() {
		if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
			if err != context.Canceled {
				sp.logger.Error("Error consuming stream", "error", err)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			sp.outputChan <- "\n"
			return nil, ctx.Err()

		case event, ok := <-eventCh:
			if !ok {
				sp.outputChan <- "\n"
				return chatResponse, nil
			}

			switch event.Type {
			case "text_delta":
				sp.outputChan <- fmt.Sprintf("%v", event.Delta)
			case "message_stop":
				chatResponse = event.Message
			case "error":
				chatResponse = event.Message
				sp.outputChan <- "\n"
				return chatResponse, fmt.Errorf("error in stream: %v", event.Message)
			}
		}
	}
}
