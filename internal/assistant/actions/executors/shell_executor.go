package executors

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"syscall"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type ShellExecutor struct {
	safePatterns   []*regexp.Regexp
	logger         *slog.Logger
	commandTimeout time.Duration
}

func NewShellExecutor(
	safePatterns []string,
	logger *slog.Logger,
) *ShellExecutor {
	compiled := make([]*regexp.Regexp, len(safePatterns))
	for i, p := range safePatterns {
		compiled[i] = regexp.MustCompile(p)
	}

	return &ShellExecutor{
		safePatterns:   compiled,
		logger:         logger,
		commandTimeout: 2 * time.Minute,
	}
}

func (s *ShellExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.ShellCommandAction)
	return ok
}

func (s *ShellExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actions.Action, []actions.Action, error) {
	cmdAction := action.Payload.(actions.ShellCommandAction)

	if !s.isCommandAllowed(cmdAction.Command) {
		return action, []actions.Action{
			actions.NewLogAction(
				&action,
				fmt.Sprintf("Blocked unsafe command: %s", cmdAction.Command),
			),
		}, fmt.Errorf("unsafe command")
	}

	if cmdAction.NeedsConfirm && !cmdAction.Confirmed {
		return action, []actions.Action{
			actions.NewUserInput(action,
				actions.ConfirmInput,
				"Do you want to run this?").
				WithParent(&action),
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, s.commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", cmdAction.Command)
	output, err := cmd.CombinedOutput()

	// Update action state
	cmdAction.Executed = true
	cmdAction.Output = string(output)
	cmdAction.ExitCode = cmd.ProcessState.ExitCode()
	cmdAction.Success = (cmdAction.ExitCode == 0)

	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			cmdAction.ExitCode = status.ExitStatus()
		}
	}

	action.Payload = cmdAction
	return action, []actions.Action{
		actions.NewLogAction(&action, fmt.Sprintf("Command output:\n%s", string(output))),
	}, err
}

func (s *ShellExecutor) isCommandAllowed(cmd string) bool {
	for _, pattern := range s.safePatterns {
		if pattern.MatchString(cmd) {
			return true
		}
	}
	return false
}
