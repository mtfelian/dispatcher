package counter

import (
	"sync"
)

// Synced counter
type Synced struct {
	count int
	sync.Mutex
}

// New returns a new synced counter initialized by initialValue
func New(initialValue int) Synced {
	return Synced{
		initialValue,
		sync.Mutex{},
	}
}

// Inc increases counter value by 1
func (c *Synced) Inc() {
	c.Lock()
	c.count++
	c.Unlock()
}

// Add adds i to a counter's value
func (c *Synced) Add(i int) {
	c.Lock()
	c.count += i
	c.Unlock()
}

// Dec decreases counter value by 1
func (c *Synced) Dec() {
	c.Lock()
	c.count--
	c.Unlock()
}

// Get returns current counter value
func (c *Synced) Get() int {
	c.Lock()
	defer c.Unlock()
	return c.count
}
