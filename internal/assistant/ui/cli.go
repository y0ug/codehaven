package ui

import (
	"bufio"
	"fmt"
	"strings"
	"sync"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
)

type UserInterface interface {
	Publish(eventbus.Event) error
	Output(eventbus.Event) error
	Error(eventbus.Event) error
	Status(eventbus.Event) error
	RequestConfirmation(eventbus.Event) (eventbus.Event, error)
	Shutdown(eventbus.Event) error
	FileNotification(eventbus.Event) error
}

type CliUI struct {
	bus *eventbus.EventBus
	mu  sync.Mutex
}

var _ UserInterface = &CliUI{}

// NewCliUI creates a new CLI UI
func NewCliUI(bus *eventbus.EventBus) *CliUI {
	cli := &CliUI{
		bus: bus,
	}
	if bus != nil {
		go cli.listenEvents()
	}
	return cli
}

func Publish(u UserInterface, event eventbus.Event) error {
	switch event.Type {
	case eventbus.EventOutput:
		return u.Output(event)
	case eventbus.EventError:
		return u.Error(event)
	case eventbus.EventStatusUpdate:
		return u.Status(event)
	case eventbus.EventUserInputRequest:
		// Response should be handle if they used Publish but how?
		_, err := u.RequestConfirmation(event)
		return err
	case eventbus.EventFileNotification:
		return u.FileNotification(event)
	case eventbus.EventShutdown:
		return u.Shutdown(event)
	}
	return nil
}

func (u *CliUI) Publish(event eventbus.Event) error {
	return Publish(u, event)
}

func (u *CliUI) Output(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	message, ok := event.Payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Print(message)
	return nil
}

func (u *CliUI) Error(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *CliUI) Status(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *CliUI) RequestConfirmation(event eventbus.Event) (response eventbus.Event, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	request, ok := event.Payload.(eventbus.UserInputRequest)
	if !ok {
		return response, fmt.Errorf("invalid payload type")
	}

	action := request.Action.(actions.UserInputAction)
	fmt.Printf("Confirmation [%s]: %s (y/n): ", request.ID, action.Message)
	var userResponse string
	_, err = fmt.Scanln(&userResponse)
	if err != nil {
		return
	}
	approved := userResponse == "y" || userResponse == "Y"

	response = eventbus.NewEvent(
		eventbus.EventUserInputResponse,
		eventbus.UserInputResponse{
			ID:       request.ID,
			Approved: approved,
		},
	)

	fmt.Printf("Response %t", approved)
	fmt.Printf("Response: %v\n", response)
	return
}

func (u *CliUI) FileNotification(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	payload, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("File FileOperation %s: %s\n", payload.Type, payload.Files)
	return nil
}

func (u *CliUI) Shutdown(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Printf("Shutdown\n")
	return nil
}

func (c *CliUI) listenEvents() {
	sub := c.bus.Subscribe(100)
	defer c.bus.Unsubscribe(sub)

	for evt := range sub {
		c.Publish(evt)
	}
}

func (c *CliUI) handleConfirmation(
	requestID, message string,
	scanner *bufio.Scanner,
) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		fmt.Printf("Confirmation [%s]: %s (y/n): ", requestID, message)
		if !scanner.Scan() {
			return false, fmt.Errorf("failed to read input")
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "y" || input == "Y" {
			return true, nil
		} else if input == "n" || input == "N" {
			return false, nil
		} else {
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}
