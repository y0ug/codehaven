package validation

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
)

type DryRunValidator struct{}

func NewDryRunValidator() *DryRunValidator {
	return &DryRunValidator{}
}

func (v *DryRunValidator) Name() string { return "dryrun_validator" }

func (v *DryRunValidator) Validate(
	ctx context.Context,
	filePath string,
	original string,
	updated string,
) ValidationResult {
	result := ValidationResult{Valid: true}

	// Parse original and updated content to check syntax
	if _, err := parser.ParseFile(token.NewFileSet(), filePath, original, parser.AllErrors); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("original content syntax error: %v", err))
		result.Valid = false
	}

	if _, err := parser.ParseFile(token.NewFileSet(), filePath, updated, parser.AllErrors); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("updated content syntax error: %v", err))
		result.Valid = false
	}

	return result
}
