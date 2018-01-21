package dispatcher

import (
	"sync"
	"time"

	"github.com/mtfelian/dispatcher/counter"
	"github.com/mtfelian/dispatcher/queue"
)

type (
	// Result consists of an inputChan and output data
	Result struct {
		In    interface{}
		Out   interface{}
		Error error
	}

	// TreatFunc is a func for treating generic inputChan element
	TreatFunc func(element interface{}) (interface{}, error)

	// OnResultFunc is a func to perform with task resultChan when at task finish
	OnResultFunc func(result Result)
)

// Dispatcher implements a task-dispatching functionality
type Dispatcher struct {
	workers    counter.Synced
	maxWorkers int
	tasksDone  counter.Synced
	errorCount counter.Synced

	// inputChan is an input data channel
	inputChan chan interface{}
	// resultChan <- send here when worker stops
	resultChan chan Result
	// stopChan <- send here to stop dispatcher
	stopChan chan bool

	treatFunc TreatFunc

	onResultFunc  OnResultFunc
	onResultMutex sync.Mutex
}

// New returns new dispatcher with max workers
func New(max int, treatFunc TreatFunc, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		workers:      counter.New(0),
		maxWorkers:   max,
		tasksDone:    counter.New(0),
		errorCount:   counter.New(0),
		inputChan:    make(chan interface{}),
		resultChan:   make(chan Result),
		stopChan:     make(chan bool),
		treatFunc:    treatFunc,
		onResultFunc: onResultFunc,
	}
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// ErrorCount returns an error count
func (d *Dispatcher) ErrorCount() int { return d.errorCount.Get() }

// Stop the dispatcher
func (d *Dispatcher) Stop() { d.stopChan <- true }

// AddWork adds work inputChan to dispatcher
func (d *Dispatcher) AddWork(data interface{}) { d.inputChan <- data }

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// WaitUntilNoTasks stopChan the dispatcher making checks for "0 tasks now" every period
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
	d.resultChan <- Result{In: element, Out: result, Error: err}
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *queue.Synced) {
	popped, err := q.Pop()
	if err != nil {
		d.resultChan <- Result{In: nil, Out: nil, Error: err}
		return
	}
	go d.treat(popped)
}

// onResult is called on resultChan receiving from the appropriate channel.
// a func treating the resultChan called synchronized therefore to treat resultChan is a thread-safe operation
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
		case r := <-d.resultChan: // worker is done
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
		case data := <-d.inputChan: // dispatcher receives inputChan
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
		case <-d.stopChan: // stopChan signal received
			// cleaning up, i think it should be implemented better
			t := time.NewTimer(time.Second)
			for {
				select {
				case <-d.resultChan:
				case <-d.inputChan:
				case <-t.C:
					return
				}
			}
		}
	}
}
