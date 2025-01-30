package validation

import (
	"context"
	"fmt"
	"os/exec"
)

type TestValidator struct {
	testCommand string
	testArgs    []string
	workingDir  string
}

func NewTestValidator(workingDir string) *TestValidator {
	return &TestValidator{
		testCommand: "go",
		testArgs:    []string{"test", "./..."},
		workingDir:  workingDir,
	}
}

func (v *TestValidator) Name() string { return "test_validator" }

func (v *TestValidator) Validate(
	ctx context.Context,
	filePath string,
	original string,
	updated string,
) ValidationResult {
	result := ValidationResult{Valid: true}

	// Run tests
	cmd := exec.CommandContext(ctx, v.testCommand, v.testArgs...)
	cmd.Dir = v.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("tests failed: %v\n%s", err, output))
		result.Valid = false
	}

	return result
}
