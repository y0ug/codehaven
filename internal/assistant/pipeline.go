package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/actions/executors"
	"github.com/y0ug/codehaven/internal/assistant/actions/middleware"
	"github.com/y0ug/codehaven/internal/assistant/actions/queue"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
)

type Processor[T any] func(context.Context, T) (T, []T, error)

type Pipeline struct {
	eventBus      *eventbus.EventBus
	cb            Processor[actions.Action]
	logger        *slog.Logger
	actionManager *actions.ActionManager
	registry      *executors.Registry
}

func NewPipeline(
	logger *slog.Logger,
	actionManager *actions.ActionManager,
	registry *executors.Registry,
) *Pipeline {
	p := &Pipeline{
		eventBus:      eventbus.GetEventBus(),
		logger:        logger,
		actionManager: actionManager,
		registry:      registry,
	}
	chain := middleware.NewMiddlewareChain(
		p.baseHandler,
		middleware.NewLoggerMiddleware(p.logger),
		middleware.NewTimeoutMiddleware(time.Second*1),
	)
	p.cb = chain.Process
	return p
}

func (p *Pipeline) Execute(ctx context.Context, action actions.Action) {
	p.startAction(ctx, action)
}

func (p *Pipeline) processSingleAction(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	var results []actions.Action
	var err error

	p.actionManager.RegisterAction(action)

	action, results, err = p.cb(ctx, action)
	if err != nil {
		p.logger.Error("Error handling action", "error", err, "context", action)
	}

	if action.Result == nil {
		action.Result = &actions.ActionResult{}
		action.Result.Success = true
		if err != nil {
			action.Result.Error = err
			action.Result.Success = false
		}
	}

	// Update the action results
	p.actionManager.RegisterAction(action)

	// Register follow-up actions
	for _, result := range results {
		p.actionManager.RegisterAction(result)
	}

	// p.actionManager.AddResult(action.Context.ChainID,
	// 	fmt.Sprintf("ACTION: %s", action.String())) // Store action summary
	return action, results, err
}

func (p *Pipeline) startAction(ctx context.Context, action actions.Action) {
	q := queue.NewActionQueue()

	q.Enqueue(action)
	for !q.IsEmpty() {
		action, _ := q.Dequeue()

		_, results, _ := p.processSingleAction(ctx, action)

		if len(results) > 0 {
			q.Enqueue(results...)
		}
	}
}

func (p *Pipeline) baseHandler(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	handler := p.registry.GetHandler(action)
	if handler == nil {

		errMsg := fmt.Errorf("no handler for %s action", action.Type)
		p.logger.Error("No handler", "error", errMsg, "context", action)
		p.actionManager.AddError(action.Context.ChainID, action, errMsg)

		// Create error followup action
		errorAction := actions.NewLogAction(&action, errMsg.Error())

		// we don't return the error we handle it here
		return action, []actions.Action{
			errorAction,
		}, nil // fmt.Errorf("no handler for action type %s", action.Type)
	}
	return handler.Handle(ctx, action)
}
