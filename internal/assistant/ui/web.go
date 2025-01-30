package ui

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
)

type WebUI struct {
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]bool
	mu       sync.Mutex
}

var _ UserInterface = &WebUI{}

func NewWebUI() *WebUI {
	return &WebUI{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Adjust as needed
		},
		clients: make(map[*websocket.Conn]bool),
	}
}

func (u *WebUI) Publish(event eventbus.Event) error {
	return Publish(u, event)
}

func (u *WebUI) Output(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := event.Payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebUI) Error(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *WebUI) Status(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebUI) RequestConfirmation(event eventbus.Event) (response eventbus.Event, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
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

	return
}

func (u *WebUI) FileNotification(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	u.broadcast(event)
	return nil
}

func (u *WebUI) Shutdown(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.broadcast(event)
	return nil
}

func (w *WebUI) broadcast(message interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for client := range w.clients {
		err := client.WriteJSON(message)
		if err != nil {
			client.Close()
			delete(w.clients, client)
		}
	}
}

func (w *WebUI) HandleWebSocketEndpoint(wr http.ResponseWriter, r *http.Request) {
	conn, err := w.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	w.mu.Lock()
	w.clients[conn] = true
	w.mu.Unlock()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			w.mu.Lock()
			delete(w.clients, conn)
			w.mu.Unlock()
			break
		}

		// Handle incoming messages, e.g., confirmations
		if msgType, ok := msg["type"].(string); ok && msgType == "confirmation_response" {
			// requestID, _ := msg["id"].(string)
			// approved, _ := msg["approved"].(bool)
			// Here you would notify the core logic about the confirmation
			// This could be via channels, callbacks, or an event bus
		}
	}
}
