package eventbus

import (
	"time"
)

type EventType int

const (
	EventInput EventType = iota
	EventError
	EventOutput
	EventStatusUpdate
	EventFileNotification
	EventAddFile
	EventRemoveFile
	EventShutdown
	EventUserInputRequest
	EventUserInputResponse
)

type Event struct {
	Type     EventType
	Payload  interface{}
	Metadata map[string]string
}

// Generic event constructor
func NewEvent(t EventType, payload interface{}) Event {
	return Event{
		Type:    t,
		Payload: payload,
		Metadata: map[string]string{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"source":    "source", // runtime.Caller(2).Function,
		},
	}
}

type StatusUpdate struct {
	Old string
	New string
}

type UserInput struct {
	Source  string
	Content string
}

type FileOperation struct {
	Type     FileOperationType
	Files    []string
	ReadOnly bool
}

type FileOperationType string

var (
	FileOperationTypeAdd    FileOperationType = "add"
	FileOperationTypeRemove FileOperationType = "remove"
)

type UserInputRequest struct {
	ID        string
	Action    interface{}
	ExpiresAt time.Time
}

type UserInputResponse struct {
	ID       string
	Approved bool
	Input    string
}
