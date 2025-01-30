package prompts

import (
	"strings"

	"github.com/y0ug/llmhaven/chat"
)

const BlockFence = "```"

func New(name string) Prompter {
	switch strings.ToLower(name) {
	case "architect":
		return NewArchitectPrompts()
	case "ask":
		return NewAskPrompts()
	case "editblock":
		return NewEditBlockPrompts()
	case "editblockfenced":
		return NewEditBlockFencedPrompts()
	case "editblockfunction":
		return NewEditBlockFunctionPrompts()
	case "editoreditblock":
		return NewEditorEditBlockPrompts()
	case "editorwholefile":
		return NewEditorWholeFilePrompts()
	case "help":
		return NewHelpPrompts()
	case "singlewholefilefunction":
		return NewSingleWholeFileFunctionPrompts()
	case "wholefile":
		return NewWholeFilePrompts()
	default:
		return nil
	}
}

// BasePrompts represents the base prompts configuration
// A interface is generated from this struct
// It's should contains all the fields that all the prompts needs
type BasePrompts struct {
	Name                             string    `yaml:"name"`
	SystemReminder                   string    `yaml:"system_reminder"`
	FilesContentGPTEdits             string    `yaml:"files_content_gpt_edits"`
	FilesContentGPTEditsNoRepo       string    `yaml:"files_content_gpt_edits_no_repo"`
	FilesContentGPTNoEdits           string    `yaml:"files_content_gpt_no_edits"`
	FilesContentLocalEdits           string    `yaml:"files_content_local_edits"`
	LazyPrompt                       string    `yaml:"lazy_prompt"`
	ExampleMessages                  []Message `yaml:"example_messages"`
	FilesContentPrefix               string    `yaml:"files_content_prefix"`
	FilesContentAssistantReply       string    `yaml:"files_content_assistant_reply"`
	FilesNoFullFiles                 string    `yaml:"files_no_full_files"`
	FilesNoFullFilesWithRepoMap      string    `yaml:"files_no_full_files_with_repo_map"`
	FilesNoFullFilesWithRepoMapReply string    `yaml:"files_no_full_files_with_repo_map_reply"`
	RepoContentPrefix                string    `yaml:"repo_content_prefix"`
	ReadOnlyFilesPrefix              string    `yaml:"read_only_files_prefix"`
	ShellCmdPrompt                   string    `yaml:"shell_cmd_prompt"`
	ShellCmdReminder                 string    `yaml:"shell_cmd_reminder"`
	NoShellCmdPrompt                 string    `yaml:"no_shell_cmd_prompt"`
	NoShellCmdReminder               string    `yaml:"no_shell_cmd_reminder"`
	MainSystem                       string    `yaml:"main_system"`
	RedactedEditMessage              string    `yaml:"redacted_edit_message"`
	EditFormat                       string    `yaml:"edit_format"`
}

// ArchitectPrompts represents the architect-specific prompts
type ArchitectPrompts struct {
	BasePrompts
}

// AskPrompts represents the ask-specific prompts
type AskPrompts struct {
	BasePrompts
}

// EditBlockPrompts represents the editblock-specific prompts
type EditBlockPrompts struct {
	BasePrompts
}

// EditBlockFencedPrompts represents the editblock-fenced-specific prompts
type EditBlockFencedPrompts struct {
	EditBlockPrompts
}

// EditBlockFunctionPrompts represents the editblock-function-specific prompts
type EditBlockFunctionPrompts struct {
	BasePrompts
}

// EditorEditBlockPrompts represents the editor-editblock-specific prompts
type EditorEditBlockPrompts struct {
	EditBlockPrompts
}

// EditorWholeFilePrompts represents the editor-wholefile-specific prompts
type EditorWholeFilePrompts struct {
	WholeFilePrompts
}

// HelpPrompts represents the help-specific prompts
type HelpPrompts struct {
	BasePrompts
}

// SingleWholeFileFunctionPrompts represents the single-wholefile-function-specific prompts
type SingleWholeFileFunctionPrompts struct {
	BasePrompts
}

// WholeFilePrompts represents the wholefile-specific prompts
type UdiffPrompts struct {
	BasePrompts
}

// WholeFilePrompts represents the wholefile-specific prompts
type WholeFilePrompts struct {
	BasePrompts
}

// Message represents a chat message
type Message struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

func (m *Message) ToChatMessage() *chat.ChatMessage {
	return chat.NewMessage(m.Role, chat.NewTextContent(m.Content))
}
