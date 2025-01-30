package eventbus

import (
	"sync"
	"time"
)

type TestableEventBus struct {
	*EventBus
	history     []Event
	historyLock sync.Mutex
}

func NewTestableEventBus() *TestableEventBus {
	return &TestableEventBus{
		EventBus: GetEventBus(),
	}
}

func (b *TestableEventBus) RecordEvent(event Event) {
	b.historyLock.Lock()
	defer b.historyLock.Unlock()
	b.history = append(b.history, event)
}

func (b *TestableEventBus) GetHistory() []Event {
	b.historyLock.Lock()
	defer b.historyLock.Unlock()
	return append([]Event{}, b.history...)
}

func (b *TestableEventBus) WaitForEventType(t EventType, timeout time.Duration) bool {
	sub := b.Subscribe(10)
	defer b.Unsubscribe(sub)

	deadline := time.After(timeout)
	for {
		select {
		case event := <-sub:
			if event.Type == t {
				return true
			}
		case <-deadline:
			return false
		}
	}
}
