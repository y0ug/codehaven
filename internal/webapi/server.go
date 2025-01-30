package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/y0ug/codehaven/internal/assistant"
	"github.com/y0ug/codehaven/internal/assistant/eventbus"
	"github.com/y0ug/codehaven/internal/assistant/ui"
)

type WebServer struct {
	assistant           assistant.Assister
	eventBus            *eventbus.EventBus
	upgrader            websocket.Upgrader
	clients             sync.Map     // thread-safe map for websocket clients
	server              *http.Server // Add HTTP server reference
	shutdown            chan struct{}
	confirmationManager *ui.ConfirmationManager
}

func NewWebServer(assistant assistant.Assister, bus *eventbus.EventBus) *WebServer {
	return &WebServer{
		assistant: assistant,
		eventBus:  bus,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure as needed
			},
		},
		shutdown:            make(chan struct{}),
		confirmationManager: ui.NewConfirmationManager(bus),
	}
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}

func (s *WebServer) Start(addr string) error {
	// Using Go 1.22 routing patterns
	mux := http.NewServeMux()

	// Serve static files
	fsys, err := getFileSystem()
	if err != nil {
		return err
	}

	// Serve the frontend at root
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			content, err := fs.ReadFile(fsys, "index.html")
			if err != nil {
				http.Error(w, "Could not load frontend", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
			return
		}
		http.FileServer(http.FS(fsys)).ServeHTTP(w, r)
	})

	// Chat endpoints
	mux.HandleFunc("POST /api/chat", s.handleChat)

	// User confirmation
	mux.HandleFunc("POST /api/confirmation", s.handleConfirmation)

	// File management
	mux.HandleFunc("POST /api/files", s.handleAddFiles)
	mux.HandleFunc("DELETE /api/files/{filename}", s.handleRemoveFile)
	mux.HandleFunc("GET /api/files", s.handleListFiles)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	// Start the server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.eventBus.SubscribeFunc(func(event eventbus.Event) error {
		switch event.Type {
		case eventbus.EventShutdown:
			s.Shutdow()
		}
		return nil
	})
	// Channel for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-s.shutdown:
		return s.performCleanShutdown()
	}
}

func (s *WebServer) performCleanShutdown() error {
	fmt.Println("\nShutting down gracefully the api...")
	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Notify clients of shutdown
	s.notifyClientsOfShutdown()

	// Close all WebSocket connections
	s.closeAllWebSocketConnections()

	// Shutdown the HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return nil
}

func (s *WebServer) notifyClientsOfShutdown() {
	s.clients.Range(func(key, value interface{}) bool {
		if client, ok := key.(*WSClient); ok {
			// Send shutdown message to client
			shutdownMsg := []byte(`{"type":"shutdown","message":"Server is shutting down"}`)
			client.send <- shutdownMsg
		}
		return true
	})
}

func (s *WebServer) closeAllWebSocketConnections() {
	s.clients.Range(func(key, value interface{}) bool {
		if client, ok := key.(*WSClient); ok {
			close(client.send)
			client.conn.Close()
		}
		return true
	})
}

// Shutdown initiates server shutdown
func (s *WebServer) Shutdow() {
	close(s.shutdown)
}

func generateRequestID() string {
	return uuid.New().String()
}

func (s *WebServer) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Add request ID
		ctx := context.WithValue(r.Context(), "requestID", generateRequestID())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *WebServer) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a channel to receive the response
	// responseChan := make(chan string, 1)

	// Subscribe to output events
	sub := s.eventBus.Subscribe(100)
	defer s.eventBus.Unsubscribe(sub)

	// Publish input event
	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventInput,
		eventbus.UserInput{
			Source:  "api",
			Content: req.Message,
		},
	))

	// Wait for response with timeout
	select {
	case event := <-sub:
		switch event.Type {
		case eventbus.EventOutput:
			if output, ok := event.Payload.(string); ok {
				json.NewEncoder(w).Encode(ChatResponse{
					Response: output,
					Status:   s.assistant.GetStatus().Current(),
				})
				return
			}
		}

	case <-time.After(30 * time.Second):
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}
}

func (s *WebServer) handleAddFiles(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	fileNames := make([]string, 0, len(files))

	for _, fileHeader := range files {
		// Save file to temp location
		fileName := fileHeader.Filename
		fileNames = append(fileNames, fileName)
	}

	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventAddFile,
		eventbus.FileOperation{
			Files: fileNames,
		},
	))

	w.WriteHeader(http.StatusOK)
}

func (s *WebServer) handleRemoveFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")

	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventRemoveFile,
		eventbus.FileOperation{
			Files: []string{filename},
		},
	))

	w.WriteHeader(http.StatusOK)
}

func (s *WebServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files := s.assistant.GetRM().ListFiles(0)
	json.NewEncoder(w).Encode(files)
}

// WebSocket handling
type WSClient struct {
	conn      *websocket.Conn
	server    *WebServer
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func (s *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WSClient{
		conn:      conn,
		server:    s,
		send:      make(chan []byte, 256),
		done:      make(chan struct{}),
		closeOnce: sync.Once{},
	}

	// Store client
	s.clients.Store(client, struct{}{})

	// Start client routines
	go client.writePump()
	go client.readPump()

	// Subscribe to events
	sub := s.eventBus.Subscribe(100)
	go func() {
		defer s.eventBus.Unsubscribe(sub)
		for event := range sub {
			select {
			case <-client.done:
				return
			default:
				// Convert event to JSON and send to client
				if data, err := json.Marshal(event); err == nil {
					client.send <- data
				}
			}
		}
	}()

	go func() {
		// Listen for confirmation notifications
		for {
			select {
			case <-client.done:
				return
			case confirmation, ok := <-s.confirmationManager.Notifications():
				if !ok {
					return
				}
				msg := struct {
					Type    string      `json:"Type"`
					Payload interface{} `json:"Payload"`
				}{
					Type:    "confirmation_request",
					Payload: confirmation,
				}
				if data, err := json.Marshal(msg); err == nil {
					select {
					case <-client.done:
						return
					case client.send <- data:
					}
				}
			}
		}
	}()
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeOnce.Do(func() {
			close(c.done)
		})
		c.conn.Close()
		c.server.clients.Delete(c)
	}()

	// Add done channel for shutdown
	done := make(chan struct{})
	go func() {
		select {
		case <-c.server.shutdown:
			close(done)
		}
	}()

	for {
		select {
		case <-done:
			return
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) readPump() {
	defer func() {
		c.conn.Close()
		c.closeOnce.Do(func() {
			close(c.done)
		})
		c.server.clients.Delete(c)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Add done channel for shutdown
	done := make(chan struct{})
	go func() {
		select {
		case <-c.server.shutdown:
			close(done)
		}
	}()

	for {
		select {
		case <-done:
			return
		default:
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
				) {
					// c.server.logger.Error("websocket error", "error", err)
				}
				return
			} // Handle incoming WebSocket messages
			var input struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}

			if err := json.Unmarshal(message, &input); err != nil {
				continue
			}

			switch input.Type {
			case "chat":
				var chatReq ChatRequest
				if err := json.Unmarshal(input.Payload, &chatReq); err != nil {
					continue
				}
				c.server.eventBus.Publish(eventbus.NewEvent(
					eventbus.EventInput,
					eventbus.UserInput{
						Source:  "websocket",
						Content: chatReq.Message,
					},
				))
			case "user_response":
				var response eventbus.UserInputResponse
				if err := json.Unmarshal(input.Payload, &response); err != nil {
					continue
				}
				c.server.eventBus.Publish(eventbus.NewEvent(
					eventbus.EventUserInputResponse,
					response,
				))
			}
		}
	}
}

func (s *WebServer) handleConfirmation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Approved bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.confirmationManager.RespondToConfirmation(req.ID, req.Approved); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
}
