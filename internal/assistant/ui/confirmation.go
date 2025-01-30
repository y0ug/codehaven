package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
)

type Confirmation struct {
	ID        string
	Message   string
	Time      time.Time
	ExpiresAt time.Time
}

type ConfirmationManager struct {
	pendingConfirmations sync.Map
	eventBus             *eventbus.EventBus
	notifications        chan Confirmation
}

func NewConfirmationManager(bus *eventbus.EventBus) *ConfirmationManager {
	cm := &ConfirmationManager{
		eventBus:      bus,
		notifications: make(chan Confirmation, 10),
	}
	cm.registerEventHandlers()
	go cm.cleanupRoutine()
	return cm
}

func (cm *ConfirmationManager) registerEventHandlers() {
	sub := cm.eventBus.Subscribe(100)
	go func() {
		for event := range sub {
			switch event.Type {
			case eventbus.EventUserInputRequest:
				cm.handleUserConfirm(event)
			case eventbus.EventUserInputResponse:
				cm.handleUserResponse(event)
			}
		}
	}()
}

func (cm *ConfirmationManager) handleUserResponse(event eventbus.Event) {
	resp := event.Payload.(eventbus.UserInputResponse)
	cm.pendingConfirmations.Delete(resp.ID)
}

func (cm *ConfirmationManager) handleUserConfirm(event eventbus.Event) {
	req := event.Payload.(eventbus.UserInputRequest)
	action := req.Action.(actions.UserInputAction)

	confirmation := Confirmation{
		ID:        req.ID,
		Message:   action.Message,
		Time:      time.Now(),
		ExpiresAt: req.ExpiresAt,
	}
	cm.pendingConfirmations.Store(req.ID, confirmation)
	select {
	case cm.notifications <- confirmation:
	default:
	}
}

func (cm *ConfirmationManager) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		cm.pendingConfirmations.Range(func(key, value interface{}) bool {
			confirmation := value.(Confirmation)
			if now.After(confirmation.ExpiresAt) {
				cm.pendingConfirmations.Delete(key)
			}
			return true
		})
	}
}

func (cm *ConfirmationManager) GetPendingConfirmations() []Confirmation {
	var confirmations []Confirmation
	now := time.Now()
	cm.pendingConfirmations.Range(func(key, value interface{}) bool {
		confirmation := value.(Confirmation)
		if !now.After(confirmation.ExpiresAt) {
			confirmations = append(confirmations, confirmation)
		}
		return true
	})
	return confirmations
}

func (cm *ConfirmationManager) GetConfirmation(id string) (Confirmation, bool) {
	if value, exists := cm.pendingConfirmations.Load(id); exists {
		confirmation := value.(Confirmation)
		if !time.Now().After(confirmation.ExpiresAt) {
			return confirmation, true
		}
	}
	return Confirmation{}, false
}

func (cm *ConfirmationManager) Notifications() <-chan Confirmation {
	return cm.notifications
}

func (cm *ConfirmationManager) RespondToConfirmation(id string, approved bool) error {
	if confirmation, exists := cm.GetConfirmation(id); !exists {
		return fmt.Errorf("no pending confirmation with ID %s", id)
	} else if time.Now().After(confirmation.ExpiresAt) {
		return fmt.Errorf("confirmation %s has expired", id)
	}

	cm.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventUserInputResponse,
		eventbus.UserInputResponse{
			ID:       id,
			Approved: approved,
		},
	))
	return nil
}
