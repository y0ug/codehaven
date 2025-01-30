package validation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type LintValidator struct {
	command     string
	args        []string
	ignoreRules []string
	workingDir  string
}

func NewLintValidator(command string, args []string, workingDir string) *LintValidator {
	return &LintValidator{
		command:    command,
		args:       args,
		workingDir: workingDir,
	}
}

func (v *LintValidator) Name() string { return "lint_validator" }

func (v *LintValidator) Validate(
	ctx context.Context,
	filePath string,
	original string,
	updated string,
) ValidationResult {
	result := ValidationResult{Valid: true}

	// Create temp file with updated content
	tmpFile, err := os.CreateTemp("", "validate-*.go")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create temp file: %v", err))
		return result
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(updated); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to write temp file: %v", err))
		return result
	}
	tmpFile.Close()

	// Run linter command
	cmdArgs := append(v.args, tmpFile.Name())
	cmd := exec.CommandContext(ctx, v.command, cmdArgs...)
	cmd.Dir = v.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			result.Errors = append(result.Errors, parseLintOutput(output, filePath)...)
			result.Valid = false
		}
	}

	return result
}

func parseLintOutput(output []byte, originalPath string) []string {
	// Implementation depends on linter output format
	// Example for golangci-lint:
	var errors []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, originalPath) {
			errors = append(errors, line)
		}
	}
	return errors
}
