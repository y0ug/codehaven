package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/y0ug/codehaven/internal/assistant"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/assistant/llm"
	modelinfocoder "github.com/y0ug/codehaven/internal/assistant/llm/models"
	"github.com/y0ug/codehaven/internal/assistant/prompt/prompts"
	"github.com/y0ug/codehaven/internal/assistant/repomanager"
	"github.com/y0ug/codehaven/internal/assistant/settings"
	"github.com/y0ug/codehaven/internal/assistant/ui"
	"github.com/y0ug/codehaven/internal/consolecoder"
	"github.com/y0ug/codehaven/internal/filemanager"
	"github.com/y0ug/codehaven/internal/webapi"
	"github.com/y0ug/codehaven/internal/gitrepo"
	"github.com/y0ug/codehaven/internal/highlighter"
	"github.com/y0ug/llmhaven"
	"github.com/y0ug/llmhaven/chat"
	"github.com/y0ug/llmhaven/http/options"
	"github.com/y0ug/llmhaven/modelinfo"
)

func main() {
	ctx := context.Background()
	verbose := flag.Bool("v", false, "Show verbose output")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	}))

	// Setup model info provider
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "codehaven")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Error creating config directory", "error", err)
		os.Exit(1)
	}

	// Setup the model info provider with a cache json file
	infoProviderCacheFile := filepath.Join(configDir, "provider_cache.json")
	modelInfoProvider, err := modelinfo.New(ctx, infoProviderCacheFile)
	if err != nil {
		logger.Error("Error creating model info providers", "error", err)
		os.Exit(1)
	}

	// Get settings from environment variables
	model := os.Getenv("AI_MODEL")
	if model == "" {
		logger.Error("AI_MODEL environment variable not set")
		os.Exit(1)
	}

	promptName := os.Getenv("AI_PROMPT")
	if promptName == "" {
		promptName = "EditBlock"
	}

	// Find model info and provider from the model string
	modelInfo, err := modelinfo.Get(model, modelInfoProvider)
	if err != nil {
		logger.Error("Error parsing model info", "error", err)
		os.Exit(1)
	}

	// Define the model with the settings need for the assistant
	modelCoder, err := modelinfocoder.NewModel(*modelInfo, nil, nil, "")
	if err != nil {
		logger.Error("Error creating model coder", "error", err)
		os.Exit(1)
	}

	fm := filemanager.NewLocalFileManager()
	// Setup chat parameters
	_ = chat.NewChatParams()
	rootPath, err := os.Getwd()
	if err != nil {
		logger.Error("Error getting current working directory", "error", err)
		os.Exit(1)
	}
	logger.Info("rootPath", "rootPath", rootPath)
	requestOpts := []options.RequestOption{
		// options.WithMiddleware(middleware.LoggingMiddleware()),
		// options.WithMiddleware(middleware.TimeitMiddleware(logger)),
	}
	llmProvider, err := llmhaven.New(modelInfo.Provider, requestOpts...)
	if err != nil {
		logger.Error("Error creating llm client", "error", err)
		os.Exit(1)
	}

	coderSettings := settings.NewCoderSettings(modelCoder)

	llm := llm.New(
		llmProvider,
		coderSettings,
		logger,
	)

	gitRepo, err := gitrepo.NewGitRepo(logger, nil, rootPath)
	if err != nil {
		logger.Error("Error creating git repo", "error", err)
	}

	rm := repomanager.NewRepoManager(rootPath, logger, fm, gitRepo, llm.SendMessages)

	repomap := rm.GenerateRepoMap()
	fmt.Println("rm", repomap.String())

	pts := prompts.New(promptName)
	if pts == nil {
		logger.Error("Error creating prompts", "prompt_name", promptName)
		os.Exit(1)
	}

	h := highlighter.NewHighlighter(os.Stdout)

	// Capture Ctrl-C (SIGINT)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	eventBus := eventbus.GetEventBus()

	// ui := ui.NewCliUI(eventBus)
	ui := ui.NewEventBusUI(eventBus)
	coderOpts := assistant.AssistantOptions{
		Logger:      logger,
		Llm:         llm,
		RepoManager: rm,
		Prompts:     pts,
		Settings:    coderSettings,
		Stream:      true,
		EventBus:    eventBus,
		UI:          ui,
	}
	coder := assistant.NewAssistantOrchestrator(coderOpts)

	// // Create and start web server
	server := webapi.NewWebServer(coder, eventBus)

	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}
	}()

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		eventBus.Publish(eventbus.NewEvent(eventbus.EventShutdown, nil))
		// Shut down or let it process the event?
		// server.Shutdown()
	}()

	// startCLI(ctx, coder)
	console := consolecoder.New(coder, h, eventBus)
	console.Run()
}

func startCLI(ctx context.Context, orchestrator *assistant.AssistantOrchestrator) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		input = strings.Trim(input, " \r\n\t")
		if input == "" {
			continue
		}
		args := strings.Split(input, " ")
		switch args[0] {
		case "/dump":
			fmt.Println(orchestrator.DumpActionChain())
		case "/add":
			files := args[1:]
			orchestrator.AddFiles(ctx, false, files...)
		case "/remove":
			files := args[1:]
			orchestrator.RemoveFiles(ctx, files...)
		default:
			orchestrator.InputEvent(
				ctx,
				eventbus.NewEvent(
					eventbus.EventInput,
					eventbus.UserInput{Source: "console", Content: input},
				),
			)
		}
	}
}
