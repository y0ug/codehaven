package validation

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Validator interface {
	Validate(
		ctx context.Context,
		filePath string,
		originalContent string,
		updatedContent string,
	) ValidationResult
	AddStep(step ValidationStep)
}

type ValidationLevel int

const (
	LevelWarning ValidationLevel = iota
	LevelError
)

// Add String() method for ValidationLevel
func (l ValidationLevel) String() string {
	return [...]string{"warning", "error"}[l]
}

// Add this array for level names
var levelNames = []string{
	"warning",
	"error",
}

type ValidationIssue struct {
	Level   ValidationLevel
	Message string
	Line    int
	Column  int
}

// Add safe String() method for ValidationIssue
func (i ValidationIssue) String() string {
	if int(i.Level) >= len(levelNames) || i.Level < 0 {
		return fmt.Sprintf("unknown:%s", i.Message)
	}
	return fmt.Sprintf("%s:%s", levelNames[i.Level], i.Message)
}

type ValidationResult struct {
	Valid    bool
	Issues   []ValidationIssue
	Warnings []string
	Errors   []string
}

type ValidationStep interface {
	Validate(
		ctx context.Context,
		filePath string,
		originalContent string,
		updatedContent string,
	) ValidationResult
	Name() string
}

type ValidationPipeline struct {
	steps   []ValidationStep
	logger  *slog.Logger
	timeout time.Duration
}

// Ensure ValidationPipeline implements Validator
var _ Validator = (*ValidationPipeline)(nil)

func NewValidationPipeline(logger *slog.Logger) *ValidationPipeline {
	return &ValidationPipeline{
		steps:   make([]ValidationStep, 0),
		logger:  logger,
		timeout: 30 * time.Second,
	}
}

func (p *ValidationPipeline) AddStep(step ValidationStep) {
	p.steps = append(p.steps, step)
}

func (p *ValidationPipeline) Validate(
	ctx context.Context,
	filePath string,
	original string,
	updated string,
) ValidationResult {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	result := ValidationResult{Valid: true}

	for _, step := range p.steps {
		select {
		case <-ctx.Done():
			result.Errors = append(
				result.Errors,
				fmt.Sprintf("validation timeout for %s", step.Name()),
			)
			result.Valid = false
			return result
		default:
			stepResult := step.Validate(ctx, filePath, original, updated)
			result.Issues = append(result.Issues, stepResult.Issues...)
			result.Warnings = append(result.Warnings, stepResult.Warnings...)
			result.Errors = append(result.Errors, stepResult.Errors...)

			if !stepResult.Valid {
				result.Valid = false
			}
		}
	}

	return result
}
