package middleware

import (
	"context"
	"fmt"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type ValidationMiddleware struct {
	*BaseMiddleware
	validator ActionValidator
}

type ActionValidator interface {
	Validate(action actions.Action) error
}

func NewValidationMiddleware(validator ActionValidator) *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: validator,
	}
}

func (m *ValidationMiddleware) Process(
	ctx context.Context,
	action actions.Action,
	next ActionHandler,
) (actions.Action, []actions.Action, error) {
	if err := m.validator.Validate(action); err != nil {
		return action, []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Validation failed: %v", err)),
		}, err
	}
	return next(ctx, action)
}
