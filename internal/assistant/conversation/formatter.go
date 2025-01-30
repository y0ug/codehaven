package conversation

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/codehaven/internal/assistant/prompt/prompts"
	"github.com/y0ug/codehaven/internal/assistant/repomanager"
	"github.com/y0ug/codehaven/internal/assistant/settings"
	"github.com/y0ug/llmhaven/chat"
)

type PromptFormatter struct {
	logger          *slog.Logger
	templateHandler *prompts.TemplateHandler
	rm              repomanager.RepoManagerInterface
	prompts         prompts.Prompter
	settings        *settings.CoderSettings
}

func (mf *PromptFormatter) FormatMessages(history *ChatHistory) *PromptChunks {
	mf.rm.ChooseFence()

	mf.settings.Update(mf.rm.GetGit() != nil, mf.rm.GetRoot(), mf.rm.GetFence())
	chunks := &PromptChunks{}

	// Add system messages
	exampleMessages := make([]*chat.ChatMessage, 0)

	mainSystem := mf.RenderPrompt(mf.prompts.GetMainSystem())
	if mf.settings.MainModel().ExamplesAsSysMsg {
		if len(mf.prompts.GetExampleMessages()) > 0 {
			mainSystem += "\n# Examples conversations:\n\n"
		}
		for _, msg := range mf.prompts.GetExampleMessages() {
			role := strings.ToUpper(msg.Role)
			content := mf.RenderPrompt(msg.Content)
			mainSystem += fmt.Sprintf("## %s: %s\n\n", role, content)
		}
	} else {
		for _, msg := range mf.prompts.GetExampleMessages() {
			msg.Content = mf.RenderPrompt(msg.Content)
			exampleMessages = append(exampleMessages, msg.ToChatMessage())
		}

		if len(exampleMessages) > 0 {
			msg := []*chat.ChatMessage{
				chat.NewMessage("user", chat.NewTextContent("I switched to a new code base. Please don't consider the above files  or try to edit them any longer")),
				chat.NewMessage("assistant", chat.NewTextContent("Ok.")),
			}
			exampleMessages = append(exampleMessages, msg...)
		}
	}
	systemReminder := mf.RenderPrompt(mf.prompts.GetSystemReminder())
	if len(systemReminder) > 0 {
		mainSystem += "\n" + systemReminder
	}

	if mf.settings.MainModel().UseSystemPrompt {
		chunks.System = []*chat.ChatMessage{
			chat.NewMessage("system", chat.NewTextContent(mainSystem)),
		}
	} else {
		chunks.System = []*chat.ChatMessage{
			chat.NewMessage("user", chat.NewTextContent(mainSystem)),
			chat.NewMessage("assistant", chat.NewTextContent("Ok.")),
		}
	}

	chunks.Examples = exampleMessages

	// Summarize end call

	chunks.Done = history.GetDoneMessages()
	chunks.Repo = mf.GetRepoMessages()
	chunks.ReadOnlyFiles = mf.GetReadOnlyFilesMessages()
	chunks.ChatFiles = mf.GetChatFilesMessages()
	chunks.Cur = history.GetCurrentMessages()

	var finalMessage *chat.ChatMessage
	if len(chunks.Cur) > 0 {
		finalMessage = chunks.Cur[len(chunks.Cur)-1]
	}

	// msgTokens := mf.settings.MainModel().TokenCount(chunks.AllMessages())
	// reminderTokens := mf.settings.MainModel().TokenCount(reminderMessage)
	// curTokens := mf.settings.MainModel().TokenCount(chunks.Cur)
	// totalTokens := msgTokens + reminderTokens + curTokens
	// if totalTokens < maxInputTokens and len(systemReminder)
	// Count if we have enought token to add the reminder
	if len(systemReminder) > 0 {
		if mf.settings.MainModel().Reminder == "sys" {
			chunks.Reminder = []*chat.ChatMessage{
				chat.NewMessage("system", chat.NewTextContent(systemReminder)),
			}
		} else if mf.settings.MainModel().Reminder == "user" && finalMessage != nil && finalMessage.Role == "user" {
			finalMessage.Content[0].Text = fmt.Sprintf("%s\n\n%s", finalMessage.Content[0].Text, systemReminder)
		}
	}

	chunks.AddCacheControlHeaders()
	return chunks
}

func NewPromptFormatter(logger *slog.Logger, rm repomanager.RepoManagerInterface,
	p prompts.Prompter, settings *settings.CoderSettings,
) *PromptFormatter {
	return &PromptFormatter{
		logger:          logger,
		rm:              rm,
		prompts:         p,
		settings:        settings,
		templateHandler: prompts.NewTemplateHandler(p, settings, logger),
	}
}

func (mf *PromptFormatter) RenderPrompt(tmpl string) string {
	return mf.templateHandler.Render(tmpl)
}

func (mf *PromptFormatter) RenderPromptData(tmpl string, data map[string]interface{}) string {
	return mf.templateHandler.RenderData(tmpl, data)
}

func (mf *PromptFormatter) GetRepoMessages() []*chat.ChatMessage {
	repoContent := mf.rm.GetRepoMap()
	if repoContent == "" {
		return nil
	}

	return []*chat.ChatMessage{
		chat.NewMessage("user", chat.NewTextContent(repoContent)),
		chat.NewMessage(
			"assistant",
			chat.NewTextContent("Ok, I won't try and edit those files without asking first."),
		),
	}
}

func (mf *PromptFormatter) GetReadOnlyFilesMessages() []*chat.ChatMessage {
	content := mf.rm.GetReadOnlyFilesContent()
	if content == "" {
		return nil
	}

	return []*chat.ChatMessage{
		chat.NewMessage(
			"user",
			chat.NewTextContent(mf.RenderPrompt(mf.prompts.GetReadOnlyFilesPrefix())+"\n"+content),
		),
		chat.NewMessage(
			"assistant",
			chat.NewTextContent("Ok, I will use these files as references."),
		),
	}
}

func (mf *PromptFormatter) GetChatFilesMessages() []*chat.ChatMessage {
	if len(mf.rm.ListFiles(0)) == 0 {
		if mf.rm.GetRepoMap() != "" && mf.prompts.GetFilesNoFullFilesWithRepoMap() != "" {
			return []*chat.ChatMessage{
				chat.NewMessage(
					"user",
					chat.NewTextContent(
						mf.RenderPrompt(mf.prompts.GetFilesNoFullFilesWithRepoMap()),
					),
				),
				chat.NewMessage(
					"assistant",
					chat.NewTextContent(
						mf.RenderPrompt(mf.prompts.GetFilesNoFullFilesWithRepoMapReply()),
					),
				),
			}
		}
		return []*chat.ChatMessage{
			chat.NewMessage(
				"user",
				chat.NewTextContent(mf.RenderPrompt(mf.prompts.GetFilesNoFullFiles())),
			),
			chat.NewMessage("assistant", chat.NewTextContent("Ok.")),
		}
	}

	content := mf.RenderPrompt(mf.prompts.GetFilesContentPrefix()) + "\n" + mf.rm.GetFilesContent()

	return []*chat.ChatMessage{
		chat.NewMessage("user", chat.NewTextContent(content)),
		chat.NewMessage(
			"assistant",
			chat.NewTextContent(mf.RenderPrompt(mf.prompts.GetFilesContentAssistantReply())),
		),
	}
}
