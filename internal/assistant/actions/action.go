package actions

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/y0ug/llmhaven/chat"
)

type ActionType string

const (
	ActionTypeExtractor     ActionType = "extractor"
	ActionTypeFileEdit      ActionType = "file_edit"
	ActionTypeFileBatchEdit ActionType = "file_batch_edit"
	ActionTypeShellCommand  ActionType = "shell_command"
	ActionTypeUserInput     ActionType = "user_input"
	ActionTypeUserResponse  ActionType = "user_response"
	ActionTypeCommit        ActionType = "commit"
	ActionTypeLog           ActionType = "log"
	ActionTypeLLMRequest    ActionType = "llm_request"
	ActionTypeLLMResponse   ActionType = "llm_response"
	ActionTypeAddMessage    ActionType = "add_message"
)

type ActionContext struct {
	ToolCallID string
	ParentID   uuid.UUID
	ChainID    uuid.UUID
	CreatedAt  time.Time
}

type ActionResult struct {
	Success  bool
	Output   string
	Error    error
	Metadata map[string]interface{}
	Payload  interface{}
}

type Action struct {
	ID      uuid.UUID
	Type    ActionType
	Payload interface{}
	Context ActionContext
	Result  *ActionResult
}

func (a *Action) IsToolCall() bool {
	return a.Context.ToolCallID != ""
}

type FileEditType string

var (
	CreateFile FileEditType = "create"
	RemoveFile FileEditType = "remove"
	AppendFile FileEditType = "append"
	PatchFile  FileEditType = "patch"
)

type FileEditAction struct {
	Filename string
	Original string
	Updated  string
	Type     FileEditType
}

type ShellCommandAction struct {
	Command      string
	NeedsConfirm bool
	Confirmed    bool
	Output       string
	Executed     bool // Add execution state
	Success      bool // Add success stat
	ExitCode     int  // Add exit code
}

func (s ShellCommandAction) WithConfirmed(parentAction *Action) Action {
	s.Confirmed = true
	return NewAction(ActionTypeShellCommand, s).WithParent(parentAction)
}

type UserInputType string

var (
	ConfirmInput UserInputType = "confirm"
	MessageInput UserInputType = "message"
)

type UserInputAction struct {
	Message      string
	Type         UserInputType
	ParentAction Action
	Context      ActionContext
}

type UserResponseAction struct {
	Allowed bool
	Input   string
	Type    UserInputType
	Context ActionContext
}

type CommitAction struct {
	Message string
	Hash    string
}

type LogAction struct {
	Message string
}

func (a Action) String() string {
	// Add indication of completion status
	status := ""
	if a.Result != nil && a.Result.Success {
		status = "✓ "
	}
	switch v := a.Payload.(type) {
	case BatchEditAction:
		return fmt.Sprintf("%sBATCHEDIT: %s", status, v.String())
	case FileEditAction:
		return fmt.Sprintf("%sEDIT %s: %q → %q", status,
			v.Filename, shorten(v.Original, DefaultShorten), shorten(v.Updated, DefaultShorten))
	case CommitAction:
		return fmt.Sprintf("%sCOMMIT: %s", status, v.Message)
	case ShellCommandAction:
		return fmt.Sprintf("%sCMD: %s (confirmed:%v)", status,
			v.Command, v.Confirmed)
	case LogAction:
		return fmt.Sprintf("%sLOG: %s", status, v.Message)
	default:
		return fmt.Sprintf("%sACTION -%s", status, a.Type)
	}
}

func (a Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(a.Type)),
		slog.String("action_id", shorten(a.ID.String(), DefaultShorten)),
		slog.String("chain_id", shorten(a.Context.ChainID.String(), DefaultShorten)),
		// slog.String("parent_id", a.Context.ParentID.String()),
		// slog.String("tool_call_id", a.Context.ToolCallID),
		// slog.String("created_at", a.Context.CreatedAt.Format(time.RFC3339)),
	)
}

// New helper method to maintain chain IDs

func (a Action) WithToolCallID(id string) Action {
	a.Context.ToolCallID = id
	return a
}

func (a Action) WithParent(parent *Action) Action {
	if parent != nil {
		a.Context.ChainID = parent.Context.ChainID
		a.Context.ParentID = parent.ID
		// TODO: check if we want this
		// we only copy the tool call ID if the parent is a tool call
		if parent.IsToolCall() {
			a.Context.ToolCallID = parent.Context.ToolCallID
		}
	}
	return a
}

var DefaultShorten = 20

func shorten(s string, n int) string {
	if len(s) > n {
		return s[:n-3] + "..."
	}
	return s
}

// Helper to convert an action to an Array of actions
func Slice(action ...Action) []Action {
	return action
}

func NewAction(actionType ActionType, payload interface{}) Action {
	return Action{
		ID:      uuid.New(),
		Type:    actionType,
		Payload: payload,
		Context: ActionContext{
			ChainID:   uuid.New(),
			CreatedAt: time.Now(),
		},
	}
}

func NewApplyEdit(filename, original, updated string) Action {
	return NewAction(ActionTypeFileEdit, FileEditAction{
		Filename: filename,
		Original: original,
		Updated:  updated,
	})
}

func NewShellCommand(command string, needsConfirm bool) Action {
	return NewAction(ActionTypeShellCommand, ShellCommandAction{
		Command:      command,
		NeedsConfirm: needsConfirm,
	})
}

func NewUserInput(parentAction Action, inputType UserInputType, message string) Action {
	return NewAction(ActionTypeUserInput, UserInputAction{
		Message:      message,
		Type:         inputType,
		ParentAction: parentAction,
		Context:      parentAction.Context,
	})
}

func NewUserResponseConfirm(parentAction Action, isAllowed bool) Action {
	return NewAction(ActionTypeUserResponse, UserResponseAction{
		Allowed: isAllowed,
		Type:    ConfirmInput,
		Context: parentAction.Context,
	})
}

func NewUserInputResponse(
	parentAction Action,
	inputType UserInputType,
	isAllowed bool,
	input string,
) Action {
	return NewAction(ActionTypeUserResponse, UserResponseAction{
		Allowed: isAllowed,
		Input:   input,
		Type:    inputType,
		Context: parentAction.Context,
	})
}

func NewCommitAction(parentAction *Action, message string) Action {
	return NewAction(ActionTypeCommit, CommitAction{
		Message: message,
	}).WithParent(parentAction)
}

func NewLogAction(parentAction *Action, message string) Action {
	return NewAction(ActionTypeLog, LogAction{
		Message: message,
	}).WithParent(parentAction)
}

type LLMRequestAction struct {
	Prompt string
}
type LLMResponseAction struct {
	Resp chat.ChatResponse
}

type AddMessageAction struct {
	Msg chat.ChatMessage
}

func NewLLMRequestAction(prompt string) Action {
	return NewAction(ActionTypeLLMRequest, LLMRequestAction{Prompt: prompt})
}

func NewLLMResponseAction(resp chat.ChatResponse) Action {
	return NewAction(ActionTypeLLMResponse, LLMResponseAction{Resp: resp})
}

func NewAddMessageAction(msg chat.ChatMessage) Action {
	return NewAction(ActionTypeAddMessage, AddMessageAction{Msg: msg})
}

type BatchEditAction struct {
	Edits []BatchEdit // Each edit now includes its tool_call_id
}

type BatchEdit struct {
	Edit       FileEditAction
	ToolCallID string // Track the tool_call_id for each edit
}

func NewBatchEditAction(edits []BatchEdit) Action {
	return NewAction(ActionTypeFileBatchEdit, BatchEditAction{Edits: edits})
}

func (b BatchEditAction) String() string {
	filenames := []string{}
	for _, edit := range b.Edits {
		filenames = append(filenames, edit.Edit.Filename)
	}

	return fmt.Sprintf("[%s]", strings.Join(filenames, ", "))
}

type ActionExtractor struct {
	Msg chat.ChatMessage
}

func NewActionExtractor(msg chat.ChatMessage) Action {
	return NewAction(ActionTypeExtractor, ActionExtractor{Msg: msg})
}
