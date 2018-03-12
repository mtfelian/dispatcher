package synced

import "sync"

// Counter that is thread-safe
type Counter struct {
	count int
	sync.Mutex
}

// NewCounter returns a new synced counter initialized by initialValue
func NewCounter(initialValue int) Counter { return Counter{initialValue, sync.Mutex{}} }

// Inc increases counter by 1
func (c *Counter) Inc() {
	c.Lock()
	c.count++
	c.Unlock()
}

// Add i to counter
func (c *Counter) Add(i int) {
	c.Lock()
	c.count += i
	c.Unlock()
}

// Dec decreases counter by 1
func (c *Counter) Dec() {
	c.Lock()
	c.count--
	c.Unlock()
}

// Set counter to i
func (c *Counter) Set(i int) {
	c.Lock()
	c.count = i
	c.Unlock()
}

// Get returns current counter value
func (c *Counter) Get() int {
	c.Lock()
	defer c.Unlock()
	return c.count
}
