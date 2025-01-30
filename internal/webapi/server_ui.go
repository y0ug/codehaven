package webapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/assistant/ui"
)

var _ ui.UserInterface = &WebServer{}

func (u *WebServer) Publish(event eventbus.Event) error {
	return ui.Publish(u, event)
}

func (u *WebServer) Output(event eventbus.Event) (err error) {
	_, ok := event.Payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebServer) Error(event eventbus.Event) (err error) {
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *WebServer) Status(event eventbus.Event) (err error) {
	_, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebServer) RequestConfirmation(event eventbus.Event) (response eventbus.Event, err error) {
	request, ok := event.Payload.(eventbus.UserInputRequest)
	if !ok {
		return response, fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)

	responseChan := make(chan bool)
	defer close(responseChan)

	select {
	case approved := <-responseChan:
		response = eventbus.NewEvent(
			eventbus.EventUserInputResponse,
			eventbus.UserInputResponse{
				ID:       request.ID,
				Approved: approved,
			},
		)
		return response, nil
	case <-time.After(2 * time.Minute):
		return response, errors.New("confirmation timeout")
	}
}

func (u *WebServer) FileNotification(event eventbus.Event) (err error) {
	_, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebServer) Shutdown(event eventbus.Event) (err error) {
	u.broadcast(event)
	return nil
}

func (w *WebServer) broadcast(event interface{}) {
	w.clients.Range(func(key, value interface{}) bool {
		if client, ok := key.(*WSClient); ok {
			if data, err := json.Marshal(event); err == nil {
				client.send <- data
			}
		}
		return true
	})
}
