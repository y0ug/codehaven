package queue

import (
	"sync"

	"github.com/y0ug/codehaven/internal/assistant/actions"
)

type (
	Queuer[T any] interface {
		Enqueue(actions ...T)
		Dequeue() (T, bool)
		IsEmpty() bool
		Len() int
	}
	ActionQueuer = Queuer[actions.Action]
	ActionQueue  struct {
		mu    sync.RWMutex
		items []actions.Action
		cond  *sync.Cond
	}
)

func NewActionQueue() ActionQueuer {
	q := &ActionQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *ActionQueue) Enqueue(actions ...actions.Action) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(q.items, actions...)
	q.cond.Broadcast()
}

func (q *ActionQueue) Dequeue() (actions.Action, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	action := q.items[0]
	q.items = q.items[1:]
	return action, true
}

func (q *ActionQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

func (q *ActionQueue) IsEmpty() bool {
	return q.Len() == 0
}
