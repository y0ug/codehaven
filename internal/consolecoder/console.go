package consolecoder

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/y0ug/codehaven/internal/assistant"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/assistant/ui"
	"github.com/y0ug/codehaven/internal/highlighter"
)

type Console struct {
	coder               *assistant.AssistantOrchestrator
	h                   *highlighter.Highlighter
	commands            map[string]Command
	eventBus            *eventbus.EventBus
	status              *ui.StatusManager
	input               *ui.InputHandler
	pt                  *prompt.Prompt
	historyFile         string
	confirmationManager *ui.ConfirmationManager
}

type ConsoleConfirmation struct {
	ID      string
	Message string
	Time    time.Time
}

func getHistoryFilePath() string {
	usr, err := user.Current()
	if err != nil {
		return ".codehaven-history"
	}
	return filepath.Join(usr.HomeDir, ".ai-coder-history")
}

func New(
	coder *assistant.AssistantOrchestrator,
	h *highlighter.Highlighter,
	bus *eventbus.EventBus,
) *Console {
	output := ui.NewOutputHandler(h)
	input := ui.NewInputHandler()
	confirmationManager := ui.NewConfirmationManager(bus)

	c := &Console{
		coder:               coder,
		h:                   h,
		historyFile:         getHistoryFilePath(),
		eventBus:            bus,
		status:              coder.GetStatus(),
		input:               input,
		confirmationManager: confirmationManager,
	}

	c.setCommands()

	c.pt = prompt.New(
		c.executor,
		c.completer,
		prompt.OptionLivePrefix(c.UpdatePrompt),
		prompt.OptionTitle("Chat"),
		prompt.OptionPrefix(fmt.Sprintf(" ➜ ")),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionPrefixTextColor(prompt.Blue),
		prompt.OptionMaxSuggestion(5),
		prompt.OptionHistory(c.loadHistory()),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn:  func(*prompt.Buffer) { c.shutdown() },
		}),
	)

	// Wire components
	bus.Use(
	// ui.LoggingMiddleware(coder.logger),
	// ui.ErrorHandlingMiddleware(),
	)

	// Start subsystems
	output.Start()
	input.Start()

	c.registerEventHandlers()
	return c
}

func (c *Console) registerEventHandlers() {
	sub := c.eventBus.Subscribe(100)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for event := range sub {
			switch event.Type {
			case eventbus.EventFileNotification:
				c.handleFileNotification(ctx, event)
			}
		}
	}()

	go func() {
		for confirmation := range c.confirmationManager.Notifications() {
			fmt.Printf("\nNew confirmation request [%s]: %s\n",
				confirmation.ID[:8],
				confirmation.Message)
			c.printPrompt() // Reprint the prompt
		}
	}()
}

func (c *Console) handleConfirmCommand(args []string) {
	if len(args) != 2 {
		fmt.Println("Usage: /confirm <id> <y/n>")
		return
	}

	id := args[0]
	response := strings.ToLower(args[1])
	approved := response == "y"

	if err := c.confirmationManager.RespondToConfirmation(id, approved); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Responded to confirmation %s: %v\n", id, approved)
}

func (c *Console) handlePendingCommand(args []string) {
	fmt.Println("\nPending confirmations:")
	count := 0
	for _, confirmation := range c.confirmationManager.GetPendingConfirmations() {
		fmt.Printf("[%s] %s (received: %s)\n",
			confirmation.ID[:8],
			confirmation.Message,
			confirmation.Time.Format("15:04:05"),
		)
		count++
	}

	if count == 0 {
		fmt.Println("No pending confirmations")
	}
}

func (c *Console) printPrompt() {
	p, _ := c.UpdatePrompt()
	fmt.Printf("%s ", p)
}

func (c *Console) handleFileNotification(ctx context.Context, event eventbus.Event) {
	fileOp := event.Payload.(eventbus.FileOperation)
	switch fileOp.Type {
	case eventbus.FileOperationTypeAdd:
		fmt.Printf("Added file: %s\n", fileOp.Files)
	case eventbus.FileOperationTypeRemove:
		fmt.Printf("Remove file: %s\n", fileOp.Files)
	}
}

func (c *Console) UpdatePrompt() (string, bool) {
	// status := <-c.statusChan
	return fmt.Sprintf("[%s]  ➜ ", c.status.Current()), true
}

func (c *Console) Run() {
	c.pt.Run()
}

func (c *Console) handleHelp(args []string) {
	fmt.Println("Available commands:")
	for _, cmd := range c.commands {
		fmt.Printf("/%s - %s\n", cmd.name, cmd.description)
	}
}

func (c *Console) shutdown() {
	fmt.Println("\nShutting down gracefully...")
	c.eventBus.Publish(eventbus.NewEvent(eventbus.EventShutdown, nil))
	c.eventBus.Shutdown()
	os.Exit(0)
}

//	func (c *Console) handleAddFile(args []string) {
//		if len(args) == 0 {
//			fmt.Println("Please specify file(s) to add")
//			return
//		}
//
//		for _, file := range args {
//			err := c.coder.GetRM().GetFM().Add(file, false)
//			if err != nil {
//				fmt.Printf("Error loading file %s: %v\n", file, err)
//				continue
//			}
//			fmt.Printf("Added file: %s\n", file)
//		}
//	}
//
//	func (c *Console) handleRemoveFile(args []string) {
//		if len(args) == 0 {
//			fmt.Println("Please specify file(s) to remove")
//			return
//		}
//
//		for _, file := range args {
//			fmt.Printf("Removed file: %s\n", file)
//			err := c.coder.GetRM().GetFM().Remove(file)
//			if err != nil {
//				fmt.Printf("Error removing file %s: %v\n", file, err)
//			}
//		}
//	}

func (c *Console) handleAddFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to add")
		return
	}
	c.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventAddFile,
		eventbus.FileOperation{Files: args},
	))
}

func (c *Console) handleRemoveFile(args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify file(s) to remove")
		return
	}
	c.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventRemoveFile,
		eventbus.FileOperation{Files: args},
	))
}

func (c *Console) handleListFiles(args []string) {
	files := c.coder.GetRM().ListFiles(0)
	if len(files) == 0 {
		fmt.Println("No files currently attached")
		return
	}
	fmt.Println("Currently attached files:")
	for fileName, file := range files {
		fmt.Printf(
			"- %s %s (%t) %s\n",
			fileName,
			file.LastUpdate,
			file.ReadOnly,
			string(file.Status.String()),
		)
	}
}

func (c *Console) send(args []string) {
}

func (c *Console) executor(input string) {
	input = strings.TrimSpace(input)

	if input == "" {
		return
	}

	// Save to history
	c.appendHistory(input)

	// Handle commands
	if strings.HasPrefix(input, "/") {
		parts := strings.Fields(input)
		cmd, exists := c.commands[parts[0]]
		if exists {
			cmd.handler(parts[1:])
			return
		}
		fmt.Printf("Unknown command: %s\n", parts[0])
		return
	}

	// Publish input event instead of direct channel access
	c.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventInput,
		eventbus.UserInput{
			Source:  "console",
			Content: input,
		},
	))
}
