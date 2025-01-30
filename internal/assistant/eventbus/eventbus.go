package eventbus

import (
	"sync"

	"github.com/google/uuid"
)

type EventBus struct {
	subscribers     []chan Event
	funcSubscribers map[string]EventHandler
	middleware      []Middleware
	shutdownChan    chan struct{}
	mu              sync.RWMutex
	funcMu          sync.RWMutex
}

var (
	instance *EventBus
	once     sync.Once
)

type (
	Middleware   func(next EventHandler) EventHandler
	EventHandler func(Event) error
)

func GetEventBus() *EventBus {
	once.Do(func() {
		instance = &EventBus{
			shutdownChan: make(chan struct{}),
		}
	})
	return instance
}

func (b *EventBus) Subscribe(bufferSize int) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, bufferSize)
	b.subscribers = append(b.subscribers, ch)
	return ch
}

func (b *EventBus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		// if reflect.DeepEqual(sub, ch) {
		if (<-chan Event)(sub) == ch {
			b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Publish to channel subscribers
	for _, sub := range b.subscribers {
		select {
		case sub <- event:
		case <-b.shutdownChan:
			return
		default:
			// Handle channel overflow
		}
	}

	// Publish to function subscribers
	b.funcMu.RLock()
	defer b.funcMu.RUnlock()

	for id, handler := range b.funcSubscribers {
		go func(id string, h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic in subscriber
				}
			}()
			if err := h(event); err != nil {
				// Handle error from subscriber
			}
		}(id, handler)
	}
}

// SubscribeFunc registers a callback function to handle events
func (b *EventBus) SubscribeFunc(handler EventHandler) string {
	b.funcMu.Lock()
	defer b.funcMu.Unlock()

	// Apply middleware to the handler
	wrappedHandler := b.Process(handler)

	// Generate unique ID for this subscription
	id := uuid.New().String()

	if b.funcSubscribers == nil {
		b.funcSubscribers = make(map[string]EventHandler)
	}
	b.funcSubscribers[id] = wrappedHandler

	return id
}

// UnsubscribeFunc removes a callback subscription
func (b *EventBus) UnsubscribeFunc(id string) {
	b.funcMu.Lock()
	defer b.funcMu.Unlock()

	delete(b.funcSubscribers, id)
}

func (b *EventBus) Use(middleware ...Middleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.middleware = append(b.middleware, middleware...)
}

func (b *EventBus) Process(handler EventHandler) EventHandler {
	for i := len(b.middleware) - 1; i >= 0; i-- {
		handler = b.middleware[i](handler)
	}
	return handler
}

func (b *EventBus) Shutdown() {
	close(b.shutdownChan)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Close channel subscribers
	for _, sub := range b.subscribers {
		close(sub)
	}
	b.subscribers = nil

	// Clear function subscribers
	b.funcMu.Lock()
	defer b.funcMu.Unlock()
	b.funcSubscribers = nil
}
