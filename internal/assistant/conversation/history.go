package conversation

import (
	"sync"

	"github.com/y0ug/llmhaven/chat"
)

type ChatHistory struct {
	mu             sync.RWMutex
	curMessages    []*chat.ChatMessage
	doneMessages   []*chat.ChatMessage
	lastStopReason string
}

func NewChatHistory() *ChatHistory {
	return &ChatHistory{
		curMessages:    make([]*chat.ChatMessage, 0),
		doneMessages:   make([]*chat.ChatMessage, 0),
		lastStopReason: "",
	}
}

// Add getter and setter for lastStopReason
func (ch *ChatHistory) GetLastStopReason() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.lastStopReason
}

func (ch *ChatHistory) SetLastStopReason(reason string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.lastStopReason = reason
}

// Existing methods with added synchronization for lastStopReason
func (ch *ChatHistory) AddMessage(msg *chat.ChatMessage) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.curMessages = append(ch.curMessages, msg)
}

func (ch *ChatHistory) GetCurrentMessages() []*chat.ChatMessage {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.curMessages
}

func (ch *ChatHistory) GetDoneMessages() []*chat.ChatMessage {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.doneMessages
}

func (ch *ChatHistory) MoveCurrentToDone(responseMsg string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.doneMessages = append(ch.doneMessages, ch.curMessages...)
	ch.curMessages = make([]*chat.ChatMessage, 0)

	if responseMsg != "" {
		ch.doneMessages = append(
			ch.doneMessages,
			chat.NewMessage("user", chat.NewTextContent(responseMsg)),
			chat.NewMessage("assistant", chat.NewTextContent("Ok.")),
		)
	}
}
