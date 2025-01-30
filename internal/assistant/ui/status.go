package ui

import (
	"sync/atomic"

	"github.com/y0ug/codehaven/internal/assistant/eventbus"
)

type StatusManager struct {
	value atomic.Value
}

const (
	StatusReady      = "ready"
	StatusProcessing = "processing"
	StatusError      = "error"
	StatusWaiting    = "waiting"
)

func NewStatusManager(initial string) *StatusManager {
	s := &StatusManager{}
	s.value.Store(initial)
	return s
}

func (s *StatusManager) Current() string {
	return s.value.Load().(string)
}

func (s *StatusManager) Update(newStatus string) {
	s.value.Store(newStatus)
	// Automatic status event generation
	eventbus.GetEventBus().Publish(eventbus.NewEvent(
		eventbus.EventStatusUpdate,
		eventbus.StatusUpdate{Old: s.Current(), New: newStatus},
	))
}
