package extractors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/llmhaven/chat"
)

type FuncWholeFileExtractor struct {
	logger *slog.Logger
	tools  map[string]ToolerOuput[actions.Action]
}

type WriteFileInput struct {
	Explanation string `json:"explanation" jsonschema_description:"Step by step plan for the changes to be made to the code (future tense, markdown format)"`
	Filename    string `json:"filename"    jsonschema_description:"Name of the file with the path to write to"`
	Content     string `json:"content"     jsonschema_description:"Content to write to the file"`
}

func NewFuncWholeFileExtractor(
	logger *slog.Logger,
	format ExtractorType,
	fence Fence,
) *FuncWholeFileExtractor {
	c := &FuncWholeFileExtractor{
		logger: logger,
		tools:  make(map[string]ToolerOuput[actions.Action]),
	}
	tool := NewToolAction(
		"write_file",
		"Write content to a file",
		c.WriteFileHandler,
	)
	c.tools[tool.GetName()] = tool
	return c
}

func (c *FuncWholeFileExtractor) Name() string {
	return "FuncWholeFileExtractor"
}

func (c *FuncWholeFileExtractor) SupportedActions() []actions.ActionType {
	return []actions.ActionType{"apply_edit"}
}

func (c *FuncWholeFileExtractor) Type() ExtractorType {
	return TypeFunc
}

func (c *FuncWholeFileExtractor) GetChatTools() []chat.Tool {
	tools := make([]chat.Tool, 0)
	for _, tool := range c.tools {
		tools = append(tools, tool.GetChatTool())
	}
	return tools
}

func (c *FuncWholeFileExtractor) SetFence(fence Fence) {
}

func (c *FuncWholeFileExtractor) WriteFileHandler(
	ctx context.Context,
	input WriteFileInput,
) (actions.Action, error) {
	action := actions.NewApplyEdit(input.Filename, "", input.Content)
	return action, nil
}

func (c *FuncWholeFileExtractor) Extract(
	msg *chat.ChatMessage,
) ([]actions.Action, error) {
	results := make([]actions.Action, 0)
	var batchEdits []actions.BatchEdit

	for _, content := range msg.Content {
		if content.Type == chat.ContentTypeToolUse {
			action, err := c.processToolCall(content)
			if err != nil {
				c.logger.Error("Error processing tool call", "error", err)
			} else {
				// Add the edit with its tool_call_id to the batch
				batchEdits = append(batchEdits, actions.BatchEdit{
					Edit:       action.Payload.(actions.FileEditAction),
					ToolCallID: content.ID, // Preserve the tool_call_id
				})
			}
		}
	}

	// If there are multiple edits, create a single action for the batch
	if len(batchEdits) > 0 {
		batchAction := actions.NewBatchEditAction(batchEdits)
		results = append(results, batchAction)
	}

	return results, nil
}

func (tp *FuncWholeFileExtractor) processToolCall(
	content *chat.MessageContent,
) (actions.Action, error) {
	ctx := context.TODO()
	var null actions.Action
	if content.GetType() != string(chat.ContentTypeToolUse) {
		return null, fmt.Errorf("invalid tool call: no tool call data")
	}

	logger := tp.logger.With("content", content.Name)
	tool, exists := tp.tools[content.Name]
	if !exists {
		return null, fmt.Errorf("unknown tool: %s", content.Name)
	}

	logger.Debug("Tool call",
		"name", content.Name,
		"id", content.ID,
		"input", string(content.Input))

	action, err := tool.ExecuteTyped(ctx, content.Input)
	if err != nil {
		return null, fmt.Errorf("error executing tool: %w", err)
	}

	return action.WithToolCallID(content.ID), nil
}
