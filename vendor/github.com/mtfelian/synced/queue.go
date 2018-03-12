package synced

import (
	"errors"
	"sync"
)

// Queue that is thread-safe
type Queue struct {
	queue []interface{}
	sync.Mutex
}

// NewQueue returns a new synced queue
func NewQueue() Queue { return Queue{[]interface{}{}, sync.Mutex{}} }

// Push pushed an object to a queue
func (q *Queue) Push(object interface{}) {
	q.Lock()
	q.queue = append(q.queue, object)
	q.Unlock()
}

// Len returns a queue current length
func (q *Queue) Len() int {
	q.Lock()
	defer q.Unlock()
	return len(q.queue)
}

// Pop returns an object from a queue
func (q *Queue) Pop() (interface{}, error) {
	q.Lock()
	defer q.Unlock()
	if len(q.queue) == 0 {
		return nil, errors.New("Очередь пуста")
	}
	popped := q.queue[0]
	q.queue = q.queue[1:]
	return popped, nil
}
