package executors

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/assistant/ui"
)

type UserInteractionExecutor struct {
	pendingRequests sync.Map
	logger          *slog.Logger
	timeout         time.Duration
	eventBus        *eventbus.EventBus
	responseChan    chan actions.UserResponseAction
	ui              ui.UserInterface
}

type PendingRequest struct {
	OriginalAction actions.Action
	ReceivedAt     time.Time
	ResponseChan   chan actions.UserResponseAction
}

func NewUserInteractionExecutor(
	logger *slog.Logger,
	eventBus *eventbus.EventBus,
	ui ui.UserInterface,
) *UserInteractionExecutor {
	e := &UserInteractionExecutor{
		logger:       logger,
		timeout:      5 * time.Minute,
		eventBus:     eventBus,
		responseChan: make(chan actions.UserResponseAction),
		ui:           ui,
	}

	if eventBus != nil {
		sub := eventBus.Subscribe(100)
		go e.handleEvents(sub)
	}
	go e.cleanupRoutine()
	return e
}

func (e *UserInteractionExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.UserInputAction)
	return ok
}

func (e *UserInteractionExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	logger := actions.GetLogger(ctx)

	confirmAction := action.Payload.(actions.UserInputAction)

	requestID := uuid.New().String()
	responseChan := make(chan actions.UserResponseAction)

	now := time.Now()
	expiresAt := now.Add(e.timeout)

	// Store the pending request
	e.pendingRequests.Store(requestID, &PendingRequest{
		OriginalAction: confirmAction.ParentAction,
		ReceivedAt:     now,
		ResponseChan:   responseChan,
	})

	req := eventbus.NewEvent(
		eventbus.EventUserInputRequest,
		eventbus.UserInputRequest{
			ID:        requestID,
			Action:    confirmAction,
			ExpiresAt: expiresAt,
		})

	// Publish confirmation request event
	if e.eventBus != nil {
		e.eventBus.Publish(req)
	} else {
		response, err := e.ui.RequestConfirmation(req)
		_, ok := response.Payload.(eventbus.UserInputResponse)
		if !ok {
			return action, nil, fmt.Errorf("confirmation request failed wrong response %T: %w", response, err)
		}
		if err != nil {
			return action, nil, fmt.Errorf("confirmation request failed: %w", err)
		}
		go e.handleResponse(response)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		// Update original action based on response
		switch originalAction := confirmAction.ParentAction.Payload.(type) {
		case actions.ShellCommandAction:
			if response.Allowed {
				newConfirmedaction := originalAction.WithConfirmed(&action)
				return action, []actions.Action{newConfirmedaction}, nil
			}
		default:
			logger.Error("Unsupported action type for confirmation",
				"action_type", confirmAction.ParentAction.Type)
			return action, nil, fmt.Errorf("unsupported action type for confirmation")
		}
	case <-time.After(e.timeout):
		e.pendingRequests.Delete(requestID)
		return action, nil, fmt.Errorf("confirmation timeout")
	case <-ctx.Done():
		e.pendingRequests.Delete(requestID)
		return action, nil, ctx.Err()
	}

	return action, nil, nil
}

func (e *UserInteractionExecutor) handleResponse(event eventbus.Event) {
	response, ok := event.Payload.(eventbus.UserInputResponse)
	if ok {
		if req, ok := e.pendingRequests.Load(response.ID); ok {
			if pendingReq, ok := req.(*PendingRequest); ok {
				// Send response to the waiting Handle function
				pendingReq.ResponseChan <- actions.UserResponseAction{
					Allowed: response.Approved,
					// Input:   response.Input,
				}
			}
			e.pendingRequests.Delete(response.ID)
		}
	}
}

func (e *UserInteractionExecutor) handleEvents(sub <-chan eventbus.Event) {
	for event := range sub {
		if event.Type == eventbus.EventUserInputResponse {
			e.handleResponse(event)
		}
	}
}

func (e *UserInteractionExecutor) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		e.pendingRequests.Range(func(key, value interface{}) bool {
			req := value.(*PendingRequest)
			if time.Since(req.ReceivedAt) > e.timeout {
				e.pendingRequests.Delete(key)
				e.logger.Debug("Cleaned up expired request",
					"age", time.Since(req.ReceivedAt),
					"action_id", req.OriginalAction.ID,
				)
				// Close response channel to prevent goroutine leak
				close(req.ResponseChan)
			}
			return true
		})
	}
}
