package conversation

import (
	"log/slog"

	"github.com/y0ug/codehaven/internal/assistant/prompt/prompts"
	"github.com/y0ug/codehaven/internal/assistant/repomanager"
	"github.com/y0ug/codehaven/internal/assistant/settings"
	"github.com/y0ug/llmhaven/chat"
)

// ConversationManager manages conversation turns, summarization, and chunk building.
type ConversationManager struct {
	logger  *slog.Logger
	history *ChatHistory
	rm      repomanager.RepoManagerInterface
	// summarizer Summarizer // e.g. DefaultSummarizer or nil if summarization is off
	tokenLimit int

	prompts   prompts.Prompter
	settings  *settings.CoderSettings
	formatter *PromptFormatter

	lastMsgChunk *PromptChunks
}

// NewConversationManager creates a new ConversationManager
func NewConversationManager(
	logger *slog.Logger,
	history *ChatHistory,
	rm repomanager.RepoManagerInterface,
	// summarizer Summarizer,
	tokenLimit int,
	prompts prompts.Prompter,
	settings *settings.CoderSettings,
) *ConversationManager {
	fm := NewPromptFormatter(logger, rm, prompts, settings) // or pass RepoManager
	return &ConversationManager{
		logger:  logger,
		history: history,
		rm:      rm,
		// summarizer: summarizer,
		tokenLimit: tokenLimit,
		prompts:    prompts,
		settings:   settings,
		formatter:  fm,
	}
}

// StartTurn is called at the beginning of a new user request.
func (cm *ConversationManager) StartTurn() {
	// If the last turn ended, you might finalize it.
	if cm.history.GetLastStopReason() == "end_turn" {
		cm.logger.Info("Finishing previous turn...")
		cm.finishPreviousTurn()
	}
	// (Optionally, you could do other setup for a new turn here)
}

// finishPreviousTurn is internal: move current messages into done, clear state, etc.
func (cm *ConversationManager) finishPreviousTurn() {
	// Move current messages to done
	cm.history.MoveCurrentToDone("")
	cm.history.SetLastStopReason("")
	// Re-generate chunk or set it to nil
	cm.lastMsgChunk = nil
}

// AddUserMessage adds a user’s message to the current turn
func (cm *ConversationManager) AddUserMessage(content string) {
	userMsg := chat.NewMessage("user", chat.NewTextContent(content))
	cm.history.AddMessage(userMsg)
}

func (cm *ConversationManager) AddMessage(msg chat.ChatMessage) {
	cm.history.AddMessage(&msg)
}

// AddAssistantMessage adds an assistant message (could be final or partial).
func (cm *ConversationManager) AddAssistantMessage(content string) {
	assistantMsg := chat.NewMessage("assistant", chat.NewTextContent(content))
	cm.history.AddMessage(assistantMsg)
}

// MaybeSummarize checks if we exceed token usage and replaces older done messages with a summary
// func (cm *ConversationManager) MaybeSummarize(ctx context.Context) error {
// 	if cm.summarizer == nil {
// 		// Summarization is disabled
// 		return nil
// 	}
//
// 	totalTokens := cm.estimateTokenUsage()
// 	if totalTokens > cm.tokenLimit {
// 		cm.logger.Info("Token limit exceeded. Summarizing older messages.")
// 		done := cm.history.GetDoneMessages()
// 		if len(done) == 0 {
// 			return nil
// 		}
//
// 		summary, err := cm.summarizer.Summarize(ctx, done)
// 		if err != nil {
// 			return err
// 		}
//
// 		// Replace all doneMessages with a single summary message
// 		summarizedDone := []*chat.ChatMessage{
// 			chat.NewMessage("assistant", chat.NewTextContent(summary)),
// 		}
// 		cm.history.SetDoneMessages(summarizedDone)
// 	}
// 	return nil
// }

// BuildPrompt builds or updates cm.lastMsgChunk with the latest conversation state.
func (cm *ConversationManager) BuildPrompt() *PromptChunks {
	cm.lastMsgChunk = cm.formatter.FormatMessages(cm.history)
	return cm.lastMsgChunk
}

// EndTurn marks that the LLM has produced a final answer for this turn.
func (cm *ConversationManager) EndTurn() {
	cm.history.SetLastStopReason("end_turn")
}

func (cm *ConversationManager) SetTurn(val string) {
	cm.history.SetLastStopReason(val)
}

// estimateTokenUsage is a placeholder. In real usage, you'd do more robust counting.
func (cm *ConversationManager) estimateTokenUsage() int {
	// done := cm.history.GetDoneMessages()
	// cur := cm.history.GetCurrentMessages()
	// If your model or summarizer provides a method to count tokens, call that:
	// return cm.settings.MainModel().CountTokens(append(done, cur...))
	// or approximate:
	return 100 // approximateTokens(append(done, cur...))
}

// getOrBuildPromptChunk is sometimes convenient if you only want to build the chunk once per turn.
func (cm *ConversationManager) GetOrBuildPromptChunk() *PromptChunks {
	if cm.lastMsgChunk == nil {
		cm.lastMsgChunk = cm.formatter.FormatMessages(cm.history)
	}
	return cm.lastMsgChunk
}
