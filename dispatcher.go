package dispatcher

import (
	"sync"
	"time"

	"github.com/mtfelian/dispatcher/counter"
	"github.com/mtfelian/dispatcher/queue"
)

type (
	// Result is a testOutput data consisting of input and output data
	Result struct {
		In    interface{}
		Out   interface{}
		Error error
	}

	// channels used by dispatcher
	channels struct {
		// input is an input data
		input chan interface{}
		// result <- send here when worker stops
		result chan Result
		// stop <- send here to stop dispatcher
		stop chan bool
	}

	// TreatFunc is a func for treating generic input element
	TreatFunc func(element interface{}) (interface{}, error)

	// OnResultFunc is a func to perform with testOutput on task end
	OnResultFunc func(result Result)
)

// dispatcher
type Dispatcher struct {
	workers    counter.Synced
	maxWorkers int
	tasksDone  counter.Synced
	errorCount counter.Synced

	signals      channels
	signalsMutex sync.Mutex

	treatFunc TreatFunc

	onResultFunc  OnResultFunc
	onResultMutex sync.Mutex
}

// New returns new dispatcher with max workers
func New(max int, treatFunc TreatFunc, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		workers:    counter.New(0),
		maxWorkers: max,
		tasksDone:  counter.New(0),
		errorCount: counter.New(0),
		signals: channels{
			input:  make(chan interface{}),
			result: make(chan Result),
			stop:   make(chan bool),
		},
		treatFunc:    treatFunc,
		onResultFunc: onResultFunc,
	}
}

// _signals returns signals struct thread-safely
func (d *Dispatcher) _signals() *channels {
	d.signalsMutex.Lock()
	defer d.signalsMutex.Unlock()
	return &d.signals
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// ErrorCount returns an error count
func (d *Dispatcher) ErrorCount() int { return d.errorCount.Get() }

// Stop the dispatcher
func (d *Dispatcher) Stop() { d._signals().stop <- true }

// AddWork adds work input to dispatcher
func (d *Dispatcher) AddWork(data interface{}) { d._signals().input <- data }

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// WaitUntilNoTasks stop the dispatcher making checks for "0 tasks now" every period
func (d *Dispatcher) WaitUntilNoTasks(period time.Duration) {
	for {
		time.Sleep(period)
		if d.Workers() == 0 {
			break
		}
	}
	d.Stop()
}

// treat the element
func (d *Dispatcher) treat(element interface{}) {
	result, err := d.treatFunc(element)
	d._signals().result <- Result{In: element, Out: result, Error: err}
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *queue.Synced) {
	popped, err := q.Pop()
	if err != nil {
		d._signals().result <- Result{In: nil, Out: nil, Error: err}
		return
	}
	go d.treat(popped)
}

// onResult is called on result receving from the appropriate channel.
// a func treating the result called synchronized therefore to treat result is thread-safe
func (d *Dispatcher) onResult(result Result) {
	d.onResultMutex.Lock()
	d.onResultFunc(result)
	d.onResultMutex.Unlock()
}

// Run starts dispatching
func (d *Dispatcher) Run() {
	q := queue.New()
	for {
		select {
		case r := <-d._signals().result: // worker is done
			if r.Error != nil {
				d.errorCount.Inc()
			}
			go d.onResult(r)

			d.tasksDone.Inc()
			d.workers.Dec()
			// queue is not empty -- can add worker
			if q.Len() > 0 {
				d.workers.Inc()
				d.popTreat(&q)
			}
		case data := <-d._signals().input: // dispatcher receives input
			if d.workers.Get() >= d.maxWorkers {
				q.Push(data)
				continue
			}
			d.workers.Inc()

			// queue is not empty, FIFO - push, pop
			if q.Len() > 0 {
				q.Push(data)
				d.popTreat(&q)
				continue
			}

			// queue is empty, simply treat the url
			go d.treat(data)
		case <-d._signals().stop: // stop signal received
			// cleaning up, i think it should be implemented better
			t := time.NewTimer(time.Second)
			for {
				select {
				case <-d._signals().result:
				case <-t.C:
					break
				}
			}
			t = time.NewTimer(time.Second)
			for {
				select {
				case <-d._signals().input:
				case <-t.C:
					break
				}
			}
			return
		}
	}
}
