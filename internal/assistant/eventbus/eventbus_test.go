package eventbus

import (
	"fmt"
	"testing"
	"time"
)

func TestEventBusPublishing(t *testing.T) {
	bus := NewTestableEventBus()
	sub := bus.Subscribe(10)

	expected := NewEvent(EventInput, "test")
	bus.Publish(expected)

	select {
	case received := <-sub:
		fmt.Println(received)
		// if received != expected {
		// 	t.Errorf("Expected %v, got %v", expected, received)
		// }
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for event")
	}
}

func TestEventHistoryRecording(t *testing.T) {
	bus := NewTestableEventBus()
	bus.Publish(NewEvent(EventStatusUpdate, "ready"))

	history := bus.GetHistory()
	if len(history) != 1 {
		t.Errorf("Expected 1 recorded event, got %d", len(history))
	}
}
