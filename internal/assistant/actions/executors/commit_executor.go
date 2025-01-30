package executors

import (
	"context"
	"fmt"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/repomanager"
)

// Create new commit_executor.go
type CommitExecutor struct {
	repo repomanager.RepoManagerInterface
}

func NewCommitExecutor(repo repomanager.RepoManagerInterface) *CommitExecutor {
	return &CommitExecutor{repo: repo}
}

func (e *CommitExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.CommitAction)
	return ok
}

func (e *CommitExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	logger := actions.GetLogger(ctx)

	commitAction := action.Payload.(actions.CommitAction)

	hash, msg, err := e.repo.AutoCommit(ctx)
	if err != nil {
		return action, []actions.Action{
			actions.NewLogAction(&action, fmt.Sprintf("Commit failed: %v", err)),
		}, err
	}

	logger.Debug("commit", "message", msg, "hash", hash)

	commitAction.Message = msg
	commitAction.Hash = hash

	action.Payload = commitAction
	return action, []actions.Action{
		actions.NewLogAction(&action, fmt.Sprintf("Changes committed successfully %q", msg)),
	}, nil
}
