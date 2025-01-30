package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/highlighter"
)

type InputHandler struct {
	eventBus         *eventbus.EventBus
	ctx              context.Context
	cancel           context.CancelFunc
	consoleInputChan chan string
}

func NewInputHandler() *InputHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &InputHandler{
		consoleInputChan: make(chan string),
		eventBus:         eventbus.GetEventBus(),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (h *InputHandler) Start() {
	go h.listenConsole()
	go h.listenAPI()
}

func (h *InputHandler) listenConsole() {
	// Integration with console input
	for {
		select {
		case <-h.ctx.Done():
			return
		case input := <-h.consoleInputChan:
			h.eventBus.Publish(eventbus.NewEvent(
				eventbus.EventInput,
				eventbus.UserInput{Source: "console", Content: input},
			))
		}
	}
}

func (h *InputHandler) listenAPI() {
	// WebSocket/HTTP API integration
}

type OutputHandler struct {
	eventBus    *eventbus.EventBus
	highlighter *highlighter.Highlighter
}

func NewOutputHandler(h *highlighter.Highlighter) *OutputHandler {
	return &OutputHandler{
		eventBus:    eventbus.GetEventBus(),
		highlighter: h,
	}
}

func (h *OutputHandler) Start() {
	sub := h.eventBus.Subscribe(100)
	go h.processOutput(sub)
}

func (h *OutputHandler) processOutput(ch <-chan eventbus.Event) {
	for event := range ch {
		switch event.Type {
		case eventbus.EventOutput:
			if output, ok := event.Payload.(string); ok {
				fmt.Fprint(h.highlighter, output)
			}

		case eventbus.EventError:
			if errData, ok := event.Payload.(map[string]interface{}); ok {
				var errMsg strings.Builder
				if err, exists := errData["error"].(error); exists {
					errMsg.WriteString(fmt.Sprintf("Error: %v", err))
				}
				if ctx, exists := errData["context"].(string); exists {
					errMsg.WriteString(fmt.Sprintf(" (Context: %s)", ctx))
				}
				if file, exists := errData["file"].(string); exists {
					errMsg.WriteString(fmt.Sprintf(" (File: %s)", file))
				}
				fmt.Fprintf(h.highlighter, "\x1b[31m%s\x1b[0m\n", errMsg.String())
			}
		}

		// Some idea to handle different output types
		// switch event.Metadata["output_type"] {
		// case "stream":
		// case "final":
		// case "error":
		// }
	}
}
