package dispatcher

import (
	"sync"
	"time"

	"github.com/mtfelian/synced"
)

type (
	// Result consists of an input and output data
	Result struct {
		In    interface{}
		Out   interface{}
		Error error
	}

	// TreatFunc is a func for treating generic input element
	TreatFunc func(element interface{}) (interface{}, error)

	// OnResultFunc is a func to do something with task result on it's finish
	OnResultFunc func(result Result)
)

// Dispatcher implements a task-dispatching functionality
type Dispatcher struct {
	taskQueue synced.Queue

	workers    synced.Counter
	maxWorkers int
	tasksDone  synced.Counter
	errorCount synced.Counter

	// inputChan is an input data channel
	inputChan chan interface{}
	// resultChan <- send here when worker stops
	resultChan chan Result
	// stopChan <- send here to stop dispatcher
	stopChan      chan bool
	stopping      bool
	stoppingMutex sync.Mutex

	treatFunc TreatFunc

	onResultFunc  OnResultFunc
	onResultMutex sync.Mutex
}

// New returns new dispatcher with max workers
func New(max int, treatFunc TreatFunc, onResultFunc OnResultFunc) *Dispatcher {
	return &Dispatcher{
		taskQueue:    synced.NewQueue(),
		workers:      synced.NewCounter(0),
		maxWorkers:   max,
		tasksDone:    synced.NewCounter(0),
		errorCount:   synced.NewCounter(0),
		inputChan:    make(chan interface{}),
		resultChan:   make(chan Result),
		stopChan:     make(chan bool),
		treatFunc:    treatFunc,
		onResultFunc: onResultFunc,
	}
}

// setStopping sets state to stopping
func (d *Dispatcher) setStopping() {
	d.stoppingMutex.Lock()
	d.stopping = true
	d.stoppingMutex.Unlock()
}

// isStopping returns whether the dispatcher is stopping
func (d *Dispatcher) isStopping() bool {
	d.stoppingMutex.Lock()
	defer d.stoppingMutex.Unlock()
	return d.stopping
}

// Workers returns current worker count
func (d *Dispatcher) Workers() int { return d.workers.Get() }

// SetMaxWorkers to n
func (d *Dispatcher) SetMaxWorkers(n int) { d.maxWorkers = n }

// ErrorCount returns an error count
func (d *Dispatcher) ErrorCount() int { return d.errorCount.Get() }

// Stop the dispatcher
func (d *Dispatcher) Stop() {
	d.stopChan <- true
	close(d.stopChan)
}

// AddWork adds work to the dispatcher
func (d *Dispatcher) AddWork(data interface{}) {
	if !d.isStopping() {
		d.inputChan <- data
	}
}

// TasksDone returns done tasks count
func (d *Dispatcher) TasksDone() int { return d.tasksDone.Get() }

// WaitUntilNoTasks stops the dispatcher making checks for "0 tasks now" every period
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
	d.sendResult(Result{In: element, Out: result, Error: err})
}

// sendResult to results channel
func (d *Dispatcher) sendResult(result Result) {
	if !d.isStopping() {
		d.resultChan <- result
	}
}

// popTreat pops an element from queue q and treats it
func (d *Dispatcher) popTreat(q *synced.Queue) {
	popped, err := q.Pop()
	if err != nil {
		d.sendResult(Result{In: nil, Out: nil, Error: err})
		return
	}
	go d.treat(popped)
}

// onResult is called on result receiving from the appropriate channel.
// a func treating the result called synchronized, therefore to treat result is a thread-safe operation
func (d *Dispatcher) onResult(result Result) {
	d.onResultMutex.Lock()
	d.onResultFunc(result)
	d.onResultMutex.Unlock()
}

// QueueLen returns current length of the queue
func (d *Dispatcher) QueueLen() int { return d.taskQueue.Len() }

// Run starts dispatching
func (d *Dispatcher) Run() {
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
			if d.taskQueue.Len() > 0 {
				d.workers.Inc()
				d.popTreat(&d.taskQueue)
			}
		case data := <-d.inputChan: // dispatcher receives some data
			if d.workers.Get() >= d.maxWorkers {
				d.taskQueue.Push(data)
				continue
			}
			d.workers.Inc()

			// queue is not empty, FIFO - push, pop
			if d.taskQueue.Len() > 0 {
				d.taskQueue.Push(data)
				d.popTreat(&d.taskQueue)
				continue
			}

			// queue is empty, simply treat the url
			go d.treat(data)
		case <-d.stopChan: // stop signal received
			d.setStopping()
			// cleaning up, may be
			t := time.NewTicker(100 * time.Millisecond)
			for {
				select {
				case <-d.resultChan:
				case <-d.inputChan:
				case <-t.C:
					if d.Workers() == 0 {
						t.Stop()
						return
					}
				}
			}
		}
	}
}
