package executors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/repomanager"
	"github.com/y0ug/codehaven/internal/assistant/validation"
	"github.com/y0ug/llmhaven/chat"
)

type EditExecutor struct {
	repo      repomanager.RepoManagerInterface
	validator validation.Validator
	logger    *slog.Logger
}

func NewEditExecutor(
	repo repomanager.RepoManagerInterface,
	validator validation.Validator,
	logger *slog.Logger,
) *EditExecutor {
	return &EditExecutor{
		repo:      repo,
		validator: validator,
		logger:    logger,
	}
}

func (e *EditExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.FileEditAction)
	_, okBatch := action.Payload.(actions.BatchEditAction)
	return ok || okBatch
}

func (e *EditExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	switch v := action.Payload.(type) {
	case actions.FileEditAction:
		return e.handleSingleEdit(ctx, action, v)
	case actions.BatchEditAction:
		return e.handleBatchEdit(ctx, action, v)
	default:
		return action, nil, fmt.Errorf("unsupported edit type")
	}
}

func (e *EditExecutor) handleSingleEdit(
	ctx context.Context,
	action actions.Action,
	edit actions.FileEditAction,
) (actions.Action, []actions.Action, error) {
	var followUps []actions.Action

	validationResult := e.validator.Validate(ctx, edit.Filename, edit.Original, edit.Updated)
	if !validationResult.Valid {
		for _, issue := range validationResult.Issues {
			followUps = append(followUps, actions.NewLogAction(&action,
				fmt.Sprintf("Validation %s: %s (line %d)",
					strings.ToLower(issue.Level.String()),
					issue.Message,
					issue.Line,
				),
			))
		}
		if action.IsToolCall() {
			followUps = append(
				followUps,
				NewActionAddMsgToolResult(
					action.Context.ToolCallID,
					"Edit rejected due to validation errors",
				).WithParent(&action),
			)
		}
		followUps = append(
			followUps,
			actions.NewLogAction(&action, "Edit rejected due to validation errors"),
		)
		return action, followUps, fmt.Errorf("validation failed")
	}

	if err := e.repo.ApplyEdit(edit); err != nil {
		if action.IsToolCall() {
			followUps = append(
				followUps,
				NewActionAddMsgToolResult(
					action.Context.ToolCallID,
					"failed to apply edit",
				).WithParent(&action),
			)
		}

		followUps = append(
			followUps,
			actions.NewLogAction(&action, fmt.Sprintf("Edit failed: %v", err)),
		)
		return action, followUps, fmt.Errorf("failed to apply edit: %w", err)
	}

	if action.IsToolCall() {
		followUps = append(
			followUps,
			NewActionAddMsgToolResult(
				action.Context.ToolCallID,
				"Edit applied successfully",
			).WithParent(&action),
		)
	}
	followUps = append(
		followUps,
		actions.NewCommitAction(&action, fmt.Sprintf("Applied edit to %s", edit.Filename)),
		actions.NewLogAction(&action, "Edit applied successfully"),
	)
	return action, followUps, nil
}

func (e *EditExecutor) handleBatchEdit(
	ctx context.Context,
	action actions.Action,
	batch actions.BatchEditAction,
) (actions.Action, []actions.Action, error) {
	var followUps []actions.Action

	// logger := actions.GetLogger(ctx)
	chatMsg := chat.NewMessage("tool")
	// if chatMsg.Content == nil {
	// 	chatMsg.Content = make([]*chat.MessageContent, 0)
	// }

	// Process each edit in the batch
	for _, batchEdit := range batch.Edits {

		edit := batchEdit.Edit
		toolCallID := batchEdit.ToolCallID

		validationResult := e.validator.Validate(ctx, edit.Filename, edit.Original, edit.Updated)
		if !validationResult.Valid {
			for _, issue := range validationResult.Issues {
				followUps = append(followUps, actions.NewLogAction(&action,
					fmt.Sprintf("Validation %s: %s (line %d)",
						strings.ToLower(issue.Level.String()),
						issue.Message,
						issue.Line,
					),
				))
			}
			msg := "Edit rejected due to validation errors"
			if toolCallID != "" {
				chatMsg.Content = append(chatMsg.Content,
					chat.NewToolResultContent(toolCallID, msg),
				)

				followUps = append(
					followUps,
					actions.NewAddMessageAction(*chatMsg).
						WithParent(&action),
				)
			}
			followUps = append(
				followUps,
				actions.NewLogAction(&action, msg),
			)
			return action, followUps, fmt.Errorf("validation failed")
		}

		if err := e.repo.ApplyEdit(edit); err != nil {
			msg := fmt.Sprintf("Edit failed: %v", err)
			if toolCallID != "" {
				chatMsg.Content = append(chatMsg.Content,
					chat.NewToolResultContent(toolCallID, msg),
				)

				followUps = append(
					followUps,
					actions.NewAddMessageAction(*chatMsg).
						WithParent(&action).
						WithToolCallID(toolCallID),
				)
			}
			followUps = append(
				followUps,
				actions.NewLogAction(&action, msg),
			)
			return action, followUps, fmt.Errorf("failed to apply edit: %w", err)
		}

		// Respond to the tool call
		msg := fmt.Sprintf("Edit applied successfully")
		if toolCallID != "" {
			chatMsg.Content = append(chatMsg.Content,
				chat.NewToolResultContent(toolCallID, msg),
			)
		}

	}

	if len(chatMsg.Content) > 0 {
		followUps = append(
			followUps,
			actions.NewAddMessageAction(*chatMsg).WithParent(&action),
		)
	}

	// Commit all edits
	followUps = append(
		followUps,
		actions.NewCommitAction(&action, ""),
	// actions.NewLogAction(&action, "Batch edits applied successfully"),
	)
	return action, followUps, nil
}

func NewActionAddMsgToolResult(toolCallID string, content string) actions.Action {
	return actions.NewAddMessageAction(
		*chat.NewMessage("tool",
			chat.NewToolResultContent(
				toolCallID,
				content,
			)))
}

func NewToolResult(toolCallID string, content string) actions.Action {
	return actions.NewAddMessageAction(
		*chat.NewMessage("tool",
			chat.NewToolResultContent(
				toolCallID,
				content,
			)))
}
