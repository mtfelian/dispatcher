package queue

import (
	"errors"
	"sync"
)

// Synced queue
type Synced struct {
	queue []interface{}
	sync.Mutex
}

// New returns a new synced queue
func New() Synced {
	return Synced{
		[]interface{}{},
		sync.Mutex{},
	}
}

// Push pushed an object to a queue
func (q *Synced) Push(object interface{}) {
	q.Lock()
	q.queue = append(q.queue, object)
	q.Unlock()
}

// Len returns a queue current length
func (q *Synced) Len() int {
	q.Lock()
	defer q.Unlock()
	return len(q.queue)
}

// Pop returns an object from a queue
func (q *Synced) Pop() (interface{}, error) {
	q.Lock()
	defer q.Unlock()
	if len(q.queue) == 0 {
		return nil, errors.New("Очередь пуста")
	}
	popped := q.queue[0]
	q.queue = q.queue[1:]
	return popped, nil
}
