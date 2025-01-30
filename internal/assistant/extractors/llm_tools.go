package extractors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/llmhaven/chat"
)

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	return reflector.Reflect(v)
}

func StrToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type ToolerOuput[O any] interface {
	ExecuteTyped(ctx context.Context, input json.RawMessage) (O, error)
	GetName() string
	GetChatTool() chat.Tool
}

type Tooler interface {
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
	GetName() string
	GetChatTool() chat.Tool
}

type ToolHandler[T any, O any] func(ctx context.Context, input T) (O, error)

type ToolBase[T any, O any] struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	InputSchema interface{}       `json:"input_schema,omitempty"`
	handler     ToolHandler[T, O] `json:"-"`
}

func (t ToolBase[T, O]) GetName() string {
	return t.Name
}

func (t ToolBase[T, O]) GetChatTool() chat.Tool {
	return chat.Tool{
		Name:        t.Name,
		Description: StrToPtr(t.Description),
		InputSchema: t.InputSchema,
	}
}

func (t ToolBase[T, O]) ExecuteTyped(
	ctx context.Context,
	input json.RawMessage,
) (O, error) {
	var args T
	var out O
	err := json.Unmarshal(input, &args)
	if err != nil {
		return out, fmt.Errorf("%s failed to unmarshal input: %w", t.Name, err)
	}

	output, err := t.handler(ctx, args)
	if err != nil {
		return out, fmt.Errorf("%s failed to execute: %w", t.Name, err)
	}
	return output, nil
}

func (t ToolBase[T, O]) Execute(
	ctx context.Context,
	input json.RawMessage,
) (json.RawMessage, error) {
	var args T
	err := json.Unmarshal(input, &args)
	if err != nil {
		return nil, fmt.Errorf("%s failed to unmarshal input: %w", t.Name, err)
	}

	output, err := t.handler(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("%s failed to execute: %w", t.Name, err)
	}
	result, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("%s failed to marshal output: %w", t.Name, err)
	}
	return result, nil
}

func NewToolAction[T any, O actions.Action](
	name string,
	description string,
	handler ToolHandler[T, O],
) *ToolBase[T, O] {
	inputSchema := GenerateSchema[T]()
	return &ToolBase[T, O]{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		handler:     handler,
	}
}
