package extractors

import (
	"fmt"
	"log/slog"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/llmhaven/chat"
)

type (
	ExtractorType string
)

type ExtractorError struct {
	Name          string
	ExtractorType string
	Err           error
	Line          int
}

func (e *ExtractorError) Error() string {
	return fmt.Sprintf("%s (line %d) - %s %+w", e.Name, e.Line, e.ExtractorType, e.Err)
}

func NewExtractorError(
	name string,
	extractorType ExtractorType,
	err error,
	line int,
) *ExtractorError {
	return &ExtractorError{
		Name:          name,
		ExtractorType: string(extractorType),
		Err:           err,
		Line:          line,
	}
}

const (
	TypeDiff       ExtractorType = "block-diff"
	TypeDiffFenced ExtractorType = "block-diff-fenced"
	TypeFunc       ExtractorType = "func-whole"
)

type Extractor interface {
	Extract(msg *chat.ChatMessage) ([]actions.Action, error)
	GetChatTools() []chat.Tool
	SetFence(Fence)
	Name() string
	Type() ExtractorType
	SupportedActions() []actions.ActionType
}

func New(editFormat ExtractorType, logger *slog.Logger) Extractor {
	fence := DefaultFences[0]
	switch editFormat {
	case TypeDiff:
		return NewBlockExtractor(logger, editFormat, fence)
	case TypeDiffFenced:
		return NewBlockExtractor(logger, editFormat, fence)
	case TypeFunc:
		return NewFuncWholeFileExtractor(logger, editFormat, fence)
	default:
		return nil
	}
}

type Fence [2]string

// DefaultFences defines all possible fencing options in order of preference
var DefaultFences = []Fence{
	{"```", "```"},
	{"````", "````"},
	{"<source>", "</source>"},
	{"<code>", "</code>"},
	{"<pre>", "</pre>"},
	{"<codeblock>", "</codeblock>"},
	{"<sourcecode>", "</sourcecode>"},
}
